package scaffold

import (
	"fmt"
	"time"

	"github.com/Abraxas-365/manifesto-cli/internal/config"
	"github.com/Abraxas-365/manifesto-cli/internal/remote"
	"github.com/Abraxas-365/manifesto-cli/internal/ui"
)

type InstallOptions struct {
	ProjectRoot string
	ModuleName  string
	Ref         string
}

func InstallModule(opts InstallOptions) error {
	manifest, err := config.LoadManifest(opts.ProjectRoot)
	if err != nil {
		return fmt.Errorf("not a manifesto project: %w", err)
	}

	if mc, ok := manifest.Modules[opts.ModuleName]; ok {
		return fmt.Errorf("module '%s' already installed (version: %s)", opts.ModuleName, mc.Version)
	}

	if _, ok := config.ModuleRegistry[opts.ModuleName]; !ok {
		return fmt.Errorf("unknown module: '%s'. Run 'manifesto modules' to see available modules", opts.ModuleName)
	}

	// Resolve deps, find what's missing.
	allNeeded := config.ResolveDeps([]string{opts.ModuleName})
	var toInstall []string
	for _, name := range allNeeded {
		if _, ok := manifest.Modules[name]; !ok {
			toInstall = append(toInstall, name)
		}
	}

	// Collect paths.
	var allPaths []string
	for _, name := range toInstall {
		allPaths = append(allPaths, config.ModuleRegistry[name].Paths...)
	}

	// Determine ref.
	ref := opts.Ref
	if ref == "" {
		ref = manifest.Project.Version
	}
	if ref == "" {
		client := remote.NewClient("")
		ref, _ = client.GetLatestVersion()
		if ref == "" {
			ref = remote.DefaultRef
		}
	}

	// Fetch.
	spin := ui.NewSpinner(fmt.Sprintf("Installing %s from manifesto@%s...", opts.ModuleName, ref))
	spin.Start()

	client := remote.NewClient("")
	if err := client.FetchModulePaths(ref, allPaths, opts.ProjectRoot, ManifestoGoModule, manifest.Project.GoModule); err != nil {
		spin.Stop(false)
		return fmt.Errorf("fetch module: %w", err)
	}
	spin.Stop(true)

	// Update manifest.
	for _, name := range toInstall {
		manifest.Modules[name] = config.ModuleConfig{
			Version:     ref,
			InstalledAt: time.Now(),
		}
	}

	if err := manifest.Save(opts.ProjectRoot); err != nil {
		return fmt.Errorf("save manifesto.yaml: %w", err)
	}

	ui.PrintInstallSuccess(opts.ModuleName, toInstall)
	return nil
}
