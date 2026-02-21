#!/bin/bash
set -e

# Run this from the root of your manifesto-cli repo
# It moves flat files into the correct Go package directories

echo "🔧 Restructuring manifesto-cli..."

# Create directory structure
mkdir -p cmd/manifesto
mkdir -p internal/cli
mkdir -p internal/config
mkdir -p internal/remote
mkdir -p internal/scaffold
mkdir -p internal/templates/domain
mkdir -p internal/templates/project
mkdir -p .github/workflows

# Move files to correct locations
# Entry point
[ -f main.go ] && mv main.go cmd/manifesto/main.go

# CLI commands
[ -f root.go ] && mv root.go internal/cli/root.go
[ -f init.go ] && mv init.go internal/cli/init.go
[ -f add.go ] && mv add.go internal/cli/add.go
[ -f install.go ] && mv install.go internal/cli/install.go
[ -f modules.go ] && mv modules.go internal/cli/modules.go

# Config
[ -f manifest.go ] && mv manifest.go internal/config/manifest.go

# Remote
[ -f github.go ] && mv github.go internal/remote/github.go

# Scaffold
[ -f domain.go ] && mv domain.go internal/scaffold/domain.go
[ -f project.go ] && mv project.go internal/scaffold/project.go
[ -f module.go ] && mv module.go internal/scaffold/module.go

# Templates
[ -f embed.go ] && mv embed.go internal/templates/embed.go
[ -f entity.go.tmpl ] && mv entity.go.tmpl internal/templates/domain/entity.go.tmpl
[ -f port.go.tmpl ] && mv port.go.tmpl internal/templates/domain/port.go.tmpl
[ -f errors.go.tmpl ] && mv errors.go.tmpl internal/templates/domain/errors.go.tmpl
[ -f service.go.tmpl ] && mv service.go.tmpl internal/templates/domain/service.go.tmpl
[ -f postgres.go.tmpl ] && mv postgres.go.tmpl internal/templates/domain/postgres.go.tmpl
[ -f handler.go.tmpl ] && mv handler.go.tmpl internal/templates/domain/handler.go.tmpl
[ -f kernel_ids.go.tmpl ] && mv kernel_ids.go.tmpl internal/templates/domain/kernel_ids.go.tmpl
[ -f env.example.tmpl ] && mv env.example.tmpl internal/templates/project/env.example.tmpl

# Release workflow
[ -f release.yaml ] && mv release.yaml .github/workflows/release.yaml

echo ""
echo "✓ Structure fixed. Verify with: tree"
echo ""
echo "Now run:"
echo "  go mod tidy"
echo "  go build ./..."
echo "  git add -A && git commit -m 'fix: restore directory structure' && git push"
