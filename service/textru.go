package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gitlab.scorum.com/blog/api/common"
)

const (
	textRUCheckPostUrl = "http://api.text.ru/post"
	retryDelay         = time.Minute
)

var (
	errTextRuTextIsTooShort = errors.New("checking text is too short")
)

type errResponse struct {
	ErrorCode int    `json:"error_code"`
	ErrorDesc string `json:"error_desc"`
}

func (r errResponse) checkForErr() error {
	if r.ErrorDesc != "" {
		if r.ErrorCode == 112 {
			return errTextRuTextIsTooShort
		}

		return fmt.Errorf("api response contains err code:%d err:%s", r.ErrorCode, r.ErrorDesc)
	}
	return nil
}

type startPlagiarismCheckResponse struct {
	errResponse
	TextUID string `json:"text_uid"`
}

type plagiarismCheckResultResponse struct {
	errResponse
	TextUnique string          `json:"text_unique"`
	ResultJson json.RawMessage `json:"result_json"`
}

type PlagiarismCheckResult struct {
	DateCheck PlagiarismTime  `json:"date_check"`
	Unique    float32         `json:"unique"`
	Urls      []PlagiarismUrl `json:"urls"`
	Status    string          `json:"status"`
}

type PlagiarismUrl struct {
	Url     string  `json:"url"`
	Plagiat float32 `json:"plagiat"`
	Title   string  `json:"title"`
}

type PlagiarismTime struct {
	time.Time
}

func (ct *PlagiarismTime) UnmarshalJSON(b []byte) (err error) {
	j := string(b) //make string parsable, we are gettings time in a very stange format from text.ru
	j = strings.Trim(j, `"`)

	ct.Time, err = time.Parse("02.01.2006 15:04:05", j)
	if err != nil {
		ct.Time, err = time.Parse(time.RFC3339, j)
	}
	return
}

func (ct *PlagiarismTime) MarshalJSON() ([]byte, error) {
	return ct.Time.MarshalJSON()
}

type TextRUClient struct {
	http.Client
	key string
}

func (a *TextRUClient) submitPostForCheck(text, domainToIgnore string) (string, error) {
	form := url.Values{}
	form.Set("text", text)
	form.Set("userkey", a.key)

	if domainToIgnore != "" {
		form.Set("exceptdomain", domainToIgnore)
	}

	var r startPlagiarismCheckResponse
	err := common.TryDo(func(attempt int) (retry bool, err error) {
		retry = true
		resp, err := a.PostForm(textRUCheckPostUrl, form)
		if err != nil || resp.StatusCode != http.StatusOK {
			time.Sleep(retryDelay)
			return
		}

		if err = json.NewDecoder(resp.Body).Decode(&r); err != nil { // text.ru can return html in case of error instead of json, so we have to retry it
			time.Sleep(retryDelay)
			return
		}

		return false, nil
	},
	)

	if err != nil {
		return "", err
	}

	return r.TextUID, r.checkForErr()
}

func (a *TextRUClient) checkResults(uid string) (string, error) {
	form := url.Values{}
	form.Set("uid", uid)
	form.Set("userkey", a.key)

	var result plagiarismCheckResultResponse
	err := common.TryDo(func(attempt int) (retry bool, err error) {
		retry = true
		resp, err := a.PostForm(textRUCheckPostUrl, form)
		if err != nil {
			time.Sleep(retryDelay)
			return
		}

		if resp.StatusCode != http.StatusOK {
			err = errors.New("status code not OK")
			time.Sleep(retryDelay)
			return
		}

		var temp plagiarismCheckResultResponse
		if err = json.NewDecoder(resp.Body).Decode(&temp); err != nil {
			time.Sleep(retryDelay)
			return
		}

		if err = temp.checkForErr(); err != nil {
			time.Sleep(retryDelay)
			return
		}

		result = temp
		return
	})
	j := string(result.ResultJson) // we are getting result json as a string instead json object
	j = strings.Trim(j, `"`)       // we are doint that manipulations to make it parsable
	j = strings.Replace(j, `\`, "", -1)

	return j, err
}
