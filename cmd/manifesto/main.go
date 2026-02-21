package main

import "github.com/Abraxas-365/manifesto-cli/internal/cli"

var version = "dev"

func main() {
	cli.Version = version
	cli.Execute()
}
