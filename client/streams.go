package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/rybesh/zulip-cli/types"
)

// GetStreamsRequest represents a request to list streams.
//
// Optional flags throughout this package are pointers: a nil field is left out
// of the request so the server's default applies, while a non-nil false is sent
// as false rather than silently dropped.
type GetStreamsRequest struct {
	IncludePublic          *bool `json:"include_public,omitempty"`
	IncludeSubscribed      *bool `json:"include_subscribed,omitempty"`
	IncludeAllActive       *bool `json:"include_all_active,omitempty"`
	IncludeDefault         *bool `json:"include_default,omitempty"`
	IncludeOwnerSubscribed *bool `json:"include_owner_subscribed,omitempty"`
}

// GetStreamsResponse represents streams list response
type GetStreamsResponse struct {
	types.Response
	Streams []types.Stream `json:"streams"`
}

// GetStreams retrieves all streams
func (c *Client) GetStreams(req GetStreamsRequest) (*GetStreamsResponse, error) {
	params := map[string]interface{}{}
	if req.IncludePublic != nil {
		params["include_public"] = *req.IncludePublic
	}
	if req.IncludeSubscribed != nil {
		params["include_subscribed"] = *req.IncludeSubscribed
	}
	if req.IncludeAllActive != nil {
		params["include_all_active"] = *req.IncludeAllActive
	}
	if req.IncludeDefault != nil {
		params["include_default"] = *req.IncludeDefault
	}
	if req.IncludeOwnerSubscribed != nil {
		params["include_owner_subscribed"] = *req.IncludeOwnerSubscribed
	}

	body, err := c.Get("streams", params)
	if err != nil {
		return nil, err
	}

	var resp GetStreamsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetStreamIDResponse represents stream ID response
type GetStreamIDResponse struct {
	types.Response
	StreamID int `json:"stream_id"`
}

// GetStreamID gets the ID of a stream by name
func (c *Client) GetStreamID(streamName string) (*GetStreamIDResponse, error) {
	endpoint := fmt.Sprintf("get_stream_id?stream=%s", url.QueryEscape(streamName))
	body, err := c.Get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp GetStreamIDResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetStreamResponse represents a single channel response
type GetStreamResponse struct {
	types.Response
	Stream types.Stream `json:"stream"`
}

// GetStream gets one channel by ID, with everything the server knows about it.
// GetStreamID is the cheaper way to turn a name into an ID alone.
func (c *Client) GetStream(streamID int) (*GetStreamResponse, error) {
	body, err := c.Get(fmt.Sprintf("streams/%d", streamID), nil)
	if err != nil {
		return nil, err
	}

	var resp GetStreamResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// ChannelCreateFeatureLevel is the first server feature level with
// POST /channels/create, added in Zulip 11.0. Older servers create a channel as
// a side effect of subscribing to one that does not exist yet.
const ChannelCreateFeatureLevel = 417

// ChannelSettings is the initial configuration of a new channel. Both creation
// endpoints accept the same set, so the same values travel either way.
//
// The can_*_group fields are group-setting values; see types.GroupSetting.
// Several of them were added after the settings themselves, so an older server
// will ignore some — which it reports, and the client warns about.
type ChannelSettings struct {
	InviteOnly                        *bool               `json:"invite_only,omitempty"`
	IsWebPublic                       *bool               `json:"is_web_public,omitempty"`
	IsDefaultStream                   *bool               `json:"is_default_stream,omitempty"`
	HistoryPublicToSubscribers        *bool               `json:"history_public_to_subscribers,omitempty"`
	DefaultPushNotifications          *bool               `json:"default_push_notifications,omitempty"`
	MessageRetentionDays              interface{}         `json:"message_retention_days,omitempty"`
	FolderID                          *int                `json:"folder_id,omitempty"`
	TopicsPolicy                      *string             `json:"topics_policy,omitempty"`
	CanAddSubscribersGroup            *types.GroupSetting `json:"can_add_subscribers_group,omitempty"`
	CanAdministerChannelGroup         *types.GroupSetting `json:"can_administer_channel_group,omitempty"`
	CanCreateTopicGroup               *types.GroupSetting `json:"can_create_topic_group,omitempty"`
	CanDeleteAnyMessageGroup          *types.GroupSetting `json:"can_delete_any_message_group,omitempty"`
	CanDeleteOwnMessageGroup          *types.GroupSetting `json:"can_delete_own_message_group,omitempty"`
	CanMoveMessagesOutOfChannelGroup  *types.GroupSetting `json:"can_move_messages_out_of_channel_group,omitempty"`
	CanMoveMessagesWithinChannelGroup *types.GroupSetting `json:"can_move_messages_within_channel_group,omitempty"`
	CanRemoveSubscribersGroup         *types.GroupSetting `json:"can_remove_subscribers_group,omitempty"`
	CanResolveTopicsGroup             *types.GroupSetting `json:"can_resolve_topics_group,omitempty"`
	CanSendMessageGroup               *types.GroupSetting `json:"can_send_message_group,omitempty"`
	CanSubscribeGroup                 *types.GroupSetting `json:"can_subscribe_group,omitempty"`
}

// addTo adds the settings the caller asked for to a request's parameters.
func (s ChannelSettings) addTo(params map[string]interface{}) {
	if s.InviteOnly != nil {
		params["invite_only"] = *s.InviteOnly
	}
	if s.IsWebPublic != nil {
		params["is_web_public"] = *s.IsWebPublic
	}
	if s.IsDefaultStream != nil {
		params["is_default_stream"] = *s.IsDefaultStream
	}
	if s.HistoryPublicToSubscribers != nil {
		params["history_public_to_subscribers"] = *s.HistoryPublicToSubscribers
	}
	if s.DefaultPushNotifications != nil {
		params["default_push_notifications"] = *s.DefaultPushNotifications
	}
	if s.MessageRetentionDays != nil {
		params["message_retention_days"] = s.MessageRetentionDays
	}
	if s.FolderID != nil {
		params["folder_id"] = *s.FolderID
	}
	if s.TopicsPolicy != nil {
		params["topics_policy"] = *s.TopicsPolicy
	}
	for name, group := range map[string]*types.GroupSetting{
		"can_add_subscribers_group":              s.CanAddSubscribersGroup,
		"can_administer_channel_group":           s.CanAdministerChannelGroup,
		"can_create_topic_group":                 s.CanCreateTopicGroup,
		"can_delete_any_message_group":           s.CanDeleteAnyMessageGroup,
		"can_delete_own_message_group":           s.CanDeleteOwnMessageGroup,
		"can_move_messages_out_of_channel_group": s.CanMoveMessagesOutOfChannelGroup,
		"can_move_messages_within_channel_group": s.CanMoveMessagesWithinChannelGroup,
		"can_remove_subscribers_group":           s.CanRemoveSubscribersGroup,
		"can_resolve_topics_group":               s.CanResolveTopicsGroup,
		"can_send_message_group":                 s.CanSendMessageGroup,
		"can_subscribe_group":                    s.CanSubscribeGroup,
	} {
		if group != nil {
			params[name] = group
		}
	}
}

// RetentionDays renders a message retention setting for the wire. Zulip reads
// the parameter as JSON, so a number of days travels as a number while
// "realm_default" and "unlimited" have to arrive quoted.
func RetentionDays(value string) interface{} {
	if days, err := strconv.Atoi(value); err == nil {
		return days
	}
	return json.RawMessage(strconv.Quote(value))
}

// CreateChannelRequest represents a channel creation request
type CreateChannelRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	// Subscribers are the user IDs to subscribe to the new channel. Empty means
	// the caller alone, which is what creating a channel has always done.
	Subscribers []int `json:"subscribers,omitempty"`
	Announce    *bool `json:"announce,omitempty"`
	ChannelSettings
}

// CreateChannelResponse represents channel creation response. The dedicated
// endpoint returns the new channel's ID; the older path returns who ended up
// subscribed, so which fields are filled in depends on the server.
type CreateChannelResponse struct {
	SubscribeResponse
	ID int `json:"id,omitempty"`
}

// CreateChannel creates a channel and subscribes Subscribers to it, or the
// caller alone when that list is empty. Servers from feature level 417 get the
// dedicated endpoint; older ones fall back to creating the channel by
// subscribing to it.
func (c *Client) CreateChannel(req CreateChannelRequest) (*CreateChannelResponse, error) {
	level, err := c.FeatureLevel()
	if err != nil {
		return nil, err
	}
	if level < ChannelCreateFeatureLevel {
		return c.createChannelBySubscribing(req)
	}

	subscribers := req.Subscribers
	if len(subscribers) == 0 {
		profile, err := c.GetProfile()
		if err != nil {
			return nil, fmt.Errorf("failed to look up your own user ID to subscribe you to the new channel: %w", err)
		}
		subscribers = []int{profile.UserID}
	}

	params := map[string]interface{}{
		"name":        req.Name,
		"subscribers": subscribers,
	}
	if req.Description != nil {
		params["description"] = *req.Description
	}
	if req.Announce != nil {
		params["announce"] = *req.Announce
	}
	req.ChannelSettings.addTo(params)

	body, err := c.Post("channels/create", params)
	if err != nil {
		return nil, err
	}

	var resp CreateChannelResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// createChannelBySubscribing creates a channel the way every client had to
// before feature level 417: by subscribing to one that does not exist yet. The
// two endpoints take the same channel settings, so nothing is given up beyond
// learning the new channel's ID.
func (c *Client) createChannelBySubscribing(req CreateChannelRequest) (*CreateChannelResponse, error) {
	sub := SubscribeRequest{
		Announce:        req.Announce,
		ChannelSettings: req.ChannelSettings,
	}
	sub.Subscriptions = append(sub.Subscriptions, ChannelSubscription{Name: req.Name})
	if req.Description != nil {
		sub.Subscriptions[0].Description = *req.Description
	}
	for _, userID := range req.Subscribers {
		sub.Principals = append(sub.Principals, userID)
	}

	resp, err := c.Subscribe(sub)
	if err != nil {
		return nil, err
	}

	return &CreateChannelResponse{SubscribeResponse: *resp}, nil
}

// UpdateStreamRequest represents a stream update request
type UpdateStreamRequest struct {
	StreamID                   int         `json:"stream_id"`
	Description                *string     `json:"description,omitempty"`
	NewName                    *string     `json:"new_name,omitempty"`
	IsPrivate                  *bool       `json:"is_private,omitempty"`
	IsWebPublic                *bool       `json:"is_web_public,omitempty"`
	HistoryPublicToSubscribers *bool       `json:"history_public_to_subscribers,omitempty"`
	MessageRetentionDays       interface{} `json:"message_retention_days,omitempty"`
	// CanSendMessageGroup is who may post in the channel, replacing the
	// stream_post_policy the server dropped at feature level 333. It is sent as
	// a plain replacement: the request says what the value should become and
	// not what the caller believed it was, so a concurrent change is overwritten
	// rather than reported.
	CanSendMessageGroup *types.GroupSetting `json:"can_send_message_group,omitempty"`
}

// UpdateStream updates stream settings
func (c *Client) UpdateStream(req UpdateStreamRequest) (*types.Response, error) {
	params := map[string]interface{}{}
	if req.Description != nil {
		params["description"] = *req.Description
	}
	if req.NewName != nil {
		params["new_name"] = *req.NewName
	}
	if req.IsPrivate != nil {
		params["is_private"] = *req.IsPrivate
	}
	if req.IsWebPublic != nil {
		params["is_web_public"] = *req.IsWebPublic
	}
	if req.HistoryPublicToSubscribers != nil {
		params["history_public_to_subscribers"] = *req.HistoryPublicToSubscribers
	}
	if req.MessageRetentionDays != nil {
		params["message_retention_days"] = req.MessageRetentionDays
	}
	if req.CanSendMessageGroup != nil {
		params["can_send_message_group"] = map[string]interface{}{"new": req.CanSendMessageGroup}
	}

	body, err := c.Patch(fmt.Sprintf("streams/%d", req.StreamID), params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeleteStream deletes a stream
func (c *Client) DeleteStream(streamID int) (*types.Response, error) {
	body, err := c.Delete(fmt.Sprintf("streams/%d", streamID), nil)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetStreamTopicsResponse represents stream topics response
type GetStreamTopicsResponse struct {
	types.Response
	Topics []types.Topic `json:"topics"`
}

// GetStreamTopics retrieves all topics in a stream
func (c *Client) GetStreamTopics(streamID int) (*GetStreamTopicsResponse, error) {
	body, err := c.Get(fmt.Sprintf("users/me/%d/topics", streamID), nil)
	if err != nil {
		return nil, err
	}

	var resp GetStreamTopicsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetStreamEmailAddressResponse represents stream email response
type GetStreamEmailAddressResponse struct {
	types.Response
	Email string `json:"email"`
}

// GetStreamEmailAddress gets the email address for a stream
func (c *Client) GetStreamEmailAddress(streamID int) (*GetStreamEmailAddressResponse, error) {
	body, err := c.Get(fmt.Sprintf("streams/%d/email_address", streamID), nil)
	if err != nil {
		return nil, err
	}

	var resp GetStreamEmailAddressResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetSubscribersResponse represents subscribers list response
type GetSubscribersResponse struct {
	types.Response
	Subscribers []int `json:"subscribers"`
}

// GetSubscribers gets all subscribers to a stream
func (c *Client) GetSubscribers(streamID int) (*GetSubscribersResponse, error) {
	body, err := c.Get(fmt.Sprintf("streams/%d/members", streamID), nil)
	if err != nil {
		return nil, err
	}

	var resp GetSubscribersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetSubscriptionsRequest represents subscriptions list request
type GetSubscriptionsRequest struct {
	IncludeSubscribers *bool `json:"include_subscribers,omitempty"`
}

// GetSubscriptionsResponse represents user subscriptions response
type GetSubscriptionsResponse struct {
	types.Response
	Subscriptions []types.Subscription `json:"subscriptions"`
}

// GetSubscriptions gets all streams the user is subscribed to
func (c *Client) GetSubscriptions(req GetSubscriptionsRequest) (*GetSubscriptionsResponse, error) {
	params := map[string]interface{}{}
	if req.IncludeSubscribers != nil {
		params["include_subscribers"] = *req.IncludeSubscribers
	}

	body, err := c.Get("users/me/subscriptions", params)
	if err != nil {
		return nil, err
	}

	var resp GetSubscriptionsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// ChannelSubscription names a channel to subscribe to, creating it with the
// given description if it does not exist yet.
type ChannelSubscription struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// SubscribeRequest represents a subscription request. The channel settings
// apply only to channels the request creates.
type SubscribeRequest struct {
	Subscriptions            []ChannelSubscription `json:"subscriptions"`
	Principals               []interface{}         `json:"principals,omitempty"`
	AuthorizationErrorsFatal *bool                 `json:"authorization_errors_fatal,omitempty"`
	Announce                 *bool                 `json:"announce,omitempty"`
	ChannelSettings
}

// SubscribeResponse represents subscription response
type SubscribeResponse struct {
	types.Response
	Subscribed        map[string][]string `json:"subscribed,omitempty"`
	AlreadySubscribed map[string][]string `json:"already_subscribed,omitempty"`
	Unauthorized      []string            `json:"unauthorized,omitempty"`
}

// Subscribe subscribes users to channels, creating any that do not exist
func (c *Client) Subscribe(req SubscribeRequest) (*SubscribeResponse, error) {
	params := map[string]interface{}{
		"subscriptions": req.Subscriptions,
	}
	if len(req.Principals) > 0 {
		params["principals"] = req.Principals
	}
	if req.AuthorizationErrorsFatal != nil {
		params["authorization_errors_fatal"] = *req.AuthorizationErrorsFatal
	}
	if req.Announce != nil {
		params["announce"] = *req.Announce
	}
	req.ChannelSettings.addTo(params)

	body, err := c.Post("users/me/subscriptions", params)
	if err != nil {
		return nil, err
	}

	var resp SubscribeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UnsubscribeRequest represents an unsubscription request
type UnsubscribeRequest struct {
	Subscriptions []string      `json:"subscriptions"`        // Stream names
	Principals    []interface{} `json:"principals,omitempty"` // User emails or IDs
}

// UnsubscribeResponse represents unsubscription response
type UnsubscribeResponse struct {
	types.Response
	Removed    []string `json:"removed"`
	NotRemoved []string `json:"not_removed"`
}

// Unsubscribe unsubscribes users from streams
func (c *Client) Unsubscribe(req UnsubscribeRequest) (*UnsubscribeResponse, error) {
	params := map[string]interface{}{
		"subscriptions": req.Subscriptions,
	}
	if len(req.Principals) > 0 {
		params["principals"] = req.Principals
	}

	body, err := c.Delete("users/me/subscriptions", params)
	if err != nil {
		return nil, err
	}

	var resp UnsubscribeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetSubscriptionStatusResponse represents subscription status response
type GetSubscriptionStatusResponse struct {
	types.Response
	IsSubscribed bool `json:"is_subscribed"`
}

// GetSubscriptionStatus checks if a user is subscribed to a stream
func (c *Client) GetSubscriptionStatus(userID, streamID int) (*GetSubscriptionStatusResponse, error) {
	body, err := c.Get(fmt.Sprintf("users/%d/subscriptions/%d", userID, streamID), nil)
	if err != nil {
		return nil, err
	}

	var resp GetSubscriptionStatusResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateSubscriptionSettingsRequest represents subscription settings update
type UpdateSubscriptionSettingsRequest struct {
	SubscriptionData []map[string]interface{} `json:"subscription_data"`
}

// UpdateSubscriptionSettings updates subscription properties
func (c *Client) UpdateSubscriptionSettings(req UpdateSubscriptionSettingsRequest) (*types.Response, error) {
	params := map[string]interface{}{
		"subscription_data": req.SubscriptionData,
	}

	body, err := c.Post("users/me/subscriptions/properties", params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UserTopicFeatureLevel is the first server feature level with
// POST /user_topics, added in Zulip 7.0. It deprecates
// PATCH /users/me/subscriptions/muted_topics, which knows only whether a topic
// is muted, and which upstream says may be removed in a future release.
const UserTopicFeatureLevel = 170

// FollowedTopicFeatureLevel is the first server feature level that understands
// the followed visibility policy, added later in the same Zulip 7.0 series.
const FollowedTopicFeatureLevel = 219

// UpdateUserTopicRequest represents a change to the caller's own preferences
// for one topic.
type UpdateUserTopicRequest struct {
	StreamID         int                         `json:"stream_id"`
	Topic            string                      `json:"topic"`
	VisibilityPolicy types.TopicVisibilityPolicy `json:"visibility_policy"`
}

// UpdateUserTopic sets the caller's visibility policy for a topic. Servers
// from feature level 170 get POST /user_topics; older ones fall back to the
// endpoint it deprecates, which can only mute and unmute.
func (c *Client) UpdateUserTopic(req UpdateUserTopicRequest) (*types.Response, error) {
	level, err := c.FeatureLevel()
	if err != nil {
		return nil, err
	}
	if level < UserTopicFeatureLevel {
		return c.updateUserTopicByMuting(req)
	}
	if req.VisibilityPolicy == types.VisibilityFollowed && level < FollowedTopicFeatureLevel {
		return nil, fmt.Errorf(
			"this server, at feature level %d, cannot follow topics: that needs feature level %d",
			level, FollowedTopicFeatureLevel)
	}

	params := map[string]interface{}{
		"stream_id":         req.StreamID,
		"topic":             req.Topic,
		"visibility_policy": int(req.VisibilityPolicy),
	}

	body, err := c.Post("user_topics", params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// updateUserTopicByMuting sets a visibility policy the way every client had to
// before feature level 170. That endpoint is binary, so the policies that
// arrived with POST /user_topics have nothing here to fall back to.
func (c *Client) updateUserTopicByMuting(req UpdateUserTopicRequest) (*types.Response, error) {
	var op string
	switch req.VisibilityPolicy {
	case types.VisibilityMuted:
		op = "add"
	case types.VisibilityInherit:
		op = "remove"
	default:
		return nil, fmt.Errorf(
			"this server, below feature level %d, cannot set the %s visibility policy: it can only mute and unmute topics",
			UserTopicFeatureLevel, req.VisibilityPolicy)
	}

	params := map[string]interface{}{
		"stream_id": req.StreamID,
		"topic":     req.Topic,
		"op":        op,
	}

	body, err := c.Patch("users/me/subscriptions/muted_topics", params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// AddDefaultStream adds a stream to the default streams
func (c *Client) AddDefaultStream(streamID int) (*types.Response, error) {
	params := map[string]interface{}{
		"stream_id": streamID,
	}

	body, err := c.Post("default_streams", params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// MoveTopicRequest represents a topic move request
type MoveTopicRequest struct {
	StreamID                    int                     `json:"stream_id"`
	NewStreamID                 int                     `json:"new_stream_id,omitempty"`
	Topic                       string                  `json:"topic"`
	NewTopic                    *string                 `json:"new_topic,omitempty"`
	MessageID                   int                     `json:"message_id,omitempty"`
	PropagateMode               types.EditPropagateMode `json:"propagate_mode,omitempty"`
	SendNotificationToOldThread *bool                   `json:"send_notification_to_old_thread,omitempty"`
	SendNotificationToNewThread *bool                   `json:"send_notification_to_new_thread,omitempty"`
}

// MoveTopic moves a topic to another stream and/or renames it
func (c *Client) MoveTopic(req MoveTopicRequest) (*types.Response, error) {
	// If no message ID provided, get the latest message in the topic
	messageID := req.MessageID
	if messageID == 0 {
		getReq := GetMessagesRequest{
			Anchor:    "newest",
			NumBefore: 1,
			NumAfter:  0,
			Narrow: []types.Narrow{
				{Operator: "stream", Operand: req.StreamID},
				{Operator: "topic", Operand: req.Topic},
			},
		}
		msgs, err := c.GetMessages(getReq)
		if err != nil {
			return nil, err
		}
		if len(msgs.Messages) == 0 {
			return nil, fmt.Errorf("no messages found in topic")
		}
		messageID = msgs.Messages[0].ID
	}

	params := map[string]interface{}{}
	if req.NewStreamID > 0 {
		params["stream_id"] = req.NewStreamID
	}
	if req.NewTopic != nil {
		params["topic"] = *req.NewTopic
	}
	if req.PropagateMode != "" {
		params["propagate_mode"] = req.PropagateMode
	} else {
		params["propagate_mode"] = types.ChangeAll
	}
	if req.SendNotificationToOldThread != nil {
		params["send_notification_to_old_thread"] = *req.SendNotificationToOldThread
	}
	if req.SendNotificationToNewThread != nil {
		params["send_notification_to_new_thread"] = *req.SendNotificationToNewThread
	}

	body, err := c.Patch(fmt.Sprintf("messages/%d", messageID), params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
