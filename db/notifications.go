package db

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"gitlab.scorum.com/blog/core/domain"
)

const (
	StartedFollowNotificationType         NotificationType = "started_follow"
	PostVotedNotificationType             NotificationType = "post_voted"
	CommentVotedNotificationType          NotificationType = "comment_voted"
	PostFlaggedNotificationType           NotificationType = "post_flagged"
	CommentFlaggedNotificationType        NotificationType = "comment_flagged"
	PostRepliedNotificationType           NotificationType = "post_replied"
	CommentRepliedNotificationType        NotificationType = "comment_replied"
	PostUniquenessCheckedNotificationType NotificationType = "post_uniqueness_checked"
)

type NotificationType string

type Notification struct {
	ID        uuid.UUID        `db:"id"`
	Account   string           `db:"account"`
	Timestamp time.Time        `db:"timestamp"`
	IsRead    bool             `db:"is_read"`
	IsSeen    bool             `db:"is_seen"`
	Type      NotificationType `db:"type"`
	Meta      json.RawMessage  `db:"meta"`
}

type NotificationMeta interface {
	sql.Scanner
	driver.Value
	ToJson() json.RawMessage
}

type StartedFollowNotificationMeta struct {
	Account string `json:"account"`
}

type PostRelatedNotificationMeta struct {
	Account      string   `json:"account"`
	Permlink     string   `json:"permlink,omitempty"`
	PostAuthor   string   `json:"post_author,omitempty"`
	PostCategory string   `json:"category,omitempty"`
	PostTitle    string   `json:"post_title,omitempty"`
	PostImage    string   `json:"post_image,omitempty"`
	Domains      []string `json:"domain,omitempty"`
}

func (meta PostRelatedNotificationMeta) PostLink() string {
	return fmt.Sprintf("https://scorum.%s/%s/@%s/%s",
		domain.GetDomainSafe(meta.Domains),
		getCategory([]string{meta.PostCategory}),
		meta.PostAuthor,
		meta.Permlink)
}

func (m StartedFollowNotificationMeta) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.New("type assertion .([]byte) failed.")
	}

	err := json.Unmarshal(source, m)
	if err != nil {
		return err
	}

	return nil
}

func (m StartedFollowNotificationMeta) Value() (driver.Value, error) {
	v, err := json.Marshal(m)
	return v, err
}

func (m StartedFollowNotificationMeta) ToJson() json.RawMessage {
	data, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return data
}

type PlagiarismRelatedNotificationMeta struct {
	PostRelatedNotificationMeta
	Uniqueness float32 `json:"uniqueness"`
	Status     string  `json:"status"`
}

func (m PlagiarismRelatedNotificationMeta) ToJson() json.RawMessage {
	data, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return data
}

func ToStartedFollowNotificationMeta(data json.RawMessage) (*StartedFollowNotificationMeta, error) {
	var meta StartedFollowNotificationMeta

	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

func (m PostRelatedNotificationMeta) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.New("type assertion .([]byte) failed.")
	}

	err := json.Unmarshal(source, m)
	if err != nil {
		return err
	}

	return nil
}

func (m PostRelatedNotificationMeta) Value() (driver.Value, error) {
	v, err := json.Marshal(m)
	return v, err
}

func (m PostRelatedNotificationMeta) ToJson() json.RawMessage {
	data, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return data
}

type NotificationStorage struct {
	ext sqlx.Ext
}

func NewNotificationsStorage(ext sqlx.Ext) *NotificationStorage {
	return &NotificationStorage{ext}
}

func (ns *NotificationStorage) InTx(tx *sqlx.Tx) *NotificationStorage {
	return &NotificationStorage{tx}
}

func (ns *NotificationStorage) GetNotifications(account string, limit int) ([]*Notification, error) {
	var notifications []*Notification
	err := sqlx.Select(ns.ext, &notifications, `
			SELECT id, account, timestamp, is_read, is_seen, type, meta
			FROM notifications
			WHERE account = $1
			ORDER BY timestamp DESC
			LIMIT $2
		`,
		account, limit)

	return notifications, err
}

func (ns *NotificationStorage) MarkAllRead(account string) error {
	_, err := ns.ext.Exec(`UPDATE notifications SET is_read = true WHERE account = $1`, account)
	return err
}

func (ns *NotificationStorage) MarkAllSeen(account string) error {
	_, err := ns.ext.Exec(`UPDATE notifications SET is_seen = true WHERE account = $1`, account)
	return err
}

func (ns *NotificationStorage) MarkRead(account string, id uuid.UUID) error {
	_, err := ns.ext.Exec(`UPDATE notifications SET is_read = true WHERE account = $1 AND id = $2`,
		account, id)
	return err
}

func (ns *NotificationStorage) Insert(notification Notification) error {
	notification.ID = uuid.New()
	_, err := sqlx.NamedExec(ns.ext,
		`
		INSERT INTO notifications(id, account, timestamp, type, meta)
		VALUES(:id, :account, :timestamp, :type, :meta);
		`, notification)
	return err
}

func (ns *NotificationStorage) Delete(notification Notification) error {
	_, err := sqlx.NamedExec(ns.ext, `
		DELETE FROM notifications WHERE account = :account AND type = :type AND meta = :meta;
	`, notification)

	return err
}

func (ns *NotificationStorage) DeletePlagiarismNotification(account, permlink string) error {
	_, err := ns.ext.Exec(fmt.Sprintf(
		`DELETE FROM notifications WHERE account = $1 AND type = 'post_uniqueness_checked' AND meta @> '{"permlink":"%s","account":"%s"}'`,
		permlink, account),
		account,
	)
	return err
}
