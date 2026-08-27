package auth

import (
	"path/filepath"
	"testing"

	"github.com/calliope/calliope-cli/internal/output"
)

func TestElEntornoGanaAlAlmacen(t *testing.T) {
	st := NewFileStore(filepath.Join(t.TempDir(), "c.json"))
	if err := st.Save(Credential{Kind: KindAPIKey, Token: "del-almacen"}); err != nil {
		t.Fatal(err)
	}

	env := func(k string) string {
		if k == "CALLIOPE_API_KEY" {
			return "del-entorno"
		}
		return ""
	}

	c, origen, err := Resolve(env, st)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if c.Token != "del-entorno" {
		t.Errorf("token = %q, la variable de entorno debe ganar", c.Token)
	}
	if origen == "" {
		t.Error("se esperaba una descripción del origen")
	}
}

func TestSinCredencialDevuelveErrorConHint(t *testing.T) {
	st := NewFileStore(filepath.Join(t.TempDir(), "c.json"))
	vacio := func(string) string { return "" }

	_, _, err := Resolve(vacio, st)
	if err == nil {
		t.Fatal("se esperaba un error al no haber credencial")
	}
	if got := output.ExitCodeFor(err); got != 3 {
		t.Errorf("código de salida = %d, se esperaba 3 (no autorizado)", got)
	}

	var cliErr *output.CLIError
	if !asCLIError(err, &cliErr) {
		t.Fatal("se esperaba un *output.CLIError")
	}
	if cliErr.Hint == "" {
		t.Error("el error debe decir cómo autenticarse")
	}
}

func asCLIError(err error, target **output.CLIError) bool {
	e, ok := err.(*output.CLIError)
	if ok {
		*target = e
	}
	return ok
}
