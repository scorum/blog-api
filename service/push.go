package service

import (
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/rpc"
)

func (blog *Blog) RegisterPushToken(op types.Operation) *rpc.Error {
	in := op.(*types.RegisterPushTokenOperation)

	err := blog.PushRegistrationStorage.Add(in.Account, in.Token)
	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	return nil
}
