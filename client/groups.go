package client

import (
	"encoding/json"
	"fmt"

	"github.com/rybesh/zulip-cli/types"
)

// GetUserGroupsResponse represents user groups list response
type GetUserGroupsResponse struct {
	types.Response
	UserGroups []types.UserGroup `json:"user_groups"`
}

// GetUserGroups retrieves all user groups
func (c *Client) GetUserGroups() (*GetUserGroupsResponse, error) {
	body, err := c.Get("user_groups", nil)
	if err != nil {
		return nil, err
	}

	var resp GetUserGroupsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// CreateUserGroupRequest represents a user group creation request
type CreateUserGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Members     []int  `json:"members"`
}

// CreateUserGroupResponse represents user group creation response
type CreateUserGroupResponse struct {
	types.Response
	ID int `json:"id,omitempty"`
}

// CreateUserGroup creates a new user group
func (c *Client) CreateUserGroup(req CreateUserGroupRequest) (*CreateUserGroupResponse, error) {
	params := map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
		"members":     req.Members,
	}

	body, err := c.Post("user_groups/create", params)
	if err != nil {
		return nil, err
	}

	var resp CreateUserGroupResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateUserGroupRequest represents a user group update request
type UpdateUserGroupRequest struct {
	GroupID     int     `json:"group_id"`
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// UpdateUserGroup updates a user group
func (c *Client) UpdateUserGroup(req UpdateUserGroupRequest) (*types.Response, error) {
	params := map[string]interface{}{}
	if req.Name != nil {
		params["name"] = *req.Name
	}
	if req.Description != nil {
		params["description"] = *req.Description
	}

	body, err := c.Patch(fmt.Sprintf("user_groups/%d", req.GroupID), params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UserGroupDeactivateFeatureLevel is the first server feature level with
// POST /user_groups/{id}/deactivate. Older servers delete a group outright,
// and servers this new answer that DELETE with "Method Not Allowed".
const UserGroupDeactivateFeatureLevel = 290

// DeleteUserGroup deletes a user group. Servers from feature level 290
// deactivate it through the dedicated endpoint; older ones delete it.
func (c *Client) DeleteUserGroup(groupID int) (*types.Response, error) {
	level, err := c.FeatureLevel()
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("user_groups/%d", groupID)
	var body []byte
	if level < UserGroupDeactivateFeatureLevel {
		body, err = c.Delete(endpoint, nil)
	} else {
		body, err = c.Post(endpoint+"/deactivate", nil)
	}
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateUserGroupMembersRequest represents a group members update request
type UpdateUserGroupMembersRequest struct {
	GroupID int   `json:"group_id"`
	Add     []int `json:"add,omitempty"`
	Delete  []int `json:"delete,omitempty"`
}

// UpdateUserGroupMembers adds or removes members from a user group
func (c *Client) UpdateUserGroupMembers(req UpdateUserGroupMembersRequest) (*types.Response, error) {
	params := map[string]interface{}{}
	if len(req.Add) > 0 {
		params["add"] = req.Add
	}
	if len(req.Delete) > 0 {
		params["delete"] = req.Delete
	}

	body, err := c.Post(fmt.Sprintf("user_groups/%d/members", req.GroupID), params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
