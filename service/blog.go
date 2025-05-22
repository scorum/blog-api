package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/scorum/scorum-go"
	"gitlab.scorum.com/blog/api/blob"
	"gitlab.scorum.com/blog/api/common"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/push"
	"gitlab.scorum.com/blog/api/rpc"
	"gopkg.in/go-playground/validator.v9"
)

// constants
const (
	maxLargePageSize = 1000
	imageSizeLimit   = 200
)

// format
const (
	TimeLayout = `2006-01-02T15:04:05`
)

var (
	validate         *validator.Validate
	mediaNotFoundErr = errors.New("media not found")
)

func init() {
	validate = validator.New()
}

// Blog handles get and transaction requests

type Database struct {
	Write *sqlx.DB
	Read  *sqlx.DB
}

type Config struct {
	Admin                   string `yaml:"admin"`
	NotificationsLimit      int    `yaml:"notifications_limit"`
	UnsubscribeApiJwtSecret string `yaml:"unsubscribe_api_jwt_secret"`
	MaxFollow               int    `yaml:"max_follow"`
}

type Blog struct {
	DB                      Database
	Config                  Config
	Blockchain              *scorumgo.Client
	Blob                    *blob.Service
	Notifier                push.Notifier
	PushRegistrationStorage *db.PushTokensStorage
	NotificationStorage     *db.NotificationStorage
	DownvotesStorage        *db.DownvotesStorage
}

func (blog *Blog) getMediaByUrl(account, url string) (*db.Media, error) {
	var media db.Media
	err := blog.DB.Read.Get(&media,
		`SELECT * FROM media WHERE account = $1 AND url = $2`,
		account, url)

	if err == sql.ErrNoRows {
		return nil, mediaNotFoundErr
	}
	return &media, err
}

func (blog *Blog) checkAccountExists(account string) (bool, error) {
	var exists bool
	err := blog.DB.Read.Get(&exists, `SELECT EXISTS(SELECT * FROM profiles WHERE account = $1)`, account)
	return exists, err
}

func NewError(code int, message string) *rpc.Error {
	return &rpc.Error{
		Code:    code,
		Message: message,
	}
}

func WrapError(code int, err error) *rpc.Error {
	return NewError(code, err.Error())
}

func uniqueStrings(input []string) []string {
	u := make([]string, 0, len(input))
	m := make(map[string]struct{})

	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = struct{}{}
			u = append(u, val)
		}
	}

	return u
}

func isProfileAllowedContentType(t common.ContentType) bool {
	allowedTypes := []common.ContentType{
		common.ImageJpegContentType,
		common.ImagePngContentType}
	for _, v := range allowedTypes {
		if v == t {
			return true
		}
	}
	return false
}

func isMediaAllowedContentType(t common.ContentType) bool {
	allowedTypes := []common.ContentType{
		common.ImageJpegContentType,
		common.ImagePngContentType,
		common.ImageGifContentType}
	for _, v := range allowedTypes {
		if v == t {
			return true
		}
	}
	return false
}

func downloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func getParam(params []*json.RawMessage, at int, p interface{}) error {
	if at >= len(params) {
		return fmt.Errorf("no params at index %d", at)
	}

	arg := params[at]
	if arg == nil {
		p = nil
		return nil
	}

	return json.Unmarshal(*arg, p)
}
