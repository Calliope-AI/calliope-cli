// Command calliope es el punto de entrada del CLI de Calliope Data.
package main

import (
	"os"
	"slices"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/cli"
	"github.com/calliope/calliope-cli/internal/output"
)

func main() {
	if err := cli.NewRootCmd(appctx.DefaultDeps()).Execute(); err != nil {
		isJSON := slices.Contains(os.Args, "--json")
		output.WriteError(os.Stderr, err, isJSON)
		os.Exit(output.ExitCodeFor(err))
	}
}
