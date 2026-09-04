package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Bot storage is a small key-value store the server keeps for a bot, so that a
// bot can remember something between the runs that a CLI invocation implies.
// It belongs to whichever bot the credentials in the environment name, and a
// human account has none.

var getStorageCmd = &cobra.Command{
	Use:   "get-storage [keys...]",
	Short: "Read this bot's stored state",
	Long: `Read the key-value state the server keeps for this bot.

With no keys this returns everything stored. The storage belongs to the bot
whose credentials are in the environment, so a human account has none.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.GetStorage(args)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var updateStorageCmd = &cobra.Command{
	Use:   "update-storage [key=value...]",
	Short: "Write this bot's stored state",
	Long: `Write key-value state the server keeps for this bot.

Keys not named are left as they are.

  zulip-cli update-storage last-seen=1717171717 greeting="hello there"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		storage := make(map[string]string, len(args))
		for _, arg := range args {
			key, value, found := strings.Cut(arg, "=")
			if !found || key == "" {
				return fmt.Errorf("invalid entry %q: it must be key=value", arg)
			}
			storage[key] = value
		}

		resp, err := zulipClient.UpdateStorage(storage)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}
