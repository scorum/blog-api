package image

import (
	"image"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.scorum.com/blog/api/common"
)

var (
	png1200x700 []byte
	png1200x200 []byte
	jpg900x700  []byte
	gif400x300  []byte
)

func init() {
	f, _ := os.Open("testdata/1200x700.png")
	png1200x700, _ = ioutil.ReadAll(f)

	f, _ = os.Open("testdata/1200x200.png")
	png1200x200, _ = ioutil.ReadAll(f)

	f, _ = os.Open("testdata/900x700.jpg")
	jpg900x700, _ = ioutil.ReadAll(f)

	f, _ = os.Open("testdata/400x300.gif")
	gif400x300, _ = ioutil.ReadAll(f)
}

func TestNewImage(t *testing.T) {
	t.Run("1200x700 valid png image", func(t *testing.T) {
		img, err := NewImage(png1200x700, common.ImagePngContentType)

		require.NoError(t, err)
		require.NotNil(t, img)

		require.Equal(t, 1200, img.OriginalSize().X)
		require.Equal(t, 700, img.OriginalSize().Y)

		//thumbs
		img.AddThumb("384", 384, 384)
		img.AddThumb("500", 500, 500)
		require.Len(t, img.Thumbs, 2)

		require.Equal(t, image.Point{384, 224}, img.Thumbs[0].Bounds().Size())
		require.Equal(t, img.Thumbs[0].Postfix, "384")
		require.Equal(t, image.Point{500, 291}, img.Thumbs[1].Bounds().Size())
		require.Equal(t, img.Thumbs[1].Postfix, "500")
	})

	t.Run("900x700 valid jpeg image", func(t *testing.T) {
		img, err := NewImage(jpg900x700, common.ImageJpegContentType)

		require.NoError(t, err)
		require.NotNil(t, img)

		require.Equal(t, 900, img.OriginalSize().X)
		require.Equal(t, 700, img.OriginalSize().Y)

		//thumbs
		img.AddThumb("800", 800, 800)
		require.Len(t, img.Thumbs, 1)

		require.Equal(t, image.Point{800, 622}, img.Thumbs[0].Bounds().Size())
	})

	t.Run("400x300 valid gif image", func(t *testing.T) {
		img, err := NewImage(gif400x300, common.ImageGifContentType)

		require.NoError(t, err)
		require.NotNil(t, img)

		require.Equal(t, 400, img.OriginalSize().X)
		require.Equal(t, 300, img.OriginalSize().Y)

		//thumbs
		img.AddThumb("384", 384, 384)
		require.Len(t, img.Thumbs, 1)

		require.Equal(t, image.Point{384, 288}, img.Thumbs[0].Bounds().Size())
	})

	t.Run("invalid jpeg image", func(t *testing.T) {
		_, err := NewImage(png1200x700, common.ImageJpegContentType)
		require.Error(t, err)
		require.Equal(t, ErrInvalidFormat, err)
	})

	t.Run("invalid png image", func(t *testing.T) {
		_, err := NewImage(jpg900x700, common.ImagePngContentType)
		require.Error(t, err)
		require.Equal(t, ErrInvalidFormat, err)
	})

	t.Run("invalid gif image", func(t *testing.T) {
		_, err := NewImage(png1200x700, common.ImageGifContentType)
		require.Error(t, err)
		require.Equal(t, ErrInvalidFormat, err)
	})
}

func TestAddThumbNeat(t *testing.T) {
	t.Run("768 width, 176 maxHeight", func(t *testing.T) {
		img, err := NewImage(png1200x200, common.ImagePngContentType)

		require.NoError(t, err)
		require.NotNil(t, img)

		img.AddThumbNeat("", 800, 176, 768, 176)
		require.Len(t, img.Thumbs, 1)

		require.Equal(t, image.Point{768, 176}, img.Thumbs[0].Bounds().Size())
	})

	t.Run("100 width, 300 maxHeight", func(t *testing.T) {
		img, err := NewImage(png1200x200, common.ImagePngContentType)

		require.NoError(t, err)
		require.NotNil(t, img)

		img.AddThumbNeat("", 300, 400, 100, 300)
		require.Len(t, img.Thumbs, 1)

		require.Equal(t, image.Point{100, 300}, img.Thumbs[0].Bounds().Size())
	})
}
