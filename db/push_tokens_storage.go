package db

import "github.com/jmoiron/sqlx"

type PushTokensStorage struct {
	db *sqlx.DB
}

func NewPushTokensStorage(db *sqlx.DB) *PushTokensStorage {
	return &PushTokensStorage{
		db: db,
	}
}

func (pr *PushTokensStorage) Add(acc string, token string) error {
	_, err := pr.db.Exec(
		`INSERT INTO push_tokens (account, token)
                VALUES ($1, $2)
                ON CONFLICT (account, token) DO NOTHING`, acc, token)
	return err
}

func (pr *PushTokensStorage) Delete(acc string, token string) error {
	_, err := pr.db.Exec(
		`DELETE FROM push_tokens
                WHERE account = $1 AND token = $2 `, acc, token)
	return err
}

func (pr *PushTokensStorage) GetTokensByAccount(acc string) (tokens []string, err error) {
	err = pr.db.Select(&tokens,
		`SELECT token FROM push_tokens WHERE account = $1`, acc)
	return
}
