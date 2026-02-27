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

	// Collect library modules (always present, not wireable)
	var libraryNames []string
	for name := range config.ModuleRegistry {
		if !config.IsWireableModule(name) {
			libraryNames = append(libraryNames, name)
		}
	}
	sort.Strings(libraryNames)

	var libraries []ui.ModuleDisplay
	for _, name := range libraryNames {
		mod := config.ModuleRegistry[name]
		installed := false
		if manifest != nil {
			_, installed = manifest.Modules[name]
		}

		deps := ""
		if len(mod.Deps) > 0 {
			deps = strings.Join(mod.Deps, ", ")
		}

		libraries = append(libraries, ui.ModuleDisplay{
			Name:        name,
			Description: mod.Description,
			Installed:   installed,
			Core:        mod.Core,
			Deps:        deps,
		})
	}

	// Collect wireable modules
	wireableNames := config.WireableModuleNames()
	sort.Strings(wireableNames)

	var wireables []ui.WireableModuleDisplay
	for _, name := range wireableNames {
		spec := config.WireableModuleRegistry[name]
		wired := false
		if manifest != nil {
			wired = manifest.IsWired(name)
		}

		wireables = append(wireables, ui.WireableModuleDisplay{
			Name:        name,
			Description: spec.Description,
			Wired:       wired,
		})
	}

	ui.PrintModulesWithSections(libraries, wireables)
	return nil
}
