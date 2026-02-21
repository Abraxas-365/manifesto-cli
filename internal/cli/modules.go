package cli

import (
	"sort"
	"strings"

	"github.com/Abraxas-365/manifesto-cli/internal/config"
	"github.com/Abraxas-365/manifesto-cli/internal/ui"
	"github.com/spf13/cobra"
)

var modulesCmd = &cobra.Command{
	Use:   "modules",
	Short: "List available modules",
	RunE:  runModules,
}

func runModules(cmd *cobra.Command, args []string) error {
	projectRoot, _ := findProjectRoot()
	manifest, _ := config.LoadManifest(projectRoot)

	var names []string
	for name := range config.ModuleRegistry {
		names = append(names, name)
	}
	sort.Strings(names)

	var modules []ui.ModuleDisplay
	for _, name := range names {
		mod := config.ModuleRegistry[name]
		installed := false
		if manifest != nil {
			_, installed = manifest.Modules[name]
		}

		deps := ""
		if len(mod.Deps) > 0 {
			deps = strings.Join(mod.Deps, ", ")
		}

		modules = append(modules, ui.ModuleDisplay{
			Name:        name,
			Description: mod.Description,
			Installed:   installed,
			Core:        mod.Core,
			Deps:        deps,
		})
	}

	ui.PrintModules(modules)
	return nil
}
