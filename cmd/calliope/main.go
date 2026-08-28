// Command calliope es el punto de entrada del CLI de Calliope Data.
package main

import (
	"os"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/cli"
	"github.com/calliope/calliope-cli/internal/output"
)

func main() {
	d := appctx.DefaultDeps()

	// ExecuteC (no Execute) devuelve el *cobra.Command que de verdad se
	// ejecutó -con sus flags ya fusionados y parseados, incluso si la
	// ejecución falló después-, para poder resolver el modo de salida del
	// error con la misma función que usa el resto del CLI para el éxito:
	// appctx.ResolveOutputMode. Antes se adivinaba con
	// slices.Contains(os.Args, "--json"), desconectado de esa resolución
	// (C2 de la oleada final).
	executed, err := cli.ExecuteRoot(d)
	if err != nil {
		opts := appctx.ResolveOutputMode(executed, d)
		output.WriteError(d.Stderr, err, opts.IsMachineReadable())
		os.Exit(output.ExitCodeFor(err))
	}
}
