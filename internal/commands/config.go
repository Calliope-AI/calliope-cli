package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/config"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/presenter"
)

// NewConfigCmd construye el grupo `config`.
func NewConfigCmd(d appctx.Deps) *cobra.Command {
	grupo := &cobra.Command{
		Use:   "config",
		Short: "Consulta y modifica la configuración de calliope",
	}
	grupo.AddCommand(newConfigListCmd(d), newConfigGetCmd(d), newConfigSetCmd(d), newConfigPathCmd(d))
	return grupo
}

func newConfigListCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Muestra cada valor con la capa de la que proviene",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.BuildSinCredencial(cmd, d)
			if err != nil {
				return err
			}
			todos := ctx.Cfg.All()

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(todos, fmt.Sprintf("%d valores", len(todos)),
					output.Breadcrumb{Action: "cambiar", Cmd: "calliope config set <clave> <valor>"}),
				Text: func(w io.Writer) error {
					claves := make([]string, 0, len(todos))
					for k := range todos {
						claves = append(claves, k)
					}
					sort.Strings(claves)

					filas := make([][]string, 0, len(claves))
					for _, k := range claves {
						v := todos[k]
						filas = append(filas, []string{k, v.Value, string(v.Source), v.Path})
					}
					return presenter.Table(w, []string{"CLAVE", "VALOR", "ORIGEN", "FICHERO"}, filas)
				},
			})
		},
	}
}

func newConfigGetCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "get <clave>",
		Short: "Muestra el valor de una clave",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.BuildSinCredencial(cmd, d)
			if err != nil {
				return err
			}
			v := ctx.Cfg.Get(args[0])
			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(v.Value, fmt.Sprintf("%s (%s)", args[0], v.Source)),
				Text: func(w io.Writer) error {
					_, err := fmt.Fprintln(w, v.Value)
					return err
				},
			})
		},
	}
}

func newConfigSetCmd(d appctx.Deps) *cobra.Command {
	var global bool
	cmd := &cobra.Command{
		Use:   "set <clave> <valor>",
		Short: "Fija una clave en la configuración de proyecto (o global con --global)",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clave, valor := args[0], args[1]

			ruta := filepath.Join(d.Cwd, ".calliope", config.FileName)
			dirMode := os.FileMode(0o755)
			if global {
				ruta = config.GlobalPath(d.Env)
				// El directorio global es el mismo donde auth.DefaultStore
				// guarda las credenciales (internal/auth/store.go), que lo
				// trata como sensible: 0700 en vez del 0755 que basta para el
				// directorio de proyecto, que ya es público -vive dentro del
				// propio repositorio.
				dirMode = 0o700
			} else if !config.IsProjectAllowed(clave) {
				// La misma regla que aplica al leer se aplica al escribir: si
				// no se pudiera leer, escribirlo solo confunde.
				//
				// El mensaje (no solo el hint) menciona --global a propósito:
				// output.CLIError.Error() solo devuelve Message, nunca Hint
				// (contrato fijado en TestCLIErrorErrorDevolverSoloMensaje,
				// internal/output/errors_test.go), así que cualquier código que
				// inspeccione err.Error() -incluido este comando desde fuera-
				// necesita ver la explicación completa ahí, no solo en el hint.
				return output.NewError(output.CodeUsage,
					fmt.Sprintf("La clave %q no se puede fijar en la configuración de proyecto; usa --global para fijarla en la configuración global.", clave),
					fmt.Sprintf("Fíjala en la global con: calliope config set %s %s --global", clave, valor))
			}

			dir := filepath.Dir(ruta)
			if err := os.MkdirAll(dir, dirMode); err != nil {
				return err
			}
			if global {
				// Igual que auth/store.go: os.MkdirAll solo aplica el modo al
				// crear el directorio; si ya existía (p. ej. tras restaurar
				// dotfiles o extraer un tar con umask laxo) se queda con los
				// permisos que ya tuviera. Se refuerza explícitamente para que
				// no quede listable por otros usuarios del sistema.
				if err := os.Chmod(dir, 0o700); err != nil {
					return err
				}
			}
			vals := map[string]string{}
			if b, err := os.ReadFile(ruta); err == nil {
				_ = json.Unmarshal(b, &vals)
			}
			vals[clave] = valor

			b, err := json.MarshalIndent(vals, "", "  ")
			if err != nil {
				return err
			}
			if err := os.WriteFile(ruta, b, 0o600); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s = %s (%s)\n", clave, valor, ruta)
			return nil
		},
	}
	cmd.Flags().BoolVar(&global, "global", false, "escribe en la configuración global en vez de la del proyecto")
	return cmd
}

func newConfigPathCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Imprime la ruta del fichero de configuración global",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), config.GlobalPath(d.Env))
			return nil
		},
	}
}
