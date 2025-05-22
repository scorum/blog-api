package types

type OpType string

// Code returns the operation code associated with the given operation type.
func (kind OpType) Code() uint16 {
	return opCodes[kind]
}

// opCodes keeps mapping operation type -> operation code.
var opCodes map[OpType]uint16

func init() {
	opCodes = make(map[OpType]uint16, len(opTypes))
	for i, opType := range opTypes {
		opCodes[opType] = uint16(i)
	}
}

var opTypes = []OpType{
	RegisterOpType,
	UpdateProfileOpType,
	FollowOpType,
	UnfollowOpType,
	UploadMediaOpType,
	"", //MarkPostDeletedOpType is depricated now
	AddToBlacklistAdminOpType,
	RemoveFromBlacklistAdminOpType,
	AddCategoryAdminOpType,
	RemoveCategoryAdminOpType,
	UpdateCategoryAdminOpType,
	SetAccountTrustedAdminOpType,
	UpsertDraftOpType,
	RemoveDraftOpType,
	MarkNotificationReadOpType,
	MarkAllNotificationsReadOpType,
	MarkAllNotificationsSeenOpType,
	UpdateProfileSettingsOpType,
	RegisterPushTokenOpType,
	DownvoteOpType,
	RemoveDownvoteOpType,
}

const (
	RegisterOpType                 OpType = "register"
	FollowOpType                   OpType = "follow"
	UnfollowOpType                 OpType = "unfollow"
	UpdateProfileOpType            OpType = "update_profile"
	UploadMediaOpType              OpType = "upload_media"
	AddToBlacklistAdminOpType      OpType = "add_to_blacklist_admin"
	RemoveFromBlacklistAdminOpType OpType = "remove_from_blacklist_admin"
	AddCategoryAdminOpType         OpType = "add_category_admin"
	RemoveCategoryAdminOpType      OpType = "remove_category_admin"
	UpdateCategoryAdminOpType      OpType = "update_category_admin"
	SetAccountTrustedAdminOpType   OpType = "set_account_trusted_admin"
	UpsertDraftOpType              OpType = "upsert_draft"
	RemoveDraftOpType              OpType = "remove_draft"
	MarkNotificationReadOpType     OpType = "mark_notification_read"
	MarkAllNotificationsReadOpType OpType = "mark_all_notifications_read"
	MarkAllNotificationsSeenOpType OpType = "mark_all_notifications_seen"
	UpdateProfileSettingsOpType    OpType = "update_profile_settings"
	RegisterPushTokenOpType        OpType = "register_push_token"
	DownvoteOpType                 OpType = "downvote"
	RemoveDownvoteOpType           OpType = "remove_downvote"
)
