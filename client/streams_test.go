package client

import (
	"fmt"
	"net/url"
	"reflect"
	"testing"

	"github.com/rybesh/zulip-cli/types"
)

// serverSettings is what a server of the given feature level answers with.
func serverSettings(level int) map[string]string {
	return map[string]string{
		"/api/v1/server_settings": fmt.Sprintf(
			`{"result":"success","zulip_feature_level":%d}`, level),
	}
}

// A current server gets the dedicated creation endpoint, which needs the user
// IDs to subscribe — the caller's own when none were named.
func TestCreateChannelUsesTheDedicatedEndpoint(t *testing.T) {
	bodies := serverSettings(ChannelCreateFeatureLevel)
	bodies["/api/v1/users/me"] = `{"result":"success","user_id":8}`
	bodies["/api/v1/channels/create"] = `{"result":"success","id":50}`
	c, rec := testRoutedServer(t, bodies)

	description := "For music"
	resp, err := c.CreateChannel(CreateChannelRequest{
		Name:            "music",
		Description:     &description,
		ChannelSettings: ChannelSettings{CanSendMessageGroup: types.NamedGroup(15)},
	})
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}
	if resp.ID != 50 {
		t.Errorf("new channel ID = %d, want 50", resp.ID)
	}

	method, path, params, _ := rec.snapshot()
	if method != "POST" || path != "/api/v1/channels/create" {
		t.Fatalf("%s %s, want POST /api/v1/channels/create", method, path)
	}
	want := url.Values{
		"name":                   {"music"},
		"description":            {"For music"},
		"subscribers":            {"[8]"},
		"can_send_message_group": {"15"},
	}
	if !reflect.DeepEqual(params, want) {
		t.Errorf("parameters = %v, want %v", params, want)
	}
}

// Named subscribers are sent as given, without a lookup of the caller's own ID.
func TestCreateChannelSubscribesTheUsersNamed(t *testing.T) {
	bodies := serverSettings(509)
	bodies["/api/v1/users/me"] = `{"result":"error","msg":"should not be asked"}`
	c, rec := testRoutedServer(t, bodies)

	if _, err := c.CreateChannel(CreateChannelRequest{
		Name:        "music",
		Subscribers: []int{17, 12},
	}); err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}

	_, path, params, calls := rec.snapshot()
	if calls != 2 {
		t.Errorf("made %d requests, want a feature-level check and the creation", calls)
	}
	if path != "/api/v1/channels/create" {
		t.Fatalf("posted to %s, want /api/v1/channels/create", path)
	}
	if got := params.Get("subscribers"); got != "[17,12]" {
		t.Errorf("subscribers = %s, want [17,12]", got)
	}
}

// A server too old for POST /channels/create still creates channels the way it
// always did, carrying the same settings.
func TestCreateChannelFallsBackToSubscribing(t *testing.T) {
	c, rec := testRoutedServer(t, serverSettings(ChannelCreateFeatureLevel-1))

	description := "For music"
	if _, err := c.CreateChannel(CreateChannelRequest{
		Name:            "music",
		Description:     &description,
		Subscribers:     []int{17},
		ChannelSettings: ChannelSettings{CanSendMessageGroup: types.NamedGroup(15)},
	}); err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}

	method, path, params, _ := rec.snapshot()
	if method != "POST" || path != "/api/v1/users/me/subscriptions" {
		t.Fatalf("%s %s, want POST /api/v1/users/me/subscriptions", method, path)
	}
	want := url.Values{
		"subscriptions":          {`[{"name":"music","description":"For music"}]`},
		"principals":             {"[17]"},
		"can_send_message_group": {"15"},
	}
	if !reflect.DeepEqual(params, want) {
		t.Errorf("parameters = %v, want %v", params, want)
	}
}

// The feature level is settled once, however many calls need it.
func TestFeatureLevelIsFetchedOnce(t *testing.T) {
	c, rec := testRoutedServer(t, serverSettings(509))

	for i := 0; i < 3; i++ {
		level, err := c.FeatureLevel()
		if err != nil {
			t.Fatalf("FeatureLevel: %v", err)
		}
		if level != 509 {
			t.Fatalf("feature level = %d, want 509", level)
		}
	}

	if _, _, _, calls := rec.snapshot(); calls != 1 {
		t.Errorf("made %d requests, want 1", calls)
	}
}

// Group-setting values arrive as either a group ID or an anonymous group, and
// have to survive the round trip in both shapes.
func TestGroupSettingRoundTrip(t *testing.T) {
	cases := []struct {
		wire string
		want types.GroupSetting
	}{
		{"15", *types.NamedGroup(15)},
		{`{"direct_members":[10],"direct_subgroups":[11]}`,
			*types.AnonymousGroup([]int{10}, []int{11})},
	}

	for _, tc := range cases {
		var got types.GroupSetting
		if err := got.UnmarshalJSON([]byte(tc.wire)); err != nil {
			t.Fatalf("unmarshal %s: %v", tc.wire, err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("unmarshal %s = %+v, want %+v", tc.wire, got, tc.want)
		}
		back, err := got.MarshalJSON()
		if err != nil {
			t.Fatalf("marshal %+v: %v", got, err)
		}
		if string(back) != tc.wire {
			t.Errorf("marshal %+v = %s, want %s", got, back, tc.wire)
		}
	}
}
