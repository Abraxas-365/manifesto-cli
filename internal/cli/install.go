package cli

import (
	"fmt"

	"github.com/Abraxas-365/manifesto-cli/internal/ui"
	"github.com/spf13/cobra"
)

var installRef string

var installCmd = &cobra.Command{
	Use:        "install <module>",
	Short:      "Deprecated: use 'manifesto add' instead",
	Deprecated: "use 'manifesto add <module>' instead",
	Args:       cobra.ExactArgs(1),
	RunE:       runInstall,
}

func init() {
	installCmd.Flags().StringVar(&installRef, "ref", "", "Manifesto version (default: project version)")
}

func runInstall(cmd *cobra.Command, args []string) error {
	ui.StepWarn("'manifesto install' is deprecated. Use 'manifesto add' instead.")
	fmt.Println()

	// Forward to add command
	addCmd.SetArgs(args)
	return addCmd.Execute()
}
