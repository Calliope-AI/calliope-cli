package cli

import (
	"bytes"
	"errors"
	"testing"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/output"
)

func TestCatalogDevuelveSoloLasHojas(t *testing.T) {
	root := &cobra.Command{Use: "calliope"}
	grupo := &cobra.Command{Use: "docs"}
	grupo.AddCommand(&cobra.Command{Use: "list", Short: "lista", RunE: noop})
	grupo.AddCommand(&cobra.Command{Use: "show <id>", Short: "muestra", RunE: noop})
	root.AddCommand(grupo)
	root.AddCommand(&cobra.Command{Use: "ask <pregunta>", Short: "pregunta", RunE: noop})

	cat := Catalog(root)
	quiero := []string{"ask", "docs list", "docs show"}
	if len(cat) != len(quiero) {
		t.Fatalf("Catalog devolvió %d entradas, se esperaban %d: %+v", len(cat), len(quiero), cat)
	}
	for i, q := range quiero {
		if cat[i].Path != q {
			t.Errorf("cat[%d].Path = %q, se esperaba %q", i, cat[i].Path, q)
		}
	}
}

// TestGrupoDeRecursosMuestraAyudaPeladoYFallaConSubcomandoDesconocido es la
// reescritura de C3 de la oleada final. El test original
// (TestNingunGrupoDeRecursosDefineRunE) aseveraba el MECANISMO del §5 del
// spec ("ningún grupo define RunE"), no el comportamiento que ese mecanismo
// perseguía -y ahí se coló el bug: cobra decide si un comando es Runnable()
// (y por tanto si cae a flag.ErrHelp, que muestra la ayuda con exit 0) ANTES
// de validar los argumentos, así que un grupo sin RunE no podía distinguir
// "invocado sin argumentos" de "invocado con un subcomando que no existe".
// "calliope docs typo" mostraba la misma ayuda que "calliope docs" solo, con
// exit 0 en los seis grupos.
//
// Este test asevera el comportamiento correcto por sí mismo, ejecutando el
// árbol real de comandos con ambos argumentos por cada uno de los seis
// grupos, así que no puede volver a colarse un bug que preserve el mecanismo
// (RunE ausente o presente) pero rompa el comportamiento.
func TestGrupoDeRecursosMuestraAyudaPeladoYFallaConSubcomandoDesconocido(t *testing.T) {
	grupos := []string{"auth", "orgs", "config", "docs", "concepts", "rules"}

	for _, g := range grupos {
		t.Run(g+"/pelado muestra ayuda con exit 0", func(t *testing.T) {
			root := NewRootCmd(appctx.Deps{})
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&out)
			root.SetArgs([]string{g})

			if err := root.Execute(); err != nil {
				t.Fatalf("%q pelado debe salir con 0 (ayuda), dio error: %v", g, err)
			}
			if !bytes.Contains(out.Bytes(), []byte("Usage:")) {
				t.Errorf("%q pelado debe imprimir la ayuda (con \"Usage:\"), se obtuvo: %q", g, out.String())
			}
		})

		t.Run(g+"/subcomando desconocido falla con exit 2", func(t *testing.T) {
			root := NewRootCmd(appctx.Deps{})
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&out)
			root.SetArgs([]string{g, "esto-no-es-un-subcomando"})

			err := root.Execute()
			if err == nil {
				t.Fatalf("%q con un subcomando desconocido debe fallar", g)
			}

			var cliErr *output.CLIError
			if !errors.As(err, &cliErr) {
				t.Fatalf("%q: el error debería ser un *output.CLIError, fue %T (%v)", g, err, err)
			}
			if cliErr.Code != output.CodeUsage {
				t.Errorf("%q: código = %v, se esperaba %v (uso incorrecto)", g, cliErr.Code, output.CodeUsage)
			}
			if cliErr.Hint == "" {
				t.Errorf("%q: el error debería traer un hint con los subcomandos disponibles", g)
			}
			if got := output.ExitCodeFor(err); got != 2 {
				t.Errorf("%q: código de salida = %d, se esperaba 2 (uso incorrecto)", g, got)
			}
		})
	}
}

func noop(cmd *cobra.Command, args []string) error { return nil }
