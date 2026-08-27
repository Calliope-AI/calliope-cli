// Command calliope es el punto de entrada del CLI de Calliope Data.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"github.com/calliope/calliope-cli/internal/cli"
	"github.com/calliope/calliope-cli/internal/output"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		isJSON := slices.Contains(os.Args, "--json")

		// Si es un CLIError, manejarlo especialmente para incluir el hint.
		var cliErr *output.CLIError
		if ok := isError(err, &cliErr); ok {
			if isJSON {
				// En modo JSON, imprimir el envelope completo.
				envelope := cliErr.Envelope()
				b, _ := json.Marshal(envelope)
				fmt.Fprintln(os.Stderr, string(b))
			} else {
				// En modo texto, imprimir el mensaje y el hint.
				fmt.Fprintln(os.Stderr, cliErr.Message)
				if cliErr.Hint != "" {
					fmt.Fprintln(os.Stderr, cliErr.Hint)
				}
			}
		} else {
			// Errores genéricos: usar la cadena del error.
			fmt.Fprintln(os.Stderr, err)
		}

		os.Exit(output.ExitCodeFor(err))
	}
}

// isError desenvuelve cadenas de error creadas con %w para buscar un CLIError.
func isError(err error, target **output.CLIError) bool {
	// Intentar desenvolver el error.
	for err != nil {
		if cliErr, ok := err.(*output.CLIError); ok {
			*target = cliErr
			return true
		}
		// Intenta desenvolver con Unwrap (cadenas creadas con %w).
		type unwrapper interface {
			Unwrap() error
		}
		if w, ok := err.(unwrapper); ok {
			err = w.Unwrap()
		} else {
			break
		}
	}
	return false
}
