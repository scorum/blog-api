package db

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/jmoiron/sqlx"
	"time"
)

const (
	PlagiarismStatusPending        = "pending"
	PlagiarismStatusLegacy         = "legacy"
	PlagiarismStatusChecked        = "checked"
	PlagiarismStatusFailed         = "failed"
	PlagiarismStatusInvalidTextLen = "invalid_text_len"
)

type PostPlagiarism struct {
	Author            string         `db:"author"`
	Permlink          string         `db:"permlink"`
	LastCheckAt       time.Time      `db:"last_check_at"`
	UniquenessPercent float32        `db:"uniqueness_percent"`
	Urls              PlagiarismUrls `db:"urls"`
	ChecksNum         int            `db:"checks_num"`
	Status            string         `db:"status"`
}

type PlagiarismUrls []PlagiarismUrl

func (p PlagiarismUrls) Value() (driver.Value, error) {
	j, err := json.Marshal(p)
	return j, err
}

func (p *PlagiarismUrls) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.New("type assertion .([]byte) failed.")
	}

	err := json.Unmarshal(source, p)
	return err
}

type PlagiarismUrl struct {
	Url     string  `json:"url"`
	Plagiat float32 `json:"plagiat"`
	Title   string  `json:"title"`
}

type PlagiarismStorage struct {
	db sqlx.Ext
}

func NewPlagiarismStorage(db *sqlx.DB) *PlagiarismStorage {
	return &PlagiarismStorage{db: db}
}

func (ps *PlagiarismStorage) InTx(tx *sqlx.Tx) *PlagiarismStorage {
	return &PlagiarismStorage{db: tx}
}

func (ps *PlagiarismStorage) Insert(p PostPlagiarism) error {
	_, err := sqlx.NamedExec(ps.db, `
				INSERT
				INTO posts_plagiarism (author, permlink, last_check_at, uniqueness_percent, urls, status)
				VALUES (:author, :permlink, :last_check_at, :uniqueness_percent, :urls, :status)
				ON CONFLICT (author, permlink) DO NOTHING
			`,
		p,
	)
	return err
}

func (ps *PlagiarismStorage) Upsert(p PostPlagiarism) error {
	_, err := sqlx.NamedExec(ps.db, `
				INSERT
				INTO posts_plagiarism (author, permlink, last_check_at, uniqueness_percent, urls, status)
				VALUES (:author, :permlink, :last_check_at, :uniqueness_percent, :urls, :status)
				ON CONFLICT (author, permlink) DO UPDATE
				SET last_check_at=:last_check_at, uniqueness_percent=:uniqueness_percent, urls=:urls, checks_num=posts_plagiarism.checks_num + 1, status=:status
			`,
		p,
	)
	return err
}

func (ps *PlagiarismStorage) UpdateStatus(account, permlink, status string) error {
	p := PostPlagiarism{
		Author:   account,
		Permlink: permlink,
		Status:   PlagiarismStatusPending,
	}
	_, err := sqlx.NamedExec(ps.db, `
												 UPDATE posts_plagiarism
												 SET status=:status
												 WHERE author=:author AND permlink=:permlink`,
		p)
	return err
}

func (ps *PlagiarismStorage) Get(account, permlink string) (*PostPlagiarism, error) {
	var p PostPlagiarism

	err := sqlx.Get(ps.db,
		&p,
		`SELECT author, permlink, last_check_at, uniqueness_percent, status, urls, checks_num FROM posts_plagiarism WHERE author=$1 AND permlink=$2`,
		account,
		permlink,
	)

	if err != nil {
		return nil, err
	}

	return &p, nil
}
