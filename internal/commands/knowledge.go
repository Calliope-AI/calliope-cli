package commands

import (
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/Calliope-AI/calliope-cli/internal/appctx"
	"github.com/Calliope-AI/calliope-cli/internal/output"
	"github.com/Calliope-AI/calliope-cli/internal/presenter"
)

// NewConceptsCmd construye el grupo `concepts`: qué datos existen, en lenguaje
// de negocio. Es lo primero que debe mirar un agente antes de preguntar.
// Invocado pelado muestra la ayuda (exit 0); con un subcomando que no
// existe, un error de uso (exit 2): ver groupRunE en args.go.
func NewConceptsCmd(d appctx.Deps) *cobra.Command {
	grupo := &cobra.Command{
		Use:   "concepts",
		Short: "Explora los conceptos de negocio de la ontología",
		RunE:  groupRunE,
	}
	grupo.AddCommand(newConceptsListCmd(d), newConceptsShowCmd(d))
	return grupo
}

func newConceptsListCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lista los conceptos de negocio",
		Args:  NoPositionalArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}
			grafo, err := ctx.Client.ListConcepts(cmd.Context(), ctx.Org)
			if err != nil {
				return err
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(grafo.Concepts,
					pluralize(len(grafo.Concepts), "concepto", "conceptos"),
					output.Breadcrumb{Action: "detalle", Cmd: "calliope concepts show <id>"},
					output.Breadcrumb{Action: "preguntar", Cmd: "calliope ask \"<pregunta>\""}),
				Text: func(w io.Writer) error {
					filas := make([][]string, 0, len(grafo.Concepts))
					for _, c := range grafo.Concepts {
						registros := "—"
						if c.RecordCount != nil {
							registros = strconv.Itoa(*c.RecordCount)
						}
						filas = append(filas, []string{c.ID, c.Name, registros, yesNo(c.IsActive)})
					}
					return presenter.Table(w, []string{"ID", "CONCEPTO", "REGISTROS", "ACTIVO"}, filas)
				},
			})
		},
	}
}

func newConceptsShowCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Muestra un concepto y sus atributos",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}
			det, err := ctx.Client.GetConcept(cmd.Context(), ctx.Org, args[0])
			if err != nil {
				return err
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(det,
					fmt.Sprintf("%s · %s", det.Concept.Name, pluralize(len(det.Attributes), "atributo", "atributos")),
					output.Breadcrumb{Action: "preguntar", Cmd: "calliope ask \"<pregunta sobre " + det.Concept.Name + ">\""}),
				Text: func(w io.Writer) error {
					if _, err := fmt.Fprintf(w, "%s\n", det.Concept.Name); err != nil {
						return err
					}
					if det.Concept.Description != nil && *det.Concept.Description != "" {
						if _, err := fmt.Fprintf(w, "%s\n", *det.Concept.Description); err != nil {
							return err
						}
					}
					fmt.Fprintln(w)
					filas := make([][]string, 0, len(det.Attributes))
					for _, a := range det.Attributes {
						desc := ""
						if a.Description != nil {
							desc = *a.Description
						}
						filas = append(filas, []string{a.Name, desc, yesNo(a.IsActive)})
					}
					return presenter.Table(w, []string{"ATRIBUTO", "DESCRIPCIÓN", "ACTIVO"}, filas)
				},
			})
		},
	}
}

// NewRulesCmd construye el grupo `rules`. Invocado pelado muestra la ayuda
// (exit 0); con un subcomando que no existe, un error de uso (exit 2): ver
// groupRunE en args.go.
func NewRulesCmd(d appctx.Deps) *cobra.Command {
	grupo := &cobra.Command{
		Use:   "rules",
		Short: "Consulta las reglas de negocio compartidas",
		RunE:  groupRunE,
	}
	grupo.AddCommand(newRulesListCmd(d))
	return grupo
}

func newRulesListCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lista las reglas de negocio de la organización",
		Args:  NoPositionalArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}
			reglas, err := ctx.Client.ListRules(cmd.Context(), ctx.Org)
			if err != nil {
				return err
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(reglas, pluralize(len(reglas), "regla", "reglas"),
					output.Breadcrumb{Action: "conceptos", Cmd: "calliope concepts list"}),
				Text: func(w io.Writer) error {
					filas := make([][]string, 0, len(reglas))
					for _, r := range reglas {
						cat := ""
						if r.Category != nil {
							cat = *r.Category
						}
						filas = append(filas, []string{r.Name, cat, r.Status, truncate(r.Details, 60)})
					}
					return presenter.Table(w, []string{"REGLA", "CATEGORÍA", "ESTADO", "DETALLE"}, filas)
				},
			})
		},
	}
}

func yesNo(b bool) string {
	if b {
		return "sí"
	}
	return "no"
}
