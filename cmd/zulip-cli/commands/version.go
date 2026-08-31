package commands

import (
	"github.com/rybesh/zulip-cli/client"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long: `Print the version of this build, along with the revision it was built
from and the toolchain that built it.

The version is read from the build information Go embeds in the binary. A
binary installed with "go install ...@version" reports that module version; a
local "go build" reports "devel" plus the commit it was built from.

Output follows the --output flag, so the version is available to scripts:

  zulip-cli version | jq -r .version

The --version flag prints the same version as a single line.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return printResult(client.CurrentBuild())
	},
}
