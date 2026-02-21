# Manifesto CLI

Create production-grade Go apps with DDD architecture.

## Install

```bash
go install github.com/Abraxas-365/manifesto-cli/cmd/manifesto@latest
```

## Usage

```bash
# Create a new project (interactive)
manifesto init myapp --module github.com/me/myapp

# Create with specific modules
manifesto init myapp --module github.com/me/myapp --with iam,ai

# Create with everything
manifesto init myapp --module github.com/me/myapp --all

# Add a domain package
cd myapp
manifesto add pkg/recruitment/candidate

# Install a module later
manifesto install ai

# List modules
manifesto modules
```
# manifesto-cli
