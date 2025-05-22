package push

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/domainprovider"
	. "gitlab.scorum.com/blog/core/domain"
	"gitlab.scorum.com/blog/core/locale"
)

type Notifier interface {
	NotifyPostReplied(meta db.PostRelatedNotificationMeta)
	NotifyPostVoted(meta db.PostRelatedNotificationMeta)
	NotifyCommentReplied(to string, meta db.PostRelatedNotificationMeta)
	NotifyStartedFollow(account string, who string)
	NotifyCommentVoted(to string, meta db.PostRelatedNotificationMeta)
	NotifyPostFlagged(meta db.PostRelatedNotificationMeta)
	NotifyCommentFlagged(to string, meta db.PostRelatedNotificationMeta)
}

func NewNotifier(pusher *Pusher, localizer *locale.Localizer, dp *domainprovider.DomainProvider,
	prs *db.PushTokensStorage) Notifier {
	return &notifier{
		Pusher:                  pusher,
		Localizer:               localizer,
		DomainProvider:          dp,
		PushRegistrationStorage: prs,
	}
}

type notifier struct {
	Pusher                  *Pusher
	Localizer               *locale.Localizer
	PushRegistrationStorage *db.PushTokensStorage
	DomainProvider          *domainprovider.DomainProvider
}

func (n *notifier) getAccountLocalization(account string) (Domain, *locale.Localization) {
	domain, err := n.DomainProvider.GetByAccount(account)
	if err != nil {
		log.Errorf("failed to get %s domain: %s", account, err)
		domain = DomainCom
	}
	return domain, n.Localizer.GetLocalization(domain)
}

func (n *notifier) notify(account string, push Push) {
	tokens, err := n.PushRegistrationStorage.GetTokensByAccount(account)
	if err != nil {
		log.Errorf("failed to get %s push tokens: %s", account, err)
		return
	}

	for _, token := range tokens {
		if err = n.Pusher.SendWebPush(token, push); err != nil {
			if err == ErrTokenUnregistered {
				if err1 := n.PushRegistrationStorage.Delete(account, token); err1 != nil {
					log.Errorf("failed to delete web push token=%s: %s", token, err1)
				}
				log.Infof("web push token=%s deleted, account=%s", token, account)
				continue
			}

			// firebase sometimes returns 500, push is not so important, simply warn
			log.Warningf("failed to send web push, token=%s: %s", token, err)
		}
	}
}

func (n *notifier) NotifyPostReplied(meta db.PostRelatedNotificationMeta) {
	go func() {
		_, loc := n.getAccountLocalization(meta.PostAuthor)

		n.notify(meta.PostAuthor, Push{
			Title:       meta.Account,
			Body:        fmt.Sprintf(`%s "%s"`, loc.Translate("blog.notifications.comment"), meta.PostTitle),
			ClickAction: meta.PostLink(),
		})
	}()
}

func (n *notifier) NotifyCommentReplied(parentCommentAuthor string, meta db.PostRelatedNotificationMeta) {
	go func() {
		_, loc := n.getAccountLocalization(parentCommentAuthor)

		n.notify(parentCommentAuthor, Push{
			Title:       meta.Account,
			Body:        fmt.Sprintf(`%s "%s`, loc.Translate("blog.notifications.respond"), meta.PostTitle),
			ClickAction: meta.PostLink(),
		})
	}()
}

func (n *notifier) NotifyStartedFollow(account string, who string) {
	go func() {
		domain, loc := n.getAccountLocalization(account)

		n.notify(account, Push{
			Title:       who,
			Body:        loc.Translate("blog.notifications.follow"),
			ClickAction: fmt.Sprintf("https://scorum.%s/profile/@%s", domain, who),
		})
	}()
}

func (n *notifier) NotifyPostVoted(meta db.PostRelatedNotificationMeta) {
	go func() {
		_, loc := n.getAccountLocalization(meta.PostAuthor)

		n.notify(meta.PostAuthor, Push{
			Title:       meta.Account,
			Body:        fmt.Sprintf(`%s "%s`, loc.Translate("blog.notifications.upvote-post"), meta.PostTitle),
			ClickAction: meta.PostLink(),
		})
	}()
}

func (n *notifier) NotifyCommentVoted(parentCommentAuthor string, meta db.PostRelatedNotificationMeta) {
	go func() {
		_, loc := n.getAccountLocalization(parentCommentAuthor)

		n.notify(parentCommentAuthor, Push{
			Title:       meta.Account,
			Body:        fmt.Sprintf(`%s "%s`, loc.Translate("blog.notifications.upvote-comment"), meta.PostTitle),
			ClickAction: meta.PostLink(),
		})
	}()
}

func (n *notifier) NotifyPostFlagged(meta db.PostRelatedNotificationMeta) {
	go func() {
		_, loc := n.getAccountLocalization(meta.PostAuthor)

		n.notify(meta.PostAuthor, Push{
			Title:       meta.Account,
			Body:        fmt.Sprintf(`%s "%s`, loc.Translate("blog.notifications.downvote-post"), meta.PostTitle),
			ClickAction: meta.PostLink(),
		})
	}()
}

func (n *notifier) NotifyCommentFlagged(parentCommentAuthor string, meta db.PostRelatedNotificationMeta) {
	go func() {
		_, loc := n.getAccountLocalization(parentCommentAuthor)

		n.notify(parentCommentAuthor, Push{
			Title:       meta.Account,
			Body:        fmt.Sprintf(`%s "%s`, loc.Translate("blog.notifications.downvote-comment"), meta.PostTitle),
			ClickAction: meta.PostLink(),
		})
	}()
}
