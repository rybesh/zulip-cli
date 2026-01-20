package client

import (
	"encoding/json"

	"github.com/intelligrit/zulip-cli/types"
)

// GetStorageRequest represents a storage get request
type GetStorageRequest struct {
	Keys []string `json:"keys,omitempty"`
}

// GetStorageResponse represents storage get response
type GetStorageResponse struct {
	types.Response
	Storage map[string]string `json:"storage"`
}

// GetStorage retrieves bot storage data
func (c *Client) GetStorage(keys []string) (*GetStorageResponse, error) {
	params := map[string]interface{}{}
	if len(keys) > 0 {
		params["keys"] = keys
	}

	body, err := c.Get("bot_storage", params)
	if err != nil {
		return nil, err
	}

	var resp GetStorageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateStorageRequest represents a storage update request
type UpdateStorageRequest struct {
	Storage map[string]string `json:"storage"`
}

// UpdateStorage updates bot storage data
func (c *Client) UpdateStorage(storage map[string]string) (*types.Response, error) {
	params := map[string]interface{}{
		"storage": storage,
	}

	body, err := c.Post("bot_storage", params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
