package service

import (
	"fmt"
	"time"

	"github.com/lib/pq"

	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/rpc"
	"gitlab.scorum.com/blog/api/utils/postgres"
)

func (blog *Blog) Follow(op types.Operation) (rerr *rpc.Error) {
	in := op.(*types.FollowOperation)

	tx, err := blog.DB.Write.Beginx()
	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	defer func() {
		if rerr != nil {
			tx.Rollback()
		} else if err := tx.Commit(); err != nil {
			rerr = WrapError(rpc.InternalErrorCode, err)
		}
	}()

	var followCount int
	err = tx.Get(&followCount, `SELECT COUNT(*) FROM followers WHERE account=$1`, in.Account)
	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	if followCount >= blog.Config.MaxFollow {
		return NewError(rpc.FollowsLimitReachedCode, "user reach max number of follows")
	}

	follow := struct {
		Account       string    `db:"account"`
		FollowAccount string    `db:"follow_account"`
		CreatedAt     time.Time `db:"created_at"`
	}{
		Account:       in.Account,
		FollowAccount: in.Follow,
		CreatedAt:     time.Now().UTC(),
	}

	if _, err := tx.NamedExec(
		`INSERT INTO followers VALUES (:account, :follow_account, :created_at)`, follow); err != nil {
		if uniqueError, _ := postgres.IsUniqueError(err); uniqueError {
			return NewError(rpc.ProfileAlreadyFollowedCode, fmt.Sprintf("%s already follows %s", in.Account, in.Follow))
		}

		if foreignKeyError, _ := postgres.IsForeignKeyViolationError(err); foreignKeyError {
			return NewError(rpc.ProfileNotFoundCode, fmt.Sprintf("%s not found", in.Follow))
		}
		return WrapError(rpc.InternalErrorCode, err)
	}

	notification := db.Notification{
		Account:   in.Follow,
		Timestamp: follow.CreatedAt,
		Type:      db.StartedFollowNotificationType,
		Meta:      db.StartedFollowNotificationMeta{Account: in.Account}.ToJson(),
	}

	if err := blog.NotificationStorage.InTx(tx).Insert(notification); err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	blog.Notifier.NotifyStartedFollow(follow.FollowAccount, follow.Account)
	return nil
}

func (blog *Blog) Unfollow(op types.Operation) (rerr *rpc.Error) {
	in := op.(*types.UnfollowOperation)

	tx, err := blog.DB.Write.Beginx()
	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	defer func() {
		if rerr != nil {
			tx.Rollback()
		} else if err := tx.Commit(); err != nil {
			rerr = WrapError(rpc.InternalErrorCode, err)
		}
	}()

	_, err = tx.Exec(
		`DELETE FROM followers
				WHERE account = $1 AND follow_account = $2`, in.Account, in.Unfollow)
	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	notification := db.Notification{
		Account: in.Unfollow,
		Type:    db.StartedFollowNotificationType,
		Meta:    db.StartedFollowNotificationMeta{Account: in.Account}.ToJson(),
	}

	if err := blog.NotificationStorage.InTx(tx).Delete(notification); err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	return nil
}

func (blog *Blog) GetFollowers(ctx *rpc.Context) {
	var account string
	if err := ctx.Param(0, &account); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	var from uint32
	if err := ctx.Param(1, &from); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	var limit uint32
	if err := ctx.Param(2, &limit); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	if limit > maxLargePageSize {
		ctx.WriteError(rpc.InvalidParameterCode, "invalid limit")
		return
	}

	profiles, err := blog.doGetFollowers(account, from, limit)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	ctx.WriteResult(profiles)
}

func (blog *Blog) doGetFollowers(account string, from uint32, limit uint32) ([]*Profile, *rpc.Error) {
	var profiles []*db.Profile
	err := blog.DB.Read.Select(&profiles,
		`SELECT p.account, display_name, location, bio, avatar_url, cover_url, p.created_at
				FROM profiles p INNER JOIN followers f ON p.account = f.account
				WHERE f.follow_account = $1
				ORDER BY f.created_at DESC
				LIMIT $2 OFFSET $3`,
		account, limit, from)
	if err != nil {
		return nil, WrapError(rpc.InternalErrorCode, err)
	}
	return toAPIProfiles(profiles), nil
}

func (blog *Blog) FilterFollowers(ctx *rpc.Context) {
	var account string
	if err := ctx.Param(0, &account); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	var accountsToCheck []string
	if err := ctx.Param(1, &accountsToCheck); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	profiles, err := blog.doFilterFollowers(account, accountsToCheck)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	ctx.WriteResult(profiles)
}

func (blog *Blog) doFilterFollowers(account string, accountsToCheck []string) ([]*Profile, *rpc.Error) {
	var profiles []*db.Profile

	if len(accountsToCheck) == 0 {
		return toAPIProfiles(profiles), nil
	}

	err := blog.DB.Read.Select(&profiles,
		`SELECT p.account, display_name, location, bio, avatar_url, cover_url, p.created_at
		FROM profiles p JOIN  UNNEST($2::TEXT[]) WITH ORDINALITY t(account, ord) USING (account)
		INNER JOIN followers f ON p.account = f.account
		WHERE f.follow_account = $1
		ORDER BY t.ord`,
		account, pq.Array(uniqueStrings(accountsToCheck)))
	if err != nil {
		return nil, WrapError(rpc.InternalErrorCode, err)
	}
	return toAPIProfiles(profiles), nil
}

func (blog *Blog) GetFollowing(ctx *rpc.Context) {
	var account string
	if err := ctx.Param(0, &account); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	var from uint32
	if err := ctx.Param(1, &from); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	var limit uint32
	if err := ctx.Param(2, &limit); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	if limit > maxLargePageSize {
		ctx.WriteError(rpc.InvalidParameterCode, "invalid limit")
	}

	profiles, err := blog.doGetFollowing(account, from, limit)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	ctx.WriteResult(profiles)
}

func (blog *Blog) FilterFollowing(ctx *rpc.Context) {
	var account string
	if err := ctx.Param(0, &account); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	var accountsToCheck []string
	if err := ctx.Param(1, &accountsToCheck); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	profiles, err := blog.doFilterFollowing(account, accountsToCheck)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	ctx.WriteResult(profiles)
}

func (blog *Blog) doGetFollowing(account string, from uint32, limit uint32) ([]*Profile, *rpc.Error) {
	var profiles []*db.Profile
	err := blog.DB.Read.Select(&profiles,
		`SELECT p.account, display_name, location, bio, avatar_url, cover_url, p.created_at
				FROM profiles p INNER JOIN followers f ON p.account = f.follow_account
				WHERE f.account = $1
				ORDER BY f.created_at DESC
				LIMIT $2 OFFSET $3`,
		account, limit, from)
	if err != nil {
		return nil, WrapError(rpc.InternalErrorCode, err)
	}
	return toAPIProfiles(profiles), nil
}

func (blog *Blog) doFilterFollowing(account string, accountsToCheck []string) ([]*Profile, *rpc.Error) {
	var profiles []*db.Profile

	if len(accountsToCheck) == 0 {
		return toAPIProfiles(profiles), nil
	}

	err := blog.DB.Read.Select(&profiles,
		`SELECT p.account, display_name, location, bio, avatar_url, cover_url, p.created_at
		FROM profiles p JOIN  UNNEST($2::TEXT[]) WITH ORDINALITY t(account, ord) USING (account)
		INNER JOIN followers f ON p.account = f.follow_account
		WHERE f.account = $1
		ORDER BY t.ord`,
		account, pq.Array(uniqueStrings(accountsToCheck)))
	if err != nil {
		return nil, WrapError(rpc.InternalErrorCode, err)
	}
	return toAPIProfiles(profiles), nil
}
