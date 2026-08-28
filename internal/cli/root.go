// Package cli construye el árbol de comandos de calliope.
package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/Calliope-AI/calliope-cli/internal/appctx"
	"github.com/Calliope-AI/calliope-cli/internal/commands"
	"github.com/Calliope-AI/calliope-cli/internal/version"
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
	// SetFlagErrorFunc se hereda de padres a hijos (cobra.Command.FlagErrorFunc
	// sube hasta encontrar uno fijado), así que basta con fijarlo aquí para
	// que un flag desconocido en cualquier subcomando -"calliope docs list
	// --xxx", no solo en la raíz- salga como CLIError en español con hint, en
	// vez del "unknown flag: --xxx" en inglés, sin hint, que da pflag por
	// defecto (I3 de la oleada final).
	root.SetFlagErrorFunc(flagError)

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

// ExecuteRoot construye la raíz y la ejecuta. Devuelve el *cobra.Command que
// de verdad se ejecutó -con sus flags ya fusionados y parseados, incluso si
// la ejecución acaba en error-, para que main() pueda resolver el modo de
// salida de un error con appctx.ResolveOutputMode, exactamente igual que
// resolvería el de un éxito (C2 de la oleada final).
//
// wrapUnknownCommand normaliza a CLIError el único error que Cobra puede
// producir antes de que se ejecute código nuestro -un comando de nivel
// superior que no existe-, así que main() no tiene que conocer la forma en
// que Cobra construye ese mensaje (I3 de la oleada final). Los flags
// desconocidos ya llegan como CLIError vía root.SetFlagErrorFunc, y un
// subcomando desconocido dentro de un grupo, vía groupRunE (C3): a esta
// altura ya son CLIError y wrapUnknownCommand los deja pasar intactos.
func ExecuteRoot(d appctx.Deps) (*cobra.Command, error) {
	cmd, err := NewRootCmd(d).ExecuteC()
	return cmd, wrapUnknownCommand(err)
}

func newVersionCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Muestra la versión de calliope",
		// I4 de la oleada final: sin esto, "calliope version algo-de-más" lo
		// ignoraba en silencio y salía con 0.
		Args: commands.NoPositionalArgs(),
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
