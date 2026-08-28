package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Calliope-AI/calliope-cli/internal/output"
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

// NoPositionalArgs es el equivalente de cobra.NoArgs con el mismo
// tratamiento que exactArgs: un CLIError en español, con hint y
// output.CodeUsage (salida 2), en vez del "unknown command %q for %q" en
// inglés, sin hint y con salida 1 que da cobra.NoArgs. Se aplica a los 14
// comandos hoja que no toman argumentos posicionales -algunos con flags,
// otros sin ninguno- para que un argumento de más no se ignore en
// silencio: antes, "calliope docs list READY" devolvía todos los
// documentos y salía con 0, y el agente creía que había filtrado (I4 de la
// oleada final). Exportada porque `version` vive en el paquete cli, no en
// commands.
func NoPositionalArgs() cobra.PositionalArgs {
	return exactArgs(0)
}

// groupRunE es el RunE de un grupo de recursos (auth, orgs, config, docs,
// concepts, rules): invocado pelado muestra la ayuda, con código de salida
// 0; con un subcomando que no existe, un CLIError en español con hint y
// código de salida 2 (C3 de la oleada final).
//
// El spec (§5) decía que los grupos "no definen RunE": invocarlos pelados
// muestra la ayuda porque es lo que Cobra hace solo cuando un comando tiene
// subcomandos y no RunE. Pero Cobra decide si un comando es Runnable() -y
// por tanto si cae al flag.ErrHelp que muestra la ayuda- ANTES de llegar a
// ValidateArgs (ver Command.execute en cobra: el chequeo "if
// !c.Runnable() { return flag.ErrHelp }" precede a
// "c.ValidateArgs(argWoFlags)"). Así que un grupo sin RunE no podía
// distinguir "invocado sin argumentos" de "invocado con un argumento que no
// casa con ningún subcomando": las dos rutas llegaban con Runnable()==false
// y las dos mostraban la ayuda con exit 0. "calliope docs typo" no era un
// error de ningún tipo: mostraba la misma ayuda que "calliope docs" solo.
//
// La invariante de la sección 5 era un medio (grupos sin RunE) para un fin
// (bare → ayuda), no el fin en sí; el §6.3 exige exit 2 para uso incorrecto.
// Dar RunE al grupo lo hace Runnable() y deja que Cobra llegue hasta aquí
// con los argumentos ya resueltos, así que este RunE puede por fin ver la
// diferencia entre las dos rutas y decidir el comportamiento correcto para
// cada una.
func groupRunE(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}
	return output.NewError(output.CodeUsage,
		fmt.Sprintf("%q no es un subcomando de %q.", args[0], cmd.CommandPath()),
		"Consulta los subcomandos disponibles con: "+cmd.CommandPath()+" --help")
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
