package db

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDownvote(t *testing.T) {
	defer cleanUp(t)

	permlink := "perm"

	registerAccount(t, leonarda)
	registerAccount(t, sheldon)

	ds := NewDownvotesStorage(dbWrite)
	downvote := Downvote{
		Reason:   DownvoteReasonSpam,
		Author:   leonarda,
		Account:  sheldon,
		Permlink: permlink,
	}
	require.NoError(t, ds.Downvote(downvote))
	require.NoError(t, ds.Downvote(downvote))

	downvotes, err := ds.GetDownvotesForPost(permlink, leonarda)
	require.NoError(t, err)
	require.Len(t, downvotes, 1)

	require.NoError(t, ds.Delete(downvote))
	downvotes, err = ds.GetDownvotesForPost(permlink, leonarda)
	require.NoError(t, err)
	require.Empty(t, downvotes)
}
