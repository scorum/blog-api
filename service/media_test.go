package service

import (
	"encoding/base64"
	"fmt"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/common"
	"gitlab.scorum.com/blog/api/service/image"
)

var (
	base64PNG1200x700 string
	base64PNG1200x200 string
	base64Gif400x300  string
	base64JPG900x700  string
	base64PNG800x300  string

	base64PNG767x720 string
	base64PNG767x723 string
	base64PNG768x400 string
	base64PNG900x600 string
)

func init() {
	f, _ := os.Open("image/testdata/1200x700.png")
	src, _ := ioutil.ReadAll(f)
	base64PNG1200x700 = base64.StdEncoding.EncodeToString(src)

	f, _ = os.Open("image/testdata/1200x200.png")
	src, _ = ioutil.ReadAll(f)
	base64PNG1200x200 = base64.StdEncoding.EncodeToString(src)

	f, _ = os.Open("image/testdata/900x700.jpg")
	src, _ = ioutil.ReadAll(f)
	base64JPG900x700 = base64.StdEncoding.EncodeToString(src)

	f, _ = os.Open("image/testdata/400x300.gif")
	src, _ = ioutil.ReadAll(f)
	base64Gif400x300 = base64.StdEncoding.EncodeToString(src)

	f, _ = os.Open("image/testdata/800x300.png")
	src, _ = ioutil.ReadAll(f)
	base64PNG800x300 = base64.StdEncoding.EncodeToString(src)

	f, _ = os.Open("image/testdata/high_images/767x720.png")
	src, _ = ioutil.ReadAll(f)
	base64PNG767x720 = base64.StdEncoding.EncodeToString(src)

	f, _ = os.Open("image/testdata/high_images/767x723.png")
	src, _ = ioutil.ReadAll(f)
	base64PNG767x723 = base64.StdEncoding.EncodeToString(src)

	f, _ = os.Open("image/testdata/high_images/768x400.png")
	src, _ = ioutil.ReadAll(f)
	base64PNG768x400 = base64.StdEncoding.EncodeToString(src)

	f, _ = os.Open("image/testdata/high_images/900x600.png")
	src, _ = ioutil.ReadAll(f)
	base64PNG900x600 = base64.StdEncoding.EncodeToString(src)
}

func TestBlog_UploadMedia_PNG(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	op := &types.UploadMediaOperation{
		Account:     leonarda,
		Media:       base64PNG1200x700,
		ID:          "png",
		ContentType: common.ImagePngContentType,
	}

	require.Nil(t, handler.UploadMedia(op))
	require.NotNil(t, handler.UploadMedia(op), "upload for the second time with the same ID, should fail")

	// thumbs
	exists, err := handler.Blob.DoesMediaExists(leonarda, fmt.Sprintf("%s_%d", op.ID, 96))
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = handler.Blob.DoesMediaExists(leonarda, fmt.Sprintf("%s_%d", op.ID, 384))
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = handler.Blob.DoesMediaExists(leonarda, fmt.Sprintf("%s_%d", op.ID, 1000))
	require.NoError(t, err)
	require.True(t, exists)
}

func TestBlog_UploadMedia_PNG800(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	op := &types.UploadMediaOperation{
		Account:     leonarda,
		Media:       base64PNG800x300,
		ID:          "png800",
		ContentType: common.ImagePngContentType,
	}

	require.Nil(t, handler.UploadMedia(op))

	exists, err := handler.Blob.DoesMediaExists(leonarda, fmt.Sprintf("%s_%d", op.ID, 96))
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = handler.Blob.DoesMediaExists(leonarda, fmt.Sprintf("%s_%d", op.ID, 384))
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = handler.Blob.DoesMediaExists(leonarda, fmt.Sprintf("%s_%d", op.ID, 800))
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = handler.Blob.DoesMediaExists(leonarda, fmt.Sprintf("%s_%s", op.ID, "preview"))
	require.NoError(t, err)
	require.True(t, exists)
}

func TestBlog_UploadMedia_GIF(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	op := &types.UploadMediaOperation{
		Account:     leonarda,
		Media:       base64Gif400x300,
		ID:          "gif",
		ContentType: common.ImageGifContentType,
	}

	require.Nil(t, handler.UploadMedia(op))

	exists, err := handler.Blob.DoesMediaExists(leonarda, fmt.Sprintf("%s_%d", op.ID, 384))
	require.NoError(t, err)
	require.True(t, exists)
	exists, err = handler.Blob.DoesMediaExists(leonarda, fmt.Sprintf("%s_%d", op.ID, 1000))
	require.NoError(t, err)
	require.False(t, exists)
}

func TestBlog_UploadMedia_JPEG(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	op := &types.UploadMediaOperation{
		Account:     leonarda,
		Media:       base64JPG900x700,
		ID:          "jpeg",
		ContentType: common.ImageJpegContentType,
	}

	require.Nil(t, handler.UploadMedia(op))
}

func TestBlog_UploadMedia_ValidationTest(t *testing.T) {
	defer cleanUp(t)

	op := types.UploadMediaOperation{
		Account:     leonarda,
		Media:       base64PNG1200x700,
		ID:          "someid123",
		ContentType: common.ImagePngContentType,
	}

	t.Run("valid_operation", func(t *testing.T) {
		require.NoError(t, validate.Struct(op))
	})

	t.Run("empty_id", func(t *testing.T) {
		cop := op
		cop.ID = ""
		require.Error(t, validate.Struct(cop))
	})

	t.Run("invalid_id_length", func(t *testing.T) {
		cop := op
		cop.ID = "12345678901234567"
		require.Error(t, validate.Struct(cop))
	})

	t.Run("invalid_id_symbols", func(t *testing.T) {
		cop := op
		cop.ID = "русскийид"
		require.Error(t, validate.Struct(cop))
	})

	t.Run("invalid_image_base64", func(t *testing.T) {
		cop := op
		cop.Media = "medianotbase64"
		require.Error(t, validate.Struct(cop))
	})

	t.Run("empty_account", func(t *testing.T) {
		cop := op
		cop.Account = ""
		require.Error(t, validate.Struct(cop))
	})

	t.Run("empty_content_type", func(t *testing.T) {
		cop := op
		cop.ContentType = ""
		require.Error(t, validate.Struct(cop))
	})

	t.Run("invalid_content_type", func(t *testing.T) {
		registerAccount(t, leonarda)

		cop := op
		cop.ContentType = "application/octet"
		err := handler.UploadMedia(&cop)

		require.NotNil(t, err)
		require.Equal(t, err.Message, "invalid content_type")
	})
}

func TestAddPreviewWideToImage1200x700(t *testing.T) {
	defer cleanUp(t)
	bb, err := base64.StdEncoding.DecodeString(base64PNG1200x200)
	require.NoError(t, err)
	img, err := image.NewImage(bb, common.ImagePngContentType)
	require.NoError(t, err)

	AddPreviewWideToImage(img)

	imgNameTemplate := "test_image_%s_%d_%d.png"
	for _, th := range img.Thumbs {
		imgName := fmt.Sprintf(
			imgNameTemplate,
			th.Postfix,
			th.Bounds().Max.X,
			th.Bounds().Max.Y,
		)

		f, err := os.Create(imgName)
		require.NoError(t, err)

		err = png.Encode(f, th)
		require.NoError(t, err)

		require.NoError(t, f.Close())
		require.NoError(t, os.Remove(imgName))
	}
}

func TestAddPreviewWideToImage900x700(t *testing.T) {
	defer cleanUp(t)
	bb, err := base64.StdEncoding.DecodeString(base64JPG900x700)
	require.NoError(t, err)
	img, err := image.NewImage(bb, common.ImageJpegContentType)
	require.NoError(t, err)

	AddPreviewWideToImage(img)

	imgNameTemplate := "test_image_%s_%d_%d.jpeg"
	for _, th := range img.Thumbs {
		imgName := fmt.Sprintf(
			imgNameTemplate,
			th.Postfix,
			th.Bounds().Max.X,
			th.Bounds().Max.Y,
		)

		f, err := os.Create(imgName)
		require.NoError(t, err)

		err = jpeg.Encode(f, th, nil)
		require.NoError(t, err)

		require.NoError(t, f.Close())
		require.NoError(t, os.Remove(imgName))
	}
}

func TestAddNotificationPreviewToImage1200x700(t *testing.T) {
	defer cleanUp(t)
	bb, err := base64.StdEncoding.DecodeString(base64PNG1200x200)
	require.NoError(t, err)
	img, err := image.NewImage(bb, common.ImagePngContentType)
	require.NoError(t, err)

	AddNotificationPreviewToImage(img)

	imgNameTemplate := "test_image_%s_%d_%d.png"
	for _, th := range img.Thumbs {
		imgName := fmt.Sprintf(
			imgNameTemplate,
			th.Postfix,
			th.Bounds().Max.X,
			th.Bounds().Max.Y,
		)

		require.EqualValues(t, 96, th.Bounds().Max.X)
		require.EqualValues(t, 96, th.Bounds().Max.Y)
		f, err := os.Create(imgName)
		require.NoError(t, err)

		err = png.Encode(f, th)
		require.NoError(t, err)

		require.NoError(t, f.Close())
		require.NoError(t, os.Remove(imgName))
	}
}

func TestAddProfilePreviewToImage1200x700(t *testing.T) {
	defer cleanUp(t)
	bb, err := base64.StdEncoding.DecodeString(base64PNG1200x200)
	require.NoError(t, err)
	img, err := image.NewImage(bb, common.ImagePngContentType)
	require.NoError(t, err)

	AddProfilePreviewToImage(img)

	imgNameTemplate := "test_image_%s_%d_%d.png"
	for _, th := range img.Thumbs {
		imgName := fmt.Sprintf(
			imgNameTemplate,
			th.Postfix,
			th.Bounds().Max.X,
			th.Bounds().Max.Y,
		)

		require.EqualValues(t, 548, th.Bounds().Max.X)
		require.EqualValues(t, 184, th.Bounds().Max.Y)
		f, err := os.Create(imgName)
		require.NoError(t, err)

		err = png.Encode(f, th)
		require.NoError(t, err)

		require.NoError(t, f.Close())
		require.NoError(t, os.Remove(imgName))
	}
}

func TestAddPreviewHighToImage1200x700(t *testing.T) {
	defer cleanUp(t)
	bb, err := base64.StdEncoding.DecodeString(base64PNG1200x200)
	require.NoError(t, err)
	img, err := image.NewImage(bb, common.ImagePngContentType)
	require.NoError(t, err)

	AddPreviewHighToImage(img)

	imgNameTemplate := "test_image_%s_%d_%d.png"
	for _, th := range img.Thumbs {
		imgName := fmt.Sprintf(
			imgNameTemplate,
			th.Postfix,
			th.Bounds().Max.X,
			th.Bounds().Max.Y,
		)

		require.EqualValues(t, previewHighWidthLarge, th.Bounds().Max.X)
		require.EqualValues(t, previewHighHeightLarge, th.Bounds().Max.Y)
		f, err := os.Create(imgName)
		require.NoError(t, err)

		err = png.Encode(f, th)
		require.NoError(t, err)

		require.NoError(t, f.Close())
		require.NoError(t, os.Remove(imgName))
	}
}

func TestAddPreviewHighToImage767x720(t *testing.T) {
	bb, err := base64.StdEncoding.DecodeString(base64PNG767x720)
	require.NoError(t, err)
	img, err := image.NewImage(bb, common.ImagePngContentType)
	require.NoError(t, err)

	AddPreviewHighToImage(img)
	for _, th := range img.Thumbs {
		imgSize := fmt.Sprintf(
			"width: %d height: %d",
			th.Bounds().Max.X,
			th.Bounds().Max.Y,
		)

		t.Log(imgSize)

		require.EqualValues(t, previewHighWidthSmall, th.Bounds().Max.X)
		require.EqualValues(t, previewHighHeightSmall, th.Bounds().Max.Y)
	}
}

func TestAddPreviewHighToImage767x723(t *testing.T) {
	bb, err := base64.StdEncoding.DecodeString(base64PNG767x723)
	require.NoError(t, err)
	img, err := image.NewImage(bb, common.ImagePngContentType)
	require.NoError(t, err)

	AddPreviewHighToImage(img)
	for _, th := range img.Thumbs {
		imgSize := fmt.Sprintf(
			"width: %d height: %d",
			th.Bounds().Max.X,
			th.Bounds().Max.Y,
		)

		t.Log(imgSize)

		require.EqualValues(t, previewHighWidthSmall, th.Bounds().Max.X)
		require.EqualValues(t, previewHighHeightSmall, th.Bounds().Max.Y)
	}
}

func TestAddPreviewHighToImage768x400(t *testing.T) {
	bb, err := base64.StdEncoding.DecodeString(base64PNG768x400)
	require.NoError(t, err)
	img, err := image.NewImage(bb, common.ImagePngContentType)
	require.NoError(t, err)

	AddPreviewHighToImage(img)
	for _, th := range img.Thumbs {
		imgSize := fmt.Sprintf(
			"width: %d height: %d",
			th.Bounds().Max.X,
			th.Bounds().Max.Y,
		)

		t.Log(imgSize)

		require.EqualValues(t, previewHighWidthLarge, th.Bounds().Max.X)
		require.EqualValues(t, previewHighHeightLarge, th.Bounds().Max.Y)
	}
}

func TestAddPreviewHighToImage900x600(t *testing.T) {
	bb, err := base64.StdEncoding.DecodeString(base64PNG900x600)
	require.NoError(t, err)
	img, err := image.NewImage(bb, common.ImagePngContentType)
	require.NoError(t, err)

	AddPreviewHighToImage(img)
	for _, th := range img.Thumbs {
		imgSize := fmt.Sprintf(
			"width: %d height: %d",
			th.Bounds().Max.X,
			th.Bounds().Max.Y,
		)

		t.Log(imgSize)

		require.EqualValues(t, previewHighWidthLarge, th.Bounds().Max.X)
		require.EqualValues(t, previewHighHeightLarge, th.Bounds().Max.Y)
	}
}
