package service

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/scorum/scorum-go/transport/http"
	"github.com/stretchr/testify/require"
	"gitlab.scorum.com/blog/api/blob"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/utils"
)

const nodeHTTPS = "https://testnet.scorum.com"

const (
	leonarda = "leonarda"
	kristie  = "kristie"
	sheldon  = "sheldon"

	notificationsLimit = 100
	followsLimit       = 1000
)

var handler Blog

func init() {
	transport := http.NewTransport(nodeHTTPS)
	client := scorumgo.NewClient(transport)

	handler = Blog{
		Blockchain: client,
		Blob: blob.NewService(blob.Config{
			Container:   "test",
			CDNDomain:   "http://cdn-blog.scorum.com",
			AccountName: "scorumblog",
			AccountKey:  "",
		}),
		Config: Config{
			Admin:              leonarda,
			NotificationsLimit: notificationsLimit,
			MaxFollow:          followsLimit,
		},
	}
}

func cleanUp(t *testing.T) {
	_, err := dbWrite.Exec("DELETE FROM comments")
	require.NoError(t, err)
	_, err = dbWrite.Exec("DELETE FROM deleted_posts")
	require.NoError(t, err)
	_, err = dbWrite.Exec("DELETE FROM posts_votes")
	require.NoError(t, err)
	_, err = dbWrite.Exec("DELETE FROM notifications")
	require.NoError(t, err)
	_, err = dbWrite.Exec("DELETE FROM profile_settings")
	require.NoError(t, err)
	_, err = dbWrite.Exec("DELETE FROM posts_plagiarism")
	require.NoError(t, err)
	_, err = dbWrite.Exec("DELETE FROM categories")
	require.NoError(t, err)
	_, err = dbWrite.Exec("DELETE FROM drafts")
	require.NoError(t, err)
	_, err = dbWrite.Exec("DELETE FROM blacklist")
	require.NoError(t, err)
	_, err = dbWrite.Exec("DELETE FROM followers")
	require.NoError(t, err)
	_, err = dbWrite.Exec("DELETE FROM media")
	require.NoError(t, err)
	_, err = dbWrite.Exec("DELETE FROM profiles")
	require.NoError(t, err)
}

var (
	dbWrite *sqlx.DB
	dbRead  *sqlx.DB
)

func registerAccount(t *testing.T, account string) {
	_, err := dbWrite.Exec(
		`INSERT INTO profiles(account, display_name) VALUES($1, $2) ON CONFLICT DO NOTHING`,
		account, account)
	require.NoError(t, err)
}

// TestMain runs a docker container with Postgres and applies db migration
func TestMain(m *testing.M) {
	utils.DockertestMain(m, func(write, read *sqlx.DB) {
		dbWrite = write
		dbRead = read

		// set db
		handler.DB.Write = dbWrite
		handler.DB.Read = dbRead

		handler.NotificationStorage = db.NewNotificationsStorage(dbWrite)
	})
}
