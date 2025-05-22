package service

import (
	"errors"
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/rpc"
	"gitlab.scorum.com/blog/api/utils/postgres"
)

func (blog *Blog) AddToBlacklistAdmin(op types.Operation) *rpc.Error {
	in := op.(*types.AddToBlacklistAdminOperation)

	if in.Account != blog.Config.Admin {
		return NewError(rpc.AccessDeniedCode, "access denied")
	}

	return blog.doAddToBlacklist(in.BlogAccount, in.Permlink)
}

func (blog *Blog) doAddToBlacklist(account, permlink string) *rpc.Error {
	_, err := blog.DB.Write.Exec(`INSERT INTO blacklist VALUES($1, $2)
								   ON CONFLICT DO NOTHING`, account, permlink)

	if err != nil {
		if isErr, constraint := postgres.IsForeignKeyViolationError(err); isErr && constraint == "blacklist_account_fk" {
			return NewError(rpc.ProfileNotFoundCode, "account doesn't exists")
		}
		return WrapError(rpc.InternalErrorCode, err)
	}

	return nil
}

func (blog *Blog) RemoveFromBlacklistAdmin(op types.Operation) *rpc.Error {
	in := op.(*types.RemoveFromBlacklistAdminOperation)

	if in.Account != blog.Config.Admin {
		return NewError(rpc.AccessDeniedCode, "access denied")
	}

	rows, err := blog.DB.Write.Exec(`DELETE FROM blacklist WHERE account = $1 AND permlink = $2`, in.BlogAccount, in.Permlink)
	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	n, err := rows.RowsAffected()
	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	if n == 0 {
		return WrapError(rpc.BlacklistEntityNotFoundCode, errors.New("blacklist entity not found"))
	}

	return nil
}

func (blog *Blog) IsBlacklisted(ctx *rpc.Context) {
	var account string
	if err := ctx.Param(0, &account); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	var permlink string
	if err := ctx.Param(1, &permlink); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	isBlacklisted, err := blog.checkIsBlacklisted(account, permlink)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	ctx.WriteResult(isBlacklisted)
}

func (blog *Blog) checkIsBlacklisted(account, permlink string) (*bool, *rpc.Error) {
	var exists bool
	err := blog.DB.Read.Get(&exists, `SELECT EXISTS(SELECT * FROM blacklist WHERE account = $1 AND permlink = $2)`, account, permlink)
	if err != nil {
		return nil, WrapError(rpc.InternalErrorCode, err)
	}

	return &exists, nil
}

func (blog *Blog) GetBlacklist(ctx *rpc.Context) {
	var from uint32
	if err := ctx.Param(0, &from); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	var limit uint16
	if err := ctx.Param(1, &limit); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	if limit > maxLargePageSize {
		ctx.WriteError(rpc.InvalidParameterCode, "invalid limit")
		return
	}

	entries, err := blog.doGetBlacklist(from, limit)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	ctx.WriteResult(entries)
}

func (blog *Blog) doGetBlacklist(from uint32, limit uint16) ([]*PostID, *rpc.Error) {
	var entries []*db.PostID

	err := blog.DB.Read.Select(&entries, `SELECT account, permlink FROM blacklist LIMIT $1 OFFSET $2`, limit, from)
	if err != nil {
		return nil, WrapError(rpc.InternalErrorCode, err)
	}

	return toAPIPostIDs(entries), nil
}
