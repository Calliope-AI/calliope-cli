package commands

import (
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/presenter"
	"github.com/calliope/calliope-cli/internal/sdk"
)

// NewDocsCmd construye el grupo `docs`. Invocado pelado muestra la ayuda
// (exit 0); con un subcomando que no existe, un error de uso (exit 2): ver
// groupRunE en args.go.
func NewDocsCmd(d appctx.Deps) *cobra.Command {
	grupo := &cobra.Command{
		Use:   "docs",
		Short: "Consulta la documentación de la organización",
		RunE:  groupRunE,
	}
	grupo.AddCommand(newDocsListCmd(d), newDocsShowCmd(d), newDocsSearchCmd(d))
	return grupo
}

func newDocsListCmd(d appctx.Deps) *cobra.Command {
	var p sdk.ListDocumentsParams
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lista los documentos disponibles",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}
			page, err := ctx.Client.ListDocuments(cmd.Context(), ctx.Org, p)
			if err != nil {
				return err
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(page.Content,
					fmt.Sprintf("%d de %d documentos", len(page.Content), page.TotalSize),
					output.Breadcrumb{Action: "detalle", Cmd: "calliope docs show <id>"},
					output.Breadcrumb{Action: "buscar", Cmd: "calliope docs search \"<consulta>\""}),
				Text: func(w io.Writer) error {
					filas := make([][]string, 0, len(page.Content))
					for _, doc := range page.Content {
						filas = append(filas, []string{doc.ID, titleOf(doc), doc.Status, strconv.FormatInt(doc.SizeBytes, 10)})
					}
					return presenter.Table(w, []string{"ID", "TÍTULO", "ESTADO", "BYTES"}, filas)
				},
			})
		},
	}
	f := cmd.Flags()
	f.StringVar(&p.Status, "status", "", "filtra por estado (READY, PROCESSING, FAILED…)")
	f.StringVar(&p.Tag, "tag", "", "filtra por etiqueta")
	f.StringVar(&p.Q, "q", "", "filtra por texto")
	f.IntVar(&p.Page, "page", 0, "página (base 1)")
	f.IntVar(&p.Size, "size", 0, "tamaño de página")
	return cmd
}

func newDocsShowCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Muestra los metadatos de un documento",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}
			doc, err := ctx.Client.GetDocument(cmd.Context(), ctx.Org, args[0])
			if err != nil {
				return err
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(doc, titleOf(*doc),
					output.Breadcrumb{Action: "buscar dentro", Cmd: "calliope docs search \"<consulta>\""}),
				Text: func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "%s\nFichero: %s\nEstado: %s\nCreado: %s\n",
						titleOf(*doc), doc.Filename, doc.Status, doc.CreatedAt)
					return err
				},
			})
		},
	}
}

func newDocsSearchCmd(d appctx.Deps) *cobra.Command {
	var limite int
	cmd := &cobra.Command{
		Use:   "search <consulta>",
		Short: "Búsqueda semántica en la documentación",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}
			res, err := ctx.Client.SearchDocuments(cmd.Context(), ctx.Org, args[0], limite)
			if err != nil {
				return err
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(res, fmt.Sprintf("%d fragmentos", len(res)),
					output.Breadcrumb{Action: "documento", Cmd: "calliope docs show <id>"}),
				Text: func(w io.Writer) error {
					filas := make([][]string, 0, len(res))
					for _, r := range res {
						filas = append(filas, []string{
							r.DocumentID,
							strconv.FormatFloat(r.Score, 'f', 3, 64),
							truncate(r.Excerpt, 70),
						})
					}
					return presenter.Table(w, []string{"DOCUMENTO", "SCORE", "FRAGMENTO"}, filas)
				},
			})
		},
	}
	cmd.Flags().IntVar(&limite, "limit", 10, "número máximo de fragmentos")
	return cmd
}

func titleOf(doc sdk.DocumentResponse) string {
	if doc.Title != nil && *doc.Title != "" {
		return *doc.Title
	}
	return doc.Filename
}

// truncate recorta s a n runas como máximo, añadiendo puntos suspensivos si
// se ha cortado algo. La Task 16 reutiliza esta misma función.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
