package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/rybesh/zulip-cli/types"
)

const (
	// longPollTimeout bounds a blocking events fetch. The server holds the
	// connection open until it has something to say, so this is well above the
	// ordinary request timeout.
	longPollTimeout = 90 * time.Second

	// Backoff bounds for retrying an events fetch that failed for reasons the
	// caller cannot fix, such as a network blip or a server restart.
	initialEventBackoff = 1 * time.Second
	maxEventBackoff     = 30 * time.Second
)

// RegisterRequest represents an event queue registration request
type RegisterRequest struct {
	EventTypes         []string               `json:"event_types,omitempty"`
	Narrow             [][]string             `json:"narrow,omitempty"`
	AllPublicStreams   *bool                  `json:"all_public_streams,omitempty"`
	IncludeSubscribers *bool                  `json:"include_subscribers,omitempty"`
	ClientGravatar     *bool                  `json:"client_gravatar,omitempty"`
	SlimPresence       *bool                  `json:"slim_presence,omitempty"`
	ApplyMarkdown      *bool                  `json:"apply_markdown,omitempty"`
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
	if req.AllPublicStreams != nil {
		params["all_public_streams"] = *req.AllPublicStreams
	}
	if req.IncludeSubscribers != nil {
		params["include_subscribers"] = *req.IncludeSubscribers
	}
	if req.ClientGravatar != nil {
		params["client_gravatar"] = *req.ClientGravatar
	}
	if req.SlimPresence != nil {
		params["slim_presence"] = *req.SlimPresence
	}
	if req.ApplyMarkdown != nil {
		params["apply_markdown"] = *req.ApplyMarkdown
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
	return c.GetEventsContext(context.Background(), req)
}

// GetEventsContext retrieves events from a queue, stopping early if ctx is
// cancelled.
func (c *Client) GetEventsContext(ctx context.Context, req GetEventsRequest) (*GetEventsResponse, error) {
	params := map[string]interface{}{
		"queue_id":      req.QueueID,
		"last_event_id": req.LastEventID,
	}
	if req.DontBlock {
		params["dont_block"] = true
	}

	// A blocking fetch waits on the server, so allow it more time than an
	// ordinary request.
	timeout := c.requestTimeout()
	if !req.DontBlock && timeout > 0 && timeout < longPollTimeout {
		timeout = longPollTimeout
	}

	body, err := c.doRequestContext(ctx, timeout, "GET", "events", params, nil)
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

// retryableEventError reports whether an events fetch is worth waiting out. A
// network blip or a server-side failure is transient; a request the server
// rejected will be rejected again.
func retryableEventError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode >= 500 || apiErr.StatusCode == http.StatusTooManyRequests
	}
	return true
}

// CallOnEachEvent registers an event queue and calls the callback for each
// event until ctx is cancelled. Transient failures are retried with backoff, so
// a brief outage does not end a long-running listener, and the queue is
// deregistered on the way out rather than left for the server to expire.
func (c *Client) CallOnEachEvent(ctx context.Context, callback EventCallback, eventTypes []string, narrow [][]string) error {
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

	defer func() {
		// Best effort: the caller is on its way out either way, and the server
		// expires abandoned queues eventually.
		_, _ = c.Deregister(queueID)
	}()

	backoff := initialEventBackoff

	// Continuously fetch and process events
	for {
		if ctx.Err() != nil {
			return nil
		}

		eventsReq := GetEventsRequest{
			QueueID:     queueID,
			LastEventID: lastEventID,
		}

		events, err := c.GetEventsContext(ctx, eventsReq)
		if err != nil {
			// Cancellation is how a listener normally ends, not a failure.
			if ctx.Err() != nil {
				return nil
			}

			// The queue expired or was garbage collected: get a new one.
			if APIErrorCode(err) == "BAD_EVENT_QUEUE_ID" {
				var registerErr error
				registration, registerErr = c.Register(registerReq)
				if registerErr == nil {
					queueID = registration.QueueID
					lastEventID = registration.LastEventID
					backoff = initialEventBackoff
					continue
				}
				err = registerErr
			}

			if !retryableEventError(err) {
				return err
			}

			select {
			case <-ctx.Done():
				return nil
			case <-time.After(backoff):
			}
			if backoff *= 2; backoff > maxEventBackoff {
				backoff = maxEventBackoff
			}
			continue
		}

		backoff = initialEventBackoff

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

// CallOnEachMessage registers an event queue and calls the callback for each
// message until ctx is cancelled.
func (c *Client) CallOnEachMessage(ctx context.Context, callback MessageCallback) error {
	return c.CallOnEachEvent(ctx, func(event map[string]interface{}) {
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
