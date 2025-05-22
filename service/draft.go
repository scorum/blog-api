package service

import (
	"database/sql"
	"encoding/json"
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/rpc"
	"gopkg.in/go-playground/validator.v9"
)

func (blog *Blog) UpsertDraft(op types.Operation) *rpc.Error {
	in := op.(*types.UpsertDraftOperation)

	if in.Body == "" && in.Title == "" {
		return NewError(rpc.InvalidParameterCode, "empty body and title")
	}

	v := validator.New()
	err := v.Struct(in)
	if err != nil {
		return NewError(
			rpc.InvalidParameterCode,
			err.Error(),
		)
	}

	draft := db.Draft{
		Account:      in.Account,
		ID:           in.ID,
		Title:        in.Title,
		Body:         in.Body,
		JsonMetadata: in.JsonMetadata,
	}

	if _, err := blog.DB.Write.NamedExec(
		`INSERT INTO drafts VALUES(:account, :id, :title, :body, :json_metadata)
		ON CONFLICT (account,id) DO UPDATE SET title=:title, body=:body, json_metadata = :json_metadata`, draft); err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	return nil
}

func (blog *Blog) RemoveDraft(op types.Operation) *rpc.Error {
	in := op.(*types.RemoveDraftOperation)

	result, err := blog.DB.Write.Exec(`DELETE FROM drafts WHERE account = $1 AND id =$2`, in.Account, in.ID)
	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	if rowsAffected == 0 {
		return NewError(rpc.DraftNotFoundCode, "draft not found")
	}

	return nil
}

func (blog *Blog) GetDraft(ctx *rpc.Context, account string, params []*json.RawMessage) {
	var id string
	if err := getParam(params, 0, &id); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	if id == "" {
		ctx.WriteError(rpc.InvalidParameterCode, "invalid id")
		return
	}

	draft, err := blog.doGetDraft(account, id)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	ctx.WriteResult(toAPIDraft(*draft))
}

func (blog *Blog) doGetDraft(account, id string) (*db.Draft, *rpc.Error) {
	var draft db.Draft

	if err := blog.DB.Read.Get(&draft,
		`SELECT id, account, title, body, json_metadata, updated_at, created_at FROM drafts WHERE account = $1 AND id =$2`,
		account, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, NewError(rpc.DraftNotFoundCode, "draft not found")
		}
		return nil, WrapError(rpc.InternalErrorCode, err)
	}

	return &draft, nil
}

func (blog *Blog) GetDrafts(ctx *rpc.Context, account string, params []*json.RawMessage) {
	drafts, err := blog.doGetDrafts(account)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	ctx.WriteResult(toAPIDrafts(drafts))
}

func (blog *Blog) doGetDrafts(account string) ([]*db.Draft, *rpc.Error) {
	var drafts []*db.Draft

	if err := blog.DB.Read.Select(&drafts,
		`SELECT id, account, title, body, json_metadata, updated_at, created_at FROM drafts
		WHERE account = $1 ORDER BY updated_at DESC`, account); err != nil {
		return nil, WrapError(rpc.InternalErrorCode, err)
	}

	return drafts, nil
}
