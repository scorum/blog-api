package types

import (
	"encoding/json"
	"reflect"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/scorum/scorum-go/encoding/transaction"
	"gitlab.scorum.com/blog/api/common"
)

type Operation interface {
	// Type returns unique operation type
	Type() OpType
	// GetAccount return account name
	GetAccount() string
}

type Operations []Operation

type operationTuple struct {
	Type OpType
	Data Operation
}

func (op *operationTuple) UnmarshalJSON(data []byte) error {
	// The operation object is [opType, opBody].
	raw := make([]*json.RawMessage, 2)
	if err := json.Unmarshal(data, &raw); err != nil {
		return errors.Wrapf(err, "failed to unmarshal operation object: %v", string(data))
	}
	if len(raw) != 2 {
		return errors.Errorf("invalid operation object: %v", string(data))
	}

	// Unmarshal the type.
	var opType OpType
	if err := json.Unmarshal(*raw[0], &opType); err != nil {
		return errors.Wrapf(err, "failed to unmarshal Operation.Type: %v", string(*raw[0]))
	}

	// Unmarshal the data.
	var opData Operation
	template, ok := templates[opType]
	if ok {
		opData = reflect.New(template).Interface().(Operation)
		if err := json.Unmarshal(*raw[1], opData); err != nil {
			return errors.Wrapf(err, "failed to unmarshal Operation.Data: %v", string(*raw[1]))
		}
	} else {
		opData = &UnknownOperation{opType, raw[1]}
	}

	// Update fields.
	op.Type = opType
	op.Data = opData
	return nil
}

func (ops *Operations) UnmarshalJSON(data []byte) (err error) {
	var tuples []*operationTuple
	if err := json.Unmarshal(data, &tuples); err != nil {
		return err
	}

	items := make([]Operation, 0, len(tuples))
	for _, tuple := range tuples {
		items = append(items, tuple.Data)
	}

	*ops = items
	return nil
}

var templates = map[OpType]reflect.Type{
	RegisterOpType:                 reflect.TypeOf(RegisterOperation{}),
	RegisterPushTokenOpType:        reflect.TypeOf(RegisterPushTokenOperation{}),
	UpdateProfileOpType:            reflect.TypeOf(UpdateProfileOperation{}),
	FollowOpType:                   reflect.TypeOf(FollowOperation{}),
	UnfollowOpType:                 reflect.TypeOf(UnfollowOperation{}),
	UploadMediaOpType:              reflect.TypeOf(UploadMediaOperation{}),
	AddToBlacklistAdminOpType:      reflect.TypeOf(AddToBlacklistAdminOperation{}),
	RemoveFromBlacklistAdminOpType: reflect.TypeOf(RemoveFromBlacklistAdminOperation{}),
	AddCategoryAdminOpType:         reflect.TypeOf(AddCategoryAdminOperation{}),
	RemoveCategoryAdminOpType:      reflect.TypeOf(RemoveCategoryAdminOperation{}),
	UpdateCategoryAdminOpType:      reflect.TypeOf(UpdateCategoryAdminOperation{}),
	SetAccountTrustedAdminOpType:   reflect.TypeOf(SetAccountTrustedAdminOperation{}),
	UpsertDraftOpType:              reflect.TypeOf(UpsertDraftOperation{}),
	RemoveDraftOpType:              reflect.TypeOf(RemoveDraftOperation{}),
	MarkNotificationReadOpType:     reflect.TypeOf(MarkNotificationReadOperation{}),
	MarkAllNotificationsReadOpType: reflect.TypeOf(MarkAllNotificationsReadOperation{}),
	MarkAllNotificationsSeenOpType: reflect.TypeOf(MarkAllNotificationsSeenOperation{}),
	UpdateProfileSettingsOpType:    reflect.TypeOf(UpdateProfileSettingsOperation{}),
	DownvoteOpType:                 reflect.TypeOf(DownvoteOperation{}),
	RemoveDownvoteOpType:           reflect.TypeOf(RemoveDownvoteOperation{}),
}

// UnknownOperation
type UnknownOperation struct {
	kind OpType
	Data *json.RawMessage
}

func (op *UnknownOperation) Type() OpType       { return op.kind }
func (op *UnknownOperation) GetAccount() string { panic("not implemented") }

// FollowOperation
type FollowOperation struct {
	Account string `validate:"required"`
	Follow  string `validate:"required,nefield=Account"`
}

func (op *FollowOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	enc.Encode(op.Follow)
	return enc.Err()
}

func (op *FollowOperation) Type() OpType {
	return FollowOpType
}

func (op *FollowOperation) GetAccount() string { return op.Account }

// UnfollowOperation
type UnfollowOperation struct {
	Account  string `validate:"required"`
	Unfollow string `validate:"required,nefield=Account"`
}

func (op *UnfollowOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	enc.Encode(op.Unfollow)
	return enc.Err()
}

func (op *UnfollowOperation) Type() OpType {
	return UnfollowOpType
}

func (op *UnfollowOperation) GetAccount() string { return op.Account }

// RegisterOperation
type RegisterOperation struct {
	Account string `json:"account" validate:"required"`
}

func (op *RegisterOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	return enc.Err()
}

func (op *RegisterOperation) Type() OpType {
	return RegisterOpType
}

func (op *RegisterOperation) GetAccount() string { return op.Account }

// RegisterPushTokenOperation
type RegisterPushTokenOperation struct {
	Account string `json:"account" validate:"required"`
	Token   string `json:"token" validate:"required"`
}

func (op *RegisterPushTokenOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	enc.Encode(op.Token)
	return enc.Err()
}

func (op *RegisterPushTokenOperation) Type() OpType {
	return RegisterPushTokenOpType
}

func (op *RegisterPushTokenOperation) GetAccount() string { return op.Account }

// UpdateProfileOperation
type UpdateProfileOperation struct {
	Account     string `json:"account" validate:"required"`
	DisplayName string `json:"display_name" validate:"omitempty,max=50"`
	Location    string `json:"location" validate:"omitempty,max=25"`
	Bio         string `json:"bio" validate:"omitempty,max=160"`
	AvatarUrl   string `json:"avatar_url" validate:"omitempty,uri"`
	CoverUrl    string `json:"cover_url" validate:"omitempty,uri"`
}

func (op *UpdateProfileOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	enc.Encode(op.DisplayName)
	enc.Encode(op.Location)
	enc.Encode(op.Bio)
	enc.Encode(op.AvatarUrl)
	enc.Encode(op.CoverUrl)
	return enc.Err()
}

func (op *UpdateProfileOperation) Type() OpType {
	return UpdateProfileOpType
}

func (op *UpdateProfileOperation) GetAccount() string { return op.Account }

// UploadMediaOperation
type UploadMediaOperation struct {
	Account string `json:"account" validate:"required"`
	// Unique media ID
	ID string `json:"id" validate:"required,max=16,alphanum"`
	// Content as base64 string
	Media string `json:"media" validate:"base64,required"`

	ContentType common.ContentType `json:"content_type" validate:"required"`
}

func (op *UploadMediaOperation) Type() OpType {
	return UploadMediaOpType
}

func (op *UploadMediaOperation) GetAccount() string { return op.Account }

func (op *UploadMediaOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	enc.Encode(op.ID)
	enc.Encode(op.Media)
	enc.Encode(string(op.ContentType))
	return enc.Err()
}

// AddToBlacklistAdminOperation
type AddToBlacklistAdminOperation struct {
	Account     string `json:"account" validate:"required"`
	BlogAccount string `json:"blog_account" validate:"required"`
	Permlink    string `json:"permlink" validate:"required"`
}

func (op *AddToBlacklistAdminOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	enc.Encode(op.BlogAccount)
	enc.Encode(op.Permlink)
	return enc.Err()
}

func (op *AddToBlacklistAdminOperation) Type() OpType {
	return AddToBlacklistAdminOpType
}

func (op *AddToBlacklistAdminOperation) GetAccount() string { return op.Account }

// RemoveFromBlacklistAdminOperation
type RemoveFromBlacklistAdminOperation struct {
	Account     string `json:"account" validate:"required"`
	BlogAccount string `json:"blog_account" validate:"required"`
	Permlink    string `json:"permlink" validate:"required"`
}

func (op *RemoveFromBlacklistAdminOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	enc.Encode(op.BlogAccount)
	enc.Encode(op.Permlink)
	return enc.Err()
}

func (op *RemoveFromBlacklistAdminOperation) Type() OpType {
	return RemoveFromBlacklistAdminOpType
}

func (op *RemoveFromBlacklistAdminOperation) GetAccount() string { return op.Account }

// AddCategoryAdminOperation
type AddCategoryAdminOperation struct {
	Account         string `json:"account" validate:"required"`
	Domain          string `json:"domain" validate:"required"`
	Label           string `json:"label" validate:"required"`
	LocalizationKey string `json:"localization_key" validate:"required"`
}

func (op *AddCategoryAdminOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	enc.Encode(op.Domain)
	enc.Encode(op.Label)
	enc.Encode(op.LocalizationKey)
	return enc.Err()
}

func (op *AddCategoryAdminOperation) Type() OpType {
	return AddCategoryAdminOpType
}

func (op *AddCategoryAdminOperation) GetAccount() string { return op.Account }

// RemoveCategoryAdminOperation
type RemoveCategoryAdminOperation struct {
	Account string `json:"account" validate:"required"`
	Domain  string `json:"domain" validate:"required"`
	Label   string `json:"label" validate:"required"`
}

func (op *RemoveCategoryAdminOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	enc.Encode(op.Domain)
	enc.Encode(op.Label)
	return enc.Err()
}

func (op *RemoveCategoryAdminOperation) Type() OpType {
	return RemoveCategoryAdminOpType
}

func (op *RemoveCategoryAdminOperation) GetAccount() string { return op.Account }

// UpdateCategoryAdminOperation
type UpdateCategoryAdminOperation struct {
	Account         string `json:"account" validate:"required"`
	Domain          string `json:"domain" validate:"required"`
	Label           string `json:"label" validate:"required"`
	Order           uint32 `json:"order"`
	LocalizationKey string `json:"localization_key" validate:"required"`
}

func (op *UpdateCategoryAdminOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	enc.Encode(op.Domain)
	enc.Encode(op.Label)
	enc.Encode(op.Order)
	enc.Encode(op.LocalizationKey)
	return enc.Err()
}

func (op *UpdateCategoryAdminOperation) Type() OpType {
	return UpdateCategoryAdminOpType
}

func (op *UpdateCategoryAdminOperation) GetAccount() string { return op.Account }

// SetAccountTrustedAdminOperation
type SetAccountTrustedAdminOperation struct {
	Account     string `json:"account" validate:"required"`
	BlogAccount string `json:"blog_account" validate:"required"`
	IsTrusted   bool   `json:"is_trusted"`
}

func (op *SetAccountTrustedAdminOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	enc.Encode(op.BlogAccount)
	enc.EncodeBool(op.IsTrusted)
	return enc.Err()
}

func (op *SetAccountTrustedAdminOperation) Type() OpType {
	return SetAccountTrustedAdminOpType
}

func (op *SetAccountTrustedAdminOperation) GetAccount() string { return op.Account }

// UpsertDraftOperation
type UpsertDraftOperation struct {
	Account string `json:"account" validate:"required"`
	// Unique draft ID
	ID string `json:"id" validate:"required,max=16,alphanum"`

	Title string `json:"title" validate:"max=255"`
	Body  string `json:"body" validate:"max=45000"`

	JsonMetadata string `json:"json_metadata"`
}

func (op *UpsertDraftOperation) Type() OpType {
	return UpsertDraftOpType
}

func (op *UpsertDraftOperation) GetAccount() string { return op.Account }

func (op *UpsertDraftOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	enc.Encode(op.ID)
	enc.Encode(op.Title)
	enc.Encode(op.Body)
	enc.Encode(op.JsonMetadata)
	return enc.Err()
}

// RemoveDraftOperation
type RemoveDraftOperation struct {
	Account string `json:"account" validate:"required"`
	// Unique draft ID
	ID string `json:"id" validate:"required,max=16,alphanum"`
}

func (op *RemoveDraftOperation) Type() OpType {
	return RemoveDraftOpType
}

func (op *RemoveDraftOperation) GetAccount() string { return op.Account }

func (op *RemoveDraftOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	enc.Encode(op.ID)
	return enc.Err()
}

// MarkNotificationReadOperation
type MarkNotificationReadOperation struct {
	Account string `json:"account" validate:"required"`
	// Notification ID
	ID uuid.UUID `json:"id" validate:"required"`
}

func (op *MarkNotificationReadOperation) Type() OpType {
	return MarkNotificationReadOpType
}

func (op *MarkNotificationReadOperation) GetAccount() string { return op.Account }

func (op *MarkNotificationReadOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	enc.Encode(op.ID.String())
	return enc.Err()
}

// MarkAllNotificationsReadOperation
type MarkAllNotificationsReadOperation struct {
	Account string `json:"account" validate:"required"`
}

func (op *MarkAllNotificationsReadOperation) Type() OpType {
	return MarkAllNotificationsReadOpType
}

func (op *MarkAllNotificationsReadOperation) GetAccount() string { return op.Account }

func (op *MarkAllNotificationsReadOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	return enc.Err()
}

// MarkAllNotificationsSeenOperation
type MarkAllNotificationsSeenOperation struct {
	Account string `json:"account" validate:"required"`
}

func (op *MarkAllNotificationsSeenOperation) Type() OpType {
	return MarkAllNotificationsSeenOpType
}

func (op *MarkAllNotificationsSeenOperation) GetAccount() string { return op.Account }

func (op *MarkAllNotificationsSeenOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	return enc.Err()
}

// UpdateProfileSettingsOperation
type UpdateProfileSettingsOperation struct {
	Account                        string `json:"account" validate:"required"`
	EnableEmailUnseenNotifications bool   `json:"enable_email_unseen_notifications"`
}

func (op *UpdateProfileSettingsOperation) Type() OpType {
	return UpdateProfileSettingsOpType
}

func (op *UpdateProfileSettingsOperation) GetAccount() string { return op.Account }

func (op *UpdateProfileSettingsOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	enc.EncodeBool(op.EnableEmailUnseenNotifications)
	return enc.Err()
}

type DownvoteOperation struct {
	Account  string `json:"account" validate:"required"`
	Author   string `json:"author" validate:"required"`
	Permlink string `json:"permlink" validate:"required"`
	Reason   string `json:"reason" validate:"required"`
	Comment  string `json:"comment" validate:"omitempty,max=500"`
}

func (op *DownvoteOperation) Type() OpType {
	return DownvoteOpType
}

func (op *DownvoteOperation) GetAccount() string { return op.Account }

func (op *DownvoteOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	enc.Encode(op.Author)
	enc.Encode(op.Permlink)
	enc.Encode(op.Reason)
	enc.Encode(op.Comment)
	return enc.Err()
}

type RemoveDownvoteOperation struct {
	Account  string `json:"account" validate:"required"`
	Author   string `json:"author" validate:"required"`
	Permlink string `json:"permlink" validate:"required"`
}

func (op *RemoveDownvoteOperation) Type() OpType {
	return RemoveDownvoteOpType
}

func (op *RemoveDownvoteOperation) GetAccount() string { return op.Account }

func (op *RemoveDownvoteOperation) MarshalTransaction(encoder *transaction.Encoder) error {
	enc := transaction.NewRollingEncoder(encoder)
	enc.EncodeUVarint(uint64(op.Type().Code()))
	enc.Encode(op.Account)
	enc.Encode(op.Author)
	enc.Encode(op.Permlink)
	return enc.Err()
}
