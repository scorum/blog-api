package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.scorum.com/blog/api/broadcast/types"
)

func TestBlog_AddToBlacklistAdmin(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, sheldon)
	op := &types.AddToBlacklistAdminOperation{
		Account:     leonarda,
		BlogAccount: sheldon,
		Permlink:    "permlink",
	}

	// It is normal to blacklist one post twice
	require.Nil(t, handler.AddToBlacklistAdmin(op))
	require.Nil(t, handler.AddToBlacklistAdmin(op))
	list, err := handler.doGetBlacklist(0, 100)
	require.Nil(t, err)
	require.Len(t, list, 1)
}

func TestBlog_AddToBlacklistAdmin_ValidationTest(t *testing.T) {
	op := types.AddToBlacklistAdminOperation{
		Account:     "some_account_name",
		BlogAccount: "some_acc",
		Permlink:    "permlink",
	}

	t.Run("valid_operation", func(t *testing.T) {
		require.NoError(t, validate.Struct(op))
	})

	t.Run("empty_account", func(t *testing.T) {
		cop := op
		cop.Account = ""
		require.Error(t, validate.Struct(cop))
	})

	t.Run("empty_blog_account", func(t *testing.T) {
		cop := op
		cop.BlogAccount = ""
		require.Error(t, validate.Struct(cop))
	})

	t.Run("empty_permlink", func(t *testing.T) {
		cop := op
		cop.Permlink = ""
		require.Error(t, validate.Struct(cop))
	})
}

func TestBlog_RemoveFromBlacklistAdmin(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, sheldon)
	op := &types.RemoveFromBlacklistAdminOperation{
		Account:     leonarda,
		BlogAccount: sheldon,
		Permlink:    "permlink",
	}

	// Remove not existing post
	require.NotNil(t, handler.RemoveFromBlacklistAdmin(op))

	// Ban post
	require.Nil(t, handler.AddToBlacklistAdmin(&types.AddToBlacklistAdminOperation{
		Account:     leonarda,
		BlogAccount: op.BlogAccount,
		Permlink:    op.Permlink,
	}))
	list, err := handler.doGetBlacklist(0, 100)
	require.Nil(t, err)
	require.Len(t, list, 1)

	// Remove existing post
	require.Nil(t, handler.RemoveFromBlacklistAdmin(op))

	list, err = handler.doGetBlacklist(0, 100)
	require.Nil(t, err)
	require.Empty(t, list)
}

func TestBlog_RemoveFromBlacklistAdmin_ValidationTest(t *testing.T) {
	op := types.RemoveFromBlacklistAdminOperation{
		Account:     "some_account_name",
		BlogAccount: "some_acc",
		Permlink:    "permlink",
	}

	t.Run("valid_operation", func(t *testing.T) {
		require.NoError(t, validate.Struct(op))
	})

	t.Run("empty_account", func(t *testing.T) {
		cop := op
		cop.Account = ""
		require.Error(t, validate.Struct(cop))
	})

	t.Run("empty_blog_account", func(t *testing.T) {
		cop := op
		cop.BlogAccount = ""
		require.Error(t, validate.Struct(cop))
	})

	t.Run("empty_permlink", func(t *testing.T) {
		cop := op
		cop.Permlink = ""
		require.Error(t, validate.Struct(cop))
	})
}

func TestBlog_GetBlacklist(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	registerAccount(t, sheldon)
	op := &types.AddToBlacklistAdminOperation{
		Account:     leonarda,
		BlogAccount: sheldon,
		Permlink:    "permlink",
	}

	require.Nil(t, handler.AddToBlacklistAdmin(op))
	list, err := handler.doGetBlacklist(0, 100)
	require.Nil(t, err)
	require.Len(t, list, 1)
	require.Equal(t, list[0].Permlink, op.Permlink)
	require.Equal(t, list[0].Account, op.BlogAccount)
}

func TestBlog_IsBlacklisted(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	op := &types.AddToBlacklistAdminOperation{
		Account:     leonarda,
		BlogAccount: leonarda,
		Permlink:    "permlink",
	}

	require.Nil(t, handler.AddToBlacklistAdmin(op))

	isBlacklisted, err := handler.checkIsBlacklisted(op.Account, op.Permlink)
	require.Nil(t, err)
	require.True(t, *isBlacklisted)

	require.Nil(t, handler.RemoveFromBlacklistAdmin(&types.RemoveFromBlacklistAdminOperation{
		Account:     leonarda,
		BlogAccount: op.Account,
		Permlink:    op.Permlink,
	}))
	isBlacklisted, err = handler.checkIsBlacklisted(op.Account, op.Permlink)
	require.Nil(t, err)
	require.False(t, *isBlacklisted)
}
