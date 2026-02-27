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
	Project   ProjectConfig           `yaml:"project"`
	Modules   map[string]ModuleConfig `yaml:"modules"`
	CreatedAt time.Time               `yaml:"created_at"`
	UpdatedAt time.Time               `yaml:"updated_at"`
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
		Paths: []string{"pkg/fsx"}, Deps: []string{"errx"},
	},
	"ai": {
		Name: "ai", Description: "LLM, embeddings, vector store, OCR, speech",
		Paths: []string{"pkg/ai"}, Deps: []string{"errx", "fsx"},
	},
}

// QuickProjectRef is the default Git ref for quick projects.
const QuickProjectRef = "quick-project"

// quickCoreModules are always included in a quick project.
var quickCoreModules = []string{"kernel", "errx", "logx", "ptrx", "config", "server"}

// quickOptionalModules can be selected when creating a quick project.
var quickOptionalModules = []string{"fsx", "asyncx", "ai"}

func CoreModules(quick bool) []string {
	if quick {
		return append([]string{}, quickCoreModules...)
	}
	var core []string
	for name, mod := range ModuleRegistry {
		if mod.Core {
			core = append(core, name)
		}
	}
	return core
}

func OptionalModules(quick bool) []string {
	if quick {
		return append([]string{}, quickOptionalModules...)
	}
	var optional []string
	for name, mod := range ModuleRegistry {
		if !mod.Core {
			optional = append(optional, name)
		}
	}
	return optional
}

// IsQuickModule returns true if the module is available in quick projects.
func IsQuickModule(name string) bool {
	for _, m := range quickCoreModules {
		if m == name {
			return true
		}
	}
	for _, m := range quickOptionalModules {
		if m == name {
			return true
		}
	}
	return false
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
