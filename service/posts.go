package service

import (
	"fmt"

	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/rpc"
	. "gitlab.scorum.com/blog/core/domain"
)

func (blog *Blog) IsPostDeleted(ctx *rpc.Context) {
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

	isDeleted, err := blog.checkIsPostDeleted(account, permlink)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	ctx.WriteResult(isDeleted)
}

func (blog *Blog) checkIsPostDeleted(account, permlink string) (*bool, *rpc.Error) {
	var exists bool
	err := blog.DB.Read.Get(&exists, `SELECT EXISTS(SELECT * FROM deleted_posts WHERE account = $1 AND permlink = $2)`, account, permlink)
	if err != nil {
		return nil, WrapError(rpc.InternalErrorCode, err)
	}

	return &exists, nil
}

func (blog *Blog) GetDeletedPosts(ctx *rpc.Context) {
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

	entries, err := blog.doGetDeletedPosts(from, limit)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	ctx.WriteResult(entries)
}

func (blog *Blog) doGetDeletedPosts(from uint32, limit uint16) ([]*PostID, *rpc.Error) {
	var entries []*db.PostID

	err := blog.DB.Read.Select(&entries, `SELECT account, permlink FROM deleted_posts LIMIT $1 OFFSET $2`, limit, from)
	if err != nil {
		return nil, WrapError(rpc.InternalErrorCode, err)
	}

	return toAPIPostIDs(entries), nil
}

func (blog *Blog) GetPostsFromNetwork(ctx *rpc.Context) {
	var account string
	if err := ctx.Param(0, &account); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	var domain string
	if err := ctx.Param(1, &domain); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	var from uint32
	if err := ctx.Param(2, &from); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	var limit uint32
	if err := ctx.Param(3, &limit); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	if limit > maxLargePageSize {
		ctx.WriteError(rpc.InvalidParameterCode, "invalid limit")
		return
	}

	if !IsValidDomain(domain) {
		ctx.WriteError(rpc.InvalidParameterCode, fmt.Sprintf("%s is not a valid domain", domain))
		return
	}

	posts, err := blog.doGetPostsFromNetwork(account, Domain(domain), from, limit)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	ctx.WriteResult(posts)
}

func (blog *Blog) doGetPostsFromNetwork(account string, domain Domain, from uint32, limit uint32) ([]*PostID, *rpc.Error) {
	var entries []*db.PostID

	// Take followers' posts ordered by created date descending.
	// Exclude both deleted and blacklisted posts
	// Return paged result
	err := blog.DB.Read.Select(&entries,
		`SELECT comments.author AS account, comments.permlink
				FROM comments
				INNER JOIN followers f ON comments.author = f.follow_account
				WHERE
					f.account = $1 AND comments.parent_author IS NULL AND comments.domain = $2
					AND NOT EXISTS (
						SELECT * FROM blacklist WHERE comments.author = blacklist.account AND comments.permlink = blacklist.permlink)
					AND NOT EXISTS (
						SELECT* FROM deleted_posts WHERE comments.author = deleted_posts.account AND comments.permlink = deleted_posts.permlink)
				ORDER BY comments.created_at DESC
				LIMIT $3 OFFSET $4`, account, string(domain), limit, from)
	if err != nil {
		return nil, WrapError(rpc.InternalErrorCode, err)
	}

	return toAPIPostIDs(entries), nil
}
