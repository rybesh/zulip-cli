package commands

import (
	"fmt"
	"strconv"

	"github.com/intelligrit/zulip-cli/client"
	"github.com/spf13/cobra"
)

var listUsersCmd = &cobra.Command{
	Use:   "list-users",
	Short: "List all users",
	RunE: func(cmd *cobra.Command, args []string) error {
		includeCustomProfileFields, _ := cmd.Flags().GetBool("include-custom-profile-fields")

		req := client.GetUsersRequest{
			IncludeCustomProfileFields: includeCustomProfileFields,
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

		includeCustomProfileFields, _ := cmd.Flags().GetBool("include-custom-profile-fields")

		resp, err := zulipClient.GetUser(userID, includeCustomProfileFields)
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
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid user ID: %w", err)
		}

		fullName, _ := cmd.Flags().GetString("full-name")
		role, _ := cmd.Flags().GetInt("role")

		req := client.UpdateUserRequest{
			UserID:   userID,
			FullName: fullName,
			Role:     role,
		}

		resp, err := zulipClient.UpdateUser(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
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

func init() {
	listUsersCmd.Flags().Bool("include-custom-profile-fields", false, "Include custom profile fields")
	getUserCmd.Flags().Bool("include-custom-profile-fields", false, "Include custom profile fields")
	createUserCmd.Flags().String("password", "", "User password")
	updateUserCmd.Flags().String("full-name", "", "New full name")
	updateUserCmd.Flags().Int("role", 0, "New role")
}
