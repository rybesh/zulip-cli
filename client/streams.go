package client

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/intelligrit/zulip-cli/types"
)

// GetStreamsRequest represents a request to list streams
type GetStreamsRequest struct {
	IncludePublic       bool `json:"include_public,omitempty"`
	IncludeSubscribed   bool `json:"include_subscribed,omitempty"`
	IncludeAllActive    bool `json:"include_all_active,omitempty"`
	IncludeDefault      bool `json:"include_default,omitempty"`
	IncludeOwnerSubscribed bool `json:"include_owner_subscribed,omitempty"`
}

// GetStreamsResponse represents streams list response
type GetStreamsResponse struct {
	types.Response
	Streams []types.Stream `json:"streams"`
}

// GetStreams retrieves all streams
func (c *Client) GetStreams(req GetStreamsRequest) (*GetStreamsResponse, error) {
	params := map[string]interface{}{}
	if req.IncludePublic {
		params["include_public"] = true
	}
	if req.IncludeSubscribed {
		params["include_subscribed"] = true
	}
	if req.IncludeAllActive {
		params["include_all_active"] = true
	}
	if req.IncludeDefault {
		params["include_default"] = true
	}
	if req.IncludeOwnerSubscribed {
		params["include_owner_subscribed"] = true
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

// CreateStreamRequest represents a stream creation request
type CreateStreamRequest struct {
	Subscriptions []struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	} `json:"subscriptions"`
	Principals             []interface{} `json:"principals,omitempty"` // Can be email strings or user IDs
	AuthorizationErrorsFatal bool       `json:"authorization_errors_fatal,omitempty"`
	Announce               bool          `json:"announce,omitempty"`
	InviteOnly             bool          `json:"invite_only,omitempty"`
	IsWebPublic            bool          `json:"is_web_public,omitempty"`
	HistoryPublicToSubscribers bool      `json:"history_public_to_subscribers,omitempty"`
	StreamPostPolicy       int           `json:"stream_post_policy,omitempty"`
	MessageRetentionDays   interface{}   `json:"message_retention_days,omitempty"`
}

// CreateStreamResponse represents stream creation response
type CreateStreamResponse struct {
	types.Response
	Subscribed map[string][]string `json:"subscribed,omitempty"`
	AlreadySubscribed map[string][]string `json:"already_subscribed,omitempty"`
	Unauthorized []string `json:"unauthorized,omitempty"`
}

// CreateStream creates one or more streams
func (c *Client) CreateStream(req CreateStreamRequest) (*CreateStreamResponse, error) {
	params := map[string]interface{}{
		"subscriptions": req.Subscriptions,
	}
	if len(req.Principals) > 0 {
		params["principals"] = req.Principals
	}
	if req.AuthorizationErrorsFatal {
		params["authorization_errors_fatal"] = true
	}
	if req.Announce {
		params["announce"] = true
	}
	if req.InviteOnly {
		params["invite_only"] = true
	}
	if req.IsWebPublic {
		params["is_web_public"] = true
	}
	if req.HistoryPublicToSubscribers {
		params["history_public_to_subscribers"] = true
	}
	if req.StreamPostPolicy > 0 {
		params["stream_post_policy"] = req.StreamPostPolicy
	}
	if req.MessageRetentionDays != nil {
		params["message_retention_days"] = req.MessageRetentionDays
	}

	body, err := c.Post("users/me/subscriptions", params)
	if err != nil {
		return nil, err
	}

	var resp CreateStreamResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateStreamRequest represents a stream update request
type UpdateStreamRequest struct {
	StreamID               int         `json:"stream_id"`
	Description            string      `json:"description,omitempty"`
	NewName                string      `json:"new_name,omitempty"`
	IsPrivate              *bool       `json:"is_private,omitempty"`
	IsWebPublic            *bool       `json:"is_web_public,omitempty"`
	HistoryPublicToSubscribers *bool   `json:"history_public_to_subscribers,omitempty"`
	StreamPostPolicy       int         `json:"stream_post_policy,omitempty"`
	MessageRetentionDays   interface{} `json:"message_retention_days,omitempty"`
}

// UpdateStream updates stream settings
func (c *Client) UpdateStream(req UpdateStreamRequest) (*types.Response, error) {
	params := map[string]interface{}{}
	if req.Description != "" {
		params["description"] = req.Description
	}
	if req.NewName != "" {
		params["new_name"] = req.NewName
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
	if req.StreamPostPolicy > 0 {
		params["stream_post_policy"] = req.StreamPostPolicy
	}
	if req.MessageRetentionDays != nil {
		params["message_retention_days"] = req.MessageRetentionDays
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
	IncludeSubscribers bool `json:"include_subscribers,omitempty"`
}

// GetSubscriptionsResponse represents user subscriptions response
type GetSubscriptionsResponse struct {
	types.Response
	Subscriptions []types.Subscription `json:"subscriptions"`
}

// GetSubscriptions gets all streams the user is subscribed to
func (c *Client) GetSubscriptions(req GetSubscriptionsRequest) (*GetSubscriptionsResponse, error) {
	params := map[string]interface{}{}
	if req.IncludeSubscribers {
		params["include_subscribers"] = true
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

// SubscribeRequest represents a subscription request
type SubscribeRequest struct {
	Subscriptions []struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	} `json:"subscriptions"`
	Principals             []interface{} `json:"principals,omitempty"`
	AuthorizationErrorsFatal bool       `json:"authorization_errors_fatal,omitempty"`
	Announce               bool          `json:"announce,omitempty"`
	InviteOnly             bool          `json:"invite_only,omitempty"`
	HistoryPublicToSubscribers bool      `json:"history_public_to_subscribers,omitempty"`
	StreamPostPolicy       int           `json:"stream_post_policy,omitempty"`
	MessageRetentionDays   interface{}   `json:"message_retention_days,omitempty"`
}

// Subscribe subscribes users to streams
func (c *Client) Subscribe(req SubscribeRequest) (*CreateStreamResponse, error) {
	params := map[string]interface{}{
		"subscriptions": req.Subscriptions,
	}
	if len(req.Principals) > 0 {
		params["principals"] = req.Principals
	}
	if req.AuthorizationErrorsFatal {
		params["authorization_errors_fatal"] = true
	}
	if req.Announce {
		params["announce"] = true
	}
	if req.InviteOnly {
		params["invite_only"] = true
	}
	if req.HistoryPublicToSubscribers {
		params["history_public_to_subscribers"] = true
	}
	if req.StreamPostPolicy > 0 {
		params["stream_post_policy"] = req.StreamPostPolicy
	}
	if req.MessageRetentionDays != nil {
		params["message_retention_days"] = req.MessageRetentionDays
	}

	body, err := c.Post("users/me/subscriptions", params)
	if err != nil {
		return nil, err
	}

	var resp CreateStreamResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UnsubscribeRequest represents an unsubscription request
type UnsubscribeRequest struct {
	Subscriptions []string      `json:"subscriptions"` // Stream names
	Principals    []interface{} `json:"principals,omitempty"` // User emails or IDs
}

// UnsubscribeResponse represents unsubscription response
type UnsubscribeResponse struct {
	types.Response
	Removed      []string `json:"removed"`
	NotRemoved   []string `json:"not_removed"`
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

// MuteTopicRequest represents a topic mute request
type MuteTopicRequest struct {
	Stream string `json:"stream"`
	Topic  string `json:"topic"`
	Op     string `json:"op"` // "add" or "remove"
}

// MuteTopic mutes or unmutes a topic
func (c *Client) MuteTopic(req MuteTopicRequest) (*types.Response, error) {
	params := map[string]interface{}{
		"stream": req.Stream,
		"topic":  req.Topic,
		"op":     req.Op,
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
	StreamID    int                      `json:"stream_id"`
	NewStreamID int                      `json:"new_stream_id,omitempty"`
	Topic       string                   `json:"topic"`
	NewTopic    string                   `json:"new_topic,omitempty"`
	MessageID   int                      `json:"message_id,omitempty"`
	PropagateMode types.EditPropagateMode `json:"propagate_mode,omitempty"`
	SendNotificationToOldThread bool     `json:"send_notification_to_old_thread,omitempty"`
	SendNotificationToNewThread bool     `json:"send_notification_to_new_thread,omitempty"`
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
	if req.NewTopic != "" {
		params["topic"] = req.NewTopic
	}
	if req.PropagateMode != "" {
		params["propagate_mode"] = req.PropagateMode
	} else {
		params["propagate_mode"] = types.ChangeAll
	}
	if req.SendNotificationToOldThread {
		params["send_notification_to_old_thread"] = true
	}
	if req.SendNotificationToNewThread {
		params["send_notification_to_new_thread"] = true
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
