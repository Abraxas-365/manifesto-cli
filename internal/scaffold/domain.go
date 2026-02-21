package scaffold

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/Abraxas-365/manifesto-cli/internal/templates"
)

// DomainData is the template context for domain scaffolding.
type DomainData struct {
	GoModule     string
	PackageName  string
	EntityName   string
	RegistryCode string
	TableName    string
	DomainPath   string
}

func NewDomainData(goModule, domainPath string) DomainData {
	parts := strings.Split(domainPath, "/")
	pkgName := parts[len(parts)-1]

	return DomainData{
		GoModule:     goModule,
		PackageName:  pkgName,
		EntityName:   toPascalCase(pkgName),
		RegistryCode: toUpperSnake(pkgName),
		TableName:    toPlural(pkgName),
		DomainPath:   domainPath,
	}
}

func GenerateDomain(projectRoot string, data DomainData) error {
	baseDir := filepath.Join(projectRoot, data.DomainPath)

	files := []struct {
		tmpl string
		dest string
	}{
		{"domain/entity.go.tmpl", filepath.Join(baseDir, data.PackageName+".go")},
		{"domain/port.go.tmpl", filepath.Join(baseDir, "port.go")},
		{"domain/errors.go.tmpl", filepath.Join(baseDir, "errors.go")},
		{"domain/service.go.tmpl", filepath.Join(baseDir, data.PackageName+"srv", "service.go")},
		{"domain/postgres.go.tmpl", filepath.Join(baseDir, data.PackageName+"infra", "postgres.go")},
		{"domain/handler.go.tmpl", filepath.Join(baseDir, data.PackageName+"api", "handler.go")},
	}

	for _, f := range files {
		if err := renderTemplate(f.tmpl, f.dest, data); err != nil {
			return fmt.Errorf("generate %s: %w", filepath.Base(f.dest), err)
		}
	}

	kernelSnippet, err := renderToString("domain/kernel_ids.go.tmpl", data)
	if err != nil {
		return fmt.Errorf("render kernel IDs: %w", err)
	}

	return appendKernelIDs(projectRoot, kernelSnippet)
}

func renderTemplate(tmplPath, destPath string, data any) error {
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

func renderToString(tmplPath string, data any) (string, error) {
	content, err := templates.FS.ReadFile(tmplPath)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New(filepath.Base(tmplPath)).Parse(string(content))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func appendKernelIDs(projectRoot, snippet string) error {
	idFile := filepath.Join(projectRoot, "pkg", "kernel", "proj_ids.go")

	if _, err := os.Stat(idFile); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(idFile), 0755); err != nil {
			return err
		}
		return os.WriteFile(idFile, []byte("package kernel\n"+snippet), 0644)
	}

	existing, err := os.ReadFile(idFile)
	if err != nil {
		return err
	}

	if strings.Contains(string(existing), strings.TrimSpace(snippet)) {
		return nil
	}

	f, err := os.OpenFile(idFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString("\n" + snippet)
	return err
}

// --- String helpers ---

func toPascalCase(s string) string {
	words := splitWords(s)
	var b strings.Builder
	for _, w := range words {
		if len(w) > 0 {
			b.WriteRune(unicode.ToUpper(rune(w[0])))
			b.WriteString(w[1:])
		}
	}
	return b.String()
}

func toUpperSnake(s string) string {
	words := splitWords(s)
	for i, w := range words {
		words[i] = strings.ToUpper(w)
	}
	return strings.Join(words, "_")
}

func toPlural(s string) string {
	if strings.HasSuffix(s, "s") {
		return s + "es"
	}
	if strings.HasSuffix(s, "y") && len(s) > 1 {
		return s[:len(s)-1] + "ies"
	}
	return s + "s"
}

func splitWords(s string) []string {
	s = strings.ReplaceAll(s, "-", "_")
	parts := strings.Split(s, "_")
	var words []string
	for _, p := range parts {
		if p != "" {
			words = append(words, strings.ToLower(p))
		}
	}
	return words
}
