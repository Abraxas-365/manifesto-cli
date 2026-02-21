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

// ProjectData is the template context for project-level templates.
type ProjectData struct {
	GoModule    string
	ProjectName string
	HasIAM      bool
	HasFSX      bool
	HasAI       bool
}

func InitProject(opts InitOptions) error {
	projectRoot := filepath.Join(opts.OutputDir, opts.ProjectName)

	if _, err := os.Stat(projectRoot); !os.IsNotExist(err) {
		return fmt.Errorf("directory %s already exists", projectRoot)
	}
	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}

	allModules := config.ResolveDeps(opts.Modules)

	// Collect remote paths to fetch from GitHub.
	var allPaths []string
	for _, modName := range allModules {
		mod, ok := config.ModuleRegistry[modName]
		if !ok {
			return fmt.Errorf("unknown module: %s", modName)
		}
		allPaths = append(allPaths, mod.Paths...)
	}

	client := remote.NewClient("")
	ref := opts.Ref
	if ref == "" {
		var err error
		ref, err = client.GetLatestVersion()
		if err != nil {
			ref = remote.DefaultRef
		}
	}

	// Step 1: Fetch module source from GitHub.
	if len(allPaths) > 0 {
		spin := ui.NewSpinner(fmt.Sprintf("Downloading manifesto@%s...", ref))
		spin.Start()
		err := client.FetchModulePaths(ref, allPaths, projectRoot, ManifestoGoModule, opts.GoModule)
		if err != nil {
			spin.Stop(false)
			os.RemoveAll(projectRoot)
			return fmt.Errorf("fetch modules: %w", err)
		}
		spin.Stop(true)
	}

	// Step 2: Generate go.mod.
	spin := ui.NewSpinner("Creating go.mod...")
	spin.Start()
	if err := generateGoMod(projectRoot, opts.GoModule, client, ref); err != nil {
		spin.Stop(false)
		return fmt.Errorf("generate go.mod: %w", err)
	}
	spin.Stop(true)

	// Step 3: Generate project files from templates.
	spin = ui.NewSpinner("Generating project files...")
	spin.Start()

	projData := ProjectData{
		GoModule:    opts.GoModule,
		ProjectName: opts.ProjectName,
		HasIAM:      config.HasModule(allModules, "iam"),
		HasFSX:      config.HasModule(allModules, "fsx"),
		HasAI:       config.HasModule(allModules, "ai"),
	}

	templateFiles := []struct {
		tmpl string
		dest string
	}{
		{"project/container.go.tmpl", filepath.Join(projectRoot, "cmd", "container.go")},
		{"project/server.go.tmpl", filepath.Join(projectRoot, "cmd", "server.go")},
		{"project/makefile.tmpl", filepath.Join(projectRoot, "Makefile")},
		{"project/docker-compose.yml.tmpl", filepath.Join(projectRoot, "docker-compose.yml")},
	}

	for _, tf := range templateFiles {
		if err := renderProjectTemplate(tf.tmpl, tf.dest, projData); err != nil {
			spin.Stop(false)
			return fmt.Errorf("generate %s: %w", filepath.Base(tf.dest), err)
		}
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

func renderProjectTemplate(tmplPath, destPath string, data any) error {
	content, err := templates.FS.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("read template %s: %w", tmplPath, err)
	}

	tmpl, err := template.New(filepath.Base(tmplPath)).Parse(string(content))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(destPath, buf.Bytes(), 0644)
}

func generateGoMod(projectRoot, goModule string, client *remote.Client, ref string) error {
	upstreamMod, err := client.FetchGoMod(ref)
	if err != nil {
		content := fmt.Sprintf("module %s\n\ngo 1.23\n", goModule)
		return os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte(content), 0644)
	}

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

func generateGitignore(projectRoot string) error {
	content := `.env
*.exe
*.dll
*.so
*.dylib
*.test
*.out
bin/
vendor/
tmp/
.idea/
.vscode/
coverage.out
coverage.html
uploads/
backups/
`
	return os.WriteFile(filepath.Join(projectRoot, ".gitignore"), []byte(content), 0644)
}
