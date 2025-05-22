package db

import (
	"testing"
	"time"

	"encoding/json"

	"github.com/stretchr/testify/require"
)

func TestNotificationsStorageGetNotifications(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	registerAccount(t, sheldon)

	ns := NewNotificationsStorage(dbWrite)

	require.Nil(t, ns.Insert(Notification{
		Account:   leonarda,
		Timestamp: time.Now(),
		Type:      StartedFollowNotificationType,
		Meta: StartedFollowNotificationMeta{
			Account: sheldon,
		}.ToJson(),
	}))

	require.Nil(t, ns.Insert(Notification{
		Account:   leonarda,
		Timestamp: time.Now(),
		Type:      PostFlaggedNotificationType,
		Meta: PostRelatedNotificationMeta{
			Account:  sheldon,
			Permlink: "permlink",
		}.ToJson(),
	}))

	notifications, err := ns.GetNotifications(leonarda, 100)
	require.Nil(t, err)
	require.Len(t, notifications, 2)
	require.Equal(t, notifications[0].Type, PostFlaggedNotificationType)
	require.False(t, notifications[0].IsRead)
}

func TestNotificationsStorageMarkRead(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	registerAccount(t, sheldon)

	ns := NewNotificationsStorage(dbWrite)

	require.Nil(t, ns.Insert(Notification{
		Account:   leonarda,
		Timestamp: time.Now(),
		Type:      StartedFollowNotificationType,
		Meta: PostRelatedNotificationMeta{
			Account: sheldon,
		}.ToJson(),
	}))

	notifications, err := ns.GetNotifications(leonarda, 100)
	require.Nil(t, err)
	require.Len(t, notifications, 1)
	require.False(t, notifications[0].IsRead)

	require.Nil(t, ns.MarkRead(leonarda, notifications[0].ID))

	notifications, err = ns.GetNotifications(leonarda, 100)
	require.Nil(t, err)
	require.Len(t, notifications, 1)
	require.True(t, notifications[0].IsRead)
}

func TestNotificationsStorageMarkReadAll(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	registerAccount(t, sheldon)

	ns := NewNotificationsStorage(dbWrite)

	require.Nil(t, ns.Insert(Notification{
		Account:   leonarda,
		Timestamp: time.Now(),
		Type:      StartedFollowNotificationType,
		Meta: PostRelatedNotificationMeta{
			Account: sheldon,
		}.ToJson(),
	}))
	require.Nil(t, ns.Insert(Notification{
		Account:   leonarda,
		Timestamp: time.Now(),
		Type:      PostFlaggedNotificationType,
		Meta: PostRelatedNotificationMeta{
			Account:  sheldon,
			Permlink: "somelink",
		}.ToJson(),
	}))

	require.Nil(t, ns.MarkAllRead(leonarda))

	notifications, err := ns.GetNotifications(leonarda, 100)
	require.Nil(t, err)
	require.Len(t, notifications, 2)
	require.True(t, notifications[0].IsRead)
	require.True(t, notifications[1].IsRead)
}

func TestNotificationsStorageMarkSeenAll(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	registerAccount(t, sheldon)

	ns := NewNotificationsStorage(dbWrite)

	require.Nil(t, ns.Insert(Notification{
		Account:   leonarda,
		Timestamp: time.Now(),
		Type:      StartedFollowNotificationType,
		Meta: PostRelatedNotificationMeta{
			Account: sheldon,
		}.ToJson(),
	}))
	require.Nil(t, ns.Insert(Notification{
		Account:   leonarda,
		Timestamp: time.Now(),
		Type:      PostFlaggedNotificationType,
		Meta: PostRelatedNotificationMeta{
			Account:  sheldon,
			Permlink: "somelink",
		}.ToJson(),
	}))

	// before
	notifications, err := ns.GetNotifications(leonarda, 100)
	require.Nil(t, err)
	require.False(t, notifications[0].IsSeen)
	require.False(t, notifications[1].IsSeen)

	require.Nil(t, ns.MarkAllSeen(leonarda))

	// after
	notifications, err = ns.GetNotifications(leonarda, 100)
	require.Nil(t, err)
	require.Len(t, notifications, 2)
	require.True(t, notifications[0].IsSeen)
	require.True(t, notifications[1].IsSeen)
}

func TestNotificationsStorageDeleteNotifications(t *testing.T) {
	defer cleanUp(t)
	registerAccount(t, leonarda)
	ns := NewNotificationsStorage(dbWrite)

	notification := Notification{
		Account:   leonarda,
		Timestamp: time.Now(),
		Type:      StartedFollowNotificationType,
		Meta: PostRelatedNotificationMeta{
			Account: sheldon,
		}.ToJson(),
	}

	require.Nil(t, ns.Insert(notification))
	notifications, err := ns.GetNotifications(leonarda, 100)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.NoError(t, ns.Delete(notification))
	notifications, err = ns.GetNotifications(leonarda, 100)
	require.NoError(t, err)
	require.Empty(t, notifications)
}

func TestPostLink(t *testing.T) {
	cases := []struct {
		Json string
		Link string
	}{
		{
			`{"domain": ["domain-com"], "account": "sheldon", "category": "categories-baseball", "permlink": "tigran-one-love", "post_title": "Here is 123 malka title post cyclin232323g 24.07.2018 16:00", "post_author": "malka"}`,
			"https://scorum.com/baseball/@malka/tigran-one-love",
		},
		{
			`{"domain": ["domain-com"], "account": "sheldon", "category": "categories-football", "permlink": "iytiuyuytuy-uytduyt-uytuytuyt-uytuytuyt-uytuytuyt-uytuytuyt-uytuytuyt-uytuytuyt", "post_image": "https://cdn-blog.scorum.com/dev/noelle/ed2f98bcaefad619", "post_title": "Знаменитая теннисистка", "post_author": "noelle"}`,
			"https://scorum.com/football/@noelle/iytiuyuytuy-uytduyt-uytuytuyt-uytuytuyt-uytuytuyt-uytuytuyt-uytuytuyt-uytuytuyt",
		},
	}

	for _, c := range cases {
		var meta PostRelatedNotificationMeta
		require.NoError(t, json.Unmarshal([]byte(c.Json), &meta))
		require.Equal(t, c.Link, meta.PostLink())
	}

}

func TestDeletePlagiarismNotification(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)

	ns := NewNotificationsStorage(dbWrite)

	require.Nil(t, ns.Insert(Notification{
		Account:   leonarda,
		Timestamp: time.Now(),
		Type:      PostUniquenessCheckedNotificationType,
		Meta: PostRelatedNotificationMeta{
			Account:  leonarda,
			Permlink: "permlink",
		}.ToJson(),
	}))
	notifications, err := ns.GetNotifications(leonarda, 100)
	require.NoError(t, err)
	require.Len(t, notifications, 1)

	require.NoError(t, ns.DeletePlagiarismNotification(leonarda, "permlink"))
	notifications, err = ns.GetNotifications(leonarda, 100)
	require.NoError(t, err)
	require.Empty(t, notifications)
}
