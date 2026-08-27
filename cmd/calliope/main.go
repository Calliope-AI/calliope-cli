// Command calliope es el punto de entrada del CLI de Calliope Data.
package main

import (
	"fmt"
	"os"

	"github.com/calliope/calliope-cli/internal/cli"
	"github.com/calliope/calliope-cli/internal/output"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(output.ExitCodeFor(err))
	}
}
