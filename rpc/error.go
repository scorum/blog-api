package rpc

import "fmt"

const (
	_ = iota
	InternalErrorCode
	RouteNotRegisteredCode
	InvalidRequestCode
	InvalidParameterCode
	ProfileNotFoundCode
	FollowsLimitReachedCode
	ProfileAlreadyFollowedCode
	InvalidMediaCode
	InvalidMediaTypeCode
	MediaNotFoundCode
	MediaAlreadyExistsCode
	AccessDeniedCode
	CategoryAlreadyExistsCode
	ImageTooSmallCode
	CategoryNotFoundCode
	DraftNotFoundCode
	PlagiarismDetailsNotFoundCode
	DownvoteNotFoundCode
	BlacklistEntityNotFoundCode
)

type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}
