package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Abraxas-365/manifesto-cli/internal/config"
	"github.com/spf13/cobra"
)

var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "manifesto",
	Short: "Create production-grade Go apps with DDD architecture",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(modulesCmd)
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print manifesto CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("manifesto v%s\n", Version)
	},
}

// findProjectRoot walks up from cwd looking for manifesto.yaml.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, config.ManifestoFile)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Fallback to cwd.
	return os.Getwd()
}
