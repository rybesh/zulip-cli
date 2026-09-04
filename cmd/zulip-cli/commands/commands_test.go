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
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/rybesh/zulip-cli/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// request is one call a command made, as the fake server saw it.
type request struct {
	method string
	path   string
	params url.Values
}

// fakeZulip points the commands at a fake server and records what it received.
type fakeZulip struct {
	mu       sync.Mutex
	requests []request
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
	return newFakeZulipFunc(t, func(r *http.Request, _ int) string {
		if body, found := bodies[r.URL.Path]; found {
			return body
		}
		return bodies[""]
	})
}

// newSequencedFakeZulip answers with each body in turn, and repeats the last
// one after that. A command that pages through the history gets a different
// answer each time round, which is what makes the paging visible.
func newSequencedFakeZulip(t *testing.T, bodies ...string) *fakeZulip {
	t.Helper()
	return newFakeZulipFunc(t, func(_ *http.Request, n int) string {
		return bodies[min(n, len(bodies)-1)]
	})
}

// newFakeZulipFunc serves whatever body answer returns for each request, which
// it is given along with the number of requests already answered.
func newFakeZulipFunc(t *testing.T, answer func(r *http.Request, n int) string) *fakeZulip {
	t.Helper()
	f := &fakeZulip{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		_ = r.ParseForm()
		n := len(f.requests)
		f.requests = append(f.requests, request{r.Method, r.URL.Path, r.Form})
		f.mu.Unlock()

		body := answer(r, n)
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

// snapshot returns the last request made, and how many there were.
func (f *fakeZulip) snapshot() (method, path string, params url.Values, calls int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) == 0 {
		return "", "", nil, 0
	}
	last := f.requests[len(f.requests)-1]
	return last.method, last.path, last.params, len(f.requests)
}

// all returns every request made, in order.
func (f *fakeZulip) all() []request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.requests)
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
		{
			name: "a message can be tagged for the sender's own event queue",
			args: []string{"send-message", "--to", "dana@example.com", "--content", "yo",
				"--queue-id", "q1", "--local-id", "7"},
			wantMethod: "POST",
			wantPath:   "/api/v1/messages",
			wantParams: url.Values{
				"type": {"private"}, "to": {`["dana@example.com"]`}, "content": {"yo"},
				"queue_id": {"q1"}, "local_id": {"7"},
			},
		},
		{
			name: "--narrow passes any operator through, alongside the shorthands",
			args: []string{"get-messages", "--channel", "general",
				"--narrow", "sender=dana@example.com", "--narrow", "is=unread"},
			wantMethod: "GET",
			wantPath:   "/api/v1/messages",
			wantParams: url.Values{
				"anchor": {"newest"}, "num_before": {"100"}, "num_after": {"0"},
				"narrow": {`[{"operator":"stream","operand":"general"},` +
					`{"operator":"sender","operand":"dana@example.com"},` +
					`{"operator":"is","operand":"unread"}]`},
			},
		},
		{
			name:       "a search operand keeps its commas",
			args:       []string{"get-messages", "--narrow", "search=one, two"},
			wantMethod: "GET",
			wantPath:   "/api/v1/messages",
			wantParams: url.Values{
				"anchor": {"newest"}, "num_before": {"100"}, "num_after": {"0"},
				"narrow": {`[{"operator":"search","operand":"one, two"}]`},
			},
		},
		{
			name:       "--no-markdown asks for the Markdown the sender typed",
			args:       []string{"get-messages", "--no-markdown"},
			wantMethod: "GET",
			wantPath:   "/api/v1/messages",
			wantParams: url.Values{
				"anchor": {"newest"}, "num_before": {"100"}, "num_after": {"0"},
				"apply_markdown": {"false"},
			},
		},
		{
			name:       "--no-markdown=false asks for rendered HTML",
			args:       []string{"get-messages", "--no-markdown=false", "--client-gravatar"},
			wantMethod: "GET",
			wantPath:   "/api/v1/messages",
			wantParams: url.Values{
				"anchor": {"newest"}, "num_before": {"100"}, "num_after": {"0"},
				"apply_markdown": {"true"}, "client_gravatar": {"true"},
			},
		},
		{
			name:       "renaming a thread moves every message in it",
			args:       []string{"update-message", "5", "--topic", "new", "--propagate-mode", "change_all"},
			wantMethod: "PATCH",
			wantPath:   "/api/v1/messages/5",
			wantParams: url.Values{"topic": {"new"}, "propagate_mode": {"change_all"}},
		},
		{
			name: "a single message can be moved to another channel",
			args: []string{"update-message", "5", "--channel-id", "43",
				"--send-notification-to-old-thread=false"},
			wantMethod: "PATCH",
			wantPath:   "/api/v1/messages/5",
			wantParams: url.Values{
				"stream_id": {"43"}, "send_notification_to_old_thread": {"false"},
			},
		},
		{
			name: "the channel privacy settings are tri-state",
			args: []string{"update-channel", "42", "--invite-only=false",
				"--history-public-to-subscribers", "--message-retention-days", "30"},
			wantMethod: "PATCH",
			wantPath:   "/api/v1/streams/42",
			wantParams: url.Values{
				"is_private":                    {"false"},
				"history_public_to_subscribers": {"true"},
				"message_retention_days":        {"30"},
			},
		},
		{
			name:       "a custom emoji is reachable by code",
			args:       []string{"add-reaction", "42", "--emoji-code", "7", "--reaction-type", "realm_emoji"},
			wantMethod: "POST",
			wantPath:   "/api/v1/messages/42/reactions",
			wantParams: url.Values{"emoji_code": {"7"}, "reaction_type": {"realm_emoji"}},
		},
		{
			name:       "a reaction named by hand still sends its name",
			args:       []string{"remove-reaction", "42", "tada"},
			wantMethod: "DELETE",
			wantPath:   "/api/v1/messages/42/reactions",
			wantParams: url.Values{"emoji_name": {"tada"}},
		},
		{
			name:       "flags are set on the messages named",
			args:       []string{"update-message-flags", "add", "starred", "41", "42"},
			wantMethod: "POST",
			wantPath:   "/api/v1/messages/flags",
			wantParams: url.Values{"messages": {"[41,42]"}, "op": {"add"}, "flag": {"starred"}},
		},
		{
			name:       "profile fields are sent as the list the server reads",
			args:       []string{"update-user", "7", "--profile-data", "4=Berlin", "--profile-data", "6=Team lead"},
			wantMethod: "PATCH",
			wantPath:   "/api/v1/users/7",
			wantParams: url.Values{
				"profile_data": {`[{"id":4,"value":"Berlin"},{"id":6,"value":"Team lead"}]`},
			},
		},
		{
			name:       "notification settings are tri-state too",
			args:       []string{"update-notification-settings", "--channel-desktop=false", "--offline-push"},
			wantMethod: "PATCH",
			wantPath:   "/api/v1/settings/notifications",
			wantParams: url.Values{
				"enable_stream_desktop_notifications": {"false"},
				"enable_offline_push_notifications":   {"true"},
			},
		},
		{
			name:       "bot storage is written as key and value",
			args:       []string{"update-storage", "last-seen=1717171717", "greeting=hello there"},
			wantMethod: "POST",
			wantPath:   "/api/v1/bot_storage",
			wantParams: url.Values{
				"storage": {`{"greeting":"hello there","last-seen":"1717171717"}`},
			},
		},
		{
			name:       "a queue left behind can be released",
			args:       []string{"deregister", "1518familiar"},
			wantMethod: "DELETE",
			wantPath:   "/api/v1/events",
			wantParams: url.Values{"queue_id": {"1518familiar"}},
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
			want: "either --content, --topic or --channel-id",
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
		{
			name: "a narrow that is not operator=operand is refused",
			args: []string{"get-messages", "--narrow", "sender"},
			want: "it must be operator=operand",
		},
		{
			name: "an unknown propagate mode is refused",
			args: []string{"update-message", "5", "--topic", "new", "--propagate-mode", "change_some"},
			want: "unknown propagate mode",
		},
		{
			name: "an unknown reaction type is refused",
			args: []string{"add-reaction", "42", "tada", "--reaction-type", "sticker"},
			want: "unknown reaction type",
		},
		{
			name: "a reaction has to name an emoji",
			args: []string{"add-reaction", "42"},
			want: "name the emoji",
		},
		{
			name: "a flag the server computes cannot be set",
			args: []string{"update-message-flags", "add", "mentioned", "41"},
			want: "unknown message flag",
		},
		{
			name: "an unknown flag operation is refused",
			args: []string{"update-message-flags", "toggle", "read", "41"},
			want: "it must be add or remove",
		},
		{
			name: "--all cannot walk in both directions",
			args: []string{"get-messages", "--all", "--num-after", "10"},
			want: "cannot be combined with --num-after",
		},
		{
			name: "--local-id means nothing without a queue",
			args: []string{"send-message", "--to", "dana@example.com", "--content", "hi", "--local-id", "7"},
			want: "--local-id needs --queue-id",
		},
		{
			name: "update-subscription needs something to change",
			args: []string{"update-subscription", "general"},
			want: "nothing to change",
		},
		{
			name: "update-notification-settings needs something to change",
			args: []string{"update-notification-settings"},
			want: "nothing to change",
		},
		{
			name: "set-typing-status needs a conversation",
			args: []string{"set-typing-status", "start"},
			want: "either --channel or --to",
		},
		{
			name: "set-typing-status cannot name two conversations",
			args: []string{"set-typing-status", "start", "--channel", "general", "--to", "dana@example.com"},
			want: "only one of them can be given",
		},
		{
			name: "an unknown typing operation is refused",
			args: []string{"set-typing-status", "typing", "--to", "dana@example.com"},
			want: "it must be start or stop",
		},
		{
			name: "storage has to be written as key=value",
			args: []string{"update-storage", "last-seen"},
			want: "it must be key=value",
		},
		{
			name: "profile data has to name a field by ID",
			args: []string{"update-user", "7", "--profile-data", "Location=Berlin"},
			want: "it must be field-id=value",
		},
		{
			name: "check-messages-match-narrow has nothing to match without a narrow",
			args: []string{"check-messages-match-narrow", "41"},
			want: "--narrow is required",
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

// --all keeps asking for older messages until the server says the history has
// run out, and hands back one merged result rather than a page at a time.
func TestGetMessagesAllWalksTheHistory(t *testing.T) {
	fake := newSequencedFakeZulip(t,
		`{"result":"success","anchor":30,"found_newest":true,"found_oldest":false,
		  "messages":[{"id":20},{"id":30}]}`,
		`{"result":"success","found_oldest":true,
		  "messages":[{"id":5},{"id":10}]}`,
	)

	out, err := run(t, "get-messages", "--all", "--num-before", "2")
	if err != nil {
		t.Fatal(err)
	}

	requests := fake.all()
	if len(requests) != 2 {
		t.Fatalf("made %d requests, want 2: one page, then the rest", len(requests))
	}
	if got := requests[0].params.Get("anchor"); got != "newest" {
		t.Errorf("first anchor = %q, want the anchor the user asked for", got)
	}
	// The second page anchors below the oldest message of the first, so the
	// message with ID 20 is not reported twice.
	if got := requests[1].params.Get("anchor"); got != "19" {
		t.Errorf("second anchor = %q, want 19", got)
	}

	var got client.GetMessagesResponse
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	wantIDs := []int{5, 10, 20, 30}
	var ids []int
	for _, message := range got.Messages {
		ids = append(ids, message.ID)
	}
	if !reflect.DeepEqual(ids, wantIDs) {
		t.Errorf("message IDs = %v, want %v: oldest first, no duplicates", ids, wantIDs)
	}
	if !got.FoundOldest {
		t.Error("found_oldest = false, want the last page's answer")
	}
	if got.Anchor != 30 {
		t.Errorf("anchor = %d, want 30: the first page's, which is what was asked for", got.Anchor)
	}
}

// A walk that the server never says it has finished still ends, rather than
// asking for the same empty page forever.
func TestGetMessagesAllStopsWhenAPageIsEmpty(t *testing.T) {
	fake := newSequencedFakeZulip(t,
		`{"result":"success","found_oldest":false,"messages":[{"id":10}]}`,
		`{"result":"success","found_oldest":false,"messages":[]}`,
	)

	if _, err := run(t, "get-messages", "--all"); err != nil {
		t.Fatal(err)
	}

	if calls := len(fake.all()); calls != 2 {
		t.Errorf("made %d requests, want 2: an empty page ends the walk", calls)
	}
}

// subscribe can name other people, by ID or by email, and an email costs one
// lookup however many there are.
func TestSubscribeResolvesPrincipals(t *testing.T) {
	fake := newRoutedFakeZulip(t, map[string]string{
		"/api/v1/users": `{"result":"success","members":[` +
			`{"user_id":12,"email":"dana@example.com"},{"user_id":13,"email":"sam@example.com"}]}`,
	})

	if _, err := run(t, "subscribe", "general",
		"--principals", "9,dana@example.com,sam@example.com",
		"--authorization-errors-fatal=false"); err != nil {
		t.Fatal(err)
	}

	method, path, params, calls := fake.snapshot()
	if calls != 2 {
		t.Fatalf("made %d requests, want a user lookup and a subscribe", calls)
	}
	if method != "POST" || path != "/api/v1/users/me/subscriptions" {
		t.Fatalf("%s %s, want POST /api/v1/users/me/subscriptions", method, path)
	}
	want := url.Values{
		"subscriptions":              {`[{"name":"general"}]`},
		"principals":                 {"[9,12,13]"},
		"authorization_errors_fatal": {"false"},
	}
	if !reflect.DeepEqual(params, want) {
		t.Errorf("parameters = %v, want %v", params, want)
	}
}

// A principal the server does not have is the user's mistake, and nobody
// should be subscribed because of it.
func TestSubscribeRefusesAnUnknownPrincipal(t *testing.T) {
	fake := newRoutedFakeZulip(t, map[string]string{
		"/api/v1/users": `{"result":"success","members":[]}`,
	})

	_, err := run(t, "subscribe", "general", "--principals", "nobody@example.com")
	if err == nil || !strings.Contains(err.Error(), `no user with email "nobody@example.com"`) {
		t.Fatalf("error = %v, want one naming the missing user", err)
	}
	if _, path, _, _ := fake.snapshot(); path == "/api/v1/users/me/subscriptions" {
		t.Error("the subscription was made anyway")
	}
}

// update-subscription changes your own settings for a channel, which the
// server takes as a list of property and value against a channel ID.
func TestUpdateSubscriptionBuildsSubscriptionData(t *testing.T) {
	fake := newRoutedFakeZulip(t, map[string]string{
		"/api/v1/get_stream_id": `{"result":"success","stream_id":42}`,
	})

	if _, err := run(t, "update-subscription", "general",
		"--color", "#76ce90", "--pin-to-top", "--is-muted=false"); err != nil {
		t.Fatal(err)
	}

	method, path, params, calls := fake.snapshot()
	if calls != 2 {
		t.Fatalf("made %d requests, want a channel lookup and an update", calls)
	}
	if method != "POST" || path != "/api/v1/users/me/subscriptions/properties" {
		t.Fatalf("%s %s, want POST /api/v1/users/me/subscriptions/properties", method, path)
	}
	want := url.Values{"subscription_data": {
		`[{"property":"color","stream_id":42,"value":"#76ce90"},` +
			`{"property":"pin_to_top","stream_id":42,"value":true},` +
			`{"property":"is_muted","stream_id":42,"value":false}]`,
	}}
	if !reflect.DeepEqual(params, want) {
		t.Errorf("parameters = %v, want %v", params, want)
	}
}

// A channel name is looked up wherever a command takes one, so that neither
// the ID nor the name is the only thing that works.
func TestChannelNamesAreLookedUp(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantPath string
	}{
		{
			name:     "get-channel-email-address",
			args:     []string{"get-channel-email-address", "general"},
			wantPath: "/api/v1/streams/42/email_address",
		},
		{
			name:     "add-default-channel",
			args:     []string{"add-default-channel", "general"},
			wantPath: "/api/v1/default_streams",
		},
		{
			name:     "set-typing-status",
			args:     []string{"set-typing-status", "start", "--channel", "general", "--topic", "standup"},
			wantPath: "/api/v1/typing",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := newRoutedFakeZulip(t, map[string]string{
				"/api/v1/get_stream_id": `{"result":"success","stream_id":42}`,
			})

			if _, err := run(t, tc.args...); err != nil {
				t.Fatalf("%v: %v", tc.args, err)
			}

			requests := fake.all()
			if len(requests) != 2 {
				t.Fatalf("made %d requests, want a name lookup and the command itself", len(requests))
			}
			if requests[0].path != "/api/v1/get_stream_id" {
				t.Errorf("looked up %s, want /api/v1/get_stream_id", requests[0].path)
			}
			if requests[1].path != tc.wantPath {
				t.Errorf("path = %s, want %s", requests[1].path, tc.wantPath)
			}
		})
	}
}

// get-subscription-status takes a user and a channel however the user has
// them, and asks about the pair.
func TestGetSubscriptionStatusResolvesBoth(t *testing.T) {
	fake := newRoutedFakeZulip(t, map[string]string{
		"/api/v1/users":         `{"result":"success","members":[{"user_id":12,"email":"dana@example.com"}]}`,
		"/api/v1/get_stream_id": `{"result":"success","stream_id":42}`,
	})

	if _, err := run(t, "get-subscription-status", "dana@example.com", "general"); err != nil {
		t.Fatal(err)
	}

	method, path, _, _ := fake.snapshot()
	if method != "GET" || path != "/api/v1/users/12/subscriptions/42" {
		t.Errorf("%s %s, want GET /api/v1/users/12/subscriptions/42", method, path)
	}
}
