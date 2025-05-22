package db

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"gitlab.scorum.com/blog/api/utils"
)

const leonarda = "leonarda"
const sheldon = "scheldon"

var dbWrite *sqlx.DB

func TestMain(m *testing.M) {
	utils.DockertestMain(m, func(write, read *sqlx.DB) {
		dbWrite = write
	})
}

func cleanUp(t *testing.T) {
	_, err := dbWrite.Exec("DELETE FROM push_tokens")
	require.NoError(t, err)

	_, err = dbWrite.Exec("DELETE FROM profile_settings")
	require.NoError(t, err)

	_, err = dbWrite.Exec("DELETE FROM notifications")
	require.NoError(t, err)

	_, err = dbWrite.Exec("DELETE FROM posts_plagiarism")
	require.NoError(t, err)

	_, err = dbWrite.Exec("DELETE FROM profiles")
	require.NoError(t, err)
}

func registerAccount(t *testing.T, account string) {
	storage := NewProfileStorage(dbWrite)
	require.NoError(t, storage.Create(account))
}
