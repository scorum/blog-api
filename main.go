package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/jinzhu/configor"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/mongodb/mongo-go-driver/core/connstring"
	"github.com/scorum/event-provider-go/provider"
	"github.com/scorum/scorum-go"
	bct "github.com/scorum/scorum-go/transport/http"
	log "github.com/sirupsen/logrus"
	"gitlab.scorum.com/blog/api/blob"
	"gitlab.scorum.com/blog/api/blockchain_monitor"
	"gitlab.scorum.com/blog/api/broadcast"
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/domainprovider"
	"gitlab.scorum.com/blog/api/mailer"
	"gitlab.scorum.com/blog/api/push"
	"gitlab.scorum.com/blog/api/rpc"
	"gitlab.scorum.com/blog/api/service"
	"gitlab.scorum.com/blog/core/locale"
	"gitlab.scorum.com/blog/core/sentry"
)

// version is set via `go build -ldflags "-X main.version=version"`
var version string
var versionFlag = flag.Bool("version", false, "app version")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")

type BlockchainConfig struct {
	HTTP         string
	SyncInterval time.Duration `yaml:"sync_interval"`
	// Blockchain chain id
	ChainID string `yaml:"chain_id"`
}

type RouterConfig struct {
	MaxRequestSize int64 `yaml:"max_request_size" default:"25000000"` // 25 Megabytes
}

type DBConfig struct {
	Write string
	Read  string
}

type Config struct {
	LogLevel                 string `yaml:"log_level"`
	DB                       DBConfig
	Port                     string
	Blockchain               BlockchainConfig
	Blob                     blob.Config
	Sentry                   string
	Router                   RouterConfig
	BlockchainMonitorEnabled bool `yaml:"blockchain_monitor_enabled"`
	Firebase                 push.Config
	Service                  service.Config
	// AuthConnection is a connection string to the mongo Auth db
	AuthConnection   string `yaml:"auth_connection" required:"true"`
	TextRuKey        string `yaml:"text_ru_key"`
	LocalizerBaseUrl string `yaml:"localizer_base_url"`
	NSQDAddress      string `yaml:"nsqd_address"`
}

func main() {
	// app binary version
	flag.Parse()
	if *versionFlag {
		fmt.Println(version)
		return
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	// only log the warning severity or above.
	log.SetLevel(log.WarnLevel)

	// load config
	var config Config
	if err := configor.Load(&config, "config.yml"); err != nil {
		log.Fatal(err)
	}

	if logLevel, err := log.ParseLevel(config.LogLevel); err == nil {
		log.SetLevel(logLevel)
	}

	if config.Service.Admin == "" {
		log.Fatal("admin account not present in config")
	}

	hook, err := sentry.NewHook(config.Sentry)
	if err != nil {
		log.Fatal(err)
	}
	log.AddHook(hook)

	// open dbWrite connection
	dbWrite, err := sqlx.Open("postgres", config.DB.Write)
	if err != nil {
		log.Fatal(err)
	}
	dbWrite.SetMaxIdleConns(10)
	dbWrite.SetMaxOpenConns(10)

	// open dbRead connection
	dbRead, err := sqlx.Open("postgres", config.DB.Read)
	if err != nil {
		log.Fatal(err)
	}
	dbRead.SetMaxIdleConns(10)
	dbRead.SetMaxOpenConns(10)

	transport := bct.NewTransport(config.Blockchain.HTTP)
	blockchain := scorumgo.NewClient(transport)

	// health
	http.HandleFunc("/health", func(writer http.ResponseWriter, _ *http.Request) {
		if err := dbRead.Ping(); err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		writer.Write([]byte("ok"))
	})

	// version
	http.HandleFunc("/version", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Write([]byte(version))
	})

	// application context
	ctx, cancel := context.WithCancel(context.Background())

	// anti plagiarism service
	antiPlagiarism := service.NewAntiPlagiarismService(
		config.TextRuKey,
		db.NewPlagiarismStorage(dbWrite),
		db.NewCommentsStorage(dbRead),
	)

	localizer := locale.NewLocalizer(config.LocalizerBaseUrl)
	localizer.Reload()
	// reload localization once in an hour
	go func() {
		ticker := time.NewTicker(time.Hour)
		for range ticker.C {

			localizer.Reload()
		}
	}()

	// pusher
	pusher, err := push.NewPusher(config.Firebase)
	if err != nil {
		log.Fatalf("failed to create pusher: %s", err)
	}

	cs, err := connstring.Parse(config.AuthConnection)
	if err != nil {
		log.Fatalf("parse auth conn string err: %s", err)
	}

	domainProvider, err := domainprovider.NewDomainProvider(cs)
	if err != nil {
		log.Fatalf("failed to create domain provider: %s", err)
	}

	// push notifier
	notifier := push.NewNotifier(
		pusher, localizer, domainProvider,
		db.NewPushTokensStorage(dbRead))

	blog := &service.Blog{
		DB: service.Database{
			Write: dbWrite,
			Read:  dbRead,
		},
		Blockchain:              blockchain,
		Blob:                    blob.NewService(config.Blob),
		Config:                  config.Service,
		Notifier:                notifier,
		PushRegistrationStorage: db.NewPushTokensStorage(dbWrite),
		NotificationStorage:     db.NewNotificationsStorage(dbWrite),
		DownvotesStorage:        db.NewDownvotesStorage(dbWrite),
	}

	// rpc handler
	router := configureRPCRouter(&config, blockchain, blog, antiPlagiarism)
	http.HandleFunc("/", router.Handle)
	http.HandleFunc("/unsubscribe", blog.UnsubscribeEndpoint)

	if config.BlockchainMonitorEnabled {
		log.Info("blockchain monitor enabled")

		mailer, err := mailer.NewClient(config.NSQDAddress)
		if err != nil {
			log.Warnf("can't create nsq client err:%s", err)
		}

		monitor := blockchain_monitor.BlockchainMonitor{
			DB:              dbWrite,
			CommentsStorage: db.NewCommentsStorage(dbWrite),
			Provider: provider.NewProvider(config.Blockchain.HTTP,
				provider.SyncInterval(config.Blockchain.SyncInterval)),
			Plagiarism:          antiPlagiarism,
			NotificationStorage: db.NewNotificationsStorage(dbWrite),
			PushNotifier:        notifier,
			DownvotesStorage:    db.NewDownvotesStorage(dbWrite),
			PlagiarismStorage:   db.NewPlagiarismStorage(dbWrite),
			MailerClient:        mailer,
		}

		go monitor.Monitor(ctx)
	}

	// listen
	go func() {
		if err := http.ListenAndServe(":"+config.Port, nil); err != nil {
			log.Fatal(err)
		}
	}()

	terminate := make(chan os.Signal)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGTERM)
	s := <-terminate
	cancel()
	log.Infof("app closed with signal: %s", s)
}

func configureRPCRouter(config *Config, blockchain *scorumgo.Client, blog *service.Blog, ap *service.AntiPlagiarism) *rpc.Router {
	verifier := rpc.NewVerifier(config.Blockchain.ChainID)
	transactionRouter := broadcast.NewTransactionRouter(blockchain, verifier)

	// rpc routes
	rpcRouter := rpc.NewRouter(blockchain, verifier, config.Router.MaxRequestSize)
	rpcRouter.Register(rpc.Route{"account_api", "get_profile"}, blog.GetProfile)
	rpcRouter.Register(rpc.Route{"account_api", "get_profiles"}, blog.GetProfiles)
	rpcRouter.Register(rpc.Route{"account_api", "get_profile_settings"}, rpcRouter.SignedAPI(blog.GetProfileSettings))
	rpcRouter.Register(rpc.Route{"account_api", "is_trusted"}, blog.IsAccountTrusted)
	rpcRouter.Register(rpc.Route{"account_api", "get_trusted"}, blog.GetTrusted)
	rpcRouter.Register(rpc.Route{"media_api", "get_media"}, blog.GetMedia)
	rpcRouter.Register(rpc.Route{"category_api", "get_categories"}, blog.GetCategories)
	rpcRouter.Register(rpc.Route{"category_api", "get_category"}, blog.GetCategory)
	rpcRouter.Register(rpc.Route{"follow_api", "get_followers"}, blog.GetFollowers)
	rpcRouter.Register(rpc.Route{"follow_api", "get_following"}, blog.GetFollowing)
	rpcRouter.Register(rpc.Route{"follow_api", "filter_followers"}, blog.FilterFollowers)
	rpcRouter.Register(rpc.Route{"follow_api", "filter_following"}, blog.FilterFollowing)
	rpcRouter.Register(rpc.Route{"blacklist_api", "is_blacklisted"}, blog.IsBlacklisted)
	rpcRouter.Register(rpc.Route{"blacklist_api", "get_blacklist"}, blog.GetBlacklist)
	rpcRouter.Register(rpc.Route{"draft_api", "get_draft"}, rpcRouter.SignedAPI(blog.GetDraft))
	rpcRouter.Register(rpc.Route{"draft_api", "get_drafts"}, rpcRouter.SignedAPI(blog.GetDrafts))
	rpcRouter.Register(rpc.Route{"notification_api", "get_notifications"}, rpcRouter.SignedAPI(blog.GetNotifications))
	rpcRouter.Register(rpc.Route{"post_api", "is_post_deleted"}, blog.IsPostDeleted)
	rpcRouter.Register(rpc.Route{"post_api", "get_deleted_posts"}, blog.GetDeletedPosts)
	rpcRouter.Register(rpc.Route{"post_api", "get_votes"}, blog.GetVotesForPostEndpoint)
	rpcRouter.Register(rpc.Route{"post_api", "get_plagiarism_check_details"}, ap.GetCheckResultEndpoint)
	rpcRouter.Register(rpc.Route{"post_api", "get_from_network"}, blog.GetPostsFromNetwork)
	rpcRouter.Register(rpc.Route{"post_api", "get_downvotes"}, blog.Downvotes)

	// all transaction are going through network_broadcast_api
	// redirect them to the transaction router
	rpcRouter.Register(rpc.Route{"network_broadcast_api", "broadcast_transaction_synchronous"},
		transactionRouter.Route)

	// transaction routes
	transactionRouter.Register(types.RegisterOpType, blog.Register)
	transactionRouter.Register(types.RegisterPushTokenOpType, blog.RegisterPushToken)
	transactionRouter.Register(types.UpdateProfileOpType, blog.UpdateProfile)
	transactionRouter.Register(types.UploadMediaOpType, blog.UploadMedia)
	transactionRouter.Register(types.FollowOpType, blog.Follow)
	transactionRouter.Register(types.UnfollowOpType, blog.Unfollow)
	transactionRouter.Register(types.AddToBlacklistAdminOpType, blog.AddToBlacklistAdmin)
	transactionRouter.Register(types.RemoveFromBlacklistAdminOpType, blog.RemoveFromBlacklistAdmin)
	transactionRouter.Register(types.AddCategoryAdminOpType, blog.AddCategoryAdmin)
	transactionRouter.Register(types.RemoveCategoryAdminOpType, blog.RemoveCategoryAdmin)
	transactionRouter.Register(types.UpdateCategoryAdminOpType, blog.UpdateCategoryAdmin)
	transactionRouter.Register(types.SetAccountTrustedAdminOpType, blog.SetAccountTrustedAdmin)
	transactionRouter.Register(types.UpsertDraftOpType, blog.UpsertDraft)
	transactionRouter.Register(types.RemoveDraftOpType, blog.RemoveDraft)
	transactionRouter.Register(types.MarkNotificationReadOpType, blog.MarkRead)
	transactionRouter.Register(types.MarkAllNotificationsReadOpType, blog.MarkReadAll)
	transactionRouter.Register(types.MarkAllNotificationsSeenOpType, blog.MarkSeenAll)
	transactionRouter.Register(types.UpdateProfileSettingsOpType, blog.UpdateProfileSettings)
	transactionRouter.Register(types.DownvoteOpType, blog.Downvote)
	transactionRouter.Register(types.RemoveDownvoteOpType, blog.RemoveDownvote)

	return rpcRouter
}
