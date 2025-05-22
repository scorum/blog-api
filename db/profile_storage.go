package db

import (
	"github.com/jmoiron/sqlx"
)

type ProfileStorage struct {
	db *sqlx.DB
}

func NewProfileStorage(db *sqlx.DB) *ProfileStorage {
	return &ProfileStorage{
		db: db,
	}
}

func (p *ProfileStorage) Create(account string) (err error) {
	var tx *sqlx.Tx

	tx, err = p.db.Beginx()

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if err != nil {
		return err
	}

	if _, err := tx.Exec(
		`INSERT INTO profiles(account, display_name)
                VALUES($1, $2)
                ON CONFLICT DO NOTHING`,
		account, account); err != nil {
		return err
	}

	if _, err := tx.Exec(
		`INSERT INTO profile_settings(account, enable_email_unseen_notifications)
                VALUES($1, $2)
                ON CONFLICT DO NOTHING`,
		account, true); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
