package service

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/rpc"
	. "gitlab.scorum.com/blog/core/domain"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

var (
	errTextLenValidation = errors.New("text len should be in [100..150000]")

	htmlStripRegexp     = regexp.MustCompile("<[^>]*>")
	replaceSpacesRegexp = regexp.MustCompile(`\s+`)
)

type AntiPlagiarism struct {
	client            *TextRUClient
	CommentsStorage   *db.CommentsStorage
	PlagiarismStorage *db.PlagiarismStorage
}

func NewAntiPlagiarismService(key string, ps *db.PlagiarismStorage, cs *db.CommentsStorage) *AntiPlagiarism {
	ap := &AntiPlagiarism{
		client: &TextRUClient{
			key: key,
		},
		PlagiarismStorage: ps,
		CommentsStorage:   cs,
	}
	ap.client.Timeout = time.Second * 30

	return ap
}

func (a *AntiPlagiarism) CheckPost(account, permlink, text string, domain Domain) (res *PlagiarismCheckResult, err error) {
	//setuping default result in case of error
	res = &PlagiarismCheckResult{
		DateCheck: PlagiarismTime{
			Time: time.Now().UTC(),
		},
		Unique: 1.0,
		Urls:   []PlagiarismUrl{},
		Status: db.PlagiarismStatusFailed,
	}

	text = stripHTMLTags(text)

	if len(text) <= 100 || len(text) >= 150000 {
		log.Warnf("can't check post @%s/%s because of len: %s", account, permlink, errTextLenValidation)

		res.Status = db.PlagiarismStatusInvalidTextLen

		err = a.upsertPostCheckResult(account, permlink, res)
		return
	}

	uid, err := a.client.submitPostForCheck(text, getDomainToIgnore(domain))
	if err != nil {
		if err == errTextRuTextIsTooShort {
			res.Status = db.PlagiarismStatusInvalidTextLen
		}

		log.Warnf("error while submiting post for check @%s/%s err:%s", account, permlink, err)
		err = a.upsertPostCheckResult(account, permlink, res)
		return
	}

	details, err := a.client.checkResults(uid)
	if err != nil {
		log.Warnf("error while getting check result @%s/%s err:%s", account, permlink, err)
		err = a.upsertPostCheckResult(account, permlink, res)
		return
	}

	err = json.Unmarshal([]byte(details), &res)
	if err != nil {
		log.Warnf("error while pasring check response @%s/%s check_id:%s err:%s", account, permlink, uid, err)
		err = a.upsertPostCheckResult(account, permlink, res)
		return
	}

	urls := make([]string, 0, len(res.Urls))

	// we want have uniqueness in float [0..1]
	res.Unique = float32(math.Round(float64(res.Unique)) / 100)
	if res.Unique >= 0.6 { // if uniqueness >= 60% we count this post as unique == uniqueness 100%
		res.Unique = 1
	}

	if res.Unique < 0.01 { // the lowest value is 0.01
		res.Unique = 0.01
	}

	for i, url := range res.Urls {
		res.Urls[i].Plagiat = url.Plagiat / 100
		urls = append(urls, url.Url)
	}
	res.Status = db.PlagiarismStatusChecked

	urlToTitleMap := extractTitles(urls)
	for i, url := range res.Urls {
		res.Urls[i].Title = urlToTitleMap[url.Url]
	}

	err = a.upsertPostCheckResult(account, permlink, res)
	return
}

func (a *AntiPlagiarism) GetCheckResultEndpoint(ctx *rpc.Context) {
	var author string
	if err := ctx.Param(0, &author); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	var permlink string
	if err := ctx.Param(1, &permlink); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	details, err := a.GetCheckResult(author, permlink)
	if err != nil {
		errCode := rpc.InternalErrorCode
		errMessage := err.Error()
		if err == sql.ErrNoRows {
			errCode = rpc.PlagiarismDetailsNotFoundCode
			errMessage = "plagiarism details not found for requested post"
		}

		ctx.WriteError(errCode, errMessage)
		return
	}

	ctx.WriteResult(*details)
}

func (a *AntiPlagiarism) GetCheckResult(account, permlink string) (*PlagiarismCheckResult, error) {
	dbDetails, err := a.PlagiarismStorage.Get(account, permlink)
	if err != nil {
		return nil, err
	}

	checkDetail := PlagiarismCheckResult{
		DateCheck: PlagiarismTime{dbDetails.LastCheckAt},
		Unique:    dbDetails.UniquenessPercent,
		Urls:      convertDBUrlsIntoPlagiarismUrls(dbDetails.Urls),
		Status:    dbDetails.Status,
	}

	return &checkDetail, err
}

// getPostLink returns post link if exists otherwise empty string
func (a *AntiPlagiarism) getPostLink(account, permlink string) (string, error) {
	comment, err := a.CommentsStorage.Get(account, permlink)
	if err != nil && err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return comment.PostLink(), nil
}

func (a *AntiPlagiarism) upsertPostCheckResult(account, permlink string, details *PlagiarismCheckResult) error {
	postDB := db.PostPlagiarism{
		Author:            account,
		Permlink:          permlink,
		LastCheckAt:       details.DateCheck.Time,
		UniquenessPercent: details.Unique,
		Status:            details.Status,
	}
	postDB.Urls = extractUrlsIntoDbEntity(details)
	return a.PlagiarismStorage.Upsert(postDB)
}

func stripHTMLTags(text string) string {
	text = htmlStripRegexp.ReplaceAllString(text, "") // stripping html tags
	return replaceSpacesRegexp.ReplaceAllString(text, " ")
}

func extractUrlsIntoDbEntity(p *PlagiarismCheckResult) db.PlagiarismUrls {
	d := db.PlagiarismUrls{}
	for _, u := range p.Urls {
		d = append(d, db.PlagiarismUrl{Url: u.Url, Plagiat: u.Plagiat, Title: u.Title})
	}
	return d
}

func convertDBUrlsIntoPlagiarismUrls(urls db.PlagiarismUrls) []PlagiarismUrl {
	var pURLs []PlagiarismUrl
	for _, u := range urls {
		pURLs = append(pURLs, PlagiarismUrl{Url: u.Url, Plagiat: u.Plagiat, Title: u.Title})
	}

	return pURLs
}

func extractTitles(urls []string) map[string]string {
	titles := make(map[string]string)
	lock := sync.Mutex{}
	wg := sync.WaitGroup{}
	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			resp, err := http.Get(url)
			if err != nil {
				titles[url] = ""
				log.WithField("url", url).
					Warnf("Error while getting site body to parse title err:%s", err.Error())
				return
			}
			defer resp.Body.Close()
			title := extractTitleFromBody(resp.Body)
			lock.Lock()
			titles[url] = title
			lock.Unlock()
		}(url)
	}
	wg.Wait()

	return titles
}

func getDomainToIgnore(d Domain) string {
	domainToIgnore := fmt.Sprintf("scorum.%s", d)
	if d == DomainMe {
		domainToIgnore = domainToIgnore + "," + fmt.Sprintf("scorum.%s", DomainRu)
	}

	if d == DomainRu {
		domainToIgnore = domainToIgnore + "," + fmt.Sprintf("scorum.%s", DomainMe)
	}

	return domainToIgnore
}

func extractTitleFromBody(body io.Reader) (title string) {
	// detect charset
	reader := bufio.NewReader(body)
	data, err := reader.Peek(1024)
	if err != nil {
		return ""
	}

	e, name, _ := charset.DetermineEncoding(data, "")
	tokenized := html.NewTokenizer(reader)
	if name != "utf-8" {
		tokenized = html.NewTokenizer(e.NewDecoder().Reader(reader))
	}

	// find the title tag
	var titleFound bool
	for {
		tt := tokenized.Next()
		switch tt {
		case html.ErrorToken:
			return
		case html.TextToken:
			if titleFound {
				title = string(tokenized.Text())
				return
			}
		case html.StartTagToken:
			t := tokenized.Token()
			if t.Data == "title" {
				titleFound = true
			}
		}
	}
}
