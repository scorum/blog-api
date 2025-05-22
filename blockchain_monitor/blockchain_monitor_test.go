package blockchain_monitor

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/jmoiron/sqlx"
	"github.com/scorum/event-provider-go/event"
	"github.com/stretchr/testify/require"
	"gitlab.scorum.com/blog/api/common"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/mailer"
	"gitlab.scorum.com/blog/api/push"
	"gitlab.scorum.com/blog/api/service"
	"gitlab.scorum.com/blog/api/utils"
	. "gitlab.scorum.com/blog/core/domain"
)

const (
	leonarda = "leonarda"
	kassie   = "kassie"
	johncena = "johnceeena"
	sheldon  = "sheldon"
	gina     = "gina"
)

const testText = "test test test test test test test test test test test " +
	"test test test test test test test test test test test test"

var dbWrite *sqlx.DB

func TestMain(m *testing.M) {
	utils.DockertestMain(m, func(write, read *sqlx.DB) {
		dbWrite = write
	})
}

func cleanUp(t *testing.T) {
	_, err := dbWrite.Exec("DELETE FROM posts_votes")
	require.NoError(t, err)
	_, err = dbWrite.Exec("DELETE FROM comments")
	require.NoError(t, err)
	_, err = dbWrite.Exec("DELETE FROM notifications")
	require.NoError(t, err)
	_, err = dbWrite.Exec("DELETE FROM deleted_posts")
	require.NoError(t, err)
	_, err = dbWrite.Exec("DELETE FROM posts_plagiarism")
	require.NoError(t, err)
	_, err = dbWrite.Exec("DELETE FROM profiles")
	require.NoError(t, err)
}

func createAntiPlagiarismService() *service.AntiPlagiarism {
	return service.NewAntiPlagiarismService("68c796999f45120ca4a4a420fa59327f",
		db.NewPlagiarismStorage(dbWrite),
		db.NewCommentsStorage(dbWrite),
	)

}

func TestProcessPost(t *testing.T) {
	defer cleanUp(t)

	const permlink = "perm"

	_, err := dbWrite.Exec(
		`INSERT INTO profiles(account, display_name) VALUES($1, $2)`,
		leonarda, leonarda)
	require.NoError(t, err)

	_, err = dbWrite.Exec(
		`INSERT INTO profiles(account, display_name) VALUES($1, $2)`,
		kassie, kassie)
	require.NoError(t, err)

	tx, err := dbWrite.Beginx()
	require.NoError(t, err)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	notifierMock := push.NewMockNotifier(mockCtrl)
	notifierMock.EXPECT().NotifyPostVoted(gomock.Any()).Times(1)

	bm := &BlockchainMonitor{
		DB:                  dbWrite,
		Plagiarism:          createAntiPlagiarismService(),
		PushNotifier:        notifierMock,
		NotificationStorage: db.NewNotificationsStorage(dbWrite),
		DownvotesStorage:    db.NewDownvotesStorage(dbWrite),
		CommentsStorage:     db.NewCommentsStorage(dbWrite),
		PlagiarismStorage:   db.NewPlagiarismStorage(dbWrite),
		MailerClient:        &mailer.Client{},
	}

	metadata := common.JsonMetadata{
		Domains: []string{
			"domain-com",
		},
		Categories: []string{
			"categories-pi",
		},
	}

	metadataBytes, err := json.Marshal(metadata)
	require.NoError(t, err)

	err = bm.processPost(event.PostEvent{
		CommonEvent: event.CommonEvent{
			BlockNum:  1,
			BlockID:   "abc",
			Timestamp: time.Now(),
		},
		PermLink:       permlink,
		ParentPermLink: "1232",
		Author:         leonarda,
		Body:           testText,
		JsonMetadata:   string(metadataBytes),
		Title:          "some title",
	}, tx)

	require.NoError(t, err)

	ve := event.VoteEvent{
		CommonEvent: event.CommonEvent{
			BlockID:   "312312",
			BlockNum:  2,
			Timestamp: time.Now(),
		},
		Voter:    kassie,
		Author:   leonarda,
		PermLink: permlink,
		Weight:   100,
	}
	require.NoError(t, bm.processVote(ve, tx))

	require.NoError(t, tx.Commit())

	var domain string
	require.NoError(t, dbWrite.Get(&domain, `SELECT domain FROM comments WHERE author = $1 AND permlink = $2`, leonarda, permlink))
	require.Equal(t, string(DomainCom), domain)
}

func TestProcessComment(t *testing.T) {
	defer cleanUp(t)

	const permlink = "perm"
	_, err := dbWrite.Exec(
		`INSERT INTO profiles(account, display_name) VALUES($1, $2)`,
		leonarda, leonarda)
	require.NoError(t, err)

	tx, err := dbWrite.Beginx()
	require.NoError(t, err)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	notifierMock := push.NewMockNotifier(mockCtrl)
	notifierMock.EXPECT().NotifyPostReplied(gomock.Any()).Times(1)
	notifierMock.EXPECT().NotifyCommentReplied(gomock.Any(), gomock.Any()).Times(1).Do(
		func(to interface{}, meta interface{}) {
			require.EqualValues(t, to, gina)
		},
	)

	bm := &BlockchainMonitor{
		DB:                  dbWrite,
		Plagiarism:          createAntiPlagiarismService(),
		PushNotifier:        notifierMock,
		CommentsStorage:     db.NewCommentsStorage(dbWrite),
		NotificationStorage: db.NewNotificationsStorage(dbWrite),
		PlagiarismStorage:   db.NewPlagiarismStorage(dbWrite),
		MailerClient:        &mailer.Client{},
	}

	metadata := common.JsonMetadata{
		Domains: []string{
			"domain-com",
		},
		Categories: []string{
			"categories-pi",
		},
	}

	metadataBytes, err := json.Marshal(metadata)
	require.NoError(t, err)

	err = bm.processPost(event.PostEvent{
		CommonEvent: event.CommonEvent{
			BlockNum:  1,
			BlockID:   "abc",
			Timestamp: time.Now(),
		},
		PermLink:       permlink,
		ParentPermLink: "1232",
		Author:         leonarda,
		Body:           testText,
		JsonMetadata:   string(metadataBytes),
		Title:          "some title",
	}, tx)

	require.NoError(t, err)

	_, err = dbWrite.Exec(
		`INSERT INTO profiles(account, display_name) VALUES($1, $2)`,
		gina, gina)
	require.NoError(t, err)

	commentPermlink := "bc4e25fd2609b5c343db2f877e8ba97f"

	err = bm.processComment(event.CommentEvent{
		CommonEvent: event.CommonEvent{
			BlockID:   "abs",
			BlockNum:  1,
			Timestamp: time.Now(),
		},
		PermLink:       commentPermlink,
		ParentAuthor:   leonarda,
		ParentPermLink: permlink,
		Author:         gina,
		Body:           "gesagt",
		JsonMetadata:   string(metadataBytes),
		Title:          permlink,
	}, tx)
	require.NoError(t, err)
	err = bm.processComment(event.CommentEvent{
		CommonEvent: event.CommonEvent{
			BlockID:   "abs",
			BlockNum:  1,
			Timestamp: time.Now(),
		},
		PermLink:       "bc4e25fd2609b5c343db2f877e8ba933",
		ParentAuthor:   gina,
		ParentPermLink: commentPermlink,
		Author:         leonarda,
		Body:           "gesagt2",
		JsonMetadata:   string(metadataBytes),
		Title:          permlink,
	}, tx)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	n, err := bm.NotificationStorage.GetNotifications(leonarda, 100)
	require.NoError(t, err)
	require.NotEmpty(t, n)
}

func TestSaveVote(t *testing.T) {
	defer cleanUp(t)

	_, err := dbWrite.Exec(
		`INSERT INTO profiles(account, display_name) VALUES($1, $2)`,
		kassie, kassie)
	require.NoError(t, err)

	_, err = dbWrite.Exec(
		`INSERT INTO profiles(account, display_name) VALUES($1, $2)`,
		johncena, johncena)

	tx, err := dbWrite.Beginx()
	require.NoError(t, err)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	bm := &BlockchainMonitor{
		DB:                  dbWrite,
		Plagiarism:          createAntiPlagiarismService(),
		PushNotifier:        push.NewMockNotifier(mockCtrl),
		NotificationStorage: db.NewNotificationsStorage(dbWrite),
		DownvotesStorage:    db.NewDownvotesStorage(dbWrite),
		PlagiarismStorage:   db.NewPlagiarismStorage(dbWrite),
		MailerClient:        &mailer.Client{},
	}

	ve := event.VoteEvent{
		CommonEvent: event.CommonEvent{
			BlockID:   "312312",
			BlockNum:  2,
			Timestamp: time.Now(),
		},
		Voter:    kassie,
		Author:   johncena,
		PermLink: "perm",
		Weight:   100,
	}

	require.NoError(t, bm.saveVote(ve, tx))
	require.NoError(t, bm.deleteVote(ve.Voter, ve.PermLink, tx))
	require.NoError(t, tx.Commit())
}

func TestDeletedPost(t *testing.T) {
	defer cleanUp(t)

	const permlink = "perm"

	_, err := dbWrite.Exec(
		`INSERT INTO profiles(account, display_name) VALUES($1, $2)`,
		sheldon, sheldon)
	require.NoError(t, err)

	tx, err := dbWrite.Beginx()
	require.NoError(t, err)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	bm := &BlockchainMonitor{
		DB:                  dbWrite,
		Plagiarism:          createAntiPlagiarismService(),
		PushNotifier:        push.NewMockNotifier(mockCtrl),
		NotificationStorage: db.NewNotificationsStorage(dbWrite),
		PlagiarismStorage:   db.NewPlagiarismStorage(dbWrite),
		MailerClient:        &mailer.Client{},
	}

	err = bm.processPost(event.PostEvent{
		PermLink:     permlink,
		Author:       sheldon,
		Title:        "some title",
		Body:         testText,
		JsonMetadata: "{}",
	}, tx)
	require.NoError(t, err)

	require.Nil(t, bm.NotificationStorage.Insert(db.Notification{
		Account:   sheldon,
		Timestamp: time.Now(),
		Type:      db.PostUniquenessCheckedNotificationType,
		Meta: db.PostRelatedNotificationMeta{
			Account:  sheldon,
			Permlink: permlink,
		}.ToJson(),
	}))

	err = bm.deleteComment(event.DeleteCommentEvent{
		PermLink: permlink,
		Author:   sheldon,
	}, tx)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	nots, err := bm.NotificationStorage.GetNotifications(sheldon, 100)
	require.NoError(t, err)
	require.Empty(t, nots)
}

func TestCheckPlagiarismAndNotify(t *testing.T) {
	t.Skip()
	defer cleanUp(t)

	_, err := dbWrite.Exec(
		`INSERT INTO profiles(account, display_name) VALUES($1, $2)`,
		gina, gina)
	require.NoError(t, err)

	bm := &BlockchainMonitor{
		DB:                  dbWrite,
		Plagiarism:          createAntiPlagiarismService(),
		NotificationStorage: db.NewNotificationsStorage(dbWrite),
		PlagiarismStorage:   db.NewPlagiarismStorage(dbWrite),
		MailerClient:        &mailer.Client{},
	}

	c := db.Comment{
		Permlink: "test",
		Author:   gina,
		Body:     testText,
		Title:    "test",
		JsonMetadata: common.JsonMetadata{
			Domains:    []string{"com"},
			Categories: []string{"category"},
			Image:      "test",
		},
		UpdatedAt: time.Now(),
		CreatedAt: time.Now(),
	}
	bm.checkPlagiarismAndNotify(c, DomainCom)
	notifications, err := bm.NotificationStorage.GetNotifications(gina, 100)
	require.NoError(t, err)
	require.NotEmpty(t, notifications)
}
