package db

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gitlab.scorum.com/blog/api/common"
	"gitlab.scorum.com/blog/core/domain"
)

type Profile struct {
	Account     string    `db:"account"`
	DisplayName string    `db:"display_name"`
	Location    string    `db:"location"`
	Bio         string    `db:"bio"`
	AvatarUrl   string    `db:"avatar_url"`
	CoverUrl    string    `db:"cover_url"`
	CreatedAt   time.Time `db:"created_at"`
}

type ExtendedProfile struct {
	Profile

	FollowersCount int64 `db:"followers_count"`
	FollowingCount int64 `db:"following_count"`
}

type Media struct {
	Account     string             `db:"account"`
	ID          string             `db:"id"`
	Url         string             `db:"url"`
	ContentType common.ContentType `db:"content_type"`
	Meta        PropertyMap        `db:"meta"`
	CreatedAt   time.Time          `db:"created_at"`
}

type Draft struct {
	Account      string    `db:"account"`
	ID           string    `db:"id"`
	Title        string    `db:"title"`
	Body         string    `db:"body"`
	JsonMetadata string    `db:"json_metadata"`
	UpdatedAt    time.Time `db:"updated_at"`
	CreatedAt    time.Time `db:"created_at"`
}

type PropertyMap map[string]interface{}

func (p PropertyMap) Value() (driver.Value, error) {
	j, err := json.Marshal(p)
	return j, err
}

func (p *PropertyMap) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.New("type assertion .([]byte) failed.")
	}

	var i interface{}
	err := json.Unmarshal(source, &i)
	if err != nil {
		return err
	}

	*p, ok = i.(map[string]interface{})
	if !ok {
		return errors.New("type assertion .(map[string]interface{}) failed.")
	}

	return nil
}

type PostID struct {
	Account  string `db:"account"`
	Permlink string `db:"permlink"`
}

type Category struct {
	Domain          string `db:"domain"`
	Label           string `db:"label"`
	Order           uint32 `db:"order"`
	LocalizationKey string `db:"localization_key"`
}

type Comment struct {
	Permlink string `db:"permlink"`
	Author   string `db:"author"`
	// Domain the post/comment belongs to, note might
	// be null since post/comment could be created not via blog but cli or
	// other client
	Domain         sql.NullString      `db:"domain"`
	ParentPermlink sql.NullString      `db:"parent_permlink"`
	ParentAuthor   sql.NullString      `db:"parent_author"`
	Body           string              `db:"body"`
	Title          string              `db:"title"`
	JsonMetadata   common.JsonMetadata `db:"json_metadata"`
	UpdatedAt      time.Time           `db:"updated_at"`
	CreatedAt      time.Time           `db:"created_at"`
}

func getCategory(categories []string) string {
	if len(categories) == 0 {
		return ""
	}

	return strings.Replace(categories[0], "categories-", "", -1)
}

func (c Comment) PostLink() string {
	d := domain.GetDomainSafe(c.JsonMetadata.Domains)
	loc, ok := domain.GetDomainLocalization(d)
	if !ok {
		loc = "en-us"
		log.Warnf("localization is not setted for domain %s", d)
	}
	return fmt.Sprintf("https://scorum.%s/%s/%s/@%s/%s",
		d,
		loc,
		getCategory(c.JsonMetadata.Categories),
		c.Author,
		c.Permlink)
}

type ProfileSettings struct {
	Account                        string `db:"account"`
	EnableEmailUnseenNotifications bool   `db:"enable_email_unseen_notifications"`
}

type PushRegistration struct {
	Author string `db:"author"`
	Token  string `db:"token"`
}

type Vote struct {
	Account    string  `db:"account"`
	Permlink   string  `db:"permlink"`
	Author     string  `db:"author"`
	PostUnique float32 `db:"post_unique"`
}

type PostInfo struct {
	Author         string              `db:"author"`
	Permlink       string              `db:"permlink"`
	Image          string              `db:"image"`
	Title          string              `db:"title"`
	Category       string              `db:"category"`
	JsonMetadata   common.JsonMetadata `db:"json_metadata"`
	ParentPermlink sql.NullString      `db:"parent_permlink"`
}
