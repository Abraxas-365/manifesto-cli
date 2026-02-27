package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const ManifestoFile = "manifesto.yaml"

type Manifest struct {
	Project      ProjectConfig           `yaml:"project"`
	Modules      map[string]ModuleConfig `yaml:"modules"`
	WiredModules []string                `yaml:"wired_modules,omitempty"`
	CreatedAt    time.Time               `yaml:"created_at"`
	UpdatedAt    time.Time               `yaml:"updated_at"`
}

type ProjectConfig struct {
	Name     string `yaml:"name"`
	GoModule string `yaml:"go_module"`
	Version  string `yaml:"manifesto_version"`
}

type ModuleConfig struct {
	Version     string    `yaml:"version"`
	InstalledAt time.Time `yaml:"installed_at"`
}

type Module struct {
	Name        string
	Description string
	Paths       []string // Remote paths fetched from GitHub
	Deps        []string
	Core        bool
}

var ModuleRegistry = map[string]Module{
	"kernel": {
		Name: "kernel", Description: "Domain primitives, value objects, pagination, UoW",
		Paths: []string{"pkg/kernel"}, Core: true,
	},
	"errx": {
		Name: "errx", Description: "Structured error handling with HTTP mapping",
		Paths: []string{"pkg/errx"}, Core: true,
	},
	"logx": {
		Name: "logx", Description: "Structured logging (console/JSON)",
		Paths: []string{"pkg/logx"}, Core: true,
	},
	"ptrx": {
		Name: "ptrx", Description: "Pointer utility helpers",
		Paths: []string{"pkg/ptrx"}, Core: true,
	},
	"asyncx": {
		Name: "asyncx", Description: "Async primitives: futures, fan-out, pools, retry, timeout",
		Paths: []string{"pkg/asyncx"}, Core: true,
	},
	"config": {
		Name: "config", Description: "Environment-driven configuration",
		Paths: []string{"pkg/config"}, Core: true,
	},
	"server": {
		Name: "server", Description: "Server, container, Makefile, docker-compose (templated)",
		Paths: []string{}, Core: true,
	},
	"migrations": {
		Name: "migrations", Description: "Database migration scaffolding",
		Paths: []string{"migrations"}, Core: true,
	},
	"iam": {
		Name: "iam", Description: "Auth, users, tenants, scopes, API keys",
		Paths: []string{"pkg/iam"}, Core: true,
	},
	"fsx": {
		Name: "fsx", Description: "File system abstraction (local, S3)",
		Paths: []string{"pkg/fsx"}, Core: true,
	},
	"ai": {
		Name: "ai", Description: "LLM, embeddings, vector store, OCR, speech",
		Paths: []string{"pkg/ai"}, Core: true,
	},
	"jobx": {
		Name: "jobx", Description: "Async job queue (Redis-backed dispatcher)",
		Paths: []string{"pkg/jobx"}, Core: true,
	},
	"notifx": {
		Name: "notifx", Description: "Email notifications (AWS SES)",
		Paths: []string{"pkg/notifx"}, Core: true,
	},
}

// QuickProjectRef is the default Git ref for quick projects.
const QuickProjectRef = "quick-project"

// quickExcluded are modules excluded from quick projects.
var quickExcluded = map[string]bool{
	"iam":        true,
	"migrations": true,
}

// CoreModules returns all modules to download.
// For quick mode, iam and migrations are excluded.
func CoreModules(quick bool) []string {
	var core []string
	for name, mod := range ModuleRegistry {
		if mod.Core {
			if quick && quickExcluded[name] {
				continue
			}
			core = append(core, name)
		}
	}
	return core
}

func ResolveDeps(names []string) []string {
	seen := make(map[string]bool)
	var result []string

	var resolve func(string)
	resolve = func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		if mod, ok := ModuleRegistry[name]; ok {
			for _, dep := range mod.Deps {
				resolve(dep)
			}
		}
		result = append(result, name)
	}

	for _, n := range names {
		resolve(n)
	}
	return result
}

func HasModule(modules []string, name string) bool {
	for _, m := range modules {
		if m == name {
			return true
		}
	}
	return false
}

// IsWired returns true if the given module name is in the manifest's WiredModules list.
func (m *Manifest) IsWired(name string) bool {
	for _, wm := range m.WiredModules {
		if wm == name {
			return true
		}
	}
	return false
}

func LoadManifest(projectRoot string) (*Manifest, error) {
	path := filepath.Join(projectRoot, ManifestoFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no manifesto.yaml at %s: %w", projectRoot, err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid manifesto.yaml: %w", err)
	}
	return &m, nil
}

func (m *Manifest) Save(projectRoot string) error {
	m.UpdatedAt = time.Now()
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal manifesto.yaml: %w", err)
	}
	return os.WriteFile(filepath.Join(projectRoot, ManifestoFile), data, 0644)
}

func NewManifest(name, goModule, version string) *Manifest {
	return &Manifest{
		Project: ProjectConfig{
			Name:     name,
			GoModule: goModule,
			Version:  version,
		},
		Modules:   make(map[string]ModuleConfig),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
