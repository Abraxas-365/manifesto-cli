package cli

import (
	"fmt"

	"github.com/Abraxas-365/manifesto-cli/internal/config"
	"github.com/Abraxas-365/manifesto-cli/internal/scaffold"
	"github.com/Abraxas-365/manifesto-cli/internal/ui"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <domain-path>",
	Short: "Add a new DDD domain package",
	Long: `Scaffold a full domain package with entity, repository, service,
infrastructure, and handler layers.

Examples:
  manifesto add pkg/recruitment/candidate
  manifesto add pkg/billing/invoice
  manifesto add pkg/notification`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func runAdd(cmd *cobra.Command, args []string) error {
	domainPath := args[0]

	projectRoot, err := findProjectRoot()
	if err != nil {
		return err
	}

	manifest, err := config.LoadManifest(projectRoot)
	if err != nil {
		return fmt.Errorf("not a manifesto project (no manifesto.yaml found)")
	}

	data := scaffold.NewDomainData(manifest.Project.GoModule, domainPath)

	fmt.Println()
	spin := ui.NewSpinner(fmt.Sprintf("Scaffolding %s...", data.EntityName))
	spin.Start()

	if err := scaffold.GenerateDomain(projectRoot, data); err != nil {
		spin.Stop(false)
		return err
	}
	spin.Stop(true)

	ui.PrintAddSuccess(data.EntityName, domainPath, data.PackageName)
	return nil
}
