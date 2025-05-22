package blockchain_monitor

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/scorum/event-provider-go/event"
	"github.com/scorum/event-provider-go/provider"
	log "github.com/sirupsen/logrus"
	"gitlab.scorum.com/blog/api/common"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/mailer"
	"gitlab.scorum.com/blog/api/push"
	"gitlab.scorum.com/blog/api/service"
	. "gitlab.scorum.com/blog/core/domain"
)

var types = []event.Type{
	event.AccountCreateEventType,
	event.CommentEventType,
	event.PostEventType,
	event.FlagEventType,
	event.VoteEventType,
	event.DeleteCommentEventType,
}

type BlockchainMonitor struct {
	DB                  *sqlx.DB
	CommentsStorage     *db.CommentsStorage
	Provider            *provider.Provider
	Plagiarism          *service.AntiPlagiarism
	PushNotifier        push.Notifier
	NotificationStorage *db.NotificationStorage
	DownvotesStorage    *db.DownvotesStorage
	PlagiarismStorage   *db.PlagiarismStorage
	MailerClient        *mailer.Client
}

func (bm *BlockchainMonitor) Monitor(ctx context.Context) {
	headBlockNum, err := bm.getLastProcessedBlockNum()
	if err != nil {
		log.Fatalf("failed to start transaction monitor: %s", err)
	}

	log.WithField("last_checked_block_num", headBlockNum).Info("getting events")

	bm.Provider.Provide(ctx, headBlockNum, types, func(ev event.Event, err error) {
		if err != nil {
			log.WithError(err).Error("blockchain monitor: MONITOR STOPPED")
			return
		}

		entry := getEventLogger(ev)
		entry.Info("processing event")

		tx, err := bm.DB.Beginx()
		if err != nil {
			log.Error(errors.Wrap(err, "blockchain monitor: failed to create tx"))
			return
		}

		if err := bm.processEvent(ev, tx); err != nil {
			entry.WithError(err).Error("blockchain monitor: failed to process event")
			tx.Rollback()
			return
		} else {
			if err := saveLastProcessedBlockNumber(ev.Common().BlockNum, tx); err != nil {
				entry.WithError(err).Error("blockchain monitor: failed to save last block")
				tx.Rollback()
				return
			}
			if err := tx.Commit(); err != nil {
				entry.WithError(err).Error("blockchain monitor: failed to commit tx")
				return
			}
			headBlockNum = ev.Common().BlockNum
		}
	})
}

func getEventLogger(ev event.Event) *log.Entry {
	return log.WithField("block_num", ev.Common().BlockNum).
		WithField("event", ev.Type())
}

func (bm *BlockchainMonitor) processEvent(ev event.Event, tx *sqlx.Tx) error {
	switch ev.Type() {
	case event.AccountCreateEventType:
		accountCreateEvent := ev.(*event.AccountCreateEvent)
		return bm.processAccount(accountCreateEvent, tx)
	case event.CommentEventType:
		commentEvent := ev.(*event.CommentEvent)
		return bm.processComment(*commentEvent, tx)
	case event.PostEventType:
		postEvent := ev.(*event.PostEvent)
		return bm.processPost(*postEvent, tx)
	case event.DeleteCommentEventType:
		deleteCommentEvent := ev.(*event.DeleteCommentEvent)
		return bm.deleteComment(*deleteCommentEvent, tx)
	case event.VoteEventType:
		voteEvent := ev.(*event.VoteEvent)
		return bm.processVote(*voteEvent, tx)
	case event.FlagEventType:
		flagEvent := ev.(*event.FlagEvent)
		return bm.processFlag(*flagEvent, tx)
	}
	return nil
}

func (bm *BlockchainMonitor) getLastProcessedBlockNum() (blockNum uint32, err error) {
	err = bm.DB.Get(&blockNum, `SELECT block_num FROM blockchain_monitor`)
	if err != nil {
		return 0, err
	}
	return
}

func saveLastProcessedBlockNumber(id uint32, tx *sqlx.Tx) error {
	_, err := tx.Exec(`UPDATE blockchain_monitor SET block_num = $1`, id)
	return err
}

func (bm *BlockchainMonitor) processAccount(ev *event.AccountCreateEvent, tx *sqlx.Tx) error {
	account := ev.Account

	if _, err := tx.Exec(
		`INSERT INTO profiles(account, display_name)
		        VALUES ($1::account, $1::TEXT) ON CONFLICT DO NOTHING`, account); err != nil {
		return err
	}

	_, err := tx.Exec(
		`INSERT INTO profile_settings(account, enable_email_unseen_notifications) VALUES($1, $2) ON CONFLICT DO NOTHING`,
		account, true)
	return err
}

func (bm *BlockchainMonitor) processComment(ev event.CommentEvent, tx *sqlx.Tx) error {
	var metadata common.JsonMetadata
	if err := json.Unmarshal([]byte(ev.JsonMetadata), &metadata); err != nil {
		getEventLogger(ev).Warnf("failed to unmarshal comment metadata: %s", ev.JsonMetadata)
	}

	comment := db.Comment{
		Permlink:     ev.PermLink,
		Author:       ev.Author,
		Body:         ev.Body,
		Title:        ev.Title,
		JsonMetadata: metadata,
		UpdatedAt:    ev.Timestamp,
		CreatedAt:    ev.Timestamp,
	}

	if ev.ParentPermLink != "" {
		comment.ParentPermlink.Valid = true
		comment.ParentPermlink.String = ev.ParentPermLink
	}

	if ev.ParentAuthor != "" {
		comment.ParentAuthor.Valid = true
		comment.ParentAuthor.String = ev.ParentAuthor
	}

	if _, err := tx.NamedExec(
		`INSERT INTO comments VALUES(:permlink, :author, :body, :title, :json_metadata, :parent_author,
		:parent_permlink, :updated_at, :created_at, :domain)
		ON CONFLICT(author, permlink) DO UPDATE SET body = :body, title = :title, json_metadata = :json_metadata`, comment); err != nil {
		return err
	}

	return bm.createNotificationFromComment(comment, tx)
}

func (bm *BlockchainMonitor) deleteComment(ev event.DeleteCommentEvent, tx *sqlx.Tx) error {
	var doesPostExist bool
	err := tx.Get(&doesPostExist, `SELECT EXISTS(SELECT * FROM comments WHERE author = $1 AND permlink = $2 AND parent_author IS NULL)`, ev.Author, ev.PermLink)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM comments WHERE author = $1 AND permlink = $2`, ev.Author, ev.PermLink)
	if err != nil {
		return err
	}

	if doesPostExist {
		_, err = tx.Exec(
			`INSERT INTO deleted_posts VALUES($1, $2)
			    ON CONFLICT DO NOTHING`, ev.Author, ev.PermLink)
		if err != nil {
			return err
		}

		getEventLogger(ev).Infof("post @%s/%s deleted", ev.Author, ev.PermLink)

		err = bm.NotificationStorage.DeletePlagiarismNotification(ev.Author, ev.PermLink)
		if err != nil {
			getEventLogger(ev).Errorf("can't delete plagiarism notifications err:%s", err)
		}
	}
	return nil
}

func (bm *BlockchainMonitor) processPost(ev event.PostEvent, tx *sqlx.Tx) error {
	var metadata common.JsonMetadata
	if err := json.Unmarshal([]byte(ev.JsonMetadata), &metadata); err != nil {
		getEventLogger(ev).Warnf("failed to unmarshal post metadata: %s", ev.JsonMetadata)
	}

	comment := db.Comment{
		Permlink:     ev.PermLink,
		Author:       ev.Author,
		Body:         ev.Body,
		Title:        ev.Title,
		JsonMetadata: metadata,
		UpdatedAt:    ev.Timestamp,
		CreatedAt:    ev.Timestamp,
	}

	if ev.ParentPermLink != "" {
		comment.ParentPermlink = sql.NullString{Valid: true, String: ev.ParentPermLink}
	}

	if len(metadata.Domains) != 0 {
		domain := strings.Replace(metadata.Domains[0], "domain-", "", 1)
		if IsValidDomain(domain) {
			comment.Domain = sql.NullString{Valid: true, String: domain}
		}
		// TODO: if not valid domain mark post/comment as suspicious
	}

	_, err := tx.NamedExec(
		`INSERT INTO comments (permlink, author, body, title, json_metadata, parent_permlink, domain, updated_at, created_at)
				VALUES(:permlink, :author, :body, :title, :json_metadata, :parent_permlink, :domain, :updated_at, :created_at)
		        ON CONFLICT(author, permlink)
                   DO UPDATE SET body = :body,
                                 title = :title,
                                 domain = :domain,
																 updated_at = :updated_at,
                                 json_metadata = :json_metadata`, comment)

	if err != nil {
		return err
	}

	// Special case: the author might recreate a post with the same permlink
	res, err := tx.NamedExec(`DELETE FROM deleted_posts WHERE permlink = :permlink AND account = :author`, comment)
	if err != nil {
		return err
	}

	deleted, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if deleted > 0 {
		getEventLogger(ev).Infof("post @%s/%s recreated", comment.Author, comment.Permlink)
	}

	domain := DomainCom // If domain is invalid .com will be used as a default
	if len(comment.JsonMetadata.Domains) > 0 {
		d, ok := GetDomain(comment.JsonMetadata.Domains[0])
		if !ok {
			getEventLogger(ev).Warnf("invalid domain in post @%s/%s", comment.Author, comment.Permlink)
		} else {
			domain = d
		}
	}

	initialPostPlagiarism := db.PostPlagiarism{ // all new posts should be in db with 100% uniqueness
		Author:            comment.Author,
		Permlink:          comment.Permlink,
		UniquenessPercent: 1.0,
		Status:            db.PlagiarismStatusPending,
	}
	if err = bm.PlagiarismStorage.InTx(tx).Insert(initialPostPlagiarism); err != nil {
		getEventLogger(ev).Warnf("failed to check post for plagiarism err: %s", err)
		return nil
	}

	if err = bm.PlagiarismStorage.InTx(tx).UpdateStatus(comment.Author, comment.Permlink, db.PlagiarismStatusPending); err != nil {
		getEventLogger(ev).Warnf("failed to check post for plagiarism err: %s", err)
		return nil
	}

	go bm.checkPlagiarismAndNotify(comment, domain)
	return nil
}

func (bm *BlockchainMonitor) createNotificationFromComment(comment db.Comment, tx *sqlx.Tx) error {
	parentPostInfo, err := bm.CommentsStorage.InTx(tx).GetParentPost(comment.Author, comment.Permlink)
	if err != nil {
		return err
	}

	if len(parentPostInfo.JsonMetadata.Categories) == 0 {
		log.Warnf("empty categories for post %s/%s", parentPostInfo.Author, parentPostInfo.Permlink)
		return nil
	}

	parentCommentAuthor := parentPostInfo.Author

	notificationType := db.PostRepliedNotificationType
	if comment.ParentPermlink.Valid && parentPostInfo.Permlink != comment.ParentPermlink.String {
		notificationType = db.CommentRepliedNotificationType

		parentComment, err := bm.CommentsStorage.InTx(tx).Get(comment.ParentAuthor.String, comment.ParentPermlink.String)
		if err != nil {
			return err
		}

		parentCommentAuthor = parentComment.Author
	}

	// comment/replied to own comment/posts, no need to create a notification
	if parentCommentAuthor == comment.Author {
		return nil
	}

	meta := db.PostRelatedNotificationMeta{
		Account:      comment.Author,
		Permlink:     parentPostInfo.Permlink,
		PostAuthor:   parentPostInfo.Author,
		PostCategory: parentPostInfo.JsonMetadata.Categories[0],
		PostImage:    parentPostInfo.JsonMetadata.Image,
		PostTitle:    parentPostInfo.Title,
		Domains:      parentPostInfo.JsonMetadata.Domains,
	}

	notification := db.Notification{
		ID:        uuid.New(),
		Account:   parentCommentAuthor,
		Timestamp: comment.CreatedAt,
		Type:      notificationType,
		Meta:      meta.ToJson(),
	}

	err = bm.NotificationStorage.InTx(tx).Insert(notification)
	if err != nil {
		return errors.Wrapf(err, "failed to insert comment notification %s/%s",
			comment.Author, comment.Permlink)
	}

	if notificationType == db.PostRepliedNotificationType {
		bm.PushNotifier.NotifyPostReplied(meta)
	} else {
		bm.PushNotifier.NotifyCommentReplied(parentCommentAuthor, meta)
	}

	return nil
}

func (bm *BlockchainMonitor) processFlag(flagEvent event.FlagEvent, tx *sqlx.Tx) error {
	if flagEvent.Author == flagEvent.Voter {
		return nil
	}

	comment, err := bm.CommentsStorage.Get(flagEvent.Author, flagEvent.PermLink)
	if err != nil {
		return err
	}

	notificationType := db.PostFlaggedNotificationType
	if comment.ParentAuthor.Valid {
		notificationType = db.CommentFlaggedNotificationType
	}

	parentPostInfo, err := bm.CommentsStorage.GetParentPost(comment.Author, comment.Permlink)
	if err != nil {
		return err
	}

	if len(parentPostInfo.JsonMetadata.Categories) == 0 {
		log.Warnf("empty categories for post %s/%s", parentPostInfo.Author, parentPostInfo.Permlink)
		return nil
	}

	meta := db.PostRelatedNotificationMeta{
		Account:      flagEvent.Voter,
		Permlink:     parentPostInfo.Permlink,
		PostAuthor:   parentPostInfo.Author,
		PostCategory: parentPostInfo.JsonMetadata.Categories[0],
		PostImage:    parentPostInfo.JsonMetadata.Image,
		PostTitle:    parentPostInfo.Title,
		Domains:      parentPostInfo.JsonMetadata.Domains,
	}

	notification := db.Notification{
		ID:        uuid.New(),
		Account:   comment.Author,
		Timestamp: flagEvent.Timestamp,
		Type:      notificationType,
		Meta:      meta.ToJson(),
	}

	err = bm.NotificationStorage.InTx(tx).Insert(notification)
	if err != nil {
		return err
	}

	if notificationType == db.PostFlaggedNotificationType {
		bm.PushNotifier.NotifyPostFlagged(meta)
	} else {
		bm.PushNotifier.NotifyCommentFlagged(comment.Author, meta)
	}

	return err
}

func (bm *BlockchainMonitor) processVote(voteEvent event.VoteEvent, tx *sqlx.Tx) error {
	if voteEvent.Author == voteEvent.Voter {
		return nil
	}

	var comment db.Comment
	if err := tx.Get(&comment,
		`SELECT permlink, author, parent_permlink, parent_author, body, title, json_metadata, updated_at, created_at
		FROM comments WHERE author=$1 AND permlink=$2`, voteEvent.Author, voteEvent.PermLink); err != nil {
		return err
	}

	notificationType := db.PostVotedNotificationType
	if comment.ParentAuthor.Valid {
		notificationType = db.CommentVotedNotificationType
	}

	parentPostInfo, err := bm.CommentsStorage.InTx(tx).GetParentPost(comment.Author, comment.Permlink)
	if err != nil {
		return err
	}

	isPostVote := voteEvent.PermLink == parentPostInfo.Permlink

	// unvote
	if voteEvent.Weight == 0 {
		_, err := tx.Exec(fmt.Sprintf(
			`DELETE FROM notifications WHERE
						account = $1 AND type = $2 AND
						meta @> '{"permlink":"%s","account":"%s"}'`, parentPostInfo.Permlink, voteEvent.Voter),
			voteEvent.Author, notificationType)
		if err != nil {
			return err
		}

		if isPostVote {
			return bm.deleteVote(voteEvent.Voter, voteEvent.PermLink, tx)
		}

		return nil
	}

	if len(parentPostInfo.JsonMetadata.Categories) == 0 {
		log.Warnf("empty categories for post %s/%s", parentPostInfo.Author, parentPostInfo.Permlink)
		return nil
	}

	if isPostVote {
		if err := bm.saveVote(voteEvent, tx); err != nil {
			return err
		}
	}

	meta := db.PostRelatedNotificationMeta{
		Account:      voteEvent.Voter,
		Permlink:     parentPostInfo.Permlink,
		PostAuthor:   parentPostInfo.Author,
		PostCategory: parentPostInfo.JsonMetadata.Categories[0],
		PostImage:    parentPostInfo.JsonMetadata.Image,
		Domains:      parentPostInfo.JsonMetadata.Domains,
		PostTitle:    parentPostInfo.Title,
	}

	notification := db.Notification{
		ID:        uuid.New(),
		Account:   comment.Author,
		Timestamp: voteEvent.Timestamp,
		Type:      notificationType,
		Meta:      meta.ToJson(),
	}

	err = bm.NotificationStorage.InTx(tx).Insert(notification)

	if notificationType == db.PostVotedNotificationType {
		bm.PushNotifier.NotifyPostVoted(meta)
	} else {
		bm.PushNotifier.NotifyCommentVoted(comment.Author, meta)
	}

	return err
}

func (bm *BlockchainMonitor) saveVote(ve event.VoteEvent, tx *sqlx.Tx) error {
	plagiarism, err := bm.Plagiarism.GetCheckResult(ve.Author, ve.PermLink)
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "can't get plagiarism details")
	}

	var postUniqueness float32
	if plagiarism != nil {
		postUniqueness = plagiarism.Unique
	}

	vote := db.Vote{
		Account:    ve.Voter,
		Permlink:   ve.PermLink,
		Author:     ve.Author,
		PostUnique: postUniqueness,
	}

	_, err = tx.NamedExec(
		`INSERT INTO posts_votes (account, permlink, author, post_unique)
		VALUES(:account, :permlink, :author, :post_unique)
		`,
		vote,
	)

	return err
}

func (bm *BlockchainMonitor) deleteVote(account, permlink string, tx *sqlx.Tx) error {
	_, err := tx.Exec(
		`DELETE FROM posts_votes WHERE
					account = $1 AND permlink = $2
			`, account, permlink)
	return err
}

func (bm *BlockchainMonitor) checkPlagiarismAndNotify(c db.Comment, d Domain) {
	checkDetails, err := bm.Plagiarism.CheckPost(
		c.Author,
		c.Permlink,
		c.Body,
		d,
	)

	logger := log.WithField("author", c.Author).
		WithField("permlink", c.Permlink)

	if err != nil {
		logger.Warnf("can't check plagiarism err: %s", err)
		return
	}

	meta := db.PlagiarismRelatedNotificationMeta{}
	meta.Account = c.Author
	meta.Permlink = c.Permlink
	if len(c.JsonMetadata.Categories) > 0 {
		meta.PostCategory = c.JsonMetadata.Categories[0]
	}
	meta.PostTitle = c.Title
	meta.PostImage = c.JsonMetadata.Image
	meta.Domains = []string{string(d)}
	meta.Uniqueness = checkDetails.Unique
	meta.Status = checkDetails.Status

	err = bm.NotificationStorage.DeletePlagiarismNotification(c.Author, c.Permlink) // to be sure that we have only lates notifications
	if err != nil {
		logger.Warnf("can't delete plagiarism notification err: %s", err)
	}

	notification := db.Notification{
		Account:   c.Author,
		Timestamp: time.Now().UTC(),
		Type:      db.PostUniquenessCheckedNotificationType,
		Meta:      meta.ToJson(),
	}
	err = bm.NotificationStorage.Insert(notification)
	if err != nil {
		logger.Warnf("can't insert palgiarism notification into db err: %s", err)
	}

	if checkDetails.Status != db.PlagiarismStatusChecked {
		return
	}

	err = bm.MailerClient.SendPlagiarismEmail(meta)
	if err != nil {
		logger.Warnf("can't send notification to mailer err: %s", err)
	}
}
