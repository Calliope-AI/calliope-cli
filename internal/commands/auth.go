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
)

// NewAuthCmd construye el grupo `auth`. Sin RunE: invocarlo pelado muestra la
// ayuda, que es lo que hace Cobra con un comando que solo tiene subcomandos.
func NewAuthCmd(d appctx.Deps) *cobra.Command {
	grupo := &cobra.Command{
		Use:   "auth",
		Short: "Gestiona la autenticación con Calliope Data",
	}
	grupo.AddCommand(newAuthLoginCmd(d), newAuthLogoutCmd(d), newAuthStatusCmd(d), newAuthTokenCmd(d))
	return grupo
}

func newAuthLoginCmd(d appctx.Deps) *cobra.Command {
	var apiKey string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Guarda y verifica una credencial de Calliope",
		RunE: func(cmd *cobra.Command, args []string) error {
			if apiKey == "" {
				return output.NewError(output.CodeUsage,
					"Falta la credencial.",
					"Ejecuta: calliope auth login --api-key <clave>  (créala en el UI, en Observabilidad → Claves API)")
			}

			cred := auth.Credential{Kind: auth.KindAPIKey, Token: apiKey}

			// Se valida ANTES de guardar: nunca se persiste una credencial
			// no verificada.
			ctx, err := appctx.BuildSinCredencial(cmd, d)
			if err != nil {
				return err
			}
			cliente := sdk.New(sdk.Options{BaseURL: ctx.Cfg.BaseURL(), Credential: cred})
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

func clientWith(ctx *appctx.Context, cred auth.Credential) *sdk.Client {
	return sdk.New(sdk.Options{BaseURL: ctx.Cfg.BaseURL(), Credential: cred})
}
