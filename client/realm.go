package client

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/rybesh/zulip-cli/types"
)

// GetRealmEmojiResponse represents realm emoji response
type GetRealmEmojiResponse struct {
	types.Response
	Emoji map[string]types.Emoji `json:"emoji"`
}

// GetRealmEmoji retrieves all custom emoji in the realm
func (c *Client) GetRealmEmoji() (*GetRealmEmojiResponse, error) {
	body, err := c.Get("realm/emoji", nil)
	if err != nil {
		return nil, err
	}

	var resp GetRealmEmojiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UploadCustomEmoji uploads a custom emoji
func (c *Client) UploadCustomEmoji(emojiName string, file io.Reader) (*types.Response, error) {
	files := map[string]io.Reader{
		emojiName: file,
	}

	body, err := c.PostWithFiles(fmt.Sprintf("realm/emoji/%s", emojiName), nil, files)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeleteCustomEmoji deletes a custom emoji
func (c *Client) DeleteCustomEmoji(emojiName string) (*types.Response, error) {
	body, err := c.Delete(fmt.Sprintf("realm/emoji/%s", emojiName), nil)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetRealmLinkifiersResponse represents linkifiers response
type GetRealmLinkifiersResponse struct {
	types.Response
	Linkifiers []types.Linkifier `json:"linkifiers"`
}

// GetRealmLinkifiers retrieves all linkifiers
func (c *Client) GetRealmLinkifiers() (*GetRealmLinkifiersResponse, error) {
	body, err := c.Get("realm/linkifiers", nil)
	if err != nil {
		return nil, err
	}

	var resp GetRealmLinkifiersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// AddRealmFilterRequest represents a filter addition request
type AddRealmFilterRequest struct {
	Pattern     string `json:"pattern"`
	URLTemplate string `json:"url_template"`
}

// AddRealmFilterResponse represents filter addition response
type AddRealmFilterResponse struct {
	types.Response
	ID int `json:"id"`
}

// AddRealmFilter adds a linkifier/filter
func (c *Client) AddRealmFilter(req AddRealmFilterRequest) (*AddRealmFilterResponse, error) {
	params := map[string]interface{}{
		"pattern":      req.Pattern,
		"url_template": req.URLTemplate,
	}

	body, err := c.Post("realm/filters", params)
	if err != nil {
		return nil, err
	}

	var resp AddRealmFilterResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// RemoveRealmFilter removes a linkifier/filter
func (c *Client) RemoveRealmFilter(filterID int) (*types.Response, error) {
	body, err := c.Delete(fmt.Sprintf("realm/filters/%d", filterID), nil)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetRealmProfileFieldsResponse represents profile fields response
type GetRealmProfileFieldsResponse struct {
	types.Response
	CustomFields []types.ProfileField `json:"custom_fields"`
}

// GetRealmProfileFields retrieves all custom profile fields
func (c *Client) GetRealmProfileFields() (*GetRealmProfileFieldsResponse, error) {
	body, err := c.Get("realm/profile_fields", nil)
	if err != nil {
		return nil, err
	}

	var resp GetRealmProfileFieldsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// CreateRealmProfileFieldRequest represents profile field creation
type CreateRealmProfileFieldRequest struct {
	Name      string `json:"name"`
	Hint      string `json:"hint,omitempty"`
	FieldType int    `json:"field_type"`
	FieldData string `json:"field_data,omitempty"`
}

// CreateRealmProfileFieldResponse represents profile field creation response
type CreateRealmProfileFieldResponse struct {
	types.Response
	ID int `json:"id"`
}

// CreateRealmProfileField creates a custom profile field
func (c *Client) CreateRealmProfileField(req CreateRealmProfileFieldRequest) (*CreateRealmProfileFieldResponse, error) {
	params := map[string]interface{}{
		"name":       req.Name,
		"field_type": req.FieldType,
	}
	if req.Hint != "" {
		params["hint"] = req.Hint
	}
	if req.FieldData != "" {
		params["field_data"] = req.FieldData
	}

	body, err := c.Post("realm/profile_fields", params)
	if err != nil {
		return nil, err
	}

	var resp CreateRealmProfileFieldResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateRealmProfileFieldRequest represents profile field update
type UpdateRealmProfileFieldRequest struct {
	FieldID   int    `json:"field_id"`
	Name      string `json:"name,omitempty"`
	Hint      string `json:"hint,omitempty"`
	FieldData string `json:"field_data,omitempty"`
}

// UpdateRealmProfileField updates a custom profile field
func (c *Client) UpdateRealmProfileField(req UpdateRealmProfileFieldRequest) (*types.Response, error) {
	params := map[string]interface{}{}
	if req.Name != "" {
		params["name"] = req.Name
	}
	if req.Hint != "" {
		params["hint"] = req.Hint
	}
	if req.FieldData != "" {
		params["field_data"] = req.FieldData
	}

	body, err := c.Patch(fmt.Sprintf("realm/profile_fields/%d", req.FieldID), params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeleteRealmProfileField deletes a custom profile field
func (c *Client) DeleteRealmProfileField(fieldID int) (*types.Response, error) {
	body, err := c.Delete(fmt.Sprintf("realm/profile_fields/%d", fieldID), nil)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// ReorderRealmProfileFieldsRequest represents field reordering
type ReorderRealmProfileFieldsRequest struct {
	Order []int `json:"order"`
}

// ReorderRealmProfileFields reorders custom profile fields
func (c *Client) ReorderRealmProfileFields(order []int) (*types.Response, error) {
	params := map[string]interface{}{
		"order": order,
	}

	body, err := c.Patch("realm/profile_fields", params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
