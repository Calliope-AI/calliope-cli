// Package cli construye el árbol de comandos de calliope.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/version"
)

// NewRootCmd construye el comando raíz con sus flags globales.
// Se crea uno nuevo por invocación para que los tests no compartan estado.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "calliope",
		Short:         "Interfaz de línea de comandos de Calliope Data",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Flags globales; los consumen appctx y presenter en tareas posteriores.
	f := root.PersistentFlags()
	f.String("org", "", "organización sobre la que operar")
	f.Bool("json", false, "salida JSON con envelope completo")
	f.Bool("quiet", false, "salida solo de datos, sin envelope")
	f.Bool("md", false, "salida en Markdown")
	f.String("jq", "", "filtra la salida con una expresión jq")

	root.AddCommand(newVersionCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Muestra la versión de calliope",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "calliope %s (%s, %s)\n",
				version.Version, version.Commit, version.Date)
			return nil
		},
	}
}
