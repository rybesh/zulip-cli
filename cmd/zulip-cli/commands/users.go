package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rybesh/zulip-cli/client"
	"github.com/spf13/cobra"
)

var listUsersCmd = &cobra.Command{
	Use:   "list-users",
	Short: "List all users",
	RunE: func(cmd *cobra.Command, args []string) error {
		req := client.GetUsersRequest{
			IncludeCustomProfileFields: boolFlag(cmd, "include-custom-profile-fields"),
		}

		resp, err := zulipClient.GetUsers(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var getUserCmd = &cobra.Command{
	Use:   "get-user [user-id]",
	Short: "Get a user by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid user ID: %w", err)
		}

		resp, err := zulipClient.GetUser(userID, boolFlag(cmd, "include-custom-profile-fields"))
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var getProfileCmd = &cobra.Command{
	Use:   "get-profile",
	Short: "Get current user's profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.GetProfile()
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var createUserCmd = &cobra.Command{
	Use:   "create-user [email] [full-name]",
	Short: "Create a new user",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		password, _ := cmd.Flags().GetString("password")

		req := client.CreateUserRequest{
			Email:    args[0],
			FullName: args[1],
			Password: password,
		}

		resp, err := zulipClient.CreateUser(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var updateUserCmd = &cobra.Command{
	Use:   "update-user [user-id]",
	Short: "Update a user",
	Long: `Update a user's name, role, or custom profile fields.

--profile-data takes field-id=value and may be repeated. The field IDs are the
ones list-profile-fields reports:

  zulip-cli update-user 7 --profile-data 4=Berlin --profile-data 6="Team lead"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid user ID: %w", err)
		}

		fullName := stringFlag(cmd, "full-name")
		role, _ := cmd.Flags().GetInt("role")

		profileData, err := profileDataFlag(cmd)
		if err != nil {
			return err
		}

		if fullName == nil && !cmd.Flags().Changed("role") && len(profileData) == 0 {
			return fmt.Errorf("either --full-name, --role or --profile-data is required")
		}

		req := client.UpdateUserRequest{
			UserID:      userID,
			FullName:    fullName,
			Role:        role,
			ProfileData: profileData,
		}

		resp, err := zulipClient.UpdateUser(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

// profileDataFlag reads the custom profile fields to set, written as
// field-id=value. The server wants a list of {id, value} pairs, and takes the
// field by ID rather than by name.
func profileDataFlag(cmd *cobra.Command) ([]map[string]interface{}, error) {
	values, _ := cmd.Flags().GetStringArray("profile-data")

	data := make([]map[string]interface{}, 0, len(values))
	for _, value := range values {
		field, fieldValue, found := strings.Cut(value, "=")
		id, err := strconv.Atoi(field)
		if !found || err != nil {
			return nil, fmt.Errorf(
				"invalid --profile-data %q: it must be field-id=value, with an ID from list-profile-fields", value)
		}
		data = append(data, map[string]interface{}{"id": id, "value": fieldValue})
	}

	return data, nil
}

var deactivateUserCmd = &cobra.Command{
	Use:   "deactivate-user [user-id]",
	Short: "Deactivate a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid user ID: %w", err)
		}

		resp, err := zulipClient.DeactivateUser(userID)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var reactivateUserCmd = &cobra.Command{
	Use:   "reactivate-user [user-id]",
	Short: "Reactivate a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid user ID: %w", err)
		}

		resp, err := zulipClient.ReactivateUser(userID)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var getUserPresenceCmd = &cobra.Command{
	Use:   "get-user-presence [user-id-or-email]",
	Short: "Get user presence",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Try to parse as int first, otherwise treat as email
		var userIDOrEmail interface{}
		if id, err := strconv.Atoi(args[0]); err == nil {
			userIDOrEmail = id
		} else {
			userIDOrEmail = args[0]
		}

		resp, err := zulipClient.GetUserPresence(userIDOrEmail)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var updatePresenceCmd = &cobra.Command{
	Use:   "update-presence [status]",
	Short: "Update presence status",
	Args:  cobra.ExactArgs(1),
	Long:  `Update presence status. Status must be "active" or "idle"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		status := args[0]
		if status != "active" && status != "idle" {
			return fmt.Errorf("status must be 'active' or 'idle'")
		}

		req := client.UpdatePresenceRequest{
			Status: status,
		}

		resp, err := zulipClient.UpdatePresence(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var getRealmPresenceCmd = &cobra.Command{
	Use:     "list-presence",
	Aliases: []string{"get-realm-presence"},
	Short:   "Get presence for everyone in the organization",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.GetRealmPresence()
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var setTypingStatusCmd = &cobra.Command{
	Use:   "set-typing-status [start|stop]",
	Short: "Tell others you are typing",
	Long: `Tell the people you are writing to that you are typing.

This is what makes the typing notification appear in their client. Name the
conversation the way send-message does: --to for a direct message, or
--channel and --topic for a channel message.

  zulip-cli set-typing-status start --to dana@example.com
  zulip-cli set-typing-status stop --channel general --topic standup

Zulip expects these while the message is being written, so a notification is
short-lived: clients stop showing one that is not renewed within a few
seconds.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		op := strings.ToLower(args[0])
		if op != "start" && op != "stop" {
			return fmt.Errorf("unknown operation %q: it must be start or stop", args[0])
		}

		channel, _ := cmd.Flags().GetString("channel")
		topic, _ := cmd.Flags().GetString("topic")
		to, _ := cmd.Flags().GetStringSlice("to")

		req := client.SetTypingStatusRequest{Op: op}

		switch {
		case channel != "" && len(to) > 0:
			return fmt.Errorf("--channel and --to name different conversations, so only one of them can be given")
		case channel != "":
			if topic == "" {
				return fmt.Errorf("--topic is required for channel messages")
			}
			streamID, err := resolveChannelID(zulipClient, channel)
			if err != nil {
				return err
			}
			// "stream" is the wire value; the server has not renamed it.
			req.Type, req.StreamID, req.Topic = "stream", streamID, topic
		case len(to) > 0:
			userIDs, err := resolveUserIDs(zulipClient, to)
			if err != nil {
				return err
			}
			req.Type, req.To = "private", userIDs
		default:
			return fmt.Errorf("either --channel or --to is required")
		}

		resp, err := zulipClient.SetTypingStatus(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

// notificationSettingFlags are the account-wide notification settings, each
// naming its flag and the request field it fills.
var notificationSettingFlags = []struct {
	name  string
	help  string
	field func(*client.UpdateNotificationSettingsRequest) **bool
}{
	{"channel-push", "Mobile push notifications for channel messages",
		func(r *client.UpdateNotificationSettingsRequest) **bool { return &r.EnableStreamPushNotifications }},
	{"channel-email", "Email notifications for channel messages",
		func(r *client.UpdateNotificationSettingsRequest) **bool { return &r.EnableStreamEmailNotifications }},
	{"channel-desktop", "Desktop notifications for channel messages",
		func(r *client.UpdateNotificationSettingsRequest) **bool { return &r.EnableStreamDesktopNotifications }},
	{"channel-audible", "A sound for channel messages",
		func(r *client.UpdateNotificationSettingsRequest) **bool { return &r.EnableStreamAudibleNotifications }},
	{"offline-push", "Mobile push notifications for direct messages and mentions while you are away",
		func(r *client.UpdateNotificationSettingsRequest) **bool { return &r.EnableOfflinePushNotifications }},
	{"offline-email", "Email notifications for direct messages and mentions while you are away",
		func(r *client.UpdateNotificationSettingsRequest) **bool { return &r.EnableOfflineEmailNotifications }},
	{"online-push", "Mobile push notifications even while you are active",
		func(r *client.UpdateNotificationSettingsRequest) **bool { return &r.EnableOnlinePushNotifications }},
	{"digest-emails", "The periodic digest email",
		func(r *client.UpdateNotificationSettingsRequest) **bool { return &r.EnableDigestEmails }},
}

var updateNotificationSettingsCmd = &cobra.Command{
	Use:   "update-notification-settings",
	Short: "Change your account-wide notification settings",
	Long: `Change the notification settings for your whole account.

The settings for a single channel are update-subscription instead.

Each flag is tri-state: leaving it out changes nothing, and --flag=false turns
that notification off.

  zulip-cli update-notification-settings --channel-desktop=false --offline-push`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var req client.UpdateNotificationSettingsRequest

		changed := false
		var names []string
		for _, setting := range notificationSettingFlags {
			names = append(names, setting.name)
			value := boolFlag(cmd, setting.name)
			if value == nil {
				continue
			}
			*setting.field(&req) = value
			changed = true
		}

		if !changed {
			return fmt.Errorf("nothing to change: pass --%s", strings.Join(names, ", --"))
		}

		resp, err := zulipClient.UpdateNotificationSettings(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

func init() {
	listUsersCmd.Flags().Bool("include-custom-profile-fields", false, "Include custom profile fields")
	getUserCmd.Flags().Bool("include-custom-profile-fields", false, "Include custom profile fields")
	createUserCmd.Flags().String("password", "", "User password")
	updateUserCmd.Flags().String("full-name", "", "New full name")
	updateUserCmd.Flags().Int("role", 0, "New role")
	updateUserCmd.Flags().StringArray("profile-data", nil,
		"Custom profile field as field-id=value, repeatable")

	setTypingStatusCmd.Flags().StringP("channel", "s", "", "Channel name (formerly stream)")
	setTypingStatusCmd.Flags().StringP("topic", "t", "", "Topic name (required for channel messages)")
	setTypingStatusCmd.Flags().StringSliceP("to", "u", nil,
		"Recipients of the direct message being written (comma-separated)")

	for _, setting := range notificationSettingFlags {
		updateNotificationSettingsCmd.Flags().Bool(setting.name, false, setting.help)
	}
}
