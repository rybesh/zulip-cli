package client

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/rybesh/zulip-cli/types"
)

// The request builders decide what actually reaches Zulip. The cases that
// matter most are the ones that used to be dropped: a false boolean, which the
// server would otherwise fill in with its own default, and an empty string,
// which is how a value gets cleared.
func TestRequestParameters(t *testing.T) {
	no, yes := false, true
	empty := ""
	name := "New Name"

	cases := []struct {
		name string
		do   func(*Client) error
		want url.Values
	}{
		{
			name: "list channels: unset flags are omitted",
			do:   func(c *Client) error { _, err := c.GetStreams(GetStreamsRequest{}); return err },
			want: url.Values{},
		},
		{
			name: "list channels: false is sent as false",
			do: func(c *Client) error {
				_, err := c.GetStreams(GetStreamsRequest{IncludePublic: &no, IncludeSubscribed: &no})
				return err
			},
			want: url.Values{"include_public": {"false"}, "include_subscribed": {"false"}},
		},
		{
			name: "list channels: true is sent as true",
			do: func(c *Client) error {
				_, err := c.GetStreams(GetStreamsRequest{IncludeAllActive: &yes})
				return err
			},
			want: url.Values{"include_all_active": {"true"}},
		},
		{
			name: "list subscriptions: false is sent as false",
			do: func(c *Client) error {
				_, err := c.GetSubscriptions(GetSubscriptionsRequest{IncludeSubscribers: &no})
				return err
			},
			want: url.Values{"include_subscribers": {"false"}},
		},
		{
			name: "list users: false is sent as false",
			do: func(c *Client) error {
				_, err := c.GetUsers(GetUsersRequest{ClientGravatar: &no, IncludeCustomProfileFields: &no})
				return err
			},
			want: url.Values{"client_gravatar": {"false"}, "include_custom_profile_fields": {"false"}},
		},
		{
			name: "get user: nil leaves the parameter out",
			do:   func(c *Client) error { _, err := c.GetUser(7, nil); return err },
			want: url.Values{},
		},
		{
			name: "get user: false is sent as false",
			do:   func(c *Client) error { _, err := c.GetUser(7, &no); return err },
			want: url.Values{"include_custom_profile_fields": {"false"}},
		},
		{
			name: "update channel: an empty description clears it",
			do: func(c *Client) error {
				_, err := c.UpdateStream(UpdateStreamRequest{StreamID: 1, Description: &empty})
				return err
			},
			want: url.Values{"description": {""}},
		},
		{
			name: "update channel: unset fields are omitted",
			do: func(c *Client) error {
				_, err := c.UpdateStream(UpdateStreamRequest{StreamID: 1})
				return err
			},
			want: url.Values{},
		},
		{
			name: "update channel: a channel can be made public again",
			do: func(c *Client) error {
				_, err := c.UpdateStream(UpdateStreamRequest{StreamID: 1, IsPrivate: &no})
				return err
			},
			want: url.Values{"is_private": {"false"}},
		},
		{
			name: "update group: an empty description clears it",
			do: func(c *Client) error {
				_, err := c.UpdateUserGroup(UpdateUserGroupRequest{GroupID: 2, Description: &empty})
				return err
			},
			want: url.Values{"description": {""}},
		},
		{
			name: "update message: an empty topic is sent",
			do: func(c *Client) error {
				_, err := c.UpdateMessage(UpdateMessageRequest{MessageID: 3, Topic: &empty})
				return err
			},
			want: url.Values{"topic": {""}},
		},
		{
			name: "update message: notification flags can be turned off",
			do: func(c *Client) error {
				_, err := c.UpdateMessage(UpdateMessageRequest{
					MessageID:                   3,
					Content:                     &name,
					SendNotificationToOldThread: &no,
				})
				return err
			},
			want: url.Values{"content": {"New Name"}, "send_notification_to_old_thread": {"false"}},
		},
		{
			name: "update user: a full name is sent as typed",
			do: func(c *Client) error {
				_, err := c.UpdateUser(UpdateUserRequest{UserID: 4, FullName: &name})
				return err
			},
			want: url.Values{"full_name": {"New Name"}},
		},
		{
			name: "update presence: false is sent as false",
			do: func(c *Client) error {
				_, err := c.UpdatePresence(UpdatePresenceRequest{Status: "active", PingOnly: &no})
				return err
			},
			want: url.Values{"status": {"active"}, "ping_only": {"false"}},
		},
		{
			name: "register: false is sent as false",
			do: func(c *Client) error {
				_, err := c.Register(RegisterRequest{EventTypes: []string{"message"}, ApplyMarkdown: &no})
				return err
			},
			want: url.Values{"event_types": {`["message"]`}, "apply_markdown": {"false"}},
		},
		{
			name: "get messages: anchor and counts always travel",
			do: func(c *Client) error {
				_, err := c.GetMessages(GetMessagesRequest{
					Anchor:        "newest",
					NumBefore:     5,
					ApplyMarkdown: &no,
				})
				return err
			},
			want: url.Values{
				"anchor":         {"newest"},
				"num_before":     {"5"},
				"num_after":      {"0"},
				"apply_markdown": {"false"},
			},
		},
		{
			name: "move topic: propagate mode is a bare string",
			do: func(c *Client) error {
				_, err := c.MoveTopic(MoveTopicRequest{
					StreamID:      1,
					MessageID:     9,
					Topic:         "old",
					NewTopic:      &name,
					PropagateMode: types.ChangeLater,
				})
				return err
			},
			want: url.Values{"topic": {"New Name"}, "propagate_mode": {"change_later"}},
		},
		{
			name: "move topic: the default propagate mode is change_all",
			do: func(c *Client) error {
				_, err := c.MoveTopic(MoveTopicRequest{StreamID: 1, MessageID: 9, NewStreamID: 2, Topic: "old"})
				return err
			},
			want: url.Values{"stream_id": {"2"}, "propagate_mode": {"change_all"}},
		},
		{
			name: "subscribe: false is sent as false",
			do: func(c *Client) error {
				req := SubscribeRequest{
					Subscriptions:   []ChannelSubscription{{Name: "engineering"}},
					ChannelSettings: ChannelSettings{InviteOnly: &no},
				}
				_, err := c.Subscribe(req)
				return err
			},
			want: url.Values{
				"subscriptions": {`[{"name":"engineering"}]`},
				"invite_only":   {"false"},
			},
		},
		{
			name: "subscribe: a group-setting value is sent as a bare group ID",
			do: func(c *Client) error {
				req := SubscribeRequest{
					Subscriptions: []ChannelSubscription{{Name: "engineering"}},
					ChannelSettings: ChannelSettings{ChannelPermissions: ChannelPermissions{
						CanSendMessageGroup: types.NamedGroup(15),
						CanAddSubscribersGroup: types.AnonymousGroup(
							[]int{10}, []int{11}),
					}},
				}
				_, err := c.Subscribe(req)
				return err
			},
			want: url.Values{
				"subscriptions":             {`[{"name":"engineering"}]`},
				"can_send_message_group":    {"15"},
				"can_add_subscribers_group": {`{"direct_members":[10],"direct_subgroups":[11]}`},
			},
		},
		{
			name: "subscribe: retention days travel as JSON",
			do: func(c *Client) error {
				req := SubscribeRequest{
					Subscriptions:   []ChannelSubscription{{Name: "engineering"}},
					ChannelSettings: ChannelSettings{MessageRetentionDays: RetentionDays("unlimited")},
				}
				_, err := c.Subscribe(req)
				return err
			},
			want: url.Values{
				"subscriptions":          {`[{"name":"engineering"}]`},
				"message_retention_days": {`"unlimited"`},
			},
		},
		{
			name: "update channel: who may post is sent as a group-setting update",
			do: func(c *Client) error {
				_, err := c.UpdateStream(UpdateStreamRequest{
					StreamID: 1,
					ChannelPermissions: ChannelPermissions{
						CanSendMessageGroup: types.NamedGroup(15),
					},
				})
				return err
			},
			want: url.Values{"can_send_message_group": {`{"new":15}`}},
		},
		{
			name: "update channel: every permission travels as a group-setting update",
			do: func(c *Client) error {
				_, err := c.UpdateStream(UpdateStreamRequest{
					StreamID: 1,
					ChannelPermissions: ChannelPermissions{
						CanAdministerChannelGroup: types.NamedGroup(15),
						CanResolveTopicsGroup:     types.AnonymousGroup([]int{10}, []int{11}),
					},
				})
				return err
			},
			want: url.Values{
				"can_administer_channel_group": {`{"new":15}`},
				"can_resolve_topics_group": {
					`{"new":{"direct_members":[10],"direct_subgroups":[11]}}`},
			},
		},
		{
			name: "subscribe: the topics policy travels as JSON",
			do: func(c *Client) error {
				policy := "disable_empty_topic"
				req := SubscribeRequest{
					Subscriptions:   []ChannelSubscription{{Name: "engineering"}},
					ChannelSettings: ChannelSettings{TopicsPolicy: &policy},
				}
				_, err := c.Subscribe(req)
				return err
			},
			want: url.Values{
				"subscriptions": {`[{"name":"engineering"}]`},
				"topics_policy": {`"disable_empty_topic"`},
			},
		},
		{
			name: "remove default channel: the channel travels as a query parameter",
			do: func(c *Client) error {
				_, err := c.RemoveDefaultStream(7)
				return err
			},
			want: url.Values{"stream_id": {"7"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := testServer(t, `{"result":"success"}`)

			if err := tc.do(c); err != nil {
				t.Fatalf("request failed: %v", err)
			}

			_, _, got, _ := rec.snapshot()
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parameters = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRequestMethodsAndPaths(t *testing.T) {
	cases := []struct {
		name       string
		do         func(*Client) error
		wantMethod string
		wantPath   string
	}{
		{
			name:       "list channels",
			do:         func(c *Client) error { _, err := c.GetStreams(GetStreamsRequest{}); return err },
			wantMethod: "GET",
			wantPath:   "/api/v1/streams",
		},
		{
			name:       "update channel",
			do:         func(c *Client) error { _, err := c.UpdateStream(UpdateStreamRequest{StreamID: 42}); return err },
			wantMethod: "PATCH",
			wantPath:   "/api/v1/streams/42",
		},
		{
			name:       "delete channel",
			do:         func(c *Client) error { _, err := c.DeleteStream(42); return err },
			wantMethod: "DELETE",
			wantPath:   "/api/v1/streams/42",
		},
		{
			name: "send message",
			do: func(c *Client) error {
				_, err := c.SendMessage(SendMessageRequest{Type: "stream", To: "general", Topic: "hi", Content: "yo"})
				return err
			},
			wantMethod: "POST",
			wantPath:   "/api/v1/messages",
		},
		{
			name:       "channel id lookup",
			do:         func(c *Client) error { _, err := c.GetStreamID("general"); return err },
			wantMethod: "GET",
			wantPath:   "/api/v1/get_stream_id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := testServer(t, `{"result":"success"}`)

			if err := tc.do(c); err != nil {
				t.Fatalf("request failed: %v", err)
			}

			method, path, _, _ := rec.snapshot()
			if method != tc.wantMethod || path != tc.wantPath {
				t.Errorf("%s %s, want %s %s", method, path, tc.wantMethod, tc.wantPath)
			}
		})
	}
}
