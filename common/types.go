package common

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type ContentType string

const (
	ImageJpegContentType ContentType = "image/jpeg"
	ImagePngContentType  ContentType = "image/png"
	ImageGifContentType  ContentType = "image/gif"
)

type JsonMetadata struct {
	Domains             []string `json:"domains"`
	Categories          []string `json:"categories"`
	Locales             []string `json:"locales"`
	Tags                []string `json:"tags"`
	Image               string   `json:"image"`
	ImageOriginalWidth  uint16   `json:"image_original_width"`
	ImageOriginalHeight uint16   `json:"image_original_height"`
}

func (m *JsonMetadata) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.New("type assertion .([]byte) failed.")
	}

	err := json.Unmarshal(source, m)
	if err != nil {
		return err
	}

	return nil
}

func (m JsonMetadata) Value() (driver.Value, error) {
	v, err := json.Marshal(m)
	return v, err
}
