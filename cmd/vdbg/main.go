package main

import (
	"fmt"
	"os"

	"github.com/huxinweidev-cloud/vdb-guardian/internal/version"
)

// main is the CLI entrypoint for local and automation-driven vdb-guardian usage.
// The initial scaffold supports version output so operators can verify that the
// binary and repository are wired correctly before connector commands are added.
func main() {
	info := version.Info()
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("%s %s\n", info.Name, info.Version)
		return
	}

	fmt.Printf("%s: enterprise vector database migration verifier\n", info.Name)
}
