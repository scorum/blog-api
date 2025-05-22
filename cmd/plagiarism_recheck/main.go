package main

import (
	"flag"
	"sync"
	"time"

	"gitlab.scorum.com/blog/core/sentry"

	"github.com/jinzhu/configor"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/service"
	"gitlab.scorum.com/blog/core/domain"
)

const configPath = "config.yml"

var (
	configPathFlag = flag.String("config", configPath, "path to the app config")

	// version is set via `go build -ldflags "-x main.version=version"`
	version     string
	versionFlag = flag.Bool("version", false, "app version")
)

type Config struct {
	DB        string `yaml:"db"`
	Sentry    string `yaml:"sentry"`
	TextRuKey string `yaml:"text_ru_key"`
	Limit     int    `yaml:"limit"`
}

func main() {
	flag.Parse()
	if *versionFlag {
		log.Info(version)
		return
	}

	var config Config
	if err := configor.Load(&config, *configPathFlag); err != nil {
		log.Fatal(err)
	}

	hook, err := sentry.NewHook(config.Sentry)
	if err != nil {
		log.Fatal(err)
	}
	log.AddHook(hook)

	dbConn, err := sqlx.Open("postgres", config.DB)
	if err != nil {
		log.Fatal(err)
	}

	// anti plagiarism service
	antiPlagiarism := service.NewAntiPlagiarismService(
		config.TextRuKey,
		db.NewPlagiarismStorage(dbConn),
		db.NewCommentsStorage(dbConn),
	)

	notifications := db.NewNotificationsStorage(dbConn)

	var posts []*db.Comment

	err = sqlx.Select(dbConn, &posts,
		`SELECT c.author, c.permlink, c.body, c.domain FROM posts_plagiarism pp
				INNER JOIN comments c on c.permlink=pp.permlink and c.author=pp.author
				WHERE pp.status='failed' and c.created_at > NOW() - interval '7 day'
				ORDER BY c.updated_at DESC
				LIMIT $1`, config.Limit)
	if err != nil {
		log.Fatal(err)
	}

	if len(posts) == 0 {
		log.Fatal("there are no posts to recheck")
	}

	log.Infof("%d posts to re-check for plagiarism", len(posts))

	wg := sync.WaitGroup{}
	wg.Add(len(posts))

	t := 1800 / len(posts) // Max post check can take 20 min. Our job suppose to run max 1h. We need to initiate post checks in a first half an hour to be sure all checks finished.
	ticker := time.NewTicker(time.Duration(t) * time.Second)

	for _, p := range posts {
		go func(c db.Comment) {
			defer wg.Done()

			<-ticker.C // for throttling

			log.Infof("start checking @%s/%s", c.Author, c.Permlink)

			d := domain.GetDomainSafe([]string{c.Domain.String})
			res, err := antiPlagiarism.CheckPost(c.Author, c.Permlink, c.Body, d)
			if err != nil {
				log.Errorf("can't check post @%s/%s err: %s", c.Author, c.Permlink, err)
				return
			}

			if res.Status != db.PlagiarismStatusChecked {
				return
			}

			log.Infof("check @%s/%s finished u:%g", c.Author, c.Permlink, res.Unique)

			err = notifications.DeletePlagiarismNotification(c.Author, c.Permlink)
			if err != nil {
				log.Warnf("can't delete plagiarism notification for @%s/%s err:%s", c.Author, c.Permlink, err)
				return
			}

			meta := db.PlagiarismRelatedNotificationMeta{}
			meta.Account = c.Author
			meta.Permlink = c.Permlink
			if len(c.JsonMetadata.Categories) > 0 {
				meta.PostCategory = c.JsonMetadata.Categories[0]
			}
			meta.PostTitle = c.Title
			meta.PostImage = c.JsonMetadata.Image
			meta.Domains = []string{string(d)}
			meta.Uniqueness = res.Unique
			meta.Status = res.Status

			notification := db.Notification{
				Account:   c.Author,
				Timestamp: time.Now().UTC(),
				Type:      db.PostUniquenessCheckedNotificationType,
				Meta:      meta.ToJson(),
			}

			if err := notifications.Insert(notification); err != nil {
				log.Warnf("can't insert palgiarism notification into db @%s/%s err: %s", c.Author, c.Permlink, err)
			}
		}(*p)
	}

	wg.Wait()
	log.Info("finished")
}
