package commands

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/presenter"
	"github.com/calliope/calliope-cli/internal/sdk"
)

// NewSchemaCmd construye `schema`. Es un atajo, no un grupo, así que define
// RunE. El SKILL.md obliga a los agentes a ejecutarlo antes de escribir SQL.
func NewSchemaCmd(d appctx.Deps) *cobra.Command {
	var tabla string
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Muestra el esquema de la base de datos de la organización",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}
			esquema, err := ctx.Client.Schema(cmd.Context(), ctx.Org)
			if err != nil {
				return err
			}

			tablas := esquema.Tables
			if tabla != "" {
				// El filtro es en cliente: el backend no expone filtrado por
				// tabla. Si el esquema crece mucho, ese endpoint sería el
				// siguiente paso (ver la sección 15 del spec).
				tablas = filterTables(tablas, tabla)
				if len(tablas) == 0 {
					return output.NewError(output.CodeNotFound,
						fmt.Sprintf("No existe la tabla %q en el esquema.", tabla),
						"Lista todas las tablas con: calliope schema")
				}
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(tablas, fmt.Sprintf("%d tablas", len(tablas)),
					output.Breadcrumb{Action: "consultar", Cmd: "calliope query \"SELECT …\""},
					output.Breadcrumb{Action: "preguntar", Cmd: "calliope ask \"<pregunta>\""}),
				Text: func(w io.Writer) error {
					for _, t := range tablas {
						// TableName es el identificador real que va en el
						// SQL: se muestra primero y con etiqueta explícita.
						// Si Name (el nombre de negocio) difiere, se muestra
						// aparte y también etiquetado, para que un agente no
						// confunda cuál va en la consulta.
						if _, err := fmt.Fprintf(w, "\n%s  [tabla SQL]\n", t.TableName); err != nil {
							return err
						}
						if t.Name != "" && t.Name != t.TableName {
							if _, err := fmt.Fprintf(w, "  nombre de negocio: %s\n", t.Name); err != nil {
								return err
							}
						}
						filas := make([][]string, 0, len(t.Columns))
						for _, c := range t.Columns {
							filas = append(filas, []string{c.Name, c.Type, c.Description})
						}
						if err := presenter.Table(w, []string{"COLUMNA", "TIPO", "DESCRIPCIÓN"}, filas); err != nil {
							return err
						}
					}
					return nil
				},
			})
		},
	}
	cmd.Flags().StringVar(&tabla, "table", "", "muestra solo esta tabla")
	return cmd
}

// filterTables filtra tablas por TableName -el identificador real que va en
// el SQL-. También casa contra Name -el nombre de negocio- como comodidad,
// por si alguien filtra con el nombre que ve en la interfaz en vez del
// identificador SQL. Sin distinguir mayúsculas de minúsculas.
func filterTables(tablas []sdk.SchemaTable, nombre string) []sdk.SchemaTable {
	var out []sdk.SchemaTable
	for _, t := range tablas {
		if strings.EqualFold(t.TableName, nombre) || strings.EqualFold(t.Name, nombre) {
			out = append(out, t)
		}
	}
	return out
}

// NewQueryCmd construye `query`, el SQL crudo. El SKILL.md establece `ask`
// como vía por defecto; esto es el escape para cuando `ask` no basta.
func NewQueryCmd(d appctx.Deps) *cobra.Command {
	var formato string
	var comoCSV bool

	cmd := &cobra.Command{
		Use:   "query <SQL>",
		Short: "Ejecuta SQL contra los datos de la organización",
		Long: "Ejecuta SQL contra los datos de la organización.\n\n" +
			"Ejecuta antes `calliope schema` para conocer las tablas reales.\n" +
			"Para preguntas de negocio, prefiere `calliope ask`.",
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}

			resp, err := ctx.Client.Query(cmd.Context(), ctx.Org, args[0], formato)
			if err != nil {
				return err
			}
			filas, err := resp.Rows()
			if err != nil {
				return output.NewError(output.CodeGeneric,
					"No se pudo interpretar el resultado de la consulta.",
					"Prueba con --json para ver la respuesta cruda del backend")
			}

			columnas := columnsOf(filas)

			if comoCSV {
				// --csv es render local y no toca el cuerpo de la petición.
				return writeCSV(ctx.Deps.Stdout, columnas, filas)
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(filas, fmt.Sprintf("%d filas", len(filas)),
					output.Breadcrumb{Action: "esquema", Cmd: "calliope schema"}),
				Text: func(w io.Writer) error {
					tabla := make([][]string, 0, len(filas))
					for _, f := range filas {
						fila := make([]string, 0, len(columnas))
						for _, c := range columnas {
							fila = append(fila, fmt.Sprintf("%v", f[c]))
						}
						tabla = append(tabla, fila)
					}
					return presenter.Table(w, columnas, tabla)
				},
			})
		},
	}
	cmd.Flags().StringVar(&formato, "output", "", "formato que se pide al backend (se reenvía en QueryRequest.output)")
	cmd.Flags().BoolVar(&comoCSV, "csv", false, "render local del resultado en CSV")
	return cmd
}

// columnsOf deduce las columnas de las filas, en orden estable.
func columnsOf(filas []map[string]any) []string {
	vistas := map[string]bool{}
	var cols []string
	for _, f := range filas {
		for k := range f {
			if !vistas[k] {
				vistas[k] = true
				cols = append(cols, k)
			}
		}
	}
	sort.Strings(cols)
	return cols
}

func writeCSV(w io.Writer, columnas []string, filas []map[string]any) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(columnas); err != nil {
		return err
	}
	for _, f := range filas {
		registro := make([]string, 0, len(columnas))
		for _, c := range columnas {
			registro = append(registro, fmt.Sprintf("%v", f[c]))
		}
		if err := cw.Write(registro); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
