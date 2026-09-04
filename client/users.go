package client

import (
	"encoding/json"
	"fmt"

	"github.com/rybesh/zulip-cli/types"
)

// GetUsersRequest represents a users list request
type GetUsersRequest struct {
	ClientGravatar             *bool `json:"client_gravatar,omitempty"`
	IncludeCustomProfileFields *bool `json:"include_custom_profile_fields,omitempty"`
}

// GetUsersResponse represents users list response
type GetUsersResponse struct {
	types.Response
	Members []types.User `json:"members"`
}

// GetUsers retrieves all users in the organization
func (c *Client) GetUsers(req GetUsersRequest) (*GetUsersResponse, error) {
	params := map[string]interface{}{}
	if req.ClientGravatar != nil {
		params["client_gravatar"] = *req.ClientGravatar
	}
	if req.IncludeCustomProfileFields != nil {
		params["include_custom_profile_fields"] = *req.IncludeCustomProfileFields
	}

	body, err := c.Get("users", params)
	if err != nil {
		return nil, err
	}

	var resp GetUsersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetUserResponse represents a single user response
type GetUserResponse struct {
	types.Response
	User types.User `json:"user"`
}

// GetUser retrieves a user by ID
func (c *Client) GetUser(userID int, includeCustomProfileFields *bool) (*GetUserResponse, error) {
	params := map[string]interface{}{}
	if includeCustomProfileFields != nil {
		params["include_custom_profile_fields"] = *includeCustomProfileFields
	}

	body, err := c.Get(fmt.Sprintf("users/%d", userID), params)
	if err != nil {
		return nil, err
	}

	var resp GetUserResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetProfileResponse represents the current user's profile
type GetProfileResponse struct {
	types.Response
	UserID         int                    `json:"user_id"`
	Email          string                 `json:"email"`
	FullName       string                 `json:"full_name"`
	IsAdmin        bool                   `json:"is_admin"`
	IsOwner        bool                   `json:"is_owner"`
	IsGuest        bool                   `json:"is_guest"`
	Role           int                    `json:"role,omitempty"`
	IsBillingAdmin bool                   `json:"is_billing_admin"`
	IsBot          bool                   `json:"is_bot"`
	AvatarURL      string                 `json:"avatar_url"`
	Timezone       string                 `json:"timezone"`
	ProfileData    map[string]interface{} `json:"profile_data,omitempty"`
}

// GetProfile retrieves the current user's profile
func (c *Client) GetProfile() (*GetProfileResponse, error) {
	body, err := c.Get("users/me", nil)
	if err != nil {
		return nil, err
	}

	var resp GetProfileResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// CreateUserRequest represents a user creation request
type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	FullName string `json:"full_name"`
}

// CreateUserResponse represents user creation response
type CreateUserResponse struct {
	types.Response
	UserID int `json:"user_id"`
}

// CreateUser creates a new user
func (c *Client) CreateUser(req CreateUserRequest) (*CreateUserResponse, error) {
	params := map[string]interface{}{
		"email":     req.Email,
		"full_name": req.FullName,
	}
	if req.Password != "" {
		params["password"] = req.Password
	}

	body, err := c.Post("users", params)
	if err != nil {
		return nil, err
	}

	var resp CreateUserResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateUserRequest represents a user update request
type UpdateUserRequest struct {
	UserID   int     `json:"user_id"`
	FullName *string `json:"full_name,omitempty"`
	Role     int     `json:"role,omitempty"`
	// ProfileData is a list of {"id", "value"} pairs naming the custom profile
	// fields to set. It is a list and not a map because that is what the server
	// reads: a map keyed by field ID is rejected.
	ProfileData []map[string]interface{} `json:"profile_data,omitempty"`
}

// UpdateUser updates a user
func (c *Client) UpdateUser(req UpdateUserRequest) (*types.Response, error) {
	params := map[string]interface{}{}
	if req.FullName != nil {
		params["full_name"] = *req.FullName
	}
	if req.Role > 0 {
		params["role"] = req.Role
	}
	if len(req.ProfileData) > 0 {
		params["profile_data"] = req.ProfileData
	}

	body, err := c.Patch(fmt.Sprintf("users/%d", req.UserID), params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeactivateUser deactivates a user
func (c *Client) DeactivateUser(userID int) (*types.Response, error) {
	body, err := c.Delete(fmt.Sprintf("users/%d", userID), nil)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// ReactivateUser reactivates a user
func (c *Client) ReactivateUser(userID int) (*types.Response, error) {
	body, err := c.Post(fmt.Sprintf("users/%d/reactivate", userID), nil)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetUserPresenceResponse represents user presence response
type GetUserPresenceResponse struct {
	types.Response
	Presence types.UserPresence `json:"presence"`
}

// GetUserPresence retrieves a user's presence
func (c *Client) GetUserPresence(userIDOrEmail interface{}) (*GetUserPresenceResponse, error) {
	var endpoint string
	switch v := userIDOrEmail.(type) {
	case int:
		endpoint = fmt.Sprintf("users/%d/presence", v)
	case string:
		endpoint = fmt.Sprintf("users/%s/presence", v)
	default:
		return nil, fmt.Errorf("userIDOrEmail must be int or string")
	}

	body, err := c.Get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp GetUserPresenceResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetRealmPresenceResponse represents realm-wide presence response
type GetRealmPresenceResponse struct {
	types.Response
	Presences map[string]map[string]types.Presence `json:"presences"`
}

// GetRealmPresence retrieves presence for all users
func (c *Client) GetRealmPresence() (*GetRealmPresenceResponse, error) {
	body, err := c.Get("realm/presence", nil)
	if err != nil {
		return nil, err
	}

	var resp GetRealmPresenceResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdatePresenceRequest represents a presence update request
type UpdatePresenceRequest struct {
	Status       string `json:"status"` // "active" or "idle"
	PingOnly     *bool  `json:"ping_only,omitempty"`
	NewUserInput *bool  `json:"new_user_input,omitempty"`
}

// UpdatePresenceResponse represents presence update response
type UpdatePresenceResponse struct {
	types.Response
	ServerTimestamp float64                              `json:"server_timestamp"`
	Presences       map[string]map[string]types.Presence `json:"presences"`
}

// UpdatePresence updates the current user's presence
func (c *Client) UpdatePresence(req UpdatePresenceRequest) (*UpdatePresenceResponse, error) {
	params := map[string]interface{}{
		"status": req.Status,
	}
	if req.PingOnly != nil {
		params["ping_only"] = *req.PingOnly
	}
	if req.NewUserInput != nil {
		params["new_user_input"] = *req.NewUserInput
	}

	body, err := c.Post("users/me/presence", params)
	if err != nil {
		return nil, err
	}

	var resp UpdatePresenceResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetAlertWordsResponse represents alert words response
type GetAlertWordsResponse struct {
	types.Response
	AlertWords []string `json:"alert_words"`
}

// GetAlertWords retrieves the user's alert words
func (c *Client) GetAlertWords() (*GetAlertWordsResponse, error) {
	body, err := c.Get("users/me/alert_words", nil)
	if err != nil {
		return nil, err
	}

	var resp GetAlertWordsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// AddAlertWordsResponse represents alert words addition response
type AddAlertWordsResponse struct {
	types.Response
	AlertWords []string `json:"alert_words"`
}

// AddAlertWords adds alert words
func (c *Client) AddAlertWords(words []string) (*AddAlertWordsResponse, error) {
	params := map[string]interface{}{
		"alert_words": words,
	}

	body, err := c.Post("users/me/alert_words", params)
	if err != nil {
		return nil, err
	}

	var resp AddAlertWordsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// RemoveAlertWordsResponse represents alert words removal response
type RemoveAlertWordsResponse struct {
	types.Response
	AlertWords []string `json:"alert_words"`
}

// RemoveAlertWords removes alert words
func (c *Client) RemoveAlertWords(words []string) (*RemoveAlertWordsResponse, error) {
	params := map[string]interface{}{
		"alert_words": words,
	}

	body, err := c.Delete("users/me/alert_words", params)
	if err != nil {
		return nil, err
	}

	var resp RemoveAlertWordsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// DirectMessageTypeFeatureLevel is the first server feature level whose typing
// endpoint calls a direct message "direct". Older servers know it only as
// "private", and servers this new no longer accept that spelling.
const DirectMessageTypeFeatureLevel = 174

// SetTypingStatusRequest represents a typing status request
type SetTypingStatusRequest struct {
	Op       string `json:"op"`                  // "start" or "stop"
	To       []int  `json:"to,omitempty"`        // User IDs for direct messages
	Type     string `json:"type,omitempty"`      // "stream" or "direct"
	Topic    string `json:"topic,omitempty"`     // For stream messages
	StreamID int    `json:"stream_id,omitempty"` // For stream messages
}

// typingConversationType spells a conversation type the way this server wants
// it. "private" and "direct" name the same kind of conversation, but a server
// accepts only one of the two, so whichever the caller used is translated.
// Every other type, such as "stream", passes through untouched.
func (c *Client) typingConversationType(name string) (string, error) {
	if name != "private" && name != "direct" {
		return name, nil
	}

	level, err := c.FeatureLevel()
	if err != nil {
		return "", err
	}
	if level < DirectMessageTypeFeatureLevel {
		return "private", nil
	}
	return "direct", nil
}

// SetTypingStatus sets typing status
func (c *Client) SetTypingStatus(req SetTypingStatusRequest) (*types.Response, error) {
	params := map[string]interface{}{
		"op": req.Op,
	}
	if len(req.To) > 0 {
		params["to"] = req.To
	}
	if req.Type != "" {
		conversation, err := c.typingConversationType(req.Type)
		if err != nil {
			return nil, err
		}
		params["type"] = conversation
	}
	if req.Topic != "" {
		params["topic"] = req.Topic
	}
	if req.StreamID > 0 {
		params["stream_id"] = req.StreamID
	}

	body, err := c.Post("typing", params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateNotificationSettingsRequest represents notification settings update
type UpdateNotificationSettingsRequest struct {
	EnableStreamPushNotifications    *bool `json:"enable_stream_push_notifications,omitempty"`
	EnableStreamEmailNotifications   *bool `json:"enable_stream_email_notifications,omitempty"`
	EnableStreamDesktopNotifications *bool `json:"enable_stream_desktop_notifications,omitempty"`
	EnableStreamAudibleNotifications *bool `json:"enable_stream_audible_notifications,omitempty"`
	EnableOfflinePushNotifications   *bool `json:"enable_offline_push_notifications,omitempty"`
	EnableOfflineEmailNotifications  *bool `json:"enable_offline_email_notifications,omitempty"`
	EnableOnlinePushNotifications    *bool `json:"enable_online_push_notifications,omitempty"`
	EnableDigestEmails               *bool `json:"enable_digest_emails,omitempty"`
}

// UpdateNotificationSettings updates notification settings
func (c *Client) UpdateNotificationSettings(req UpdateNotificationSettingsRequest) (*types.Response, error) {
	params := map[string]interface{}{}
	if req.EnableStreamPushNotifications != nil {
		params["enable_stream_push_notifications"] = *req.EnableStreamPushNotifications
	}
	if req.EnableStreamEmailNotifications != nil {
		params["enable_stream_email_notifications"] = *req.EnableStreamEmailNotifications
	}
	if req.EnableStreamDesktopNotifications != nil {
		params["enable_stream_desktop_notifications"] = *req.EnableStreamDesktopNotifications
	}
	if req.EnableStreamAudibleNotifications != nil {
		params["enable_stream_audible_notifications"] = *req.EnableStreamAudibleNotifications
	}
	if req.EnableOfflinePushNotifications != nil {
		params["enable_offline_push_notifications"] = *req.EnableOfflinePushNotifications
	}
	if req.EnableOfflineEmailNotifications != nil {
		params["enable_offline_email_notifications"] = *req.EnableOfflineEmailNotifications
	}
	if req.EnableOnlinePushNotifications != nil {
		params["enable_online_push_notifications"] = *req.EnableOnlinePushNotifications
	}
	if req.EnableDigestEmails != nil {
		params["enable_digest_emails"] = *req.EnableDigestEmails
	}

	body, err := c.Patch("settings/notifications", params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
