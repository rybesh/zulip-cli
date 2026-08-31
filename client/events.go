package client

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/rybesh/zulip-cli/types"
)

// RegisterRequest represents an event queue registration request
type RegisterRequest struct {
	EventTypes         []string               `json:"event_types,omitempty"`
	Narrow             [][]string             `json:"narrow,omitempty"`
	AllPublicStreams   bool                   `json:"all_public_streams,omitempty"`
	IncludeSubscribers bool                   `json:"include_subscribers,omitempty"`
	ClientGravatar     bool                   `json:"client_gravatar,omitempty"`
	SlimPresence       bool                   `json:"slim_presence,omitempty"`
	ApplyMarkdown      bool                   `json:"apply_markdown,omitempty"`
	ClientCapabilities map[string]interface{} `json:"client_capabilities,omitempty"`
}

// RegisterResponse represents event queue registration response
type RegisterResponse struct {
	types.Response
	QueueID           string `json:"queue_id"`
	LastEventID       int    `json:"last_event_id"`
	ZulipVersion      string `json:"zulip_version"`
	ZulipFeatureLevel int    `json:"zulip_feature_level"`
	ZulipMergeBase    string `json:"zulip_merge_base,omitempty"`
	MaxMessageID      int    `json:"max_message_id"`
}

// Register registers an event queue
func (c *Client) Register(req RegisterRequest) (*RegisterResponse, error) {
	params := map[string]interface{}{}
	if len(req.EventTypes) > 0 {
		params["event_types"] = req.EventTypes
	}
	if len(req.Narrow) > 0 {
		params["narrow"] = req.Narrow
	}
	if req.AllPublicStreams {
		params["all_public_streams"] = true
	}
	if req.IncludeSubscribers {
		params["include_subscribers"] = true
	}
	if req.ClientGravatar {
		params["client_gravatar"] = true
	}
	if req.SlimPresence {
		params["slim_presence"] = true
	}
	if req.ApplyMarkdown {
		params["apply_markdown"] = true
	}
	if len(req.ClientCapabilities) > 0 {
		params["client_capabilities"] = req.ClientCapabilities
	}

	body, err := c.Post("register", params)
	if err != nil {
		return nil, err
	}

	var resp RegisterResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetEventsRequest represents an events fetch request
type GetEventsRequest struct {
	QueueID     string `json:"queue_id"`
	LastEventID int    `json:"last_event_id"`
	DontBlock   bool   `json:"dont_block,omitempty"`
}

// GetEventsResponse represents events response
type GetEventsResponse struct {
	types.Response
	Events []map[string]interface{} `json:"events"`
}

// GetEvents retrieves events from a queue
func (c *Client) GetEvents(req GetEventsRequest) (*GetEventsResponse, error) {
	params := map[string]interface{}{
		"queue_id":      req.QueueID,
		"last_event_id": req.LastEventID,
	}
	if req.DontBlock {
		params["dont_block"] = true
	}

	// Use a longer timeout for long-polling
	originalTimeout := c.HTTPClient.Timeout
	if !req.DontBlock {
		c.HTTPClient.Timeout = 90 * time.Second
	}
	defer func() {
		c.HTTPClient.Timeout = originalTimeout
	}()

	body, err := c.Get("events", params)
	if err != nil {
		return nil, err
	}

	var resp GetEventsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Deregister removes an event queue
func (c *Client) Deregister(queueID string) (*types.Response, error) {
	params := map[string]interface{}{
		"queue_id": queueID,
	}

	body, err := c.Delete("events", params)
	if err != nil {
		return nil, err
	}

	var resp types.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// EventCallback is a function type for handling events
type EventCallback func(event map[string]interface{})

// CallOnEachEvent registers an event queue and calls the callback for each event
func (c *Client) CallOnEachEvent(callback EventCallback, eventTypes []string, narrow [][]string) error {
	// Register event queue
	registerReq := RegisterRequest{
		EventTypes: eventTypes,
		Narrow:     narrow,
	}

	registration, err := c.Register(registerReq)
	if err != nil {
		return err
	}

	queueID := registration.QueueID
	lastEventID := registration.LastEventID

	// Continuously fetch and process events
	for {
		eventsReq := GetEventsRequest{
			QueueID:     queueID,
			LastEventID: lastEventID,
		}

		events, err := c.GetEvents(eventsReq)
		if err != nil {
			// If we get a bad event queue error, re-register
			// Check if error message contains BAD_EVENT_QUEUE_ID
			if strings.Contains(err.Error(), "BAD_EVENT_QUEUE_ID") {
				registration, err = c.Register(registerReq)
				if err != nil {
					return err
				}
				queueID = registration.QueueID
				lastEventID = registration.LastEventID
				continue
			}
			return err
		}

		for _, event := range events.Events {
			if eventType, ok := event["type"].(string); ok {
				// Skip heartbeat events
				if eventType == "heartbeat" {
					continue
				}
			}

			if eventID, ok := event["id"].(float64); ok {
				if int(eventID) > lastEventID {
					lastEventID = int(eventID)
				}
			}

			callback(event)
		}
	}
}

// MessageCallback is a function type for handling messages
type MessageCallback func(message types.Message)

// CallOnEachMessage registers an event queue and calls the callback for each message
func (c *Client) CallOnEachMessage(callback MessageCallback) error {
	return c.CallOnEachEvent(func(event map[string]interface{}) {
		if eventType, ok := event["type"].(string); ok && eventType == "message" {
			if msgData, ok := event["message"]; ok {
				msgJSON, _ := json.Marshal(msgData)
				var msg types.Message
				if err := json.Unmarshal(msgJSON, &msg); err == nil {
					callback(msg)
				}
			}
		}
	}, []string{"message"}, nil)
}
