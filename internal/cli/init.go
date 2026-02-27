package cli

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Abraxas-365/manifesto-cli/internal/config"
	"github.com/Abraxas-365/manifesto-cli/internal/scaffold"
	"github.com/Abraxas-365/manifesto-cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	initGoModule string
	initModules  []string
	initRef      string
	initAll      bool
	initQuick    bool
)

var initCmd = &cobra.Command{
	Use:   "init <project-name>",
	Short: "Create a new Manifesto app",
	Long: `Create a new Go project with the Manifesto architecture.

All libraries are included by default (kernel, errx, logx, ptrx,
asyncx, config, fsx, ai, etc).

Wireable modules can be added during init or later with 'manifesto add':
  jobx    Async job processing (Redis-backed dispatcher)
  notifx  Email notifications (AWS SES)
  iam     Identity & Access Management

Use --quick for a lightweight project without IAM or migrations:
  manifesto init myapp --module github.com/me/myapp --quick

Examples:
  manifesto init myapp --module github.com/me/myapp
  manifesto init myapp --module github.com/me/myapp --with jobx,iam
  manifesto init myapp --module github.com/me/myapp --all
  manifesto init myapp --module github.com/me/myapp --quick
  manifesto init myapp --module github.com/me/myapp --quick --with jobx`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initGoModule, "module", "", "Go module path (e.g. github.com/user/project)")
	initCmd.Flags().StringSliceVar(&initModules, "with", nil, "Wireable modules to include (comma-separated: jobx,notifx,iam)")
	initCmd.Flags().StringVar(&initRef, "ref", "", "Manifesto version (tag or branch, default: latest)")
	initCmd.Flags().BoolVar(&initAll, "all", false, "Wire all available modules")
	initCmd.Flags().BoolVar(&initQuick, "quick", false, "Create a lightweight project (no IAM, no migrations)")
	_ = initCmd.MarkFlagRequired("module")
}

func runInit(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	// --- CRA-style banner ---
	ui.PrintBanner()
	if initQuick {
		ui.PrintCreateHeaderQuick(projectName, initGoModule)
	} else {
		ui.PrintCreateHeader(projectName, initGoModule)
	}

	// Build module list (all core modules).
	selected := config.CoreModules(initQuick)

	// Deduplicate.
	seen := make(map[string]bool)
	var deduped []string
	for _, m := range selected {
		if !seen[m] {
			seen[m] = true
			deduped = append(deduped, m)
		}
	}

	// Resolve with deps.
	resolved := config.ResolveDeps(deduped)

	// Show what will be installed.
	fmt.Printf("  Installing %s libraries:\n\n", ui.Bold.Sprintf("%d", len(resolved)))
	for _, name := range resolved {
		fmt.Printf("    %s %s\n", ui.Green.Sprint("+"), name)
	}
	fmt.Println()

	// Determine which modules to wire.
	var wireModules []string

	wireableNames := config.WireableModuleNames()
	sort.Strings(wireableNames)

	// Filter wireable modules based on quick mode (iam not available in quick)
	availableWireable := wireableNames
	if initQuick {
		var filtered []string
		for _, name := range wireableNames {
			if name != "iam" {
				filtered = append(filtered, name)
			}
		}
		availableWireable = filtered
	}

	if initAll {
		wireModules = availableWireable
	} else if len(initModules) > 0 {
		for _, m := range initModules {
			m = strings.TrimSpace(m)
			if !config.IsWireableModule(m) {
				return fmt.Errorf("unknown wireable module: '%s'. Available: %s", m, strings.Join(wireableNames, ", "))
			}
			if initQuick && m == "iam" {
				return fmt.Errorf("module 'iam' is not available for quick projects")
			}
			wireModules = append(wireModules, m)
		}
	} else {
		// Interactive selection.
		if len(availableWireable) > 0 {
			fmt.Println("  Which modules would you like to wire?")
			fmt.Println()
			for _, name := range availableWireable {
				spec := config.WireableModuleRegistry[name]
				fmt.Printf("    %s  %-8s  %s\n", ui.Cyan.Sprint("▸"), ui.Bold.Sprint(name), ui.Dim.Sprint(spec.Description))
			}
			fmt.Println()
			fmt.Print("  Enter modules (comma-separated), 'all', or press Enter to skip: ")

			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "all" {
				wireModules = availableWireable
			} else if input != "" {
				for _, m := range strings.Split(input, ",") {
					m = strings.TrimSpace(m)
					if m == "" {
						continue
					}
					if !config.IsWireableModule(m) {
						return fmt.Errorf("unknown wireable module: '%s'", m)
					}
					if initQuick && m == "iam" {
						return fmt.Errorf("module 'iam' is not available for quick projects")
					}
					wireModules = append(wireModules, m)
				}
			}
			fmt.Println()
		}
	}

	if len(wireModules) > 0 {
		fmt.Printf("  Wiring %s modules:\n\n", ui.Bold.Sprintf("%d", len(wireModules)))
		for _, name := range wireModules {
			fmt.Printf("    %s %s\n", ui.Cyan.Sprint("⚡"), name)
		}
		fmt.Println()
	}

	// For quick projects, default to quick-project branch.
	ref := initRef
	if initQuick && ref == "" {
		ref = config.QuickProjectRef
	}

	// Run scaffold.
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if err := scaffold.InitProject(scaffold.InitOptions{
		ProjectName: projectName,
		GoModule:    initGoModule,
		OutputDir:   cwd,
		Modules:     resolved,
		Ref:         ref,
		WireModules: wireModules,
	}); err != nil {
		return err
	}

	ui.PrintSuccess(projectName)
	return nil
}
