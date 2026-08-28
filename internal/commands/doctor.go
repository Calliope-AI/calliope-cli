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
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.BuildSinCredencial(cmd, d)
			if err != nil {
				return err
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
			if org == "" {
				org = cred.Org
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
					Detail: org + " (" + string(ctx.Cfg.Get(config.KeyOrg).Source) + ")",
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
	return fmt.Sprintf("%d de %d comprobaciones fallan", fallos, len(cs))
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
