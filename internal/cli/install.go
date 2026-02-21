package cli

import (
	"fmt"

	"github.com/Abraxas-365/manifesto-cli/internal/scaffold"
	"github.com/spf13/cobra"
)

var installRef string

var installCmd = &cobra.Command{
	Use:   "install <module>",
	Short: "Install a module into an existing project",
	Long: `Install a manifesto module and its dependencies.

Examples:
  manifesto install ai
  manifesto install iam
  manifesto install fsx --ref v1.2.0`,
	Args: cobra.ExactArgs(1),
	RunE: runInstall,
}

func init() {
	installCmd.Flags().StringVar(&installRef, "ref", "", "Manifesto version (default: project version)")
}

func runInstall(cmd *cobra.Command, args []string) error {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("find project root: %w", err)
	}

	return scaffold.InstallModule(scaffold.InstallOptions{
		ProjectRoot: projectRoot,
		ModuleName:  args[0],
		Ref:         installRef,
	})
}
