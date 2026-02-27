package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

var (
	Bold    = color.New(color.Bold)
	Green   = color.New(color.FgGreen, color.Bold)
	Cyan    = color.New(color.FgCyan)
	Yellow  = color.New(color.FgYellow)
	Red     = color.New(color.FgRed, color.Bold)
	Dim     = color.New(color.Faint)
	White   = color.New(color.FgWhite, color.Bold)
	Magenta = color.New(color.FgMagenta, color.Bold)
)

const banner = `
                        _  __          _
  _ __ ___   __ _ _ __ (_)/ _| ___ ___| |_ ___
 | '_ ` + "`" + ` _ \ / _` + "`" + ` | '_ \| | |_ / _ / __| __/ _ \
 | | | | | | (_| | | | | |  _|  __\__ | || (_) |
 |_| |_| |_|\__,_|_| |_|_|_|  \___|___/\__\___/
`

func PrintBanner() {
	Cyan.Print(banner)
}

func PrintCreateHeader(projectName, goModule string) {
	fmt.Println()
	Magenta.Println("  Creating a new Manifesto app in", Bold.Sprint("./"+projectName))
	fmt.Println()
	Dim.Printf("  module:  %s\n", goModule)
	fmt.Println()
}

func PrintCreateHeaderQuick(projectName, goModule string) {
	fmt.Println()
	Magenta.Println("  Creating a new Manifesto", Yellow.Sprint("quick"), "app in", Bold.Sprint("./"+projectName))
	fmt.Println()
	Dim.Printf("  module:  %s\n", goModule)
	Dim.Println("  mode:    quick (no IAM, no migrations)")
	fmt.Println()
}

// Spinner provides a CRA-style animated spinner.
type Spinner struct {
	message string
	done    chan bool
	mu      sync.Mutex
	stopped bool
}

var frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		done:    make(chan bool),
	}
}

func (s *Spinner) Start() {
	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				return
			default:
				frame := frames[i%len(frames)]
				Cyan.Printf("\r  %s %s", frame, s.message)
				time.Sleep(80 * time.Millisecond)
				i++
			}
		}
	}()
}

func (s *Spinner) Stop(success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	s.stopped = true
	s.done <- true

	// Clear the line.
	fmt.Printf("\r%s\r", strings.Repeat(" ", len(s.message)+10))

	if success {
		Green.Printf("  ✓ %s\n", s.message)
	} else {
		Red.Printf("  ✗ %s\n", s.message)
	}
}

func StepDone(msg string) {
	Green.Printf("  ✓ %s\n", msg)
}

func StepInfo(msg string) {
	Cyan.Printf("  ℹ %s\n", msg)
}

func StepWarn(msg string) {
	Yellow.Printf("  ⚠ %s\n", msg)
}

func PrintSuccess(projectName string) {
	fmt.Println()
	Green.Println("  Success!", White.Sprintf(" Created %s", projectName))
	fmt.Println()
	Dim.Println("  Inside that directory, you can run:")
	fmt.Println()
	Cyan.Println("    go mod tidy")
	Dim.Println("    Install dependencies")
	fmt.Println()
	Cyan.Println("    go build ./...")
	Dim.Println("    Build the project")
	fmt.Println()
	Cyan.Println("    manifesto add pkg/mymodule/entity")
	Dim.Println("    Scaffold a new DDD domain package")
	fmt.Println()
	Cyan.Println("    manifesto add jobx")
	Dim.Println("    Wire a module into the project")
	fmt.Println()
	Dim.Println("  We suggest that you begin by typing:")
	fmt.Println()
	Cyan.Printf("    cd %s\n", projectName)
	Cyan.Println("    go mod tidy")
	fmt.Println()
	Dim.Println("  Happy hacking!")
	fmt.Println()
}

func PrintAddSuccess(entityName, domainPath, pkgName string) {
	fmt.Println()
	Green.Println("  Success!", White.Sprintf(" Created domain %s", entityName))
	fmt.Println()
	Dim.Println("  Generated files:")
	fmt.Println()
	printFile(domainPath+"/"+pkgName+".go", "Entity + DTOs")
	printFile(domainPath+"/port.go", "Repository interface")
	printFile(domainPath+"/errors.go", "Error registry")
	printFile(domainPath+"/"+pkgName+"srv/service.go", "Service layer")
	printFile(domainPath+"/"+pkgName+"infra/postgres.go", "Postgres repository")
	printFile(domainPath+"/"+pkgName+"api/handler.go", "HTTP handlers")
	printFile(domainPath+"/"+pkgName+"container/container.go", "Module container (DI wiring)")
	fmt.Println()
	Dim.Printf("  + kernel.%sID added to pkg/kernel/proj_ids.go\n", entityName)
	Dim.Printf("  + %s injected into cmd/container.go\n", entityName)
	Dim.Printf("  + %s routes injected into cmd/server.go\n", entityName)
	fmt.Println()
	Dim.Println("  Next steps:")
	fmt.Println()
	Cyan.Println("    1. Customize entity fields")
	Cyan.Println("    2. Review wiring in " + domainPath + "/" + pkgName + "container/container.go")
	Cyan.Println("    3. Create migration in migrations/")
	fmt.Println()
}

func PrintWireSuccess(moduleName string, modifiedFiles []string) {
	fmt.Println()
	Green.Println("  Success!", White.Sprintf(" Wired %s", moduleName))
	fmt.Println()
	if len(modifiedFiles) > 0 {
		Dim.Println("  Modified files:")
		for _, f := range modifiedFiles {
			fmt.Printf("    %s %s\n", Green.Sprint("~"), Cyan.Sprint(f))
		}
		fmt.Println()
	}
}

func printFile(path, desc string) {
	fmt.Printf("    %s %s  %s\n", Green.Sprint("✓"), Cyan.Sprint(path), Dim.Sprint(desc))
}

type ModuleDisplay struct {
	Name        string
	Description string
	Installed   bool
	Core        bool
	Deps        string
}

type WireableModuleDisplay struct {
	Name        string
	Description string
	Wired       bool
}

func PrintModulesWithSections(libraries []ModuleDisplay, wireables []WireableModuleDisplay) {
	fmt.Println()
	Bold.Println("  Core Libraries")
	fmt.Println()

	for _, m := range libraries {
		status := Dim.Sprint("○")
		if m.Installed {
			status = Green.Sprint("●")
		}

		deps := ""
		if m.Deps != "" {
			deps = Dim.Sprintf(" → %s", m.Deps)
		}

		fmt.Printf("    %s  %-12s %s%s\n",
			status,
			Bold.Sprint(m.Name),
			m.Description,
			deps,
		)
	}

	fmt.Println()
	Bold.Println("  Wireable Modules")
	fmt.Println()

	for _, m := range wireables {
		status := Dim.Sprint("○ not wired")
		if m.Wired {
			status = Green.Sprint("● wired")
		}

		fmt.Printf("    %s  %-8s  %s\n",
			status,
			Bold.Sprint(m.Name),
			m.Description,
		)
	}

	fmt.Println()
	fmt.Printf("    %s installed/wired   %s available\n", Green.Sprint("●"), Dim.Sprint("○"))
	fmt.Println()
}

func PrintInstallSuccess(moduleName string, installed []string) {
	fmt.Println()
	Green.Println("  Success!", White.Sprintf(" Installed %s", moduleName))
	if len(installed) > 1 {
		Dim.Printf("  (with dependencies: %s)\n", strings.Join(installed, ", "))
	}
	fmt.Println()
	Dim.Println("  Run 'go mod tidy' to sync dependencies.")
	fmt.Println()
}
