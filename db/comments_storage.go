package db

import "github.com/jmoiron/sqlx"

type CommentsStorage struct {
	db sqlx.Ext
}

func NewCommentsStorage(db *sqlx.DB) *CommentsStorage {
	return &CommentsStorage{
		db: db,
	}
}

func (c *CommentsStorage) InTx(tx *sqlx.Tx) *CommentsStorage {
	return &CommentsStorage{db: tx}
}

func (c *CommentsStorage) Get(author, permlink string) (*Comment, error) {
	var comment Comment
	err := sqlx.Get(c.db, &comment,
		`SELECT permlink, author, parent_permlink, parent_author, body,
				title, json_metadata, updated_at, created_at
		        FROM comments
                WHERE author = $1 AND permlink = $2`, author, permlink)
	return &comment, err
}

func (c *CommentsStorage) GetParentPost(author, permlink string) (*PostInfo, error) {
	var info PostInfo
	err := sqlx.Get(c.db, &info, `
		WITH RECURSIVE parents AS
		(
			SELECT p.* FROM comments p WHERE p.author=$1 AND p.permlink=$2
			UNION ALL
			SELECT p.* FROM parents JOIN comments p ON parents.parent_permlink=p.permlink AND parents.parent_author=p.author
		)
		SELECT permlink, author, json_metadata, title, parent_permlink AS category FROM parents WHERE parent_author IS NULL;
		`, author, permlink)
	return &info, err
}
