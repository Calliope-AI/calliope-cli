// Package commands define los comandos de calliope, uno por fichero de grupo.
package commands

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/presenter"
	"github.com/calliope/calliope-cli/internal/sdk"
	"github.com/calliope/calliope-cli/internal/version"
)

// NewAuthCmd construye el grupo `auth`. Invocado pelado muestra la ayuda
// (exit 0); con un subcomando que no existe, un error de uso (exit 2): ver
// groupRunE en args.go.
func NewAuthCmd(d appctx.Deps) *cobra.Command {
	grupo := &cobra.Command{
		Use:   "auth",
		Short: "Gestiona la autenticación con Calliope Data",
		RunE:  groupRunE,
	}
	grupo.AddCommand(newAuthLoginCmd(d), newAuthLogoutCmd(d), newAuthStatusCmd(d), newAuthTokenCmd(d))
	return grupo
}

func newAuthLoginCmd(d appctx.Deps) *cobra.Command {
	var apiKey string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Guarda y verifica una credencial de Calliope",
		Args:  NoPositionalArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if apiKey == "" {
				return output.NewError(output.CodeUsage,
					"Falta la credencial.",
					"Ejecuta: calliope auth login --api-key <clave>  (créala en el UI, en Observabilidad → Claves API)")
			}

			cred := auth.Credential{Kind: auth.KindAPIKey, Token: apiKey}

			// Se valida ANTES de guardar: nunca se persiste una credencial
			// no verificada.
			ctx, err := appctx.BuildWithoutCredential(cmd, d)
			if err != nil {
				return err
			}
			cliente := clientWith(ctx, cred)
			me, err := cliente.Me(cmd.Context())
			if err != nil {
				return err
			}

			if err := d.Store.Save(cred); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Sesión iniciada como %s.\n", me.Email)
			return nil
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "", "clave de API de Calliope")
	return cmd
}

func newAuthLogoutCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Borra la credencial almacenada",
		Args:  NoPositionalArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.Store.Delete(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Credencial borrada.")
			return nil
		},
	}
}

func newAuthStatusCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Muestra quién eres y de dónde sale la credencial",
		Args:  NoPositionalArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}
			me, err := ctx.Client.Me(cmd.Context())
			if err != nil {
				return err
			}

			datos := map[string]string{
				"email":        me.Email,
				"userId":       me.ID,
				"credencial":   string(ctx.Cred.Kind),
				"origen":       ctx.CredSource,
				"organizacion": ctx.Org,
			}
			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(datos, "autenticado como "+me.Email,
					output.Breadcrumb{Action: "organizaciones", Cmd: "calliope orgs list"}),
				Text: func(w io.Writer) error {
					// El token nunca se imprime aquí; para eso está `auth token`.
					_, err := fmt.Fprintf(w, "Autenticado como %s\nCredencial: %s (%s)\nOrganización: %s\n",
						me.Email, ctx.Cred.Kind, ctx.CredSource, ctx.Org)
					return err
				},
			})
		},
	}
}

func newAuthTokenCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "token",
		Short: "Imprime la credencial almacenada (para scripts)",
		Args:  NoPositionalArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			cred, _, err := auth.Resolve(d.Env, d.Store)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), cred.Token)
			return nil
		},
	}
}

// authResolve y clientWith evitan repetir el mismo wiring en los comandos que
// necesitan cliente sin exigir organización.
func authResolve(d appctx.Deps) (auth.Credential, string, error) {
	return auth.Resolve(d.Env, d.Store)
}

// clientWith construye un cliente SDK con la credencial dada -en vez de la
// que resolvería appctx.Build- para los dos comandos que necesitan hablar
// con el backend sin exigir todavía una organización u otra credencial ya
// resuelta: auth login valida la credencial antes de guardarla, y orgs list
// lista organizaciones sin exigir una ya seleccionada.
//
// M1 de la oleada final: antes se construía sin Timeout ni UserAgent -a
// diferencia del cliente que monta appctx.Build-, así que auth login y orgs
// list ignoraban CALLIOPE_TIMEOUT y mandaban un User-Agent sin versión al
// backend. Ahora usa los mismos appctx.TimeoutOf(ctx.Cfg) y
// "calliope-cli/"+version.Version que appctx.Build, para que solo haya un
// sitio que sepa cómo se construye un cliente SDK "de verdad".
func clientWith(ctx *appctx.Context, cred auth.Credential) *sdk.Client {
	return sdk.New(sdk.Options{
		BaseURL:    ctx.Cfg.BaseURL(),
		Credential: cred,
		Timeout:    appctx.TimeoutOf(ctx.Cfg),
		UserAgent:  "calliope-cli/" + version.Version,
	})
}
