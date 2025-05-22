package service

import (
	"database/sql"
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/rpc"
)

func (blog *Blog) Downvote(op types.Operation) *rpc.Error {
	in := op.(*types.DownvoteOperation)

	reason := db.DownvoteReason(in.Reason)

	if !reason.IsValid() {
		return &rpc.Error{Code: rpc.InvalidParameterCode, Message: "invalid reason"}
	}

	downvote := db.Downvote{
		Account:  in.Account,
		Author:   in.Author,
		Permlink: in.Permlink,
		Reason:   reason,
		Comment:  in.Comment,
	}

	err := blog.DownvotesStorage.Downvote(downvote)
	if err != nil {
		return &rpc.Error{Code: rpc.InternalErrorCode, Message: err.Error()}
	}

	return nil
}

func (blog *Blog) RemoveDownvote(op types.Operation) *rpc.Error {
	in := op.(*types.RemoveDownvoteOperation)

	downvote := db.Downvote{
		Account:  in.Account,
		Author:   in.Author,
		Permlink: in.Permlink,
	}

	err := blog.DownvotesStorage.Delete(downvote)
	if err != nil && err == sql.ErrNoRows {
		return &rpc.Error{Code: rpc.DownvoteNotFoundCode, Message: err.Error()}
	}
	if err != nil {
		return &rpc.Error{Code: rpc.InternalErrorCode, Message: err.Error()}
	}

	return nil
}

func (blog *Blog) Downvotes(ctx *rpc.Context) {
	var author string
	if err := ctx.Param(0, &author); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	var permlink string
	if err := ctx.Param(1, &permlink); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	downvotes, err := blog.DownvotesStorage.GetDownvotesForPost(permlink, author)
	if err != nil {
		ctx.WriteError(rpc.InternalErrorCode, err.Error())
		return
	}

	ctx.WriteResult(downvotes)
}
