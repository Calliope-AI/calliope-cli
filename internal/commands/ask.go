package commands

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/Calliope-AI/calliope-cli/internal/appctx"
	"github.com/Calliope-AI/calliope-cli/internal/output"
	"github.com/Calliope-AI/calliope-cli/internal/presenter"
	"github.com/Calliope-AI/calliope-cli/internal/sdk"
)

// NewAskCmd construye `ask`, la vía por defecto para consultar los datos.
func NewAskCmd(d appctx.Deps) *cobra.Command {
	var accion string
	cmd := &cobra.Command{
		Use:   "ask <pregunta>",
		Short: "Pregunta en lenguaje natural sobre tus datos y tu documentación",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}

			resp, err := ctx.Client.Ask(cmd.Context(), ctx.Org, args[0], accion)
			if err != nil {
				return err
			}
			if !resp.Success {
				// El mensaje es fijo a propósito, aunque el backend traiga
				// uno en resp.Error: la política de todo el SDK (ver
				// mapStatus y transportError en internal/sdk/client.go)
				// nunca reenvía el cuerpo de una respuesta del backend tal
				// cual, porque puede filtrar nombres de tabla internos,
				// hostnames o puertos de su infraestructura por el stdout
				// de un CLI que corre en la máquina del cliente. Esta era
				// la única grieta en esa política: resp.Error SÍ es cuerpo
				// de respuesta, solo que llega dentro de un 200 en vez de
				// en un status de error.
				return output.NewError(output.CodeGeneric,
					"Calliope no pudo responder a la pregunta.",
					"Reformula la pregunta, o mira qué datos existen con: calliope concepts list")
			}

			resumen := pluralize(len(resp.Sources), "fuente citada", "fuentes citadas")
			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(resp, resumen,
					output.Breadcrumb{Action: "documento", Cmd: "calliope docs show <id>"},
					output.Breadcrumb{Action: "conceptos", Cmd: "calliope concepts list"}),
				Text:     func(w io.Writer) error { return writeAskText(w, resp) },
				Markdown: func(w io.Writer) error { return writeAskMarkdown(w, resp) },
			})
		},
	}
	cmd.Flags().StringVar(&accion, "action", "", "tipo de análisis a solicitar")
	return cmd
}

func writeAskText(w io.Writer, r *sdk.AskResponse) error {
	if _, err := fmt.Fprintln(w, r.Text); err != nil {
		return err
	}
	if len(r.Sources) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, "\nFuentes:"); err != nil {
		return err
	}
	for _, s := range r.Sources {
		if _, err := fmt.Fprintf(w, "  · %s (%s)\n", s.Citation, s.DocumentID); err != nil {
			return err
		}
	}
	return nil
}

func writeAskMarkdown(w io.Writer, r *sdk.AskResponse) error {
	if _, err := fmt.Fprintf(w, "%s\n", r.Text); err != nil {
		return err
	}
	if len(r.Sources) == 0 {
		return nil
	}
	// Las fuentes se citan siempre: es la invariante 5 del SKILL.md.
	if _, err := fmt.Fprint(w, "\n### Fuentes\n\n"); err != nil {
		return err
	}
	for _, s := range r.Sources {
		titulo := s.Filename
		if s.DocumentTitle != nil && *s.DocumentTitle != "" {
			titulo = *s.DocumentTitle
		}
		if _, err := fmt.Fprintf(w, "- **%s** — %s (`%s`)\n", titulo, s.Citation, s.DocumentID); err != nil {
			return err
		}
	}
	return nil
}
