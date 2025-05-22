package service

import (
	"testing"
	"time"

	"encoding/json"

	"github.com/stretchr/testify/require"
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/common"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/core/domain"
)

var (
	permlink = "test"
	testText = `Грамотность проверяемого текста играет важную роль при создании
				 качественного текста, и данный пункт позволит проверить.`
	textRUKey = "68c796999f45120ca4a4a420fa59327f"
)

func TestCheckPostPlagiarism(t *testing.T) {
	t.Skip()
	defer cleanUp(t)

	require.Nil(t, handler.Register(&types.RegisterOperation{leonarda}))
	ap := NewAntiPlagiarismService(textRUKey, db.NewPlagiarismStorage(dbWrite), db.NewCommentsStorage(dbWrite))
	_, err := ap.CheckPost(
		leonarda,
		permlink,
		testText,
		"com",
	)
	require.NoError(t, err)

	_, err = ap.CheckPost(
		leonarda,
		permlink,
		testText,
		"com",
	)
	require.NoError(t, err)
}

func TestGetPostPlagiarismCheckDetails(t *testing.T) {
	t.Skip()
	defer cleanUp(t)

	require.Nil(t, handler.Register(&types.RegisterOperation{leonarda}))
	ap := NewAntiPlagiarismService(textRUKey, db.NewPlagiarismStorage(dbWrite), db.NewCommentsStorage(dbWrite))
	_, err := ap.CheckPost(
		leonarda,
		permlink,
		testText,
		"com",
	)
	require.NoError(t, err)

	_, err = ap.GetCheckResult(leonarda, permlink)
	require.NoError(t, err)
}

func TestGetPostLink(t *testing.T) {
	defer cleanUp(t)

	ap := NewAntiPlagiarismService(textRUKey, db.NewPlagiarismStorage(dbWrite), db.NewCommentsStorage(dbWrite))
	finalLink := "https://scorum.com/en-us/baseball/@abel/test-123perm"

	meta := common.JsonMetadata{
		Tags:       []string{"baseball-tag-4"},
		Domains:    []string{"domain-com"},
		Locales:    []string{"locale-en-us"},
		Categories: []string{"categories-baseball"},
	}
	metaBytes, err := json.Marshal(&meta)
	require.NoError(t, err)

	permlink := "test-123perm"
	author := "abel"

	require.Nil(t, handler.Register(&types.RegisterOperation{author}))
	_, err = handler.DB.Write.Exec(`INSERT INTO comments(permlink, author, json_metadata, body, title, updated_at, created_at)
		VALUES($1, $2, $3, $4, $5, $6, $7)`, permlink, author, string(metaBytes), "123", "ti", time.Now(), time.Now())
	require.NoError(t, err)

	link, err := ap.getPostLink(author, permlink)
	require.NoError(t, err)
	require.EqualValues(t, finalLink, link)
}

func TestGetDomainToIgnore(t *testing.T) {
	dom := getDomainToIgnore(domain.DomainCom)
	require.EqualValues(t, dom, "scorum.com")

	dom = getDomainToIgnore(domain.DomainRu)
	require.EqualValues(t, dom, "scorum.ru,scorum.me")

	dom = getDomainToIgnore(domain.DomainMe)
	require.EqualValues(t, dom, "scorum.me,scorum.ru")
}

func TestStipHTMLTags(t *testing.T) {
	html := `<span class="post__title-text">Web Apps — сетевая операционная система</span>`
	text := "Web Apps — сетевая операционная система"
	require.EqualValues(t, stripHTMLTags(html), text)
}
