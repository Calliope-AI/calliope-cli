package cli

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
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

func TestNingunGrupoDeRecursosDefineRunE(t *testing.T) {
	root := NewRootCmd(appctx.Deps{})

	for _, c := range root.Commands() {
		if len(c.Commands()) == 0 {
			continue // es un atajo, no un grupo
		}
		if c.RunE != nil || c.Run != nil {
			t.Errorf("%q tiene subcomandos y además define RunE; invocarlo pelado debe mostrar la ayuda", c.Name())
		}
	}
}

func noop(cmd *cobra.Command, args []string) error { return nil }
