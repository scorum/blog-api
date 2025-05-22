package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/common"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/push"
)

func TestBlog_UpdateProfile(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)

	// avatar
	err := handler.UploadMedia(&types.UploadMediaOperation{
		Account:     leonarda,
		Media:       base64PNG1200x700,
		ID:          "avatar",
		ContentType: common.ImagePngContentType,
	})
	require.Nil(t, err)

	avatar, err := handler.doGetMedia(leonarda, "avatar")
	require.Nil(t, err)

	// cover
	err = handler.UploadMedia(&types.UploadMediaOperation{
		Account:     leonarda,
		Media:       base64PNG1200x700,
		ID:          "cover",
		ContentType: common.ImagePngContentType,
	})
	require.Nil(t, err)

	cover, err := handler.doGetMedia(leonarda, "cover")
	require.Nil(t, err)

	// update
	upo := &types.UpdateProfileOperation{
		Account:     leonarda,
		Location:    "minsk",
		Bio:         "bio",
		DisplayName: "zebra",
		AvatarUrl:   avatar.Url,
		CoverUrl:    cover.Url,
	}

	require.Nil(t, handler.UpdateProfile(upo))

	// assert
	profile, err := handler.doGetProfile(leonarda)
	require.Error(t, err)
	require.Equal(t, upo.Account, profile.Account)
	require.Equal(t, upo.Location, profile.Location)
	require.Equal(t, upo.Bio, profile.Bio)
	require.Equal(t, upo.AvatarUrl, profile.AvatarUrl)
	require.Equal(t, upo.CoverUrl, profile.CoverUrl)

	// avatar thumbs
	exists, err2 := handler.Blob.DoesMediaExists(leonarda, fmt.Sprintf("%s_%d", "avatar", 64))
	require.NoError(t, err2)
	require.True(t, exists)
	exists, err2 = handler.Blob.DoesMediaExists(leonarda, fmt.Sprintf("%s_%d", "avatar", 200))
	require.NoError(t, err2)
	require.True(t, exists)
}

func TestBlog_UpdateProfile_ValidationTest(t *testing.T) {
	defer cleanUp(t)

	op := types.UpdateProfileOperation{
		Account:     leonarda,
		Location:    "minsk",
		Bio:         "bio",
		DisplayName: "zebra",
		AvatarUrl:   "",
		CoverUrl:    "",
	}

	t.Run("valid_operation", func(t *testing.T) {
		require.NoError(t, validate.Struct(op))
	})

	t.Run("invalid_account", func(t *testing.T) {
		cop := op
		cop.Account = ""
		require.Error(t, validate.Struct(cop))

	})

	t.Run("invalid_location", func(t *testing.T) {
		cop := op
		cop.Location = "morethantwentyfivesymbols."
		require.Error(t, validate.Struct(cop))
	})

	t.Run("invalid_bio", func(t *testing.T) {
		cop := op
		cop.Bio = `morethen160symbolsmorethen160symbolsmorethen160symbolsmorethen160symbolsmorethen160symbols
			"morethdn160symbolsmorethen160symbolsmorethen160symbolsmorethen160symbolsmorethen160symbolsmorethen160symbols`
		require.Error(t, validate.Struct(cop))
	})

	t.Run("invalid_display_name", func(t *testing.T) {
		cop := op
		cop.DisplayName = "morethanfiftysymbolsmorethanfiftysymbolsmorethanfiftysymbols"
		require.Error(t, validate.Struct(cop))
	})

	t.Run("invalid_avatar_url", func(t *testing.T) {
		cop := op
		cop.AvatarUrl = "noturi"
		require.Error(t, validate.Struct(cop))
	})

	t.Run("invalid_cover_url", func(t *testing.T) {
		cop := op
		cop.CoverUrl = "noturi"
		require.Error(t, validate.Struct(cop))
	})

	t.Run("media_content_type_validation", func(t *testing.T) {
		registerAccount(t, leonarda)

		err := handler.UploadMedia(&types.UploadMediaOperation{
			Account:     leonarda,
			Media:       base64Gif400x300,
			ID:          "avatar",
			ContentType: common.ImageGifContentType,
		})
		require.Nil(t, err)

		avatar, err := handler.doGetMedia(leonarda, "avatar")
		require.Nil(t, err)

		t.Run("invalid_avatar_content_type", func(t *testing.T) {
			cop := op
			cop.AvatarUrl = avatar.Url
			require.NoError(t, validate.Struct(cop))
			require.NotNil(t, handler.UpdateProfile(&cop))
		})

		t.Run("invalid_cover_content_type", func(t *testing.T) {
			cop := op
			cop.CoverUrl = avatar.Url
			require.NoError(t, validate.Struct(cop))
			require.NotNil(t, handler.UpdateProfile(&cop))
		})
	})
}

func TestGetProfile(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	registerAccount(t, kristie)
	upo := &types.UpdateProfileOperation{
		Account:     leonarda,
		Location:    "minsk",
		Bio:         "bio",
		DisplayName: "zebra",
		AvatarUrl:   "",
		CoverUrl:    "",
	}
	require.Nil(t, handler.UpdateProfile(upo))

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	notifier := push.NewMockNotifier(mockCtrl)
	notifier.EXPECT().NotifyStartedFollow(gomock.Any(), gomock.Any()).Times(2)
	handler.Notifier = notifier

	// follow to update counters
	require.Nil(t, handler.Follow(&types.FollowOperation{
		Account: leonarda,
		Follow:  kristie,
	}))

	require.Nil(t, handler.Follow(&types.FollowOperation{
		Account: kristie,
		Follow:  leonarda,
	}))

	profile, err := handler.doGetProfile(leonarda)
	require.Nil(t, err)
	require.Equal(t, profile.Account, upo.Account)
	require.Equal(t, profile.FollowersCount, int64(1))
	require.Equal(t, profile.FollowingCount, int64(1))
	require.Equal(t, profile.Location, upo.Location)
	require.Equal(t, profile.Bio, upo.Bio)
	require.Equal(t, profile.DisplayName, upo.DisplayName)
	require.Equal(t, profile.AvatarUrl, upo.AvatarUrl)
	require.Equal(t, profile.CoverUrl, upo.CoverUrl)
}

func TestGetProfiles(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	registerAccount(t, kristie)
	registerAccount(t, sheldon)

	t.Run("valid_request", func(t *testing.T) {
		profiles, err := handler.doGetProfiles([]string{leonarda, kristie, sheldon})
		require.Nil(t, err)
		require.Len(t, profiles, 3)

		// make sure order is preserved
		require.Equal(t, leonarda, profiles[0].Account)
		require.Equal(t, kristie, profiles[1].Account)
		require.Equal(t, sheldon, profiles[2].Account)
	})

	t.Run("not exists", func(t *testing.T) {
		_, err := handler.doGetProfile("not_exists")
		require.NotNil(t, err)
	})

	t.Run("empty_slice", func(t *testing.T) {
		profiles, err := handler.doGetProfiles([]string{})
		require.Nil(t, err)
		require.Empty(t, profiles)
	})

	t.Run("same_accounts", func(t *testing.T) {
		profiles, err := handler.doGetProfiles([]string{leonarda, leonarda, leonarda})
		require.Nil(t, err)
		require.Len(t, profiles, 1)
		require.Equal(t, profiles[0].Account, leonarda)
		require.Equal(t, profiles[0].DisplayName, leonarda)
	})
}

func addProfileSettings(t *testing.T, account string) {
	_, err := dbWrite.Exec(
		`INSERT INTO profile_settings(account, enable_email_unseen_notifications)
			   VALUES($1, $2) ON CONFLICT DO NOTHING`,
		account, true)
	require.NoError(t, err)
}

func TestGetProfileSettings(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	addProfileSettings(t, leonarda)

	settings, err := handler.doGetProfileSettings(leonarda)
	require.Nil(t, err)
	require.NotNil(t, settings)

	settings, err = handler.doGetProfileSettings(leonarda)
	require.Nil(t, err)
	require.NotNil(t, settings)
}

func TestUpdateProfileSettings(t *testing.T) {
	defer cleanUp(t)

	registerAccount(t, leonarda)
	addProfileSettings(t, leonarda)

	settings, err := handler.doGetProfileSettings(leonarda)
	require.Nil(t, err)
	require.NotNil(t, settings)
	require.True(t, settings.EnableEmailUnseenNotifications)

	newSettings := db.ProfileSettings{
		Account: settings.Account,
		EnableEmailUnseenNotifications: false,
	}

	err = handler.doUpsertProfileSettings(&newSettings)
	require.Nil(t, err)

	settings, err = handler.doGetProfileSettings(leonarda)
	require.Nil(t, err)
	require.NotNil(t, settings)
	require.False(t, settings.EnableEmailUnseenNotifications)
}

func TestBlog_SetAccountTrustedAdmin(t *testing.T) {
	defer cleanUp(t)
	registerAccount(t, leonarda)

	isTrusted, err := handler.checkIsAccountTrusted(leonarda)
	require.Nil(t, err)
	require.False(t, *isTrusted)

	setTrusted(t, leonarda)

	isTrusted, err = handler.checkIsAccountTrusted(leonarda)
	require.Nil(t, err)
	require.True(t, *isTrusted)
}

func TestBlog_IsAccountTrusted(t *testing.T) {
	isTrusted, err := handler.checkIsAccountTrusted("not created account")
	require.Nil(t, err)
	require.False(t, *isTrusted)
}

func setTrusted(t *testing.T, account string) {
	require.Nil(t, handler.SetAccountTrustedAdmin(&types.SetAccountTrustedAdminOperation{
		Account:     account,
		BlogAccount: account,
		IsTrusted:   true,
	}))
}

func TestBlog_GetTrusted(t *testing.T) {
	defer cleanUp(t)
	registerAccount(t, leonarda)

	trusted, err := handler.doGetTrusted(0, 10)
	require.Nil(t, err)
	require.Empty(t, trusted)

	setTrusted(t, leonarda)

	trusted, err = handler.doGetTrusted(0, 10)
	require.Nil(t, err)
	require.Len(t, trusted, 1)
	require.Equal(t, leonarda, trusted[0].Account)
}

func TestUnsubscribeEndpoint(t *testing.T) {
	defer cleanUp(t)
	require.Nil(t, handler.Register(&types.RegisterOperation{"cali4888"}))

	r := httptest.NewRequest(
		"POST",
		"https://blog-api.scorum.com/unsubscribe?jwt=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2NvdW50IjoiY2FsaTQ4ODgiLCJyZWRpcmVjdC11cmwiOiJodHRwcy8vc2NvcnVtLmNvbS91bnN1YnNjcmliZT91dG1fc291cmNlPXNjb3J1bVx1MDAyNnV0bV9tZWRpdW09ZW1haWxcdTAwMjZ1dG1fY2FtcGFpZ249dW5zdWJzY3JpYmUifQ.RkT5lGrP7qr8edjS8Qx1XjD4G7rqa-EnNVFM7NQOUSQ",
		nil,
	)
	w := httptest.ResponseRecorder{}

	handler.Config.UnsubscribeApiJwtSecret = "sosecret"
	handler.UnsubscribeEndpoint(&w, r)

	require.EqualValues(t, w.Code, http.StatusTemporaryRedirect)
}
