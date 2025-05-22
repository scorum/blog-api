package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/rpc"
)

func TestBlog_SetDraft(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	op := &types.UpsertDraftOperation{
		Account:      leonarda,
		ID:           "id",
		Title:        "title",
		Body:         "body",
		JsonMetadata: "metadata",
	}

	require.Nil(t, handler.UpsertDraft(op))

	draft, err := handler.doGetDraft(op.Account, op.ID)
	require.Nil(t, err)
	require.Equal(t, draft.Account, op.Account)
	require.Equal(t, draft.ID, op.ID)
	require.Equal(t, draft.Title, op.Title)
	require.Equal(t, draft.Body, op.Body)
	require.Equal(t, draft.JsonMetadata, op.JsonMetadata)
	require.Equal(t, draft.UpdatedAt, draft.CreatedAt)

	t.Run("invalid_op", func(t *testing.T) {
		op.Title = ""
		op.Body = ""
		require.NotNil(t, handler.UpsertDraft(op))
	})

	t.Run("update_draft", func(t *testing.T) {
		op.Title = "updated_title"

		require.Nil(t, handler.UpsertDraft(op))

		draft, err := handler.doGetDraft(op.Account, op.ID)
		require.Nil(t, err)
		require.Equal(t, draft.Account, op.Account)
		require.Equal(t, draft.ID, op.ID)
		require.Equal(t, draft.Title, op.Title)
		require.Equal(t, draft.Body, op.Body)
		require.Equal(t, draft.JsonMetadata, op.JsonMetadata)
		require.NotEqual(t, draft.UpdatedAt, draft.CreatedAt)
	})

	veryLongTitle := "test "
	for len(veryLongTitle) < 255 {
		veryLongTitle = veryLongTitle + veryLongTitle
	}
	op.Title = veryLongTitle
	require.NotNil(t, handler.UpsertDraft(op))

	op.Title = "title"

	veryLongBody := "test "
	for len(veryLongBody) < 45000 {
		veryLongBody = veryLongBody + veryLongBody
	}
	op.Body = veryLongBody
	require.NotNil(t, handler.UpsertDraft(op))
}

func TestBlog_RemoveDraft(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	op := &types.UpsertDraftOperation{
		Account:      leonarda,
		ID:           "id",
		Title:        "title",
		Body:         "body",
		JsonMetadata: "metadata",
	}

	require.Nil(t, handler.UpsertDraft(op))

	_, err := handler.doGetDraft(op.Account, op.ID)
	require.Nil(t, err)

	removeOp := &types.RemoveDraftOperation{
		Account: op.Account,
		ID:      op.ID,
	}

	require.Nil(t, handler.RemoveDraft(removeOp))

	_, err = handler.doGetDraft(op.Account, op.ID)
	require.NotNil(t, err)
	require.Equal(t, err.Code, rpc.DraftNotFoundCode)
}

func TestBlog_GetDrafts(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)

	op := &types.UpsertDraftOperation{
		Account:      leonarda,
		ID:           "id",
		Title:        "title",
		Body:         "body",
		JsonMetadata: "metadata",
	}

	require.Nil(t, handler.UpsertDraft(op))

	op.ID = "id2"
	op.Title = "title2"

	require.Nil(t, handler.UpsertDraft(op))

	drafts, err := handler.doGetDrafts(op.Account)
	require.Nil(t, err)
	require.Len(t, drafts, 2)
	require.Equal(t, drafts[0].ID, "id2")
	require.Equal(t, drafts[1].ID, "id")
}

func TestBlog_DraftMarshalling(t *testing.T) {
	registerAccount(t, leonarda)

	op := &types.UpsertDraftOperation{
		Account:      leonarda,
		ID:           "id",
		Title:        "title",
		Body:         "body",
		JsonMetadata: "metadata",
	}

	require.Nil(t, handler.UpsertDraft(op))

	draft, err := handler.doGetDraft(op.Account, op.ID)
	require.Nil(t, err)
	apiDraft := toAPIDraft(*draft)

	_, err2 := json.Marshal(apiDraft)
	require.NoError(t, err2)
}
