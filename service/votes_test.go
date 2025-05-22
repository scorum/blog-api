package service

import (
	"github.com/stretchr/testify/require"
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/db"
	"testing"
)

func TestGetPostVotes(t *testing.T) {
	testAuthor := "herald"
	testPermlink := "footballsocool"
	testAccount := "man"

	require.Nil(t, handler.Register(&types.RegisterOperation{testAuthor}))
	require.Nil(t, handler.Register(&types.RegisterOperation{testAccount}))

	vote := db.Vote{
		Account:    testAccount,
		Permlink:   testPermlink,
		Author:     testAuthor,
		PostUnique: 50,
	}

	_, err := handler.DB.Write.NamedExec(
		`INSERT INTO posts_votes (account, permlink, author, post_unique)
		VALUES(:account, :permlink, :author, :post_unique)
		`,
		vote,
	)
	require.NoError(t, err)

	m, err := handler.GetPostVotes(testAuthor, testPermlink)
	require.NoError(t, err)

	require.Len(t, m, 1)
}
