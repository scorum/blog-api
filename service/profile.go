package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/dgrijalva/jwt-go"
	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/rpc"
	"gitlab.scorum.com/blog/api/service/image"
)

// Register a new account.
// Note blockchain monitor does registration as well, but because of blockchain consensus
// it takes time to propagate data (wait for last irreversible block), therefore
// this operation should be called via frontend as soon as new account created
func (blog *Blog) Register(op types.Operation) (rpcErr *rpc.Error) {
	in := op.(*types.RegisterOperation)

	tx, err := blog.DB.Write.Beginx()

	defer func() {
		if rpcErr != nil {
			tx.Rollback()
		}
	}()

	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	if _, err := tx.Exec(
		`INSERT INTO profiles(account, display_name)
                VALUES($1, $2)
                ON CONFLICT DO NOTHING`,
		in.Account, in.Account); err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	if _, err := tx.Exec(
		`INSERT INTO profile_settings(account, enable_email_unseen_notifications)
                VALUES($1, $2)
                ON CONFLICT DO NOTHING`,
		in.Account, true); err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	if err := tx.Commit(); err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	return nil
}

func (blog *Blog) UpdateProfile(op types.Operation) (rpcErr *rpc.Error) {
	in := op.(*types.UpdateProfileOperation)

	tx, err := blog.DB.Write.Beginx()

	defer func() {
		if rpcErr != nil {
			tx.Rollback()
		}
	}()

	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	var profile db.Profile
	err = tx.Get(&profile,
		`SELECT account, display_name, location, bio, avatar_url, cover_url, created_at
		FROM profiles WHERE account = $1 FOR UPDATE`,
		in.Account)
	if err != nil {
		if err == sql.ErrNoRows {
			return NewError(rpc.ProfileNotFoundCode, fmt.Sprintf("%s not found", in.Account))
		}
		return WrapError(rpc.InternalErrorCode, err)
	}

	avatar := in.AvatarUrl
	if avatar != "" {
		// validate avatar url
		media, err := blog.getMediaByUrl(in.Account, avatar)
		if err != nil {
			if err == mediaNotFoundErr {
				return NewError(rpc.MediaNotFoundCode, fmt.Sprintf("%s is not your media resource", avatar))
			}
			return WrapError(rpc.InternalErrorCode, err)
		}

		if !isProfileAllowedContentType(media.ContentType) {
			return NewError(rpc.InvalidMediaTypeCode, "invalid media content-type")
		}

		// create avatar images from origin
		if avatar != profile.AvatarUrl {
			err := blog.makeAndUploadAvatars(*media)
			if err != nil {
				return err
			}
		}
	}

	cover := in.CoverUrl
	if cover != "" {
		// validate cover url
		media, err := blog.getMediaByUrl(in.Account, cover)
		if err != nil {
			if err == mediaNotFoundErr {
				return NewError(rpc.MediaNotFoundCode, fmt.Sprintf("%s is not your media resource", cover))
			}
			return WrapError(rpc.InternalErrorCode, err)
		}

		if !isProfileAllowedContentType(media.ContentType) {
			return NewError(rpc.InvalidMediaTypeCode, "invalid media content-type")
		}
	}

	profile = db.Profile{
		Account:     in.Account,
		DisplayName: in.DisplayName,
		Location:    in.Location,
		Bio:         in.Bio,
		AvatarUrl:   avatar,
		CoverUrl:    cover,
	}

	_, err = tx.NamedExec(
		`UPDATE profiles
				SET
				  location = :location,
				  display_name= :display_name,
				  bio = :bio,
				  avatar_url = :avatar_url,
				  cover_url = :cover_url
				WHERE account = :account`, profile)
	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	if err := tx.Commit(); err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	return nil
}

func (blog *Blog) GetProfile(ctx *rpc.Context) {
	var account string
	if err := ctx.Param(0, &account); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	profile, err := blog.doGetProfile(account)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	ctx.WriteResult(profile)
}

func (blog *Blog) GetProfiles(ctx *rpc.Context) {
	var accounts []string
	if err := ctx.Param(0, &accounts); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	profiles, err := blog.doGetProfiles(accounts)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	ctx.WriteResult(profiles)
}

func (blog *Blog) GetProfileSettings(ctx *rpc.Context, account string, params []*json.RawMessage) {
	profileSettings, err := blog.doGetProfileSettings(account)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	ctx.WriteResult(profileSettings)
}

func (blog *Blog) UnsubscribeEndpoint(w http.ResponseWriter, r *http.Request) {
	jwtToken := r.URL.Query().Get("jwt")
	if jwtToken == "" {
		w.WriteHeader(http.StatusBadRequest)
		log.Debugf("jwt not found in %s", r.URL.String())
		return
	}

	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(blog.Config.UnsubscribeApiJwtSecret), nil
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Debugf("error while parsing jwt err:%s", err)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims.Valid() != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Debug("jwt is not valid")
		return
	}

	account, ok := claims["account"].(string)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing account"))
		return
	}

	redirect, ok := claims["redirect-url"].(string)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing redirect-url"))
		return
	}

	settings := &db.ProfileSettings{
		Account: account,
		EnableEmailUnseenNotifications: false,
	}

	updateErr := blog.doUpsertProfileSettings(settings)
	if updateErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Errorf("error while updating profile settings err:%s", updateErr)
		return
	}

	http.Redirect(w, r, redirect, http.StatusTemporaryRedirect)
}

func (blog *Blog) UpdateProfileSettings(op types.Operation) *rpc.Error {
	in := op.(*types.UpdateProfileSettingsOperation)

	settings := &db.ProfileSettings{
		Account: in.Account,
		EnableEmailUnseenNotifications: in.EnableEmailUnseenNotifications,
	}

	err := blog.doUpsertProfileSettings(settings)
	if err != nil {
		return WrapError(err.Code, err)
	}

	return nil
}

func (blog *Blog) doGetProfile(account string) (*ExtendedProfile, *rpc.Error) {
	var profile db.ExtendedProfile
	err := blog.DB.Read.Get(&profile,
		`SELECT account, display_name, location, bio, avatar_url, cover_url, created_at,
				(SELECT COUNT(*) FROM followers WHERE follow_account=$1) followers_count,
				(SELECT COUNT(*) FROM followers WHERE account=$1) following_count
				FROM profiles WHERE account = $1`, account)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewError(rpc.ProfileNotFoundCode, fmt.Sprintf("%s not found", account))
		}
		return nil, WrapError(rpc.InternalErrorCode, err)
	}

	return toAPIExtendedProfile(&profile), nil
}

func (blog *Blog) doGetProfiles(accounts []string) ([]*Profile, *rpc.Error) {
	var profiles []*db.Profile

	if len(accounts) == 0 {
		return toAPIProfiles(profiles), nil
	}

	// select profiles, preserving order of the given `accounts` arg
	err := blog.DB.Read.Select(&profiles, `
			SELECT account, display_name, location, bio, avatar_url, cover_url, created_at
			FROM profiles JOIN  UNNEST($1::TEXT[]) WITH ORDINALITY t(account, ord) USING (account)
			ORDER  BY t.ord`,
		pq.Array(uniqueStrings(accounts)))

	if err != nil {
		return nil, WrapError(rpc.InternalErrorCode, err)
	}

	return toAPIProfiles(profiles), nil
}

func (blog *Blog) doGetProfileSettings(account string) (*ProfileSettings, *rpc.Error) {
	var profileSettings db.ProfileSettings

	err := blog.DB.Read.Get(&profileSettings,
		`
			SELECT account, enable_email_unseen_notifications
			FROM profile_settings
			WHERE account=$1`, account)

	if err != nil {
		return nil, WrapError(rpc.InternalErrorCode, err)
	}

	return toAPIProfileSettings(&profileSettings), nil
}

func (blog *Blog) doUpsertProfileSettings(settings *db.ProfileSettings) *rpc.Error {
	_, err := blog.DB.Write.NamedExec(`
													 INSERT
													 INTO profile_settings (account, enable_email_unseen_notifications)
													 VALUES (:account, :enable_email_unseen_notifications)
													 ON CONFLICT (account) DO UPDATE
													 SET account=:account, enable_email_unseen_notifications=:enable_email_unseen_notifications
													 `,
		settings)

	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	return nil
}

func (blog *Blog) SetAccountTrustedAdmin(op types.Operation) *rpc.Error {
	in := op.(*types.SetAccountTrustedAdminOperation)

	_, err := blog.DB.Write.Exec(`UPDATE profiles SET is_trusted = $2 WHERE account = $1`, in.BlogAccount, in.IsTrusted)

	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	return nil
}

func (blog *Blog) IsAccountTrusted(ctx *rpc.Context) {
	var account string
	if err := ctx.Param(0, &account); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	isTrusted, err := blog.checkIsAccountTrusted(account)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	ctx.WriteResult(isTrusted)
}

func (blog *Blog) checkIsAccountTrusted(account string) (*bool, *rpc.Error) {
	isTrusted := false
	err := blog.DB.Read.Get(&isTrusted, `SELECT is_trusted FROM profiles WHERE account = $1`, account)
	if err != nil && err != sql.ErrNoRows {
		return nil, WrapError(rpc.InternalErrorCode, err)
	}
	return &isTrusted, nil
}

func (blog *Blog) GetTrusted(ctx *rpc.Context) {
	var from uint32
	if err := ctx.Param(0, &from); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	var limit uint16
	if err := ctx.Param(1, &limit); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	if limit > maxLargePageSize {
		ctx.WriteError(rpc.InvalidParameterCode, "invalid limit")
		return
	}

	entries, err := blog.doGetTrusted(from, limit)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	ctx.WriteResult(entries)
}

func (blog *Blog) doGetTrusted(from uint32, limit uint16) ([]*Profile, *rpc.Error) {
	var profiles []*db.Profile
	err := blog.DB.Read.Select(&profiles,
		`SELECT account, display_name, location, bio, avatar_url, cover_url, created_at
			    FROM profiles
				WHERE is_trusted = TRUE
				LIMIT $1 OFFSET $2`, limit, from)
	if err != nil {
		return nil, WrapError(rpc.InternalErrorCode, err)
	}

	return toAPIProfiles(profiles), nil
}

func (blog *Blog) makeAndUploadAvatars(media db.Media) *rpc.Error {
	file, err := downloadFile(media.Url)

	if err != nil {
		return NewError(rpc.InternalErrorCode, "failed to get avatar")
	}

	img, err := image.NewImage(file, media.ContentType)
	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	thresholds := []int{64, 200}
	for _, threshold := range thresholds {
		img.AddThumb(strconv.Itoa(threshold), threshold, threshold)
	}

	_, err = blog.uploadThumbnails(media.Account, media.ID, img)
	if err != nil {
		return WrapError(rpc.InternalErrorCode, err)
	}

	return nil
}
