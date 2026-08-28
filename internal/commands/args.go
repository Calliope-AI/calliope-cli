package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/output"
)

// exactArgs envuelve cobra.ExactArgs(n) para que un número de argumentos
// incorrecto produzca un output.CLIError -mensaje en español, código
// output.CodeUsage (salida 2) y un hint con la forma de uso concreta del
// comando- en vez del "accepts N arg(s), received M" en inglés, sin hint y
// con código de salida 1 que da Cobra por defecto.
//
// Las tareas siguientes que definan comandos con un número fijo de
// argumentos (ask, docs show, concepts show, docs search, query, ...) deben
// usar este ayudante en lugar de cobra.ExactArgs directamente.
func exactArgs(n int, uso string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == n {
			return nil
		}
		return output.NewError(output.CodeUsage,
			fmt.Sprintf("Número de argumentos incorrecto: se esperaban %d, se recibieron %d.", n, len(args)),
			"Uso: "+uso)
	}
}
