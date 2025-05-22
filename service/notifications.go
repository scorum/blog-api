package service

import (
	"encoding/json"

	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/rpc"
)

func (blog *Blog) GetNotifications(ctx *rpc.Context, account string, _ []*json.RawMessage) {
	notifications, err := blog.NotificationStorage.GetNotifications(
		account,
		blog.Config.NotificationsLimit,
	)
	if err != nil {
		ctx.WriteError(rpc.InternalErrorCode, err.Error())
		return
	}

	ctx.WriteResult(toAPINotifications(notifications))
}

func (blog *Blog) MarkRead(op types.Operation) *rpc.Error {
	in := op.(*types.MarkNotificationReadOperation)

	if err := blog.NotificationStorage.MarkRead(in.Account, in.ID); err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	return nil
}

func (blog *Blog) MarkReadAll(op types.Operation) *rpc.Error {
	in := op.(*types.MarkAllNotificationsReadOperation)

	if err := blog.NotificationStorage.MarkAllRead(in.Account); err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	return nil
}

func (blog *Blog) MarkSeenAll(op types.Operation) *rpc.Error {
	in := op.(*types.MarkAllNotificationsSeenOperation)

	if err := blog.NotificationStorage.MarkAllSeen(in.Account); err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	return nil
}
