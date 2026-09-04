package commands

import (
	"fmt"
	"strconv"

	"github.com/rybesh/zulip-cli/client"
	"github.com/spf13/cobra"
)

var listLinkifiersCmd = &cobra.Command{
	Use:     "list-linkifiers",
	Aliases: []string{"list-realm-filters"},
	Short:   "List the organization's linkifiers",
	Long: `List the organization's linkifiers.

A linkifier turns a pattern in a message — an issue number, a ticket ID — into
a link.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.GetRealmLinkifiers()
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var addLinkifierCmd = &cobra.Command{
	Use:     "add-linkifier [pattern] [url-template]",
	Aliases: []string{"add-realm-filter"},
	Short:   "Add a linkifier",
	Long: `Add a linkifier.

The pattern is a regular expression with named groups, and the URL template
fills those groups in:

  zulip-cli add-linkifier '#(?P<id>[0-9]+)' 'https://example.com/issues/{id}'

Quote both, or the shell will read the # as a comment and the braces as its
own.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.AddRealmFilter(client.AddRealmFilterRequest{
			Pattern:     args[0],
			URLTemplate: args[1],
		})
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var removeLinkifierCmd = &cobra.Command{
	Use:     "remove-linkifier [linkifier-id]",
	Aliases: []string{"remove-realm-filter"},
	Short:   "Remove a linkifier",
	Long:    `Remove a linkifier by ID, as reported by list-linkifiers.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filterID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid linkifier ID: %w", err)
		}

		resp, err := zulipClient.RemoveRealmFilter(filterID)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var listProfileFieldsCmd = &cobra.Command{
	Use:     "list-profile-fields",
	Aliases: []string{"list-realm-profile-fields"},
	Short:   "List the organization's custom profile fields",
	Long: `List the organization's custom profile fields.

The IDs reported here are the ones update-user --profile-data sets.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.GetRealmProfileFields()
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

// profileFieldTypeHelp lists the field types the server documents. It is help
// text rather than validation, since the set grows with the server and a new
// type should not need a new release of this CLI to be reachable.
const profileFieldTypeHelp = `Kind of field: 1 short text, 2 long text, 3 list of options, ` +
	`4 date, 5 link, 6 person, 7 external account, 8 pronouns`

var createProfileFieldCmd = &cobra.Command{
	Use:     "create-profile-field [name]",
	Aliases: []string{"create-realm-profile-field"},
	Short:   "Create a custom profile field",
	Long: `Create a custom profile field.

Fields that offer a choice, and external account fields, are configured through
--field-data, which takes the JSON the server documents for that field type:

  zulip-cli create-profile-field Location --field-type 1 --hint "Where you work"
  zulip-cli create-profile-field Team --field-type 3 \
    --field-data '{"0":{"text":"Backend","order":"1"},"1":{"text":"Web","order":"2"}}'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fieldType, _ := cmd.Flags().GetInt("field-type")
		if fieldType == 0 {
			return fmt.Errorf("--field-type is required. %s", profileFieldTypeHelp)
		}

		hint, _ := cmd.Flags().GetString("hint")
		fieldData, _ := cmd.Flags().GetString("field-data")

		resp, err := zulipClient.CreateRealmProfileField(client.CreateRealmProfileFieldRequest{
			Name:      args[0],
			Hint:      hint,
			FieldType: fieldType,
			FieldData: fieldData,
		})
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var updateProfileFieldCmd = &cobra.Command{
	Use:     "update-profile-field [field-id]",
	Aliases: []string{"update-realm-profile-field"},
	Short:   "Update a custom profile field",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fieldID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid profile field ID: %w", err)
		}

		name, _ := cmd.Flags().GetString("name")
		hint, _ := cmd.Flags().GetString("hint")
		fieldData, _ := cmd.Flags().GetString("field-data")

		if name == "" && hint == "" && fieldData == "" {
			return fmt.Errorf("nothing to change: pass --name, --hint, or --field-data")
		}

		resp, err := zulipClient.UpdateRealmProfileField(client.UpdateRealmProfileFieldRequest{
			FieldID:   fieldID,
			Name:      name,
			Hint:      hint,
			FieldData: fieldData,
		})
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var deleteProfileFieldCmd = &cobra.Command{
	Use:     "delete-profile-field [field-id]",
	Aliases: []string{"delete-realm-profile-field"},
	Short:   "Delete a custom profile field",
	Long: `Delete a custom profile field.

Whatever every user had filled in for it goes with it.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fieldID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid profile field ID: %w", err)
		}

		resp, err := zulipClient.DeleteRealmProfileField(fieldID)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var reorderProfileFieldsCmd = &cobra.Command{
	Use:     "reorder-profile-fields [field-ids...]",
	Aliases: []string{"reorder-realm-profile-fields"},
	Short:   "Reorder the custom profile fields",
	Long: `Set the order the custom profile fields appear in.

List every field ID, in the order wanted:

  zulip-cli reorder-profile-fields 6 4 5`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		order := make([]int, 0, len(args))
		for _, arg := range args {
			fieldID, err := strconv.Atoi(arg)
			if err != nil {
				return fmt.Errorf("invalid profile field ID %q: %w", arg, err)
			}
			order = append(order, fieldID)
		}

		resp, err := zulipClient.ReorderRealmProfileFields(order)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

func init() {
	createProfileFieldCmd.Flags().Int("field-type", 0, profileFieldTypeHelp)
	createProfileFieldCmd.Flags().String("hint", "", "Short explanation shown under the field")
	createProfileFieldCmd.Flags().String("field-data", "",
		"JSON configuring the field, for the types that need it")

	updateProfileFieldCmd.Flags().String("name", "", "New name")
	updateProfileFieldCmd.Flags().String("hint", "", "New hint")
	updateProfileFieldCmd.Flags().String("field-data", "", "New JSON configuration")
}
