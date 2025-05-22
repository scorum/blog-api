package service

import (
	"testing"

	"time"

	"database/sql"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/common"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/push"
	. "gitlab.scorum.com/blog/core/domain"
)

func TestBlog_GetPostsFromNetwork(t *testing.T) {
	defer cleanUp(t)

	require.Nil(t, handler.Register(&types.RegisterOperation{leonarda}))
	require.Nil(t, handler.Register(&types.RegisterOperation{kristie}))
	require.Nil(t, handler.Register(&types.RegisterOperation{sheldon}))

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

	// kristie follows both leonarda and sheldon
	insertPost(t, leonarda, "post 1", DomainCom)
	insertPost(t, leonarda, "post 2", DomainCom)

	insertPost(t, sheldon, "post 1", DomainCom)
	insertPost(t, sheldon, "post 2", DomainCom)
	insertPost(t, sheldon, "post 3", DomainCom)

	posts, err := handler.doGetPostsFromNetwork(kristie, DomainCom, 0, 100)
	require.Nil(t, err)
	require.Len(t, posts, 5)

	// no posts on domain me
	posts, err = handler.doGetPostsFromNetwork(kristie, DomainMe, 0, 100)
	require.Nil(t, err)
	require.Empty(t, posts)

	// blacklist
	require.Nil(t, handler.AddToBlacklistAdmin(&types.AddToBlacklistAdminOperation{
		Account:     leonarda,
		BlogAccount: sheldon,
		Permlink:    "post 3",
	}))

	posts, err = handler.doGetPostsFromNetwork(kristie, DomainCom, 0, 100)
	require.Nil(t, err)
	require.Len(t, posts, 4)

	// unfollow
	require.Nil(t, handler.Unfollow(&types.UnfollowOperation{
		Account:  kristie,
		Unfollow: leonarda,
	}))

	posts, err = handler.doGetPostsFromNetwork(kristie, DomainCom, 0, 100)
	require.Nil(t, err)
	require.Len(t, posts, 2)

	require.Nil(t, handler.Unfollow(&types.UnfollowOperation{
		Account:  kristie,
		Unfollow: sheldon,
	}))

	posts, err = handler.doGetPostsFromNetwork(kristie, DomainCom, 0, 100)
	require.Nil(t, err)
	require.Empty(t, posts)
}

func insertPost(t *testing.T, author, permlink string, domain Domain) {
	post := db.Comment{
		Permlink:       permlink,
		ParentPermlink: sql.NullString{Valid: true, String: "soccer"},
		Author:         author,
		Body:           "body",
		Title:          "title",
		Domain:         sql.NullString{Valid: true, String: string(domain)},
		JsonMetadata:   common.JsonMetadata{},
		UpdatedAt:      time.Now(),
		CreatedAt:      time.Now(),
	}

	_, err := dbWrite.NamedExec(
		`INSERT INTO comments
				(permlink, author, body, title, json_metadata, parent_permlink, domain, updated_at, created_at)
		       VALUES
                (:permlink, :author, :body, :title, :json_metadata, :parent_permlink, :domain, :updated_at, :created_at)
		`, post)

	require.NoError(t, err)

}
