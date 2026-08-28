// Package cli construye el árbol de comandos de calliope.
package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/commands"
	"github.com/calliope/calliope-cli/internal/version"
)

// NewRootCmd construye el comando raíz con sus flags globales.
// Se crea uno nuevo por invocación para que los tests no compartan estado.
func NewRootCmd(d appctx.Deps) *cobra.Command {
	root := &cobra.Command{
		Use:           "calliope",
		Short:         "Interfaz de línea de comandos de Calliope Data",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	// La lista de flags vive en appctx, no aquí: así los tests de comandos
	// montan una raíz idéntica a la real sin duplicarla.
	appctx.RegisterGlobalFlags(root)

	root.AddCommand(
		newVersionCmd(d),
		commands.NewAuthCmd(d),
		commands.NewOrgsCmd(d),
		commands.NewConfigCmd(d),
		commands.NewAskCmd(d),
		commands.NewDocsCmd(d),
		commands.NewConceptsCmd(d),
		commands.NewRulesCmd(d),
		commands.NewSchemaCmd(d),
		commands.NewQueryCmd(d),
		commands.NewDoctorCmd(d),
		commands.NewSkillCmd(),
	)
	return root
}

func newVersionCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Muestra la versión de calliope",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "calliope %s (%s, %s)\n",
				version.Version, version.Commit, version.Date)

			// La línea de aviso solo sale en terminal interactivo: en una
			// tubería rompería a quien parsee la salida.
			if d.IsTTY {
				if nueva := version.LatestVersion(d.ReleasesURL, version.Version, 2*time.Second); nueva != "" {
					fmt.Fprintf(cmd.OutOrStdout(),
						"\nHay una versión más reciente: %s. Actualiza con: brew upgrade calliope\n", nueva)
				}
			}
			return nil
		},
	}
}
