// Package commands define los comandos de calliope, uno por fichero de grupo.
package commands

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/Calliope-AI/calliope-cli/internal/appctx"
	"github.com/Calliope-AI/calliope-cli/internal/auth"
	"github.com/Calliope-AI/calliope-cli/internal/output"
	"github.com/Calliope-AI/calliope-cli/internal/presenter"
	"github.com/Calliope-AI/calliope-cli/internal/sdk"
	"github.com/Calliope-AI/calliope-cli/internal/version"
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
			// No exige organización: es el comando que se ejecuta para
			// saber cuáles hay. Pedirla aquí lo dejaba inservible justo
			// cuando más se necesita -al empezar, o al no saber qué alcance
			// tiene la clave-. Mismo patrón que auth login y orgs list.
			ctx, err := appctx.BuildWithoutCredential(cmd, d)
			if err != nil {
				return err
			}
			cred, origen, err := authResolve(d)
			if err != nil {
				return err
			}
			me, err := clientWith(ctx, cred).Me(cmd.Context())
			if err != nil {
				return err
			}

			// El alcance de la credencial es lo que responde el backend; la
			// organización activa es una preferencia local. Mezclarlos bajo
			// una sola etiqueta "organizacion" hacía creer que una clave
			// estaba acotada cuando no lo estaba.
			alcance := make([]orgDelUsuario, 0, len(me.Organizations))
			for _, o := range me.Organizations {
				alcance = append(alcance, orgDelUsuario{Name: o.Name, ID: o.ID, Rol: o.UserRole})
			}
			datos := estadoDeAutenticacion{
				Email:              me.Email,
				UserID:             me.ID,
				Credencial:         string(cred.Kind),
				Origen:             origen,
				OrganizacionActiva: ctx.Org,
				Organizaciones:     alcance,
			}
			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(datos, "autenticado como "+me.Email,
					output.Breadcrumb{Action: "organizaciones", Cmd: "calliope orgs list"}),
				Text: func(w io.Writer) error {
					// El token nunca se imprime aquí; para eso está `auth token`.
					if _, err := fmt.Fprintf(w, "Autenticado como %s\nCredencial: %s (%s)\nOrganización activa: %s\n",
						me.Email, cred.Kind, origen, orgActivaODefecto(ctx.Org)); err != nil {
						return err
					}
					if len(alcance) == 0 {
						_, err := fmt.Fprintln(w, "El backend no informa de ninguna organización para esta credencial.")
						return err
					}
					if _, err := fmt.Fprintf(w, "\nLa credencial alcanza a %s:\n", pluralize(len(alcance), "organización", "organizaciones")); err != nil {
						return err
					}
					for _, o := range alcance {
						linea := "  " + o.Name
						if o.Rol != "" {
							linea += " (" + o.Rol + ")"
						}
						if _, err := fmt.Fprintln(w, linea); err != nil {
							return err
						}
					}
					return nil
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

// estadoDeAutenticacion es lo que emite `auth status`. Separa a propósito la
// organización activa -preferencia local- del alcance real de la credencial,
// que es lo que responde el backend.
type estadoDeAutenticacion struct {
	Email              string          `json:"email"`
	UserID             string          `json:"userId"`
	Credencial         string          `json:"credencial"`
	Origen             string          `json:"origen"`
	OrganizacionActiva string          `json:"organizacionActiva"`
	Organizaciones     []orgDelUsuario `json:"organizaciones"`
}

type orgDelUsuario struct {
	Name string `json:"name"`
	ID   string `json:"id,omitempty"`
	Rol  string `json:"rol,omitempty"`
}

// orgActivaODefecto evita imprimir una línea "Organización activa:" vacía
// cuando todavía no se ha elegido ninguna.
func orgActivaODefecto(org string) string {
	if org == "" {
		return "(ninguna elegida)"
	}
	return org
}
