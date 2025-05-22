package db

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
)

const (
	DownvoteReasonSpam              DownvoteReason = "spam"
	DownvoteReasonPlagiarism        DownvoteReason = "plagiarism"
	DownvoteReasonHateOrTrolling    DownvoteReason = "hate_or_trolling"
	DownvoteReasonLowQualityContent DownvoteReason = "low_quality_content"
	DownvoteReasonDisagreeOnRewards DownvoteReason = "disagree_on_rewards"
)

var (
	validDownvoteReasons = []DownvoteReason{
		DownvoteReasonSpam,
		DownvoteReasonPlagiarism,
		DownvoteReasonHateOrTrolling,
		DownvoteReasonLowQualityContent,
		DownvoteReasonDisagreeOnRewards,
	}
)

type DownvoteReason string

func (dr DownvoteReason) IsValid() bool {
	for _, r := range validDownvoteReasons {
		if dr == r {
			return true
		}
	}

	return false
}

type Downvote struct {
	Account  string         `db:"account" json:"account"`
	Permlink string         `db:"permlink" json:"permlink"`
	Author   string         `db:"author" json:"author"`
	Reason   DownvoteReason `db:"reason" json:"reason"`
	Comment  string         `db:"comment" json:"comment"`
}

type DownvotesStorage struct {
	db sqlx.Ext
}

func NewDownvotesStorage(db *sqlx.DB) *DownvotesStorage {
	return &DownvotesStorage{db: db}
}

func (ds *DownvotesStorage) GetDownvotesForPost(permlink, author string) (map[string]*Downvote, error) {
	var downvotes []*Downvote
	err := sqlx.Select(ds.db, &downvotes, `
		SELECT account, permlink, author, reason, comment
		FROM downvotes
		WHERE permlink=$1 and author=$2
	`, permlink, author)
	if err != nil {
		return nil, err
	}

	downvotesMap := make(map[string]*Downvote, len(downvotes))
	for _, d := range downvotes {
		downvotesMap[d.Account] = d
	}

	return downvotesMap, nil
}

func (ds *DownvotesStorage) Downvote(dw Downvote) error {
	_, err := sqlx.NamedExec(ds.db, `
				INSERT
				INTO downvotes (account, author, permlink, reason, comment)
				VALUES (:account, :author, :permlink, :reason, :comment)
				ON CONFLICT (account, author, permlink) DO UPDATE
				SET reason=:reason, comment=:comment
			`,
		dw,
	)

	return err
}

func (ds *DownvotesStorage) Delete(dw Downvote) error {
	rows, err := sqlx.NamedExec(ds.db, `
						DELETE FROM downvotes
						WHERE account=:account AND author=:author AND permlink=:permlink
					`,
		dw,
	)
	if err != nil {
		return err
	}

	n, err := rows.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return sql.ErrNoRows
	}

	return nil
}
