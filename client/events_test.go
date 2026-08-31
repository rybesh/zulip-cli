package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rybesh/zulip-cli/types"
)

// eventServer stands in for the event queue endpoints. respond decides what a
// fetch returns, given how many fetches have happened so far.
type eventServer struct {
	mu           sync.Mutex
	registers    int
	fetches      int
	deregistered []string
	respond      func(w http.ResponseWriter, fetch int)
}

func newEventServer(t *testing.T, respond func(w http.ResponseWriter, fetch int)) (*Client, *eventServer) {
	t.Helper()
	es := &eventServer{respond: respond}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/register"):
			es.mu.Lock()
			es.registers++
			queue := fmt.Sprintf("q%d", es.registers)
			es.mu.Unlock()
			fmt.Fprintf(w, `{"result":"success","queue_id":%q,"last_event_id":-1}`, queue)
		case r.Method == http.MethodDelete:
			es.mu.Lock()
			es.deregistered = append(es.deregistered, r.URL.Query().Get("queue_id"))
			es.mu.Unlock()
			fmt.Fprint(w, `{"result":"success"}`)
		default:
			es.mu.Lock()
			es.fetches++
			fetch := es.fetches
			es.mu.Unlock()
			es.respond(w, fetch)
		}
	}))
	t.Cleanup(srv.Close)

	return testClient(t, Config{URL: srv.URL}), es
}

func (es *eventServer) counts() (registers, fetches int, deregistered []string) {
	es.mu.Lock()
	defer es.mu.Unlock()
	return es.registers, es.fetches, append([]string(nil), es.deregistered...)
}

// writeMessageEvent writes one message event, which is what the listener is
// waiting for in most of these tests.
func writeMessageEvent(w http.ResponseWriter, id int, content string) {
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"result": "success",
		"events": []map[string]interface{}{
			{"id": id, "type": "message", "message": map[string]interface{}{"content": content}},
		},
	})
}

// runListener runs CallOnEachMessage until stop returns true for a message, and
// reports what the call returned.
func runListener(t *testing.T, c *Client, stop func(types.Message) bool) error {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- c.CallOnEachMessage(ctx, func(msg types.Message) {
			if stop(msg) {
				cancel()
			}
		})
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(30 * time.Second):
		t.Fatal("listener never returned")
		return nil
	}
}

// A brief outage used to end a long-running watcher outright.
func TestListenerRetriesTransientFailures(t *testing.T) {
	c, es := newEventServer(t, func(w http.ResponseWriter, fetch int) {
		if fetch == 1 {
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprint(w, `{"result":"error","msg":"bad gateway"}`)
			return
		}
		writeMessageEvent(w, 1, "hello")
	})

	if err := runListener(t, c, func(types.Message) bool { return true }); err != nil {
		t.Fatalf("listener returned %v, want nil after a clean stop", err)
	}

	registers, fetches, deregistered := es.counts()
	if registers != 1 {
		t.Errorf("registered %d times, want 1", registers)
	}
	if fetches < 2 {
		t.Errorf("made %d fetches, want a retry after the 502", fetches)
	}
	if len(deregistered) != 1 || deregistered[0] != "q1" {
		t.Errorf("deregistered %v, want [q1]", deregistered)
	}
}

// An expired queue is not a failure: the listener takes a new one and carries on.
func TestListenerReregistersOnBadQueueID(t *testing.T) {
	c, es := newEventServer(t, func(w http.ResponseWriter, fetch int) {
		if fetch == 1 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"result":"error","msg":"Bad event queue ID","code":"BAD_EVENT_QUEUE_ID"}`)
			return
		}
		writeMessageEvent(w, 1, "hello")
	})

	if err := runListener(t, c, func(types.Message) bool { return true }); err != nil {
		t.Fatalf("listener returned %v, want nil after a clean stop", err)
	}

	registers, _, deregistered := es.counts()
	if registers != 2 {
		t.Errorf("registered %d times, want a second registration", registers)
	}
	// The queue released at the end is the one actually in use.
	if len(deregistered) != 1 || deregistered[0] != "q2" {
		t.Errorf("deregistered %v, want [q2]", deregistered)
	}
}

// A request the server rejects will be rejected again, so it ends the listener.
func TestListenerStopsOnRejectedRequest(t *testing.T) {
	c, es := newEventServer(t, func(w http.ResponseWriter, fetch int) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"result":"error","msg":"Invalid narrow","code":"BAD_REQUEST"}`)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := c.CallOnEachEvent(ctx, func(map[string]interface{}) {}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "Invalid narrow") {
		t.Fatalf("listener returned %v, want the server's error", err)
	}

	if _, _, deregistered := es.counts(); len(deregistered) != 1 {
		t.Errorf("deregistered %v, want the queue released even on failure", deregistered)
	}
}

func TestListenerSkipsHeartbeats(t *testing.T) {
	c, _ := newEventServer(t, func(w http.ResponseWriter, fetch int) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"result": "success",
			"events": []map[string]interface{}{
				{"id": 1, "type": "heartbeat"},
				{"id": 2, "type": "message", "message": map[string]interface{}{"content": "hello"}},
			},
		})
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	var seen []string
	done := make(chan error, 1)
	go func() {
		done <- c.CallOnEachEvent(ctx, func(event map[string]interface{}) {
			mu.Lock()
			seen = append(seen, event["type"].(string))
			mu.Unlock()
			cancel()
		}, nil, nil)
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("listener never returned")
	}

	mu.Lock()
	defer mu.Unlock()
	for _, eventType := range seen {
		if eventType == "heartbeat" {
			t.Fatalf("callback saw %v, want heartbeats filtered out", seen)
		}
	}
}

// Long-polling waits on the server, so it must not inherit the ordinary
// request timeout.
func TestBlockingFetchOutlastsTheRequestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		fmt.Fprint(w, `{"result":"success","events":[]}`)
	}))
	defer srv.Close()

	c := testClient(t, Config{URL: srv.URL, Timeout: 50 * time.Millisecond})

	if _, err := c.GetEvents(GetEventsRequest{QueueID: "q1"}); err != nil {
		t.Fatalf("blocking fetch failed: %v", err)
	}
	if _, err := c.GetEvents(GetEventsRequest{QueueID: "q1", DontBlock: true}); err == nil {
		t.Fatal("a non-blocking fetch should still honor the request timeout")
	}
}
