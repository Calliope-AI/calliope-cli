package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/config"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/presenter"
)

// NewOrgsCmd construye el grupo `orgs`. Invocado pelado muestra la ayuda
// (exit 0); con un subcomando que no existe, un error de uso (exit 2): ver
// groupRunE en args.go.
func NewOrgsCmd(d appctx.Deps) *cobra.Command {
	grupo := &cobra.Command{
		Use:   "orgs",
		Short: "Lista y selecciona la organización activa",
		RunE:  groupRunE,
	}
	grupo.AddCommand(newOrgsListCmd(d), newOrgsUseCmd(d))
	return grupo
}

func newOrgsListCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lista las organizaciones accesibles con tu credencial",
		Args:  NoPositionalArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Listar organizaciones no requiere tener una elegida.
			ctx, err := appctx.BuildWithoutCredential(cmd, d)
			if err != nil {
				return err
			}
			cred, _, err := authResolve(d)
			if err != nil {
				return err
			}
			cliente := clientWith(ctx, cred)

			orgs, err := cliente.ListOrganizations(cmd.Context())
			if err != nil {
				return err
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(orgs, pluralize(len(orgs), "organización", "organizaciones"),
					output.Breadcrumb{Action: "usar", Cmd: "calliope orgs use <nombre>"}),
				Text: func(w io.Writer) error {
					filas := make([][]string, 0, len(orgs))
					for _, o := range orgs {
						filas = append(filas, []string{o.Name, o.ID})
					}
					return presenter.Table(w, []string{"NOMBRE", "ID"}, filas)
				},
			})
		},
	}
}

func newOrgsUseCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "use <organización>",
		Short: "Fija la organización activa en este directorio",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := filepath.Join(d.Cwd, ".calliope")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return output.WrapIOError("No se pudo crear el directorio .calliope del proyecto.",
					"Comprueba los permisos de escritura en el directorio actual.", err)
			}
			ruta := filepath.Join(dir, config.FileName)

			// Se conservan los valores que ya hubiera en el fichero.
			vals := map[string]string{}
			if b, err := os.ReadFile(ruta); err == nil {
				_ = json.Unmarshal(b, &vals)
			}
			vals[config.KeyOrg] = args[0]

			b, err := json.MarshalIndent(vals, "", "  ")
			if err != nil {
				return err
			}
			if err := os.WriteFile(ruta, b, 0o600); err != nil {
				return output.WrapIOError("No se pudo guardar la organización activa.",
					"Comprueba los permisos de escritura en el directorio actual.", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Organización activa: %s (%s)\n", args[0], ruta)
			return nil
		},
	}
}
