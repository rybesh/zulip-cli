package types

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Response is the common response structure from Zulip API
type Response struct {
	Result string `json:"result"`
	Msg    string `json:"msg"`
	Code   string `json:"code,omitempty"`
	// IgnoredParameters lists parameters the server did not recognize. Zulip
	// accepts such a request and reports the parameters here, so a non-empty
	// value means part of what was asked for had no effect.
	IgnoredParameters []string `json:"ignored_parameters_unsupported,omitempty"`
}

// EditPropagateMode specifies how message updates should propagate
type EditPropagateMode string

const (
	ChangeOne   EditPropagateMode = "change_one"
	ChangeAll   EditPropagateMode = "change_all"
	ChangeLater EditPropagateMode = "change_later"
)

// EditPropagateModes lists the modes in the order they are documented, for
// help text and error messages.
var EditPropagateModes = []EditPropagateMode{ChangeOne, ChangeLater, ChangeAll}

// ParseEditPropagateMode turns what someone typed into a propagate mode.
func ParseEditPropagateMode(value string) (EditPropagateMode, error) {
	mode := EditPropagateMode(strings.ToLower(value))
	if slices.Contains(EditPropagateModes, mode) {
		return mode, nil
	}
	return "", fmt.Errorf("unknown propagate mode %q: it must be one of %s",
		value, joinNames(EditPropagateModes))
}

// TopicVisibilityPolicy is a user's personal preference for one topic. Zulip
// sends it as an integer, and the zero value is the absence of a preference.
type TopicVisibilityPolicy int

const (
	// VisibilityInherit removes any policy set for the topic, leaving it to
	// follow whatever the channel does.
	VisibilityInherit TopicVisibilityPolicy = 0
	// VisibilityMuted hides the topic.
	VisibilityMuted TopicVisibilityPolicy = 1
	// VisibilityUnmuted shows the topic even though its channel is muted. In
	// an unmuted channel it does the same thing as VisibilityInherit.
	VisibilityUnmuted TopicVisibilityPolicy = 2
	// VisibilityFollowed follows the topic.
	VisibilityFollowed TopicVisibilityPolicy = 3
)

// TopicVisibilityPolicies lists the policies in the order they are documented,
// for help text and error messages.
var TopicVisibilityPolicies = []TopicVisibilityPolicy{
	VisibilityInherit, VisibilityMuted, VisibilityUnmuted, VisibilityFollowed,
}

// String names the policy the way ParseTopicVisibilityPolicy reads it back.
func (p TopicVisibilityPolicy) String() string {
	switch p {
	case VisibilityInherit:
		return "inherit"
	case VisibilityMuted:
		return "muted"
	case VisibilityUnmuted:
		return "unmuted"
	case VisibilityFollowed:
		return "followed"
	default:
		return fmt.Sprintf("visibility policy %d", int(p))
	}
}

// ParseTopicVisibilityPolicy turns what someone typed into a policy. "none" is
// accepted for inherit, since that is what the API documentation calls it.
func ParseTopicVisibilityPolicy(value string) (TopicVisibilityPolicy, error) {
	switch strings.ToLower(value) {
	case "inherit", "none":
		return VisibilityInherit, nil
	case "muted":
		return VisibilityMuted, nil
	case "unmuted":
		return VisibilityUnmuted, nil
	case "followed":
		return VisibilityFollowed, nil
	}

	names := make([]string, 0, len(TopicVisibilityPolicies))
	for _, policy := range TopicVisibilityPolicies {
		names = append(names, policy.String())
	}
	return VisibilityInherit, fmt.Errorf(
		"unknown visibility policy %q: it must be one of %s", value, strings.Join(names, ", "))
}

// EmojiType specifies the type of emoji
type EmojiType string

const (
	RealmEmoji      EmojiType = "realm_emoji"
	UnicodeEmoji    EmojiType = "unicode_emoji"
	ZulipExtraEmoji EmojiType = "zulip_extra_emoji"
)

// EmojiTypes lists the reaction types the server accepts.
var EmojiTypes = []EmojiType{UnicodeEmoji, RealmEmoji, ZulipExtraEmoji}

// ParseEmojiType turns what someone typed into a reaction type.
func ParseEmojiType(value string) (EmojiType, error) {
	emoji := EmojiType(strings.ToLower(value))
	if slices.Contains(EmojiTypes, emoji) {
		return emoji, nil
	}
	return "", fmt.Errorf("unknown reaction type %q: it must be one of %s",
		value, joinNames(EmojiTypes))
}

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

// SettableMessageFlags are the flags a client may add or remove. The others
// the server computes and reports, and it refuses to be told what they are.
var SettableMessageFlags = []MessageFlag{FlagRead, FlagStarred, FlagCollapsed}

// ParseMessageFlag turns what someone typed into a flag that can be set.
func ParseMessageFlag(value string) (MessageFlag, error) {
	flag := MessageFlag(strings.ToLower(value))
	if slices.Contains(SettableMessageFlags, flag) {
		return flag, nil
	}
	return "", fmt.Errorf("unknown message flag %q: it must be one of %s",
		value, joinNames(SettableMessageFlags))
}

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
	MessageRetentionDays       *int   `json:"message_retention_days"`
	HistoryPublicToSubscribers bool   `json:"history_public_to_subscribers"`
	FirstMessageID             *int   `json:"first_message_id"`
	TopicsPolicy               string `json:"topics_policy,omitempty"`
	FolderID                   *int   `json:"folder_id,omitempty"`
	// CanSendMessageGroup is who may post in the channel. Servers older than
	// feature level 333 do not send it; there the deprecated fields below are
	// the only answer available.
	CanSendMessageGroup *GroupSetting `json:"can_send_message_group,omitempty"`
	// StreamPostPolicy and IsAnnouncementOnly are deprecated. Since feature
	// level 333 the server computes them from CanSendMessageGroup as the
	// closest enclosing role, so they are an approximation of who may post and
	// not the setting itself. They cannot be sent back to the server.
	StreamPostPolicy   int  `json:"stream_post_policy"`
	IsAnnouncementOnly bool `json:"is_announcement_only"`
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

// GroupSetting is a value for one of Zulip's group-based permission settings,
// such as can_send_message_group. It is either the ID of a user group — a
// named one or a role:* system group — or an anonymous group, meaning the union
// of some users and some groups. See https://zulip.com/api/group-setting-values.
type GroupSetting struct {
	// GroupID names a single user group. When it is nil the value is the
	// anonymous group described by the other two fields.
	GroupID         *int
	DirectMembers   []int
	DirectSubgroups []int
}

// NamedGroup returns a group-setting value naming a single user group.
func NamedGroup(id int) *GroupSetting {
	return &GroupSetting{GroupID: &id}
}

// AnonymousGroup returns a group-setting value covering the given users and
// groups without creating a user group to hold them.
func AnonymousGroup(members, subgroups []int) *GroupSetting {
	return &GroupSetting{DirectMembers: members, DirectSubgroups: subgroups}
}

// anonymousGroup is the wire form of a group-setting value that is not a bare
// group ID. Both lists are always sent, since the server expects both.
type anonymousGroup struct {
	DirectMembers   []int `json:"direct_members"`
	DirectSubgroups []int `json:"direct_subgroups"`
}

func (g GroupSetting) MarshalJSON() ([]byte, error) {
	if g.GroupID != nil {
		return json.Marshal(*g.GroupID)
	}
	wire := anonymousGroup{DirectMembers: g.DirectMembers, DirectSubgroups: g.DirectSubgroups}
	if wire.DirectMembers == nil {
		wire.DirectMembers = []int{}
	}
	if wire.DirectSubgroups == nil {
		wire.DirectSubgroups = []int{}
	}
	return json.Marshal(wire)
}

func (g *GroupSetting) UnmarshalJSON(data []byte) error {
	var id int
	if err := json.Unmarshal(data, &id); err == nil {
		*g = GroupSetting{GroupID: &id}
		return nil
	}
	var wire anonymousGroup
	if err := json.Unmarshal(data, &wire); err != nil {
		return fmt.Errorf("group-setting value is neither a group ID nor an anonymous group: %s", data)
	}
	*g = GroupSetting{DirectMembers: wire.DirectMembers, DirectSubgroups: wire.DirectSubgroups}
	return nil
}

// joinNames lists the accepted spellings of a string-like type, for the error
// a parse function returns when it is handed something else.
func joinNames[T ~string](values []T) string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		names = append(names, string(value))
	}
	return strings.Join(names, ", ")
}
