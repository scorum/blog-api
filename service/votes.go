package service

import (
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/rpc"
)

func (b *Blog) GetVotesForPostEndpoint(ctx *rpc.Context) {
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

	votesMap, err := b.GetPostVotes(author, permlink)
	if err != nil {
		ctx.WriteError(rpc.InternalErrorCode, err.Error())
		return
	}

	ctx.WriteResult(votesMap)
}

// GetPostVotes returns votes for requested post in account to post uniqueness format
func (b *Blog) GetPostVotes(author, permlink string) (map[string]float32, error) {
	var votes []db.Vote
	err := b.DB.Read.Select(
		&votes,
		`SELECT account, permlink, author, post_unique
		 FROM posts_votes
		 WHERE author = $1 and permlink = $2`,
		author,
		permlink,
	)
	if err != nil {
		return nil, err
	}

	accountToPostUniqueness := make(map[string]float32, len(votes))
	for _, v := range votes {
		accountToPostUniqueness[v.Account] = v.PostUnique
	}

	return accountToPostUniqueness, nil
}
