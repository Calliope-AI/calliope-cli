package cli

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/calliope/calliope-cli/internal/output"
)

// I3 de la oleada final: un comando o un flag que no existen fallaban con el
// mensaje que da Cobra/pflag por defecto -en inglés, sin hint, y con exit 1
// en vez de 2 (uso incorrecto)-. Es la misma clase de fallo que ya resuelve
// exactArgs (internal/commands/args.go) para el número de argumentos, en sus
// otras dos caras: aquí se les da el mismo tratamiento.

// flagError convierte un error de parseo de flags (pflag) en un CLIError en
// español con hint y CodeUsage. Se engancha con root.SetFlagErrorFunc:
// Cobra lo llama en el momento exacto en que ParseFlags falla, antes de que
// el mensaje en inglés de pflag llegue a ningún sitio.
//
// pflag tipa sus errores de "flag no existe" en *pflag.NotExistError, con
// GetSpecifiedName()/GetSpecifiedShortnames() para reconstruir el literal
// exacto que se escribió (--xxx o -x) sin parsear el texto del mensaje.
// Otros fallos de flags (--foo sin el valor que necesita, un valor que no
// parsea, sintaxis inválida) no tienen un caso tan preciso disponible; para
// esos se conserva el mensaje técnico de pflag -que sigue siendo información
// del propio flag que el usuario escribió, no un dato ajeno como el cuerpo
// de una respuesta del backend- enmarcado en una frase en español.
func flagError(cmd *cobra.Command, err error) error {
	var notExist *pflag.NotExistError
	if errors.As(err, &notExist) {
		literal := "--" + notExist.GetSpecifiedName()
		if cortos := notExist.GetSpecifiedShortnames(); cortos != "" {
			literal = "-" + cortos
		}
		return output.NewError(output.CodeUsage,
			fmt.Sprintf("Flag desconocido: %s.", literal),
			"Consulta los flags disponibles con: "+cmd.CommandPath()+" --help")
	}
	return output.NewError(output.CodeUsage,
		fmt.Sprintf("Flag inválido: %v.", err),
		"Consulta los flags disponibles con: "+cmd.CommandPath()+" --help")
}

// unknownCommandRe reconoce el mensaje que cobra construye en legacyArgs
// (command.go, en el módulo fijado por go.mod) para un comando de nivel
// superior que no existe: fmt.Errorf("unknown command %q for %q%s", ...).
// Cobra no expone un tipo para este error -solo lo da Find(), antes de que
// se ejecute ningún RunE nuestro, así que ni groupRunE ni SetFlagErrorFunc
// pueden interceptarlo- así que se reconoce por el prefijo estable de su
// mensaje.
var unknownCommandRe = regexp.MustCompile(`^unknown command "([^"]*)" for `)

// wrapUnknownCommand convierte "unknown command %q for %q" -lo que produce
// Cobra cuando el primer argumento no casa con ningún comando de nivel
// superior, p. ej. "calliope frobnicate"- en un CLIError en español con
// hint y CodeUsage. Cualquier otro error pasa intacto.
func wrapUnknownCommand(err error) error {
	if err == nil {
		return nil
	}
	if m := unknownCommandRe.FindStringSubmatch(err.Error()); m != nil {
		return output.NewError(output.CodeUsage,
			fmt.Sprintf("%q no es un comando de calliope.", m[1]),
			"Consulta los comandos disponibles con: calliope --help")
	}
	return err
}
