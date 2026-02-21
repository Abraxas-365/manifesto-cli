package cli

import (
	"bufio"
	"fmt"
	"os"
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
)

var initCmd = &cobra.Command{
	Use:   "init <project-name>",
	Short: "Create a new Manifesto app",
	Long: `Create a new Go project with the Manifesto architecture.

Core modules (always included):
  kernel, errx, logx, ptrx, config, server, migrations

Optional modules (select during init or install later):
  iam   Identity & Access Management
  fsx   File system abstraction (S3, local)
  ai    AI toolkit (LLM, embeddings, vector store, OCR)

Examples:
  manifesto init myapp --module github.com/me/myapp
  manifesto init myapp --module github.com/me/myapp --with iam,ai
  manifesto init myapp --module github.com/me/myapp --all`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initGoModule, "module", "", "Go module path (e.g. github.com/user/project)")
	initCmd.Flags().StringSliceVar(&initModules, "with", nil, "Optional modules (comma-separated: iam,ai,fsx)")
	initCmd.Flags().StringVar(&initRef, "ref", "", "Manifesto version (tag or branch, default: latest)")
	initCmd.Flags().BoolVar(&initAll, "all", false, "Include all optional modules")
	_ = initCmd.MarkFlagRequired("module")
}

func runInit(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	// --- CRA-style banner ---
	ui.PrintBanner()
	ui.PrintCreateHeader(projectName, initGoModule)

	// Build module list.
	selected := config.CoreModules()

	if initAll {
		selected = append(selected, config.OptionalModules()...)
	} else if len(initModules) > 0 {
		for _, m := range initModules {
			m = strings.TrimSpace(m)
			if _, ok := config.ModuleRegistry[m]; !ok {
				return fmt.Errorf("unknown module: '%s'. Run 'manifesto modules' to see available", m)
			}
			selected = append(selected, m)
		}
	} else {
		// Interactive selection.
		optional := config.OptionalModules()
		if len(optional) > 0 {
			fmt.Println("  Which optional modules would you like to include?")
			fmt.Println()
			for _, name := range optional {
				mod := config.ModuleRegistry[name]
				fmt.Printf("    %s  %-6s  %s\n", ui.Cyan.Sprint("▸"), ui.Bold.Sprint(name), ui.Dim.Sprint(mod.Description))
			}
			fmt.Println()
			fmt.Print("  Enter modules (comma-separated), 'all', or press Enter to skip: ")

			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "all" {
				selected = append(selected, optional...)
			} else if input != "" {
				for _, m := range strings.Split(input, ",") {
					m = strings.TrimSpace(m)
					if m == "" {
						continue
					}
					if _, ok := config.ModuleRegistry[m]; !ok {
						return fmt.Errorf("unknown module: '%s'", m)
					}
					selected = append(selected, m)
				}
			}
			fmt.Println()
		}
	}

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
	fmt.Printf("  Installing %s modules:\n\n", ui.Bold.Sprintf("%d", len(resolved)))
	for _, name := range resolved {
		mod := config.ModuleRegistry[name]
		tag := ""
		if mod.Core {
			tag = ui.Dim.Sprint(" core")
		}
		fmt.Printf("    %s %s%s\n", ui.Green.Sprint("+"), name, tag)
	}
	fmt.Println()

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
		Ref:         initRef,
	}); err != nil {
		return err
	}

	ui.PrintSuccess(projectName)
	return nil
}
