package types

import "time"

// Response is the common response structure from Zulip API
type Response struct {
	Result string `json:"result"`
	Msg    string `json:"msg"`
	Code   string `json:"code,omitempty"`
}

// EditPropagateMode specifies how message updates should propagate
type EditPropagateMode string

const (
	ChangeOne   EditPropagateMode = "change_one"
	ChangeAll   EditPropagateMode = "change_all"
	ChangeLater EditPropagateMode = "change_later"
)

// EmojiType specifies the type of emoji
type EmojiType string

const (
	RealmEmoji      EmojiType = "realm_emoji"
	UnicodeEmoji    EmojiType = "unicode_emoji"
	ZulipExtraEmoji EmojiType = "zulip_extra_emoji"
)

// MessageFlag represents flags that can be set on messages
type MessageFlag string

const (
	FlagRead              MessageFlag = "read"
	FlagStarred           MessageFlag = "starred"
	FlagCollapsed         MessageFlag = "collapsed"
	FlagMentioned         MessageFlag = "mentioned"
	FlagWildcardMentioned MessageFlag = "wildcard_mentioned"
	FlagHasAlertWord      MessageFlag = "has_alert_word"
	FlagHistorical        MessageFlag = "historical"
)

// Message represents a Zulip message
type Message struct {
	ID                int           `json:"id"`
	Content           string        `json:"content"`
	SenderID          int           `json:"sender_id"`
	SenderEmail       string        `json:"sender_email"`
	SenderFullName    string        `json:"sender_full_name"`
	Timestamp         int64         `json:"timestamp"`
	Type              string        `json:"type"`
	DisplayRecipient  interface{}   `json:"display_recipient"`
	Subject           string        `json:"subject"`
	Flags             []MessageFlag `json:"flags"`
	Reactions         []Reaction    `json:"reactions,omitempty"`
	Submessages       []interface{} `json:"submessages,omitempty"`
	ClientID          string        `json:"client,omitempty"`
	ContentType       string        `json:"content_type,omitempty"`
	IsMeMessage       bool          `json:"is_me_message,omitempty"`
	LastEditTimestamp *int64        `json:"last_edit_timestamp,omitempty"`
	MatchContent      string        `json:"match_content,omitempty"`
	MatchSubject      string        `json:"match_subject,omitempty"`
	StreamID          int           `json:"stream_id,omitempty"`
	TopicLinks        []interface{} `json:"topic_links,omitempty"`
}

// Reaction represents an emoji reaction on a message
type Reaction struct {
	EmojiCode    string    `json:"emoji_code"`
	EmojiName    string    `json:"emoji_name"`
	ReactionType EmojiType `json:"reaction_type"`
	UserID       int       `json:"user_id"`
}

// User represents a Zulip user
type User struct {
	UserID        int                    `json:"user_id"`
	DeliveryEmail string                 `json:"delivery_email,omitempty"`
	Email         string                 `json:"email"`
	FullName      string                 `json:"full_name"`
	DateJoined    string                 `json:"date_joined,omitempty"`
	IsActive      bool                   `json:"is_active"`
	IsOwner       bool                   `json:"is_owner,omitempty"`
	IsAdmin       bool                   `json:"is_admin,omitempty"`
	IsGuest       bool                   `json:"is_guest,omitempty"`
	IsBot         bool                   `json:"is_bot"`
	BotType       *int                   `json:"bot_type,omitempty"`
	BotOwnerID    *int                   `json:"bot_owner_id,omitempty"`
	Timezone      string                 `json:"timezone,omitempty"`
	AvatarURL     string                 `json:"avatar_url,omitempty"`
	ProfileData   map[string]interface{} `json:"profile_data,omitempty"`
}

// Stream represents a Zulip stream
type Stream struct {
	StreamID                   int    `json:"stream_id"`
	Name                       string `json:"name"`
	Description                string `json:"description"`
	DateCreated                int64  `json:"date_created"`
	InviteOnly                 bool   `json:"invite_only"`
	RenderedDescription        string `json:"rendered_description,omitempty"`
	IsWebPublic                bool   `json:"is_web_public"`
	StreamPostPolicy           int    `json:"stream_post_policy"`
	MessageRetentionDays       *int   `json:"message_retention_days"`
	HistoryPublicToSubscribers bool   `json:"history_public_to_subscribers"`
	FirstMessageID             *int   `json:"first_message_id"`
	IsAnnouncementOnly         bool   `json:"is_announcement_only"`
}

// Subscription represents a user's subscription to a stream
type Subscription struct {
	StreamID               int    `json:"stream_id"`
	Name                   string `json:"name"`
	Description            string `json:"description"`
	RenderedDescription    string `json:"rendered_description"`
	DateCreated            int64  `json:"date_created"`
	InviteOnly             bool   `json:"invite_only"`
	Subscribers            []int  `json:"subscribers,omitempty"`
	DesktopNotifications   *bool  `json:"desktop_notifications"`
	EmailNotifications     *bool  `json:"email_notifications"`
	WildcardMentionsNotify *bool  `json:"wildcard_mentions_notify"`
	PushNotifications      *bool  `json:"push_notifications"`
	AudibleNotifications   *bool  `json:"audible_notifications"`
	PinToTop               bool   `json:"pin_to_top"`
	EmailAddress           string `json:"email_address"`
	IsMuted                bool   `json:"is_muted"`
	InHomeView             bool   `json:"in_home_view"`
	Color                  string `json:"color"`
	StreamWeeklyTraffic    *int   `json:"stream_weekly_traffic"`
}

// UserGroup represents a user group
type UserGroup struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Members       []int  `json:"members"`
	IsSystemGroup bool   `json:"is_system_group,omitempty"`
}

// Narrow represents a message filter
type Narrow struct {
	Operator string      `json:"operator"`
	Operand  interface{} `json:"operand"`
}

// Presence represents user presence information
type Presence struct {
	Client    string `json:"client,omitempty"`
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
	Pushable  bool   `json:"pushable,omitempty"`
}

// Event represents a Zulip event
type Event struct {
	Type string                 `json:"type"`
	ID   int                    `json:"id"`
	Data map[string]interface{} `json:"-"`
}

// Emoji represents custom emoji
type Emoji struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SourceURL   string `json:"source_url"`
	Deactivated bool   `json:"deactivated"`
	AuthorID    *int   `json:"author_id"`
}

// Linkifier represents a realm linkifier/filter
type Linkifier struct {
	ID          int    `json:"id"`
	Pattern     string `json:"pattern"`
	URLTemplate string `json:"url_template,omitempty"`
	URLFormat   string `json:"url_format,omitempty"`
}

// ProfileField represents a custom profile field
type ProfileField struct {
	ID                      int    `json:"id"`
	Type                    int    `json:"type"`
	Order                   int    `json:"order"`
	Name                    string `json:"name"`
	Hint                    string `json:"hint"`
	FieldData               string `json:"field_data,omitempty"`
	DisplayInProfileSummary bool   `json:"display_in_profile_summary,omitempty"`
}

// Attachment represents an uploaded file
type Attachment struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	PathID     string    `json:"path_id"`
	Size       int64     `json:"size"`
	CreateTime time.Time `json:"create_time"`
	Messages   []struct {
		ID       int   `json:"id"`
		DateSent int64 `json:"date_sent"`
	} `json:"messages"`
}

// Topic represents a stream topic
type Topic struct {
	Name  string `json:"name"`
	MaxID int    `json:"max_id"`
}

// ServerSettings represents server configuration
type ServerSettings struct {
	Result                      string `json:"result"`
	Msg                         string `json:"msg"`
	ZulipVersion                string `json:"zulip_version"`
	ZulipFeatureLevel           int    `json:"zulip_feature_level"`
	PushNotificationsEnabled    bool   `json:"push_notifications_enabled"`
	IsIncompatible              bool   `json:"is_incompatible,omitempty"`
	EmailAuthEnabled            bool   `json:"email_auth_enabled,omitempty"`
	RequireEmailFormatUsernames bool   `json:"require_email_format_usernames,omitempty"`
	RealmURI                    string `json:"realm_uri,omitempty"`
	RealmName                   string `json:"realm_name,omitempty"`
	RealmIcon                   string `json:"realm_icon,omitempty"`
	RealmDescription            string `json:"realm_description,omitempty"`
}
