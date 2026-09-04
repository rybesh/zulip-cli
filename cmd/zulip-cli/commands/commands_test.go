package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/rybesh/zulip-cli/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// fakeZulip points the commands at a fake server and records what it received.
type fakeZulip struct {
	mu     sync.Mutex
	method string
	path   string
	params url.Values
	calls  int
}

func newFakeZulip(t *testing.T, body string) *fakeZulip {
	t.Helper()
	return newRoutedFakeZulip(t, map[string]string{"": body})
}

// newRoutedFakeZulip answers each endpoint with the body mapped to its path,
// and with the body under "" for anything else. Commands that look something up
// before acting need more than one canned answer.
func newRoutedFakeZulip(t *testing.T, bodies map[string]string) *fakeZulip {
	t.Helper()
	f := &fakeZulip{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		_ = r.ParseForm()
		f.method, f.path, f.params = r.Method, r.URL.Path, r.Form
		f.calls++
		f.mu.Unlock()
		body, found := bodies[r.URL.Path]
		if !found {
			body = bodies[""]
		}
		if body == "" {
			body = `{"result":"success"}`
		}
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)

	previous := newClient
	newClient = func() (*client.Client, error) {
		return client.NewClientWithConfig(client.Config{
			URL: srv.URL, Email: "bot@example.com", APIKey: "key",
		})
	}
	t.Cleanup(func() { newClient = previous })

	return f
}

func (f *fakeZulip) snapshot() (method, path string, params url.Values, calls int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.method, f.path, f.params, f.calls
}

// run executes the CLI with args and returns what it printed. Flags are reset
// first, because they live in package-level state that survives between runs
// inside one test binary.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetFlags(rootCmd)

	var out bytes.Buffer
	previousStdout := stdout
	stdout = &out
	t.Cleanup(func() { stdout = previousStdout })

	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	return out.String(), err
}

func resetFlags(cmd *cobra.Command) {
	reset := func(f *pflag.Flag) {
		f.Changed = false
		if slice, ok := f.Value.(pflag.SliceValue); ok {
			_ = slice.Replace(nil)
			return
		}
		_ = f.Value.Set(f.DefValue)
	}

	cmd.Flags().VisitAll(reset)
	cmd.PersistentFlags().VisitAll(reset)
	for _, sub := range cmd.Commands() {
		resetFlags(sub)
	}
}

// What the user types has to survive all the way to the request. A flag left
// out is left out of the request; a flag set to false is sent as false.
func TestFlagsReachTheRequest(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		wantMethod string
		wantPath   string
		wantParams url.Values
	}{
		{
			name:       "no flags means no parameters",
			args:       []string{"list-channels"},
			wantMethod: "GET",
			wantPath:   "/api/v1/streams",
			wantParams: url.Values{},
		},
		{
			name:       "false is sent, not dropped",
			args:       []string{"list-channels", "--include-public=false"},
			wantMethod: "GET",
			wantPath:   "/api/v1/streams",
			wantParams: url.Values{"include_public": {"false"}},
		},
		{
			name:       "only what the user set is sent",
			args:       []string{"list-channels", "--include-subscribed=false", "--include-public=true"},
			wantMethod: "GET",
			wantPath:   "/api/v1/streams",
			wantParams: url.Values{"include_subscribed": {"false"}, "include_public": {"true"}},
		},
		{
			name:       "false on a user flag",
			args:       []string{"get-user", "7", "--include-custom-profile-fields=false"},
			wantMethod: "GET",
			wantPath:   "/api/v1/users/7",
			wantParams: url.Values{"include_custom_profile_fields": {"false"}},
		},
		{
			name:       "an empty description clears it",
			args:       []string{"update-channel", "42", "--description", ""},
			wantMethod: "PATCH",
			wantPath:   "/api/v1/streams/42",
			wantParams: url.Values{"description": {""}},
		},
		{
			name:       "an empty topic is sent",
			args:       []string{"update-message", "5", "--topic", ""},
			wantMethod: "PATCH",
			wantPath:   "/api/v1/messages/5",
			wantParams: url.Values{"topic": {""}},
		},
		{
			name:       "the retired --stream spelling still works",
			args:       []string{"send-message", "--stream", "general", "--topic", "hi", "--content", "yo"},
			wantMethod: "POST",
			wantPath:   "/api/v1/messages",
			wantParams: url.Values{
				"type": {"stream"}, "to": {"general"}, "topic": {"hi"}, "content": {"yo"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeZulip(t, `{"result":"success"}`)

			if _, err := run(t, tc.args...); err != nil {
				t.Fatalf("%v: %v", tc.args, err)
			}

			method, path, params, calls := fake.snapshot()
			if calls != 1 {
				t.Fatalf("made %d requests, want 1", calls)
			}
			if method != tc.wantMethod || path != tc.wantPath {
				t.Errorf("%s %s, want %s %s", method, path, tc.wantMethod, tc.wantPath)
			}
			if !reflect.DeepEqual(params, tc.wantParams) {
				t.Errorf("parameters = %v, want %v", params, tc.wantParams)
			}
		})
	}
}

// A command that cannot do what was asked should say so before touching the
// network, not send a request that means something else.
func TestUsageErrorsHappenBeforeAnyRequest(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "listen cannot do both modes at once",
			args: []string{"listen", "--messages-only", "--event-types", "reaction"},
			want: "cannot be combined",
		},
		{
			name: "update-channel needs something to change",
			args: []string{"update-channel", "42"},
			want: "nothing to change",
		},
		{
			name: "update-user-group needs something to change",
			args: []string{"update-user-group", "3"},
			want: "either --name or --description",
		},
		{
			name: "update-message needs something to change",
			args: []string{"update-message", "5"},
			want: "either --content or --topic",
		},
		{
			name: "send-message needs a recipient",
			args: []string{"send-message", "--content", "hi"},
			want: "either --channel or --to",
		},
		{
			name: "channel messages need a topic",
			args: []string{"send-message", "--channel", "general", "--content", "hi"},
			want: "--topic is required",
		},
		{
			name: "move-topic needs a destination",
			args: []string{"move-topic", "42", "old"},
			want: "either --new-channel-id or --new-topic",
		},
		{
			name: "an unknown visibility policy is refused",
			args: []string{"set-topic-visibility", "42", "off-topic", "hidden"},
			want: "unknown visibility policy",
		},
		{
			name: "unsupported output formats are refused",
			args: []string{"list-channels", "-o", "yaml"},
			want: "unsupported output format",
		},
		{
			name: "a negative timeout is refused",
			args: []string{"list-channels", "--timeout", "-5s"},
			want: "must not be negative",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeZulip(t, `{"result":"success"}`)

			_, err := run(t, tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want one mentioning %q", err, tc.want)
			}
			if _, _, _, calls := fake.snapshot(); calls != 0 {
				t.Errorf("made %d requests, want none", calls)
			}
		})
	}
}

func TestResultsArePrintedAsJSON(t *testing.T) {
	newFakeZulip(t, `{"result":"success","streams":[{"stream_id":7,"name":"general"}]}`)

	out, err := run(t, "list-channels")
	if err != nil {
		t.Fatal(err)
	}

	var got struct {
		Result  string `json:"result"`
		Streams []struct {
			StreamID int    `json:"stream_id"`
			Name     string `json:"name"`
		} `json:"streams"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if len(got.Streams) != 1 || got.Streams[0].Name != "general" {
		t.Errorf("got %+v, want the channel from the server", got)
	}
}

// Commands that do not talk to a server must not require credentials, or a
// connection, to run.
func TestVersionNeedsNoClient(t *testing.T) {
	previous := newClient
	newClient = func() (*client.Client, error) {
		t.Error("version built a client")
		return nil, fmt.Errorf("no client available")
	}
	t.Cleanup(func() { newClient = previous })

	out, err := run(t, "version")
	if err != nil {
		t.Fatalf("version failed: %v", err)
	}

	var build client.BuildInfo
	if err := json.Unmarshal([]byte(out), &build); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if build.Version == "" || build.GoVersion == "" {
		t.Errorf("got %+v, want a version and a toolchain", build)
	}
}

// move-topic finds the topic's latest message before editing it, and the
// retired --new-stream-id spelling still names the destination.
func TestMoveTopicUsesTheLatestMessage(t *testing.T) {
	fake := newFakeZulip(t, `{"result":"success","messages":[{"id":99}]}`)

	if _, err := run(t, "move-topic", "42", "old", "--new-stream-id", "43"); err != nil {
		t.Fatal(err)
	}

	method, path, params, calls := fake.snapshot()
	if calls != 2 {
		t.Fatalf("made %d requests, want a lookup and an edit", calls)
	}
	if method != "PATCH" || path != "/api/v1/messages/99" {
		t.Errorf("%s %s, want PATCH /api/v1/messages/99", method, path)
	}
	want := url.Values{"stream_id": {"43"}, "propagate_mode": {"change_all"}}
	if !reflect.DeepEqual(params, want) {
		t.Errorf("parameters = %v, want %v", params, want)
	}
}

// create-channel takes group names as well as IDs, and reaches the endpoint
// that can actually apply the settings.
func TestCreateChannelResolvesGroupNames(t *testing.T) {
	fake := newRoutedFakeZulip(t, map[string]string{
		"/api/v1/server_settings": `{"result":"success","zulip_feature_level":509}`,
		"/api/v1/users/me":        `{"result":"success","user_id":8}`,
		"/api/v1/user_groups": `{"result":"success","user_groups":[` +
			`{"id":3,"name":"role:administrators"},{"id":4,"name":"engineering"}]}`,
	})

	if _, err := run(t, "create-channel", "music",
		"--can-send-message-group", "role:administrators",
		"--can-administer-channel-group", "4",
		"--topics-policy", "allow_empty_topic",
		"--message-retention-days", "unlimited",
	); err != nil {
		t.Fatal(err)
	}

	method, path, params, _ := fake.snapshot()
	if method != "POST" || path != "/api/v1/channels/create" {
		t.Fatalf("%s %s, want POST /api/v1/channels/create", method, path)
	}
	want := url.Values{
		"name":                         {"music"},
		"subscribers":                  {"[8]"},
		"can_send_message_group":       {"3"},
		"can_administer_channel_group": {"4"},
		"topics_policy":                {"allow_empty_topic"},
		"message_retention_days":       {`"unlimited"`},
	}
	if !reflect.DeepEqual(params, want) {
		t.Errorf("parameters = %v, want %v", params, want)
	}
}

// A group name the server does not have is the user's mistake, and no channel
// should be created because of it.
func TestCreateChannelRefusesAnUnknownGroup(t *testing.T) {
	fake := newRoutedFakeZulip(t, map[string]string{
		"/api/v1/server_settings": `{"result":"success","zulip_feature_level":509}`,
		"/api/v1/user_groups":     `{"result":"success","user_groups":[]}`,
	})

	_, err := run(t, "create-channel", "music", "--can-send-message-group", "nobody")
	if err == nil || !strings.Contains(err.Error(), `no user group named "nobody"`) {
		t.Fatalf("error = %v, want one naming the missing group", err)
	}
	if _, path, _, _ := fake.snapshot(); path == "/api/v1/channels/create" {
		t.Error("the channel was created anyway")
	}
}

// update-channel sends who may post as a group-setting update, which is what
// replaced the stream_post_policy the server dropped.
func TestUpdateChannelSetsWhoMayPost(t *testing.T) {
	fake := newFakeZulip(t, `{"result":"success"}`)

	if _, err := run(t, "update-channel", "42", "--can-send-message-group", "15"); err != nil {
		t.Fatal(err)
	}

	method, path, params, calls := fake.snapshot()
	if calls != 1 {
		t.Fatalf("made %d requests, want 1: a group ID needs no lookup", calls)
	}
	if method != "PATCH" || path != "/api/v1/streams/42" {
		t.Fatalf("%s %s, want PATCH /api/v1/streams/42", method, path)
	}
	want := url.Values{"can_send_message_group": {`{"new":15}`}}
	if !reflect.DeepEqual(params, want) {
		t.Errorf("parameters = %v, want %v", params, want)
	}
}

// The topic commands are shorthand for a visibility policy, and take a channel
// by name as well as by ID.
func TestTopicVisibilityCommands(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		wantCalls  int
		wantPolicy string
	}{
		{
			name:       "mute-topic mutes",
			args:       []string{"mute-topic", "42", "off-topic"},
			wantCalls:  2,
			wantPolicy: "1",
		},
		{
			name:       "unmute-topic clears the policy",
			args:       []string{"unmute-topic", "42", "off-topic"},
			wantCalls:  2,
			wantPolicy: "0",
		},
		{
			name:       "follow-topic follows",
			args:       []string{"follow-topic", "42", "off-topic"},
			wantCalls:  2,
			wantPolicy: "3",
		},
		{
			name:       "unfollow-topic clears the policy",
			args:       []string{"unfollow-topic", "42", "off-topic"},
			wantCalls:  2,
			wantPolicy: "0",
		},
		{
			name:       "set-topic-visibility takes the policy by name",
			args:       []string{"set-topic-visibility", "42", "off-topic", "unmuted"},
			wantCalls:  2,
			wantPolicy: "2",
		},
		{
			name:       "a channel name is looked up first",
			args:       []string{"mute-topic", "general", "off-topic"},
			wantCalls:  3,
			wantPolicy: "1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := newRoutedFakeZulip(t, map[string]string{
				"/api/v1/server_settings": `{"result":"success","zulip_feature_level":509}`,
				"/api/v1/get_stream_id":   `{"result":"success","stream_id":42}`,
			})

			if _, err := run(t, tc.args...); err != nil {
				t.Fatalf("%v: %v", tc.args, err)
			}

			method, path, params, calls := fake.snapshot()
			if calls != tc.wantCalls {
				t.Fatalf("made %d requests, want %d", calls, tc.wantCalls)
			}
			if method != "POST" || path != "/api/v1/user_topics" {
				t.Fatalf("%s %s, want POST /api/v1/user_topics", method, path)
			}
			want := url.Values{
				"stream_id":         {"42"},
				"topic":             {"off-topic"},
				"visibility_policy": {tc.wantPolicy},
			}
			if !reflect.DeepEqual(params, want) {
				t.Errorf("parameters = %v, want %v", params, want)
			}
		})
	}
}
