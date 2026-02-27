package scaffold

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Abraxas-365/manifesto-cli/internal/config"
)

// WireOptions configures a module wiring operation.
type WireOptions struct {
	ProjectRoot  string
	ModuleName   string
	GoModule     string   // From manifest
	ProjectName  string   // From manifest
	WiredModules []string // Already wired modules (for bridge detection)
}

// WireModule wires a module into the project by injecting code at marker points
// in config.go, container.go, server.go, and Makefile. Returns the list of modified files.
func WireModule(opts WireOptions) ([]string, error) {
	spec, ok := config.WireableModuleRegistry[opts.ModuleName]
	if !ok {
		return nil, fmt.Errorf("unknown wireable module: %s", opts.ModuleName)
	}

	// Replace placeholders with actual project values.
	spec = replacePlaceholders(spec, opts.GoModule, opts.ProjectName)

	var modified []string

	// 1. Inject into pkg/config/config.go
	if spec.ConfigFields != "" || spec.ConfigLoads != "" {
		if err := injectWireConfig(opts.ProjectRoot, spec); err != nil {
			return nil, fmt.Errorf("wire config: %w", err)
		}
		modified = append(modified, "pkg/config/config.go")
	}

	// 2. Inject into cmd/container.go
	if err := injectWireContainer(opts.ProjectRoot, spec); err != nil {
		return nil, fmt.Errorf("wire container: %w", err)
	}
	modified = append(modified, "cmd/container.go")

	// 3. Inject into cmd/server.go (if module has server injections)
	if spec.PublicRoutes != "" || spec.RouteRegistration != "" || spec.AuthMiddleware != "" || spec.ServerImports != "" {
		if err := injectWireServer(opts.ProjectRoot, spec); err != nil {
			return nil, fmt.Errorf("wire server: %w", err)
		}
		modified = append(modified, "cmd/server.go")
	}

	// 4. Inject into Makefile
	if spec.MakefileEnv != "" || spec.MakefileEnvDisplay != "" {
		if err := injectIntoMakefile(opts.ProjectRoot, spec); err != nil {
			return nil, fmt.Errorf("wire makefile: %w", err)
		}
		modified = append(modified, "Makefile")
	}

	// 5. Check cross-module bridges
	for _, bridge := range spec.Bridges {
		if hasWiredModule(opts.WiredModules, bridge.RequiresModule) {
			bridgeSpec := replaceBridgePlaceholders(bridge, opts.GoModule, opts.ProjectName)
			if err := injectBridge(opts.ProjectRoot, bridgeSpec); err != nil {
				return nil, fmt.Errorf("wire bridge (%s+%s): %w", opts.ModuleName, bridge.RequiresModule, err)
			}
		}
	}

	// 6. Install external Go dependencies
	if len(spec.GoDeps) > 0 {
		if err := installGoDeps(opts.ProjectRoot, spec.GoDeps); err != nil {
			return nil, fmt.Errorf("install deps: %w", err)
		}
	}

	return modified, nil
}

// PostProcessConfigFile inserts wiring markers into the fetched config.go file.
// Called once after init to prepare the file for future module wiring.
func PostProcessConfigFile(projectRoot string) error {
	configFile := filepath.Join(projectRoot, "pkg", "config", "config.go")

	content, err := os.ReadFile(configFile)
	if err != nil {
		return nil // config.go might not exist yet
	}

	text := string(content)

	// Already processed?
	if strings.Contains(text, "// manifesto:config-fields") {
		return nil
	}

	// Insert config-fields marker before closing brace of type Config struct { ... }
	text = insertMarkerBeforeClosingBrace(text, "type Config struct {", "// manifesto:config-fields")

	// Insert config-loads marker before "return cfg" in the Load function
	returnIdx := strings.Index(text, "return cfg")
	if returnIdx != -1 {
		indent := "\t"
		text = text[:returnIdx] + indent + "// manifesto:config-loads\n\n\t" + text[returnIdx:]
	}

	return os.WriteFile(configFile, []byte(text), 0644)
}

// ---------------------------------------------------------------------------
// Config injection
// ---------------------------------------------------------------------------

func injectWireConfig(projectRoot string, spec config.WireableModule) error {
	configFile := filepath.Join(projectRoot, "pkg", "config", "config.go")

	content, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("read config.go: %w", err)
	}

	text := string(content)

	// Guard: check if already injected
	if spec.ConfigFields != "" {
		firstLine := strings.Split(strings.TrimSpace(spec.ConfigFields), "\n")[0]
		if strings.Contains(text, strings.TrimSpace(firstLine)) {
			return nil
		}
	}

	// Inject config fields
	if spec.ConfigFields != "" {
		fieldLine := spec.ConfigFields + "\n\t// manifesto:config-fields"
		text = strings.Replace(text, "// manifesto:config-fields", fieldLine, 1)
	}

	// Inject config loads
	if spec.ConfigLoads != "" {
		loadLine := spec.ConfigLoads + "\n\t// manifesto:config-loads"
		text = strings.Replace(text, "// manifesto:config-loads", loadLine, 1)
	}

	return os.WriteFile(configFile, []byte(text), 0644)
}

// ---------------------------------------------------------------------------
// Container injection
// ---------------------------------------------------------------------------

func injectWireContainer(projectRoot string, spec config.WireableModule) error {
	containerFile := filepath.Join(projectRoot, "cmd", "container.go")

	content, err := os.ReadFile(containerFile)
	if err != nil {
		return fmt.Errorf("read container.go: %w", err)
	}

	text := string(content)

	// Guard: use first import line as idempotency check
	guardStr := wireGuardString(spec)
	if guardStr != "" && strings.Contains(text, guardStr) {
		return nil
	}

	// Inject imports
	if spec.ContainerImports != "" {
		importLine := spec.ContainerImports + "\n\t// manifesto:container-imports"
		text = strings.Replace(text, "// manifesto:container-imports", importLine, 1)
	}

	// Inject fields
	if spec.ContainerFields != "" {
		fieldLine := spec.ContainerFields + "\n\t// manifesto:container-fields"
		text = strings.Replace(text, "// manifesto:container-fields", fieldLine, 1)
	}

	// Inject module init
	if spec.ModuleInit != "" {
		initLine := spec.ModuleInit + "\n\n\t// manifesto:module-init"
		text = strings.Replace(text, "// manifesto:module-init", initLine, 1)
	}

	// Inject background start
	if spec.BackgroundStart != "" {
		bgLine := spec.BackgroundStart + "\n\t// manifesto:background-start"
		text = strings.Replace(text, "// manifesto:background-start", bgLine, 1)
	}

	// Inject helpers
	if spec.ContainerHelpers != "" {
		helperLine := spec.ContainerHelpers + "\n\n// manifesto:container-helpers"
		text = strings.Replace(text, "// manifesto:container-helpers", helperLine, 1)
	}

	return os.WriteFile(containerFile, []byte(text), 0644)
}

// ---------------------------------------------------------------------------
// Server injection
// ---------------------------------------------------------------------------

func injectWireServer(projectRoot string, spec config.WireableModule) error {
	serverFile := filepath.Join(projectRoot, "cmd", "server.go")

	content, err := os.ReadFile(serverFile)
	if err != nil {
		return fmt.Errorf("read server.go: %w", err)
	}

	text := string(content)

	// Guard: check if public routes already injected
	if spec.PublicRoutes != "" {
		firstLine := strings.Split(strings.TrimSpace(spec.PublicRoutes), "\n")[0]
		if strings.Contains(text, strings.TrimSpace(firstLine)) {
			return nil
		}
	}

	// Inject server imports
	if spec.ServerImports != "" {
		importLine := spec.ServerImports + "\n\t// manifesto:server-imports"
		text = strings.Replace(text, "// manifesto:server-imports", importLine, 1)
	}

	// Inject public routes
	if spec.PublicRoutes != "" {
		routeLine := spec.PublicRoutes + "\n\n\t// manifesto:public-routes"
		text = strings.Replace(text, "// manifesto:public-routes", routeLine, 1)
	}

	// Ensure protected group exists if this module needs routes
	if spec.RouteRegistration != "" || spec.AuthMiddleware != "" {
		if !strings.Contains(text, "protected :=") {
			// Create the protected group (with auth middleware if present)
			if spec.AuthMiddleware != "" {
				groupCode := fmt.Sprintf("\tprotected := app.Group(\"/api/v1\",\n\t\t%s,\n\t)\n\n\t// manifesto:route-registration", spec.AuthMiddleware)
				text = strings.Replace(text, "// manifesto:route-registration", groupCode, 1)
			} else {
				groupCode := "\tprotected := app.Group(\"/api/v1\")\n\n\t// manifesto:route-registration"
				text = strings.Replace(text, "// manifesto:route-registration", groupCode, 1)
			}
		} else if spec.AuthMiddleware != "" {
			// Protected group already exists — add middleware
			oldGroup := `protected := app.Group("/api/v1")`
			newGroup := fmt.Sprintf("protected := app.Group(\"/api/v1\",\n\t\t%s,\n\t)", spec.AuthMiddleware)
			if !strings.Contains(text, spec.AuthMiddleware) {
				text = strings.Replace(text, oldGroup, newGroup, 1)
			}
		}
	}

	// Inject route registration
	if spec.RouteRegistration != "" {
		regLine := spec.RouteRegistration + "\n\n\t// manifesto:route-registration"
		text = strings.Replace(text, "// manifesto:route-registration", regLine, 1)
	}

	return os.WriteFile(serverFile, []byte(text), 0644)
}

// ---------------------------------------------------------------------------
// Makefile injection
// ---------------------------------------------------------------------------

func injectIntoMakefile(projectRoot string, spec config.WireableModule) error {
	makefilePath := filepath.Join(projectRoot, "Makefile")

	content, err := os.ReadFile(makefilePath)
	if err != nil {
		return nil // Makefile might not exist
	}

	text := string(content)

	// Guard: check if already injected
	if spec.MakefileEnv != "" {
		firstLine := strings.Split(strings.TrimSpace(spec.MakefileEnv), "\n")[0]
		if strings.Contains(text, strings.TrimSpace(firstLine)) {
			return nil
		}
	}

	// Inject env config block (top-level, no tab prefix)
	if spec.MakefileEnv != "" {
		envBlock := spec.MakefileEnv + "\n\n# manifesto:env-config"
		text = strings.Replace(text, "# manifesto:env-config", envBlock, 1)
	}

	// Inject env display lines (inside make recipe, needs tab prefix)
	if spec.MakefileEnvDisplay != "" {
		displayBlock := tabPrefixLines(spec.MakefileEnvDisplay) + "\n\t# manifesto:env-display"
		text = strings.Replace(text, "\t# manifesto:env-display", displayBlock, 1)
	}

	return os.WriteFile(makefilePath, []byte(text), 0644)
}

// tabPrefixLines adds a leading tab to every non-empty line.
func tabPrefixLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = "\t" + line
		}
	}
	return strings.Join(lines, "\n")
}

// ---------------------------------------------------------------------------
// Bridge injection
// ---------------------------------------------------------------------------

func injectBridge(projectRoot string, bridge config.Bridge) error {
	containerFile := filepath.Join(projectRoot, "cmd", "container.go")

	content, err := os.ReadFile(containerFile)
	if err != nil {
		return fmt.Errorf("read container.go for bridge: %w", err)
	}

	text := string(content)

	// Guard: check if bridge code already present
	firstLine := strings.Split(strings.TrimSpace(bridge.ContainerInit), "\n")[0]
	if strings.Contains(text, strings.TrimSpace(firstLine)) {
		return nil
	}

	// Inject bridge imports (if not already present)
	if bridge.ContainerImports != "" {
		for _, line := range strings.Split(bridge.ContainerImports, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.Contains(text, line) {
				importLine := "\t" + line + "\n\t// manifesto:container-imports"
				text = strings.Replace(text, "// manifesto:container-imports", importLine, 1)
			}
		}
	}

	// Inject bridge init code
	if bridge.ContainerInit != "" {
		initLine := bridge.ContainerInit + "\n\n\t// manifesto:module-init"
		text = strings.Replace(text, "// manifesto:module-init", initLine, 1)
	}

	return os.WriteFile(containerFile, []byte(text), 0644)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// insertMarkerBeforeClosingBrace finds a pattern like "type Config struct {"
// and inserts a marker comment before the matching closing brace.
func insertMarkerBeforeClosingBrace(text, opener, marker string) string {
	idx := strings.Index(text, opener)
	if idx == -1 {
		return text
	}

	// Find the opening brace
	braceIdx := strings.Index(text[idx:], "{")
	if braceIdx == -1 {
		return text
	}
	braceIdx += idx

	// Count braces to find matching close
	depth := 1
	pos := braceIdx + 1

	for pos < len(text) && depth > 0 {
		if text[pos] == '{' {
			depth++
		} else if text[pos] == '}' {
			depth--
		}
		if depth > 0 {
			pos++
		}
	}

	if depth != 0 {
		return text // unmatched braces, don't modify
	}

	// pos is at the closing brace — insert marker before it
	return text[:pos] + "\t" + marker + "\n" + text[pos:]
}

func replacePlaceholders(spec config.WireableModule, goModule, projectName string) config.WireableModule {
	r := func(s string) string {
		s = strings.ReplaceAll(s, "{{GOMODULE}}", goModule)
		s = strings.ReplaceAll(s, "{{PROJECTNAME}}", projectName)
		return s
	}
	spec.ConfigFields = r(spec.ConfigFields)
	spec.ConfigLoads = r(spec.ConfigLoads)
	spec.ContainerImports = r(spec.ContainerImports)
	spec.ContainerFields = r(spec.ContainerFields)
	spec.ModuleInit = r(spec.ModuleInit)
	spec.BackgroundStart = r(spec.BackgroundStart)
	spec.ContainerHelpers = r(spec.ContainerHelpers)
	spec.ServerImports = r(spec.ServerImports)
	spec.PublicRoutes = r(spec.PublicRoutes)
	spec.RouteRegistration = r(spec.RouteRegistration)
	spec.MakefileEnv = r(spec.MakefileEnv)
	spec.MakefileEnvDisplay = r(spec.MakefileEnvDisplay)

	for i, bridge := range spec.Bridges {
		spec.Bridges[i].ContainerImports = r(bridge.ContainerImports)
		spec.Bridges[i].ContainerInit = r(bridge.ContainerInit)
	}

	return spec
}

func replaceBridgePlaceholders(bridge config.Bridge, goModule, projectName string) config.Bridge {
	r := func(s string) string {
		s = strings.ReplaceAll(s, "{{GOMODULE}}", goModule)
		s = strings.ReplaceAll(s, "{{PROJECTNAME}}", projectName)
		return s
	}
	bridge.ContainerImports = r(bridge.ContainerImports)
	bridge.ContainerInit = r(bridge.ContainerInit)
	return bridge
}

// wireGuardString returns a string that, if present in the file, indicates
// the module is already wired. Uses the first import line as the guard.
func wireGuardString(spec config.WireableModule) string {
	if spec.ContainerImports != "" {
		lines := strings.Split(spec.ContainerImports, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				// Extract the import path (between quotes)
				if start := strings.Index(line, `"`); start != -1 {
					if end := strings.Index(line[start+1:], `"`); end != -1 {
						return line[start+1 : start+1+end]
					}
				}
				return line
			}
		}
	}
	if spec.ContainerFields != "" {
		return strings.TrimSpace(strings.Split(spec.ContainerFields, "\n")[0])
	}
	return ""
}

func hasWiredModule(wired []string, name string) bool {
	for _, m := range wired {
		if m == name {
			return true
		}
	}
	return false
}

func installGoDeps(projectRoot string, deps []string) error {
	for _, dep := range deps {
		cmd := exec.Command("go", "get", dep)
		cmd.Dir = projectRoot
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go get %s: %w", dep, err)
		}
	}
	return nil
}
