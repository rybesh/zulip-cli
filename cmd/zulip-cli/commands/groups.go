package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rybesh/zulip-cli/client"
	"github.com/spf13/cobra"
)

var listUserGroupsCmd = &cobra.Command{
	Use:   "list-user-groups",
	Short: "List all user groups",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.GetUserGroups()
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var createUserGroupCmd = &cobra.Command{
	Use:   "create-user-group [name] [description]",
	Short: "Create a new user group",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		membersStr, _ := cmd.Flags().GetString("members")
		var members []int
		if membersStr != "" {
			for _, idStr := range strings.Split(membersStr, ",") {
				id, err := strconv.Atoi(strings.TrimSpace(idStr))
				if err != nil {
					return fmt.Errorf("invalid member ID: %w", err)
				}
				members = append(members, id)
			}
		}

		req := client.CreateUserGroupRequest{
			Name:        args[0],
			Description: args[1],
			Members:     members,
		}

		resp, err := zulipClient.CreateUserGroup(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var updateUserGroupCmd = &cobra.Command{
	Use:   "update-user-group [group-id]",
	Short: "Update a user group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid group ID: %w", err)
		}

		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")

		req := client.UpdateUserGroupRequest{
			GroupID:     groupID,
			Name:        name,
			Description: description,
		}

		resp, err := zulipClient.UpdateUserGroup(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var deleteUserGroupCmd = &cobra.Command{
	Use:   "delete-user-group [group-id]",
	Short: "Delete a user group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid group ID: %w", err)
		}

		resp, err := zulipClient.DeleteUserGroup(groupID)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var addGroupMembersCmd = &cobra.Command{
	Use:   "add-group-members [group-id] [user-ids...]",
	Short: "Add members to a user group",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid group ID: %w", err)
		}

		var members []int
		for _, idStr := range args[1:] {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return fmt.Errorf("invalid user ID: %w", err)
			}
			members = append(members, id)
		}

		req := client.UpdateUserGroupMembersRequest{
			GroupID: groupID,
			Add:     members,
		}

		resp, err := zulipClient.UpdateUserGroupMembers(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var removeGroupMembersCmd = &cobra.Command{
	Use:   "remove-group-members [group-id] [user-ids...]",
	Short: "Remove members from a user group",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid group ID: %w", err)
		}

		var members []int
		for _, idStr := range args[1:] {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return fmt.Errorf("invalid user ID: %w", err)
			}
			members = append(members, id)
		}

		req := client.UpdateUserGroupMembersRequest{
			GroupID: groupID,
			Delete:  members,
		}

		resp, err := zulipClient.UpdateUserGroupMembers(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

func init() {
	createUserGroupCmd.Flags().String("members", "", "Comma-separated list of user IDs")
	updateUserGroupCmd.Flags().String("name", "", "New name")
	updateUserGroupCmd.Flags().String("description", "", "New description")
}
