package commands

import (
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/config"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/presenter"
	"github.com/calliope/calliope-cli/internal/sdk"
	"github.com/calliope/calliope-cli/internal/version"
)

// Check es el resultado de una comprobación de doctor.
type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"` // ok | aviso | error
	Detail string `json:"detail"`
}

// NewDoctorCmd construye `doctor`. Nunca devuelve error: informa. Tiene que
// funcionar precisamente cuando la autenticación está rota, así que no usa
// appctx.Build.
func NewDoctorCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnostica la instalación, la credencial y la conectividad",
		Args:  NoPositionalArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.BuildWithoutCredential(cmd, d)
			if err != nil {
				// Una configuración que no carga (p. ej. un config.json
				// corrupto) es exactamente el tipo de instalación rota que
				// doctor tiene que diagnosticar, no una que lo tumbe. Sin
				// *appctx.Context no hay ctx.Render, así que se informa con lo
				// mínimo que no depende de la configuración.
				return renderBrokenConfig(cmd, d, err)
			}

			checks := []Check{{
				Name:   "versión",
				Status: "ok",
				Detail: fmt.Sprintf("calliope %s (%s)", version.Version, version.Commit),
			}, {
				Name:   "backend",
				Status: "ok",
				Detail: ctx.Cfg.BaseURL() + " (" + string(ctx.Cfg.Get(config.KeyBaseURL).Source) + ")",
			}}

			cred, origen, errCred := auth.Resolve(d.Env, d.Store)
			if errCred != nil {
				checks = append(checks, Check{
					Name:   "credencial",
					Status: "error",
					Detail: "no hay ninguna configurada; ejecuta: calliope auth login --api-key <clave>",
				})
			} else {
				checks = append(checks, Check{
					Name:   "credencial",
					Status: "ok",
					Detail: fmt.Sprintf("%s desde %s", cred.Kind, origen),
				})
			}

			org := ctx.Org
			origenOrg := string(ctx.Cfg.Get(config.KeyOrg).Source)
			if org == "" && cred.Org != "" {
				// La organización viene del fallback a la credencial (ver
				// appctx.Build), no de la configuración: etiquetarla con
				// ctx.Cfg.Get(config.KeyOrg).Source mentiría (saldría
				// "default").
				org = cred.Org
				origenOrg = "credencial"
			}
			if org == "" {
				checks = append(checks, Check{
					Name:   "organización",
					Status: "error",
					Detail: "ninguna seleccionada; ejecuta: calliope orgs use <nombre>",
				})
			} else {
				checks = append(checks, Check{
					Name:   "organización",
					Status: "ok",
					Detail: org + " (" + origenOrg + ")",
				})
			}

			if errCred == nil {
				checks = append(checks, checkConnectivity(cmd, ctx, cred))
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(checks, summaryOf(checks)),
				Text: func(w io.Writer) error {
					filas := make([][]string, 0, len(checks))
					for _, c := range checks {
						filas = append(filas, []string{symbol(c.Status), c.Name, c.Detail})
					}
					return presenter.Table(w, []string{"", "COMPROBACIÓN", "DETALLE"}, filas)
				},
			})
		},
	}
}

// renderBrokenConfig informa de que la configuración no se pudo cargar, sin
// devolver ese error tal cual: doctor tiene que seguir informando incluso
// cuando el propio config.json está roto, que es justo el tipo de
// instalación rota que existe para diagnosticar. Emite lo mínimo que no
// depende de la configuración (la versión) más el chequeo que sí falló, y
// sale con código 0 igual que el resto de doctor.
func renderBrokenConfig(cmd *cobra.Command, d appctx.Deps, errCfg error) error {
	checks := []Check{{
		Name:   "versión",
		Status: "ok",
		Detail: fmt.Sprintf("calliope %s (%s)", version.Version, version.Commit),
	}, {
		Name:   "configuración",
		Status: "error",
		Detail: fmt.Sprintf("%s (corrígelo o bórralo y vuelve a intentarlo)", errCfg.Error()),
	}}

	return presenter.Render(presenter.Result{
		Envelope: output.OKEnvelope(checks, summaryOf(checks)),
		Text: func(w io.Writer) error {
			filas := make([][]string, 0, len(checks))
			for _, c := range checks {
				filas = append(filas, []string{symbol(c.Status), c.Name, c.Detail})
			}
			return presenter.Table(w, []string{"", "COMPROBACIÓN", "DETALLE"}, filas)
		},
	}, outputModeWithoutConfig(cmd, d))
}

// outputModeWithoutConfig arma las opciones de render sin pasar por
// *config.Config: es la única vía disponible cuando la configuración es
// precisamente lo que falló al cargar. Replica appctx.OutputMode salvo por
// la capa de cfg.Output(), que aquí no existe.
func outputModeWithoutConfig(cmd *cobra.Command, d appctx.Deps) presenter.Options {
	opts := presenter.Options{Mode: presenter.ModeAuto, IsTTY: d.IsTTY, Out: d.Stdout}

	if v, _ := cmd.Flags().GetString("jq"); v != "" {
		opts.Mode, opts.JQExpr = presenter.ModeJQ, v
		return opts
	}
	if v, _ := cmd.Flags().GetBool("json"); v {
		opts.Mode = presenter.ModeJSON
	}
	if v, _ := cmd.Flags().GetBool("quiet"); v {
		opts.Mode = presenter.ModeQuiet
	}
	if v, _ := cmd.Flags().GetBool("md"); v {
		opts.Mode = presenter.ModeMarkdown
	}
	return opts
}

func checkConnectivity(cmd *cobra.Command, ctx *appctx.Context, cred auth.Credential) Check {
	cliente := sdk.New(sdk.Options{
		BaseURL:    ctx.Cfg.BaseURL(),
		Credential: cred,
		Timeout:    10 * time.Second,
		UserAgent:  "calliope-cli/" + version.Version,
	})

	inicio := time.Now()
	me, err := cliente.Me(cmd.Context())
	transcurrido := time.Since(inicio).Round(time.Millisecond)

	if err != nil {
		return Check{Name: "conectividad", Status: "error", Detail: err.Error()}
	}
	return Check{
		Name:   "conectividad",
		Status: "ok",
		Detail: fmt.Sprintf("%s en %s", me.Email, transcurrido),
	}
}

func summaryOf(cs []Check) string {
	fallos := 0
	for _, c := range cs {
		if c.Status == "error" {
			fallos++
		}
	}
	if fallos == 0 {
		return "todo correcto"
	}
	// El verbo se queda en plural ("fallan") sin acordarlo con len(cs):
	// doctor siempre emite al menos dos comprobaciones (versión + otra),
	// así que len(cs) nunca es 1 en la práctica.
	return fmt.Sprintf("%d de %s fallan", fallos, pluralize(len(cs), "comprobación", "comprobaciones"))
}

func symbol(estado string) string {
	switch estado {
	case "ok":
		return "✓"
	case "aviso":
		return "!"
	default:
		return "✗"
	}
}
