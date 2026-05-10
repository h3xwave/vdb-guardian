package main

import (
	"fmt"

	"github.com/huxinweidev-cloud/vdb-guardian/internal/version"
)

// main is the future API server entrypoint. The first scaffold intentionally
// avoids starting a network listener until API routes, configuration loading,
// and graceful shutdown behavior are designed and tested.
func main() {
	info := version.Info()
	fmt.Printf("%s server scaffold %s\n", info.Name, info.Version)
}
