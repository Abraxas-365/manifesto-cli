package scaffold

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/Abraxas-365/manifesto-cli/internal/config"
	"github.com/Abraxas-365/manifesto-cli/internal/remote"
	"github.com/Abraxas-365/manifesto-cli/internal/templates"
	"github.com/Abraxas-365/manifesto-cli/internal/ui"
)

const ManifestoGoModule = "github.com/Abraxas-365/manifesto"

type InitOptions struct {
	ProjectName string
	GoModule    string
	OutputDir   string
	Modules     []string
	Ref         string
}

func InitProject(opts InitOptions) error {
	projectRoot := filepath.Join(opts.OutputDir, opts.ProjectName)

	if _, err := os.Stat(projectRoot); !os.IsNotExist(err) {
		return fmt.Errorf("directory %s already exists", projectRoot)
	}
	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}

	// Resolve all module dependencies.
	allModules := config.ResolveDeps(opts.Modules)

	// Collect paths to fetch.
	var allPaths []string
	for _, modName := range allModules {
		mod, ok := config.ModuleRegistry[modName]
		if !ok {
			return fmt.Errorf("unknown module: %s", modName)
		}
		allPaths = append(allPaths, mod.Paths...)
	}

	// Determine ref.
	client := remote.NewClient("")
	ref := opts.Ref
	if ref == "" {
		var err error
		ref, err = client.GetLatestVersion()
		if err != nil {
			ref = remote.DefaultRef
		}
	}

	// Step 1: Fetch modules.
	spin := ui.NewSpinner(fmt.Sprintf("Downloading manifesto@%s...", ref))
	spin.Start()
	err := client.FetchModulePaths(ref, allPaths, projectRoot, ManifestoGoModule, opts.GoModule)
	if err != nil {
		spin.Stop(false)
		// Cleanup on failure.
		os.RemoveAll(projectRoot)
		return fmt.Errorf("fetch modules: %w", err)
	}
	spin.Stop(true)

	// Step 2: Generate go.mod.
	spin = ui.NewSpinner("Creating go.mod...")
	spin.Start()
	err = generateGoMod(projectRoot, opts.GoModule, client, ref)
	if err != nil {
		spin.Stop(false)
		return fmt.Errorf("generate go.mod: %w", err)
	}
	spin.Stop(true)

	// Step 3: Generate project files.
	spin = ui.NewSpinner("Generating project files...")
	spin.Start()

	if err := generateEnvExample(projectRoot, opts.ProjectName); err != nil {
		spin.Stop(false)
		return fmt.Errorf("generate .env.example: %w", err)
	}
	if err := generateGitignore(projectRoot); err != nil {
		spin.Stop(false)
		return fmt.Errorf("generate .gitignore: %w", err)
	}

	spin.Stop(true)

	// Step 4: Write manifesto.yaml.
	spin = ui.NewSpinner("Writing manifesto.yaml...")
	spin.Start()

	manifest := config.NewManifest(opts.ProjectName, opts.GoModule, ref)
	for _, modName := range allModules {
		manifest.Modules[modName] = config.ModuleConfig{
			Version:     ref,
			InstalledAt: time.Now(),
		}
	}
	if err := manifest.Save(projectRoot); err != nil {
		spin.Stop(false)
		return fmt.Errorf("save manifesto.yaml: %w", err)
	}
	spin.Stop(true)

	return nil
}

func generateGoMod(projectRoot, goModule string, client *remote.Client, ref string) error {
	upstreamMod, err := client.FetchGoMod(ref)
	if err != nil {
		// Fallback: minimal go.mod.
		content := fmt.Sprintf("module %s\n\ngo 1.23\n", goModule)
		return os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte(content), 0644)
	}

	// Rewrite the module line, keep everything else.
	var buf bytes.Buffer
	for _, line := range strings.Split(upstreamMod, "\n") {
		if strings.HasPrefix(line, "module ") {
			buf.WriteString("module " + goModule + "\n")
		} else {
			buf.WriteString(line + "\n")
		}
	}

	return os.WriteFile(filepath.Join(projectRoot, "go.mod"), buf.Bytes(), 0644)
}

func generateEnvExample(projectRoot, projectName string) error {
	content, err := templates.FS.ReadFile("project/env.example.tmpl")
	if err != nil {
		return fmt.Errorf("read env template: %w", err)
	}

	tmpl, err := template.New("env").Parse(string(content))
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{"ProjectName": projectName}); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(projectRoot, ".env.example"), buf.Bytes(), 0644)
}

func generateGitignore(projectRoot string) error {
	content := `.env
*.exe
*.dll
*.so
*.dylib
*.test
*.out
vendor/
tmp/
.idea/
.vscode/
`
	return os.WriteFile(filepath.Join(projectRoot, ".gitignore"), []byte(content), 0644)
}
