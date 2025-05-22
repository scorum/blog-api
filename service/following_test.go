package service

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/push"
)

func TestBlog_Following(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	registerAccount(t, kristie)
	registerAccount(t, sheldon)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	notifier := push.NewMockNotifier(mockCtrl)
	handler.Notifier = notifier
	notifier.EXPECT().NotifyStartedFollow(gomock.Any(), gomock.Any()).Times(3)

	require.Nil(t, handler.Follow(&types.FollowOperation{
		Account: leonarda,
		Follow:  kristie,
	}))

	// check is kristie notified
	notifications, rerr := handler.NotificationStorage.GetNotifications(kristie, 100)
	require.Nil(t, rerr)
	require.Len(t, notifications, 1)
	require.Equal(t, notifications[0].Account, kristie)
	require.False(t, notifications[0].IsRead)
	require.Equal(t, notifications[0].Type, db.StartedFollowNotificationType)
	meta, err := db.ToStartedFollowNotificationMeta(notifications[0].Meta)
	require.NoError(t, err)
	require.Equal(t, meta.Account, leonarda)

	require.Nil(t, handler.Follow(&types.FollowOperation{
		Account: sheldon,
		Follow:  kristie,
	}))

	followers, err := handler.doGetFollowers(kristie, 0, 10)
	require.Nil(t, err)
	require.Len(t, followers, 2)

	require.Nil(t, handler.Follow(&types.FollowOperation{
		Account: kristie,
		Follow:  leonarda,
	}))

	handler.Config.MaxFollow = 3
	require.NotNil(t, handler.Follow(&types.FollowOperation{
		Account: kristie,
		Follow:  leonarda,
	}))
	handler.Config.MaxFollow = 1000

	profile, err := handler.doGetProfile(kristie)
	require.Nil(t, err)
	require.Equal(t, profile.FollowersCount, int64(2))
	require.Equal(t, profile.FollowingCount, int64(1))

	following, err := handler.doGetFollowing(leonarda, 0, 10)
	require.Nil(t, err)
	require.Len(t, following, 1)
	require.Equal(t, kristie, following[0].Account)

	require.Nil(t, handler.Unfollow(&types.UnfollowOperation{
		Account:  leonarda,
		Unfollow: kristie,
	}))

	require.Nil(t, handler.Unfollow(&types.UnfollowOperation{
		Account:  sheldon,
		Unfollow: kristie,
	}))

	require.Nil(t, handler.Unfollow(&types.UnfollowOperation{
		Account:  kristie,
		Unfollow: leonarda,
	}))

	require.Nil(t, handler.Unfollow(&types.UnfollowOperation{
		Account:  kristie,
		Unfollow: sheldon,
	}))
}

func TestBlog_FilterFollowing(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	registerAccount(t, kristie)
	registerAccount(t, sheldon)

	t.Run("filter_following", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		notifier := push.NewMockNotifier(mockCtrl)
		handler.Notifier = notifier
		notifier.EXPECT().NotifyStartedFollow(gomock.Any(), gomock.Any()).Times(2)

		require.Nil(t, handler.Follow(&types.FollowOperation{
			Account: kristie,
			Follow:  leonarda,
		}))

		require.Nil(t, handler.Follow(&types.FollowOperation{
			Account: kristie,
			Follow:  sheldon,
		}))

		filteredFolllowing, err := handler.doFilterFollowing(kristie, []string{sheldon, leonarda, "lex"})
		require.Nil(t, err)
		require.Len(t, filteredFolllowing, 2)

		require.Equal(t, filteredFolllowing[0].Account, sheldon)
		require.Equal(t, filteredFolllowing[1].Account, leonarda)
	})

	t.Run("filter_following_empty_list", func(t *testing.T) {
		filteredFolllowing, err := handler.doFilterFollowing(kristie, []string{})
		require.Nil(t, err)
		require.Len(t, filteredFolllowing, 0)
	})
}

func TestBlog_FilterFollowers(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	registerAccount(t, kristie)
	registerAccount(t, sheldon)

	t.Run("filter_followers", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		notifier := push.NewMockNotifier(mockCtrl)
		handler.Notifier = notifier
		notifier.EXPECT().NotifyStartedFollow(gomock.Any(), gomock.Any()).Times(2)

		require.Nil(t, handler.Follow(&types.FollowOperation{
			Account: kristie,
			Follow:  leonarda,
		}))

		require.Nil(t, handler.Follow(&types.FollowOperation{
			Account: sheldon,
			Follow:  leonarda,
		}))

		filteredFollowers, err := handler.doFilterFollowers(leonarda, []string{kristie, sheldon, "lex"})
		require.Nil(t, err)
		require.Len(t, filteredFollowers, 2)

		require.Equal(t, filteredFollowers[0].Account, kristie)
		require.Equal(t, filteredFollowers[1].Account, sheldon)
	})

	t.Run("filter_followers_empty_list", func(t *testing.T) {
		filteredFolllowers, err := handler.doFilterFollowers(leonarda, []string{})
		require.Nil(t, err)
		require.Len(t, filteredFolllowers, 0)
	})
}

func TestBlog_Following_ValidationTest(t *testing.T) {
	defer cleanUp(t)

	followOp := types.FollowOperation{
		Account: leonarda,
		Follow:  kristie,
	}
	unfollowOp := types.UnfollowOperation{
		Account:  leonarda,
		Unfollow: kristie,
	}

	t.Run("valid_operation_follow", func(t *testing.T) {
		require.NoError(t, validate.Struct(followOp))
	})

	t.Run("empty_account_follow", func(t *testing.T) {
		cop := followOp
		cop.Account = ""
		require.Error(t, validate.Struct(cop))
	})

	t.Run("empty_follow", func(t *testing.T) {
		cop := followOp
		cop.Follow = ""
		require.Error(t, validate.Struct(cop))
	})

	t.Run("account_equals_follow", func(t *testing.T) {
		cop := followOp
		cop.Account = cop.Follow
		require.Error(t, validate.Struct(cop))
	})

	t.Run("valid_operation_unfollow", func(t *testing.T) {
		require.NoError(t, validate.Struct(unfollowOp))
	})

	t.Run("empty_account_unfollow", func(t *testing.T) {
		cop := unfollowOp
		cop.Account = ""
		require.Error(t, validate.Struct(cop))
	})

	t.Run("empty_unfollow", func(t *testing.T) {
		cop := unfollowOp
		cop.Unfollow = ""
		require.Error(t, validate.Struct(cop))
	})

	t.Run("account_equals_unfollow", func(t *testing.T) {
		cop := unfollowOp
		cop.Account = cop.Unfollow
		require.Error(t, validate.Struct(cop))
	})
}
