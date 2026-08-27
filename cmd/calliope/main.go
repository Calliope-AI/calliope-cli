// Command calliope es el punto de entrada del CLI de Calliope Data.
package main

import (
	"fmt"
	"os"

	"github.com/calliope/calliope-cli/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1) // Task 2 sustituye esto por el mapeo real de códigos.
	}
}
