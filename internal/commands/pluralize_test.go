package commands

import "testing"

// TestPluralize cubre los Diferidos #12 y #16 de la oleada final: 1 usa la
// forma singular; 0 y cualquier valor distinto de 1 usan la plural.
func TestPluralize(t *testing.T) {
	casos := []struct {
		n      int
		quiero string
	}{
		{0, "0 filas"},
		{1, "1 fila"},
		{2, "2 filas"},
		{-1, "-1 filas"},
	}
	for _, c := range casos {
		if got := pluralize(c.n, "fila", "filas"); got != c.quiero {
			t.Errorf("pluralize(%d, \"fila\", \"filas\") = %q, se esperaba %q", c.n, got, c.quiero)
		}
	}
}
