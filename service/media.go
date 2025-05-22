package service

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/rpc"
	"gitlab.scorum.com/blog/api/service/image"
	"gitlab.scorum.com/blog/api/utils/postgres"
)

var (
	previewWidthLarge  = 768
	previewWidthSmall  = 384
	previewHeightLarge = 352
	previewHeightSmall = 176

	previewHighWidthLarge  = 768
	previewHighWidthSmall  = 384
	previewHighHeightLarge = 720
	previewHighHeightSmall = 360

	profilePreviewWidthLarge  = 1096
	profilePreviewWidthSmall  = 548
	profilePreviewHeightLarge = 368
	profilePreviewHeightSmall = 184

	previewWideWidth  = 793
	previewWideHeight = 360

	previewNotificationSizeLarge = 96
	previewNotificationSizeSmall = 48
)

func (blog *Blog) UploadMedia(op types.Operation) *rpc.Error {
	in := op.(*types.UploadMediaOperation)

	exists, err := blog.checkAccountExists(in.Account)
	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	if !exists {
		return NewError(rpc.ProfileNotFoundCode, fmt.Sprintf("%s account does not exist", in.Account))
	}

	if !isMediaAllowedContentType(in.ContentType) {
		return NewError(rpc.InvalidMediaTypeCode, "invalid content_type")
	}

	mediaID := strings.ToLower(in.ID)

	err = blog.DB.Read.Get(&exists, `SELECT EXISTS(SELECT * FROM media WHERE account = $1 AND id = $2)`, in.Account, mediaID)
	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	if exists {
		return NewError(rpc.MediaAlreadyExistsCode, "media id already exists")
	}

	// validate
	rawBytes, err := base64.StdEncoding.DecodeString(in.Media)
	if err != nil {
		return WrapError(rpc.InvalidMediaCode, err)
	}

	// Note, for the time being only images are supported
	img, err := image.NewImage(rawBytes, in.ContentType)
	if err != nil {
		if err == image.ErrInvalidFormat {
			return WrapError(rpc.InvalidParameterCode, err)
		}
		return WrapError(rpc.InternalErrorCode, err)
	}

	originalSize := img.OriginalSize()

	// check image size
	if originalSize.X < imageSizeLimit ||
		originalSize.Y < imageSizeLimit {
		return NewError(
			rpc.ImageTooSmallCode,
			fmt.Sprintf("image is too small, min size is %dx%d", imageSizeLimit, imageSizeLimit))
	}

	thresholds := []int{96, 384, 500, 800, 1000}
	for _, threshold := range thresholds {
		if img.Max() >= threshold {
			img.AddThumb(strconv.Itoa(threshold), threshold, threshold)
		}
	}

	AddPreviewToImage(img)
	AddPreviewWideToImage(img)
	AddNotificationPreviewToImage(img)
	AddProfilePreviewToImage(img)
	AddPreviewHighToImage(img)

	// upload images
	url, err := blog.uploadImage(in.Account, in.ID, img)
	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}
	_, err = blog.uploadThumbnails(in.Account, in.ID, img)
	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	log.Debugf("%s uploaded blob: %s", in.Account, url)

	meta := make(db.PropertyMap, 0)
	meta["width"] = originalSize.X
	meta["height"] = originalSize.Y

	// save to db
	_, err = blog.DB.Write.NamedExec(
		`INSERT INTO media (account, id, url, content_type, meta)
					VALUES (:account, :id, :url, :content_type, :meta)`, &db.Media{
			Account:     in.Account,
			ID:          in.ID,
			Url:         url,
			ContentType: in.ContentType,
			Meta:        meta,
		})
	if err != nil {
		if isErr, _ := postgres.IsUniqueError(err); isErr {
			return NewError(rpc.MediaAlreadyExistsCode, "media id already exists")
		}
		return WrapError(rpc.InternalErrorCode, err)
	}

	return nil
}

func AddPreviewToImage(img *image.Image) {
	const previewPrefix = "preview"

	height, cropHeight := previewHeightLarge, previewHeightLarge
	width, cropWidth := previewWidthLarge, previewWidthLarge

	originalSize := img.OriginalSize()

	p := decimal.NewFromFloat(float64(originalSize.X)).Div(decimal.NewFromFloat(float64(originalSize.Y)))

	if p.GreaterThanOrEqual(decimal.NewFromFloat(2.18)) {
		if originalSize.Y < previewHeightLarge {
			height, cropHeight = previewHeightSmall, previewHeightSmall
			cropWidth = previewWidthSmall
		}
		width = int(p.Mul(decimal.NewFromFloat(float64(height))).IntPart())
	} else {
		if originalSize.X < previewWidthLarge {
			width, cropWidth = previewWidthSmall, previewWidthSmall
			cropHeight = previewHeightSmall
		}
		height = int(decimal.NewFromFloat(float64(width)).Div(p).IntPart())
	}

	img.AddThumbNeat(previewPrefix, width, height, cropWidth, cropHeight)
}

func AddPreviewWideToImage(img *image.Image) {
	const previewPrefix = "preview_wide"

	height, cropHeight := previewWideHeight, previewWideHeight
	width, cropWidth := previewWideWidth, previewWideWidth

	originalSize := img.OriginalSize()

	p := decimal.NewFromFloat(float64(originalSize.X)).Div(decimal.NewFromFloat(float64(originalSize.Y)))

	if p.GreaterThanOrEqual(decimal.NewFromFloat(2.2)) {
		width = int(p.Mul(decimal.NewFromFloat(float64(height))).IntPart())
	} else {
		height = int(decimal.NewFromFloat(float64(width)).Div(p).IntPart())
	}

	img.AddThumbNeat(previewPrefix, width, height, cropWidth, cropHeight)
}

func AddPreviewHighToImage(img *image.Image) {
	const previewHighPostfix = "preview_high"

	height, cropHeight := previewHighHeightLarge, previewHighHeightLarge
	width, cropWidth := previewHighWidthLarge, previewHighWidthLarge

	originalSize := img.OriginalSize()

	// because of accuracy it's better to define AspectRation with width/height relation than constant
	aspectRatio := decimal.NewFromFloat(float64(previewHighWidthLarge)).
		Div(decimal.NewFromFloat(float64(previewHighHeightLarge)))

	p := decimal.NewFromFloat(float64(originalSize.X)).Div(decimal.NewFromFloat(float64(originalSize.Y)))
	if p.GreaterThanOrEqual(aspectRatio) {
		if originalSize.X < previewHighWidthLarge {
			height, cropHeight = previewHighHeightSmall, previewHighHeightSmall
			cropWidth = previewHighWidthSmall
		}
		width = int(p.Mul(decimal.NewFromFloat(float64(height))).IntPart())
	} else {
		if originalSize.X < previewHighWidthLarge {
			width, cropWidth = previewHighWidthSmall, previewHighWidthSmall
			cropHeight = previewHighHeightSmall
		}
		height = int(decimal.NewFromFloat(float64(width)).Div(p).IntPart())
	}

	img.AddThumbNeat(previewHighPostfix, width, height, cropWidth, cropHeight)
}

func (blog *Blog) uploadThumbnails(account string, id string, image *image.Image) ([]string, error) {
	thumbs := image.Thumbs
	urls := make([]string, len(thumbs))

	// upload thumbs
	for i, thumb := range thumbs {
		var buffer bytes.Buffer
		if err := image.Encode(&buffer, thumb.Image); err != nil {
			return nil, errors.Wrapf(err, "failed to encode %s thumb", thumb.Postfix)
		}

		url, err := blog.Blob.UploadMedia(account, fmt.Sprintf("%s_%s", id, thumb.Postfix), buffer.Bytes(), image.ContentType)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to upload %s thumb", thumb.Postfix)
		}

		urls[i] = url
	}

	return urls, nil
}

func AddNotificationPreviewToImage(img *image.Image) {
	const previewPostfix = "notifications_thumb"

	height, cropHeight := previewNotificationSizeLarge, previewNotificationSizeLarge
	width, cropWidth := previewNotificationSizeLarge, previewNotificationSizeLarge

	originalSize := img.OriginalSize()

	p := decimal.NewFromFloat(float64(originalSize.X)).Div(decimal.NewFromFloat(float64(originalSize.Y)))

	if p.GreaterThanOrEqual(decimal.NewFromFloat(1)) {
		if originalSize.Y < previewNotificationSizeSmall {
			height, cropHeight = previewNotificationSizeSmall, previewNotificationSizeSmall
			cropWidth = previewNotificationSizeSmall
		}
		width = int(p.Mul(decimal.NewFromFloat(float64(height))).IntPart())
	} else {
		if originalSize.X < previewNotificationSizeSmall {
			width, cropWidth = previewNotificationSizeSmall, previewNotificationSizeSmall
			cropHeight = previewNotificationSizeSmall
		}
		height = int(decimal.NewFromFloat(float64(width)).Div(p).IntPart())
	}

	img.AddThumbNeat(previewPostfix, width, height, cropWidth, cropHeight)
}

func AddProfilePreviewToImage(img *image.Image) {
	const previewPostfix = "profile_preview"

	height, cropHeight := profilePreviewHeightLarge, profilePreviewHeightLarge
	width, cropWidth := profilePreviewWidthLarge, profilePreviewWidthLarge

	originalSize := img.OriginalSize()

	p := decimal.NewFromFloat(float64(originalSize.X)).Div(decimal.NewFromFloat(float64(originalSize.Y)))

	if p.GreaterThanOrEqual(decimal.NewFromFloat(2.97)) {
		if originalSize.Y < profilePreviewHeightLarge {
			height, cropHeight = profilePreviewHeightSmall, profilePreviewHeightSmall
			cropWidth = profilePreviewWidthSmall
		}
		width = int(p.Mul(decimal.NewFromFloat(float64(height))).IntPart())
	} else {
		if originalSize.X < profilePreviewWidthLarge {
			width, cropWidth = profilePreviewWidthSmall, profilePreviewWidthSmall
			cropHeight = profilePreviewHeightSmall
		}
		height = int(decimal.NewFromFloat(float64(width)).Div(p).IntPart())
	}

	img.AddThumbNeat(previewPostfix, width, height, cropWidth, cropHeight)
}

func (blog *Blog) uploadImage(account string, id string, image *image.Image) (string, error) {
	var buffer bytes.Buffer
	if err := image.Encode(&buffer, image.Original); err != nil {
		return "", errors.Wrap(err, "failed to encode original")
	}

	// upload original
	url, err := blog.Blob.UploadMedia(account, id, buffer.Bytes(), image.ContentType)
	if err != nil {
		return "", errors.Wrap(err, "failed to upload original image")
	}

	return url, nil
}

func (blog *Blog) GetMedia(ctx *rpc.Context) {
	var account string
	if err := ctx.Param(0, &account); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	var id string
	if err := ctx.Param(1, &id); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	if err := validate.Var(id, "required,max=16,alphanum"); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	url, err := blog.doGetMedia(account, id)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	ctx.WriteResult(url)
}

func (blog *Blog) doGetMedia(account string, id string) (*GetMediaResult, *rpc.Error) {
	var out GetMediaResult
	err := blog.DB.Read.Get(&out, `SELECT url, meta FROM media WHERE account = $1 AND id = $2`, account, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewError(rpc.MediaNotFoundCode, fmt.Sprintf("media %s not found", id))
		}
		return nil, WrapError(rpc.InternalErrorCode, err)
	}
	return &out, nil
}
