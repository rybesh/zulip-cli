package client

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/rybesh/zulip-cli/types"
)

// SendMessageRequest represents a message send request
type SendMessageRequest struct {
	Type    string      `json:"type"` // "stream" or "private"
	To      interface{} `json:"to"`   // string for stream, []string or []int for private
	Content string      `json:"content"`
	Topic   string      `json:"topic,omitempty"` // For stream messages (also accepts "subject")
	QueueID string      `json:"queue_id,omitempty"`
	LocalID string      `json:"local_id,omitempty"`
}

// SendMessageResponse represents the response from sending a message
type SendMessageResponse struct {
	types.Response
	ID int `json:"id"`
}

// SendMessage sends a message
func (c *Client) SendMessage(req SendMessageRequest) (*SendMessageResponse, error) {
	params := map[string]interface{}{
		"type":    req.Type,
		"content": req.Content,
	}

	// Handle "to" field
	switch req.Type {
	case "stream":
		if str, ok := req.To.(string); ok {
			params["to"] = str
		} else {
			return nil, fmt.Errorf("for stream messages, 'to' must be a string")
		}
		if req.Topic != "" {
			params["topic"] = req.Topic
		}
	case "private":
		params["to"] = req.To
	default:
		return nil, fmt.Errorf("invalid message type: %s", req.Type)
	}

	if req.QueueID != "" {
		params["queue_id"] = req.QueueID
	}
	if req.LocalID != "" {
		params["local_id"] = req.LocalID
	}

	body, err := c.Post("messages", params)
	if err != nil {
		return nil, err
	}

	var resp SendMessageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetMessagesRequest represents a request to fetch messages
type GetMessagesRequest struct {
	Anchor               interface{}    `json:"anchor"` // "newest", "oldest", "first_unread", or message ID
	NumBefore            int            `json:"num_before"`
	NumAfter             int            `json:"num_after"`
	Narrow               []types.Narrow `json:"narrow,omitempty"`
	ClientGravatar       *bool          `json:"client_gravatar,omitempty"`
	ApplyMarkdown        *bool          `json:"apply_markdown,omitempty"`
	UseFirstUnreadAnchor *bool          `json:"use_first_unread_anchor,omitempty"`
}

// GetMessagesResponse represents messages response
type GetMessagesResponse struct {
	types.Response
	Anchor         int             `json:"anchor"`
	FoundNewest    bool            `json:"found_newest"`
	FoundOldest    bool            `json:"found_oldest"`
	FoundAnchor    bool            `json:"found_anchor"`
	HistoryLimited bool            `json:"history_limited"`
	Messages       []types.Message `json:"messages"`
}

// GetMessages retrieves messages
func (c *Client) GetMessages(req GetMessagesRequest) (*GetMessagesResponse, error) {
	params := map[string]interface{}{
		"anchor":     req.Anchor,
		"num_before": req.NumBefore,
		"num_after":  req.NumAfter,
	}

	if len(req.Narrow) > 0 {
		params["narrow"] = req.Narrow
	}
	if req.ClientGravatar != nil {
		params["client_gravatar"] = *req.ClientGravatar
	}
	if req.ApplyMarkdown != nil {
		params["apply_markdown"] = *req.ApplyMarkdown
	}
	if req.UseFirstUnreadAnchor != nil {
		params["use_first_unread_anchor"] = *req.UseFirstUnreadAnchor
	}

	body, err := c.Get("messages", params)
	if err != nil {
		return nil, err
	}

	var resp GetMessagesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetRawMessageResponse represents raw message content
type GetRawMessageResponse struct {
	types.Response
	RawContent string `json:"raw_content"`
}

// GetRawMessage retrieves the raw Markdown content of a message
func (c *Client) GetRawMessage(messageID int) (*GetRawMessageResponse, error) {
	body, err := c.Get(fmt.Sprintf("messages/%d", messageID), nil)
	if err != nil {
		return nil, err
	}

	var resp GetRawMessageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateMessageRequest represents a message update request
type UpdateMessageRequest struct {
	MessageID                   int                     `json:"message_id"`
	Content                     *string                 `json:"content,omitempty"`
	Topic                       *string                 `json:"topic,omitempty"`
	PropagateMode               types.EditPropagateMode `json:"propagate_mode,omitempty"`
	SendNotificationToOldThread *bool                   `json:"send_notification_to_old_thread,omitempty"`
	SendNotificationToNewThread *bool                   `json:"send_notification_to_new_thread,omitempty"`
	StreamID                    int                     `json:"stream_id,omitempty"`
}

// UpdateMessage updates a message
func (c *Client) UpdateMessage(req UpdateMessageRequest) (*types.Response, error) {
	params := map[string]interface{}{}

	if req.Content != nil {
		params["content"] = *req.Content
	}
	if req.Topic != nil {
		params["topic"] = *req.Topic
	}
	if req.PropagateMode != "" {
		params["propagate_mode"] = req.PropagateMode
	}
	if req.SendNotificationToOldThread != nil {
		params["send_notification_to_old_thread"] = *req.SendNotificationToOldThread
	}
	if req.SendNotificationToNewThread != nil {
		params["send_notification_to_new_thread"] = *req.SendNotificationToNewThread
	}
	if req.StreamID > 0 {
		params["stream_id"] = req.StreamID
	}

	body, err := c.Patch(fmt.Sprintf("messages/%d", req.MessageID), params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeleteMessage permanently deletes a message
func (c *Client) DeleteMessage(messageID int) (*types.Response, error) {
	body, err := c.Delete(fmt.Sprintf("messages/%d", messageID), nil)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetMessageHistoryResponse represents message edit history
type GetMessageHistoryResponse struct {
	types.Response
	MessageHistory []struct {
		UserID              int    `json:"user_id"`
		Timestamp           int64  `json:"timestamp"`
		PrevContent         string `json:"prev_content,omitempty"`
		PrevRenderedContent string `json:"prev_rendered_content,omitempty"`
		Content             string `json:"content,omitempty"`
		RenderedContent     string `json:"rendered_content,omitempty"`
		ContentHTMLDiff     string `json:"content_html_diff,omitempty"`
		PrevTopic           string `json:"prev_topic,omitempty"`
		Topic               string `json:"topic,omitempty"`
		PrevStream          int    `json:"prev_stream,omitempty"`
		Stream              int    `json:"stream,omitempty"`
	} `json:"message_history"`
}

// GetMessageHistory retrieves the edit history of a message
func (c *Client) GetMessageHistory(messageID int) (*GetMessageHistoryResponse, error) {
	body, err := c.Get(fmt.Sprintf("messages/%d/history", messageID), nil)
	if err != nil {
		return nil, err
	}

	var resp GetMessageHistoryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateMessageFlagsRequest represents a message flags update request
type UpdateMessageFlagsRequest struct {
	Messages []int             `json:"messages"`
	Op       string            `json:"op"` // "add" or "remove"
	Flag     types.MessageFlag `json:"flag"`
}

// UpdateMessageFlags adds or removes flags on messages
func (c *Client) UpdateMessageFlags(req UpdateMessageFlagsRequest) (*types.Response, error) {
	params := map[string]interface{}{
		"messages": req.Messages,
		"op":       req.Op,
		"flag":     req.Flag,
	}

	body, err := c.Post("messages/flags", params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// MarkAllAsRead marks all messages as read
func (c *Client) MarkAllAsRead() (*types.Response, error) {
	body, err := c.Post("mark_all_as_read", nil)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// MarkStreamAsRead marks all messages in a stream as read
func (c *Client) MarkStreamAsRead(streamID int) (*types.Response, error) {
	params := map[string]interface{}{
		"stream_id": streamID,
	}

	body, err := c.Post("mark_stream_as_read", params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// MarkTopicAsRead marks all messages in a topic as read
func (c *Client) MarkTopicAsRead(streamID int, topicName string) (*types.Response, error) {
	params := map[string]interface{}{
		"stream_id":  streamID,
		"topic_name": topicName,
	}

	body, err := c.Post("mark_topic_as_read", params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// AddReactionRequest represents adding a reaction
type AddReactionRequest struct {
	MessageID    int             `json:"message_id"`
	EmojiName    string          `json:"emoji_name"`
	EmojiCode    string          `json:"emoji_code,omitempty"`
	ReactionType types.EmojiType `json:"reaction_type,omitempty"`
}

// AddReaction adds an emoji reaction to a message
func (c *Client) AddReaction(req AddReactionRequest) (*types.Response, error) {
	params := map[string]interface{}{
		"emoji_name": req.EmojiName,
	}
	if req.EmojiCode != "" {
		params["emoji_code"] = req.EmojiCode
	}
	if req.ReactionType != "" {
		params["reaction_type"] = req.ReactionType
	}

	body, err := c.Post(fmt.Sprintf("messages/%d/reactions", req.MessageID), params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// RemoveReaction removes an emoji reaction from a message
func (c *Client) RemoveReaction(req AddReactionRequest) (*types.Response, error) {
	params := map[string]interface{}{
		"emoji_name": req.EmojiName,
	}
	if req.EmojiCode != "" {
		params["emoji_code"] = req.EmojiCode
	}
	if req.ReactionType != "" {
		params["reaction_type"] = req.ReactionType
	}

	body, err := c.Delete(fmt.Sprintf("messages/%d/reactions", req.MessageID), params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// RenderMessageRequest represents a render request
type RenderMessageRequest struct {
	Content string `json:"content"`
}

// RenderMessageResponse represents rendered content
type RenderMessageResponse struct {
	types.Response
	Rendered string `json:"rendered"`
}

// RenderMessage renders Markdown to HTML
func (c *Client) RenderMessage(content string) (*RenderMessageResponse, error) {
	params := map[string]interface{}{
		"content": content,
	}

	body, err := c.Post("messages/render", params)
	if err != nil {
		return nil, err
	}

	var resp RenderMessageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UploadFileResponse represents file upload response
type UploadFileResponse struct {
	types.Response
	URI string `json:"uri"`
}

// UploadFile uploads a file. Only the base name of filename is sent: the local
// path the caller happened to have is not the name the server should store.
func (c *Client) UploadFile(file io.Reader, filename string) (*UploadFileResponse, error) {
	files := map[string]FormFile{
		"file": {Filename: filepath.Base(filename), Reader: file},
	}

	body, err := c.PostWithFiles("user_uploads", nil, files)
	if err != nil {
		return nil, err
	}

	var resp UploadFileResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetAttachmentsResponse represents attachments list
type GetAttachmentsResponse struct {
	types.Response
	Attachments []types.Attachment `json:"attachments"`
}

// GetAttachments retrieves all attachments
func (c *Client) GetAttachments() (*GetAttachmentsResponse, error) {
	body, err := c.Get("attachments", nil)
	if err != nil {
		return nil, err
	}

	var resp GetAttachmentsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// CheckMessagesMatchNarrowRequest represents a narrow match check
type CheckMessagesMatchNarrowRequest struct {
	MessageIDs []int          `json:"msg_ids"`
	Narrow     []types.Narrow `json:"narrow"`
}

// CheckMessagesMatchNarrowResponse represents narrow match results
type CheckMessagesMatchNarrowResponse struct {
	types.Response
	Messages map[string]interface{} `json:"messages"`
}

// CheckMessagesMatchNarrow checks if messages match a narrow
func (c *Client) CheckMessagesMatchNarrow(req CheckMessagesMatchNarrowRequest) (*CheckMessagesMatchNarrowResponse, error) {
	params := map[string]interface{}{
		"msg_ids": req.MessageIDs,
		"narrow":  req.Narrow,
	}

	body, err := c.Get("messages/matches_narrow", params)
	if err != nil {
		return nil, err
	}

	var resp CheckMessagesMatchNarrowResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
