package service

import (
	"encoding/json"

	"github.com/google/uuid"
	"gitlab.scorum.com/blog/api/db"
)

type Profile struct {
	Account     string `json:"account"`
	DisplayName string `json:"display_name"`
	Location    string `json:"location"`
	Bio         string `json:"bio"`
	AvatarUrl   string `json:"avatar_url"`
	CoverUrl    string `json:"cover_url"`
	CreatedAt   string `json:"created"`
}

func toAPIProfile(profile *db.Profile) *Profile {
	return &Profile{
		Account:     profile.Account,
		DisplayName: profile.DisplayName,
		Location:    profile.Location,
		Bio:         profile.Bio,
		AvatarUrl:   profile.AvatarUrl,
		CoverUrl:    profile.CoverUrl,
		CreatedAt:   profile.CreatedAt.Format(TimeLayout),
	}
}

func toAPIProfiles(profiles []*db.Profile) []*Profile {
	out := make([]*Profile, len(profiles))
	for idx, profile := range profiles {
		out[idx] = toAPIProfile(profile)
	}
	return out
}

type ExtendedProfile struct {
	Profile

	FollowersCount int64 `json:"followers_count"`
	FollowingCount int64 `json:"following_count"`
}

func toAPIExtendedProfile(extendedProfile *db.ExtendedProfile) *ExtendedProfile {
	return &ExtendedProfile{
		Profile:        *(toAPIProfile(&extendedProfile.Profile)),
		FollowingCount: extendedProfile.FollowingCount,
		FollowersCount: extendedProfile.FollowersCount,
	}
}

type PostID struct {
	Account  string `json:"account"`
	Permlink string `json:"permlink"`
}

func toAPIPostIDs(entries []*db.PostID) []*PostID {
	out := make([]*PostID, len(entries))
	for idx, entry := range entries {
		out[idx] = &PostID{
			Account:  entry.Account,
			Permlink: entry.Permlink,
		}
	}
	return out
}

type Category struct {
	Domain          string `json:"domain"`
	Label           string `json:"label"`
	LocalizationKey string `json:"localization_key"`
	Order           uint32 `json:"order"`
}

func toAPICategory(category db.Category) *Category {
	return &Category{
		Domain:          string(category.Domain),
		Label:           category.Label,
		LocalizationKey: category.LocalizationKey,
		Order:           category.Order,
	}
}

func toAPICategories(categories []*db.Category) []*Category {
	out := make([]*Category, len(categories))
	for idx, category := range categories {
		out[idx] = toAPICategory(*category)
	}
	return out
}

type GetMediaResult struct {
	Url  string         `json:"url"`
	Meta db.PropertyMap `json:"meta"`
}

type Draft struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Body         string `json:"body"`
	JsonMetadata string `json:"json_metadata"`
	UpdatedAt    string `json:"updated,omitempty"`
	CreatedAt    string `json:"created"`
}

func toAPIDraft(draft db.Draft) *Draft {
	return &Draft{
		ID:           draft.ID,
		Title:        draft.Title,
		Body:         draft.Body,
		JsonMetadata: draft.JsonMetadata,
		CreatedAt:    draft.CreatedAt.Format(TimeLayout),
		UpdatedAt:    draft.UpdatedAt.Format(TimeLayout),
	}
}

func toAPIDrafts(drafts []*db.Draft) []*Draft {
	out := make([]*Draft, len(drafts))
	for idx, draft := range drafts {
		out[idx] = toAPIDraft(*draft)
	}
	return out
}

type Notification struct {
	UUID      uuid.UUID           `json:"uuid"`
	Timestamp string              `json:"timestamp"`
	IsRead    bool                `json:"is_read"`
	IsSeen    bool                `json:"is_seen"`
	Type      db.NotificationType `json:"type"`
	Meta      json.RawMessage     `json:"meta"`
}

func toAPINotifications(notifications []*db.Notification) []*Notification {
	out := make([]*Notification, len(notifications))
	for idx, notification := range notifications {
		out[idx] = &Notification{
			UUID:      notification.ID,
			Timestamp: notification.Timestamp.Format(TimeLayout),
			IsRead:    notification.IsRead,
			IsSeen:    notification.IsSeen,
			Type:      notification.Type,
			Meta:      notification.Meta,
		}
	}
	return out
}

type ProfileSettings struct {
	Account                        string `json:"account"`
	EnableEmailUnseenNotifications bool   `json:"enable_email_unseen_notifications"`
}

func toAPIProfileSettings(profileSettings *db.ProfileSettings) *ProfileSettings {
	return &ProfileSettings{
		Account: profileSettings.Account,
		EnableEmailUnseenNotifications: profileSettings.EnableEmailUnseenNotifications,
	}
}
