package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/output"
)

// exactArgs envuelve cobra.ExactArgs(n) para que un número de argumentos
// incorrecto produzca un output.CLIError -mensaje en español, código
// output.CodeUsage (salida 2) y un hint con la forma de uso concreta del
// comando- en vez del "accepts N arg(s), received M" en inglés, sin hint y
// con código de salida 1 que da Cobra por defecto.
//
// El hint se deriva de cmd.UseLine() en lugar de recibirse como parámetro:
// así no hay dos representaciones de la misma forma de uso (el Use del
// comando y un literal en la llamada) que puedan desincronizarse si alguien
// cambia el placeholder de argumentos en uno y no en el otro.
//
// Las tareas siguientes que definan comandos con un número fijo de
// argumentos (ask, docs show, concepts show, docs search, query, ...) deben
// usar este ayudante en lugar de cobra.ExactArgs directamente.
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == n {
			return nil
		}
		return output.NewError(output.CodeUsage,
			fmt.Sprintf("Número de argumentos incorrecto: se esperaban %d, se recibieron %d.", n, len(args)),
			"Uso: "+usageLine(cmd))
	}
}

// usageLine devuelve la forma de uso completa de cmd (ruta de padres + Use),
// recortando el sufijo " [flags]" que Cobra añade en cuanto el comando tiene
// algún flag disponible -aquí, siempre: los cinco flags globales son
// persistentes de la raíz y los hereda cualquier subcomando-. Ese sufijo no
// dice nada específico del comando (ya se sabe que hay flags globales) y solo
// añade ruido al hint; probado contra el binario real antes de fijarlo así.
func usageLine(cmd *cobra.Command) string {
	return strings.TrimSuffix(cmd.UseLine(), " [flags]")
}
