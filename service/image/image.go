package image

import (
	"bytes"
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"strings"

	"fmt"

	"github.com/disintegration/imaging"
	"gitlab.scorum.com/blog/api/common"
)

var ErrInvalidFormat = errors.New("invalid image format")

type thumb struct {
	image.Image
	Postfix   string
	maxWidth  int
	maxHeight int
}

type Image struct {
	Original    image.Image
	Thumbs      []*thumb
	ContentType common.ContentType

	max int

	encode func(w io.Writer, img image.Image) error
}

func (m *Image) OriginalSize() image.Point {
	return m.Original.Bounds().Size()
}

func (m *Image) Encode(w io.Writer, img image.Image) error {
	return m.encode(w, img)
}

func (m *Image) AddThumb(prefix string, maxWidth, maxHeight int) {
	m.Thumbs = append(m.Thumbs, &thumb{
		maxWidth:  maxWidth,
		maxHeight: maxHeight,
		Postfix:   prefix,
		Image:     thumbnail(m.Original, maxWidth, maxHeight),
	})
}

// AddWideThumbPreview add a thumbnail resized to fixed height and width and center-cropped
func (m *Image) AddThumbNeat(prefix string, width, height, cropWidth, cropHeight int) {
	img := thumbnailFill(m.Original, width, height)

	m.Thumbs = append(m.Thumbs, &thumb{
		maxWidth:  width,
		maxHeight: height,
		Postfix:   prefix,
		Image:     imaging.CropCenter(img, cropWidth, cropHeight),
	})
}

func (m *Image) Max() int {
	return m.max
}

// NewImage creates a new instance of Image
func NewImage(in []byte, contentType common.ContentType) (*Image, error) {
	var err error

	buffer := bytes.NewBuffer(in)

	out := &Image{
		ContentType: contentType,
	}

	switch contentType {
	case common.ImageJpegContentType:
		out.Original, err = jpeg.Decode(buffer)
		if err != nil {
			if _, ok := err.(jpeg.FormatError); ok {
				return nil, ErrInvalidFormat
			}
			return nil, err
		}
		out.encode = func(w io.Writer, img image.Image) error {
			return jpeg.Encode(w, img, &jpeg.Options{Quality: 90})
		}
	case common.ImagePngContentType:
		out.Original, err = png.Decode(buffer)
		if err != nil {
			if _, ok := err.(png.FormatError); ok {
				return nil, ErrInvalidFormat
			}
			return nil, err
		}
		out.encode = png.Encode
	case common.ImageGifContentType:
		out.Original, err = gif.Decode(buffer)
		if err != nil {
			// there is no better way to detect format error in the gif package
			if strings.Contains(err.Error(), "gif: can't recognize format") {
				return nil, ErrInvalidFormat
			}
			return nil, err
		}
		out.encode = func(w io.Writer, img image.Image) error {
			return gif.Encode(w, img, nil)
		}
	default:
		return nil, fmt.Errorf("invalid content type: %s", contentType)
	}

	size := out.OriginalSize()

	out.max = size.X
	if size.Y > out.max {
		out.max = size.Y
	}

	return out, nil
}

func thumbnail(img image.Image, maxWidth, maxHeight int) image.Image {
	return imaging.Fit(img, maxWidth, maxHeight, imaging.Lanczos)
}

func thumbnailFill(img image.Image, maxWidth, maxHeight int) image.Image {
	return imaging.Fill(img, maxWidth, maxHeight, imaging.Center, imaging.Lanczos)
}
