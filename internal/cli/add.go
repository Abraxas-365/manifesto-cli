package cli

import (
	"fmt"
	"strings"

	"github.com/Abraxas-365/manifesto-cli/internal/config"
	"github.com/Abraxas-365/manifesto-cli/internal/remote"
	"github.com/Abraxas-365/manifesto-cli/internal/scaffold"
	"github.com/Abraxas-365/manifesto-cli/internal/ui"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <module-or-domain-path>",
	Short: "Wire a module or scaffold a DDD domain package",
	Long: `Add a module to the project or scaffold a full domain package.

Module wiring (downloads source + injects into container/server):
  manifesto add fsx
  manifesto add asyncx
  manifesto add ai
  manifesto add jobx
  manifesto add notifx
  manifesto add iam

Domain scaffolding (creates entity, repo, service, handler layers):
  manifesto add pkg/recruitment/candidate
  manifesto add pkg/billing/invoice`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func runAdd(cmd *cobra.Command, args []string) error {
	arg := args[0]

	projectRoot, err := findProjectRoot()
	if err != nil {
		return err
	}

	manifest, err := config.LoadManifest(projectRoot)
	if err != nil {
		return fmt.Errorf("not a manifesto project (no manifesto.yaml found)")
	}

	// Dispatch: wireable module vs domain path
	if config.IsWireableModule(arg) {
		return runWireModule(projectRoot, manifest, arg)
	}

	// Domain scaffolding (existing behavior) — paths contain /
	if !strings.Contains(arg, "/") {
		return fmt.Errorf("unknown module '%s'. Use a module (fsx, asyncx, ai, jobx, notifx, iam) or a domain path (pkg/mymodule/entity)", arg)
	}

	return runAddDomain(projectRoot, manifest, arg)
}

func runWireModule(projectRoot string, manifest *config.Manifest, moduleName string) error {
	// Check not already wired
	if manifest.IsWired(moduleName) {
		ui.StepInfo(fmt.Sprintf("%s is already wired", moduleName))
		return nil
	}

	spec := config.WireableModuleRegistry[moduleName]

	fmt.Println()

	// Download required source modules if not already present.
	if len(spec.RequiredModules) > 0 {
		spin := ui.NewSpinner(fmt.Sprintf("Downloading %s...", moduleName))
		spin.Start()

		client := remote.NewClient("")
		ref := manifest.Project.Version
		if ref == "" {
			var err error
			ref, err = client.GetLatestVersion()
			if err != nil || ref == "" {
				ref = remote.DefaultRef
			}
		}

		if err := scaffold.EnsureModulesPresent(projectRoot, manifest, spec.RequiredModules, client, ref); err != nil {
			spin.Stop(false)
			return fmt.Errorf("download %s: %w", moduleName, err)
		}
		spin.Stop(true)
	}

	spin := ui.NewSpinner(fmt.Sprintf("Wiring %s...", moduleName))
	spin.Start()

	modified, err := scaffold.WireModule(scaffold.WireOptions{
		ProjectRoot:  projectRoot,
		ModuleName:   moduleName,
		GoModule:     manifest.Project.GoModule,
		ProjectName:  manifest.Project.Name,
		WiredModules: manifest.WiredModules,
	})
	if err != nil {
		spin.Stop(false)
		return err
	}
	spin.Stop(true)

	// Update manifest
	manifest.WiredModules = append(manifest.WiredModules, moduleName)
	if err := manifest.Save(projectRoot); err != nil {
		return fmt.Errorf("save manifesto.yaml: %w", err)
	}

	ui.PrintWireSuccess(moduleName, modified)
	return nil
}

func runAddDomain(projectRoot string, manifest *config.Manifest, domainPath string) error {
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
