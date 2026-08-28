package commands

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/output"
)

func TestOrgsEsUnGrupoSinRunE(t *testing.T) {
	cmd := NewOrgsCmd(appctx.Deps{})
	if cmd.RunE != nil || cmd.Run != nil {
		t.Error("orgs es un grupo: invocarlo pelado debe mostrar la ayuda, no ejecutar nada")
	}
	if len(cmd.Commands()) == 0 {
		t.Error("orgs debe tener subcomandos")
	}
}

func TestOrgsListDevuelveLasOrganizaciones(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations" {
			t.Errorf("ruta = %q", r.URL.Path)
		}
		w.Write([]byte(`[{"id":"o-1","name":"acme"},{"id":"o-2","name":"globex"}]`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewOrgsCmd(d), stdout)
	root.SetArgs([]string{"orgs", "list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("orgs list: %v", err)
	}

	var env struct {
		OK   bool `json:"ok"`
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if len(env.Data) != 2 || env.Data[0].Name != "acme" {
		t.Errorf("data inesperada: %q", stdout.String())
	}
}

func TestOrgsUseEscribeLaConfiguracionDeProyecto(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewOrgsCmd(d), stdout)
	root.SetArgs([]string{"orgs", "use", "globex"})
	if err := root.Execute(); err != nil {
		t.Fatalf("orgs use: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(d.Cwd, ".calliope", "config.json"))
	if err != nil {
		t.Fatalf("no se escribió la configuración de proyecto: %v", err)
	}
	var vals map[string]string
	if err := json.Unmarshal(b, &vals); err != nil {
		t.Fatal(err)
	}
	if vals["org"] != "globex" {
		t.Errorf("config = %v, se esperaba org=globex", vals)
	}
}

// TestOrgsUsePreservaLasClavesPrevias comprueba que `orgs use` hace merge con
// lo que ya hubiera en .calliope/config.json en vez de sobrescribirlo entero:
// sin el merge, la suite seguía en verde porque ningún test anterior sembraba
// el fichero antes de ejecutar el comando.
func TestOrgsUsePreservaLasClavesPrevias(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(d.Cwd, ".calliope")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	previo, err := json.Marshal(map[string]string{"output": "json", "timeout": "30s"})
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, "config.json")
	if err := os.WriteFile(ruta, previo, 0o600); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewOrgsCmd(d), stdout)
	root.SetArgs([]string{"orgs", "use", "globex"})
	if err := root.Execute(); err != nil {
		t.Fatalf("orgs use: %v", err)
	}

	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("no se pudo leer la configuración de proyecto: %v", err)
	}
	var vals map[string]string
	if err := json.Unmarshal(b, &vals); err != nil {
		t.Fatal(err)
	}
	if vals["org"] != "globex" || vals["output"] != "json" || vals["timeout"] != "30s" {
		t.Errorf("config = %v, se esperaba conservar output=json y timeout=30s junto con org=globex", vals)
	}
}

// TestOrgsUseSinArgumentoDaErrorEnEspanolConHintYCodigoDeUso comprueba que la
// validación de argumentos pasa por exactArgs: sin él, Cobra devuelve
// "accepts 1 arg(s), received 0" en inglés, sin hint y con código de salida 1
// en vez de 2 (uso incorrecto).
func TestOrgsUseSinArgumentoDaErrorEnEspanolConHintYCodigoDeUso(t *testing.T) {
	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	root := testRoot(NewOrgsCmd(d), stdout)
	root.SetArgs([]string{"orgs", "use"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error sin argumento")
	}

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T", err)
	}
	if cliErr.Hint == "" {
		t.Error("el error debería traer un hint con la forma de uso")
	}
	// Comparación exacta, no solo "contiene": esto también protege que
	// usageLine recorte el sufijo " [flags]" que Cobra añadiría de otro modo
	// (el comando hereda los flags globales persistentes de la raíz), que no
	// aporta nada a un hint sobre número de argumentos.
	if want := "Uso: calliope orgs use <organización>"; cliErr.Hint != want {
		t.Errorf("hint = %q, se esperaba %q", cliErr.Hint, want)
	}
	if strings.Contains(cliErr.Message, "accepts") || strings.Contains(cliErr.Message, "arg(s)") {
		t.Errorf("el mensaje no debe ser el de Cobra en inglés: %q", cliErr.Message)
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2 (uso incorrecto)", got)
	}
}

// TestOrgsUseConArgumentosDeMasDaErrorEnEspanolConHintYCodigoDeUso es el
// reverso de la anterior: exactArgs(1) también debe rechazar un exceso de
// argumentos (len(args) == n falla igual por arriba que por abajo), pero
// nada lo ejercitaba.
func TestOrgsUseConArgumentosDeMasDaErrorEnEspanolConHintYCodigoDeUso(t *testing.T) {
	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	root := testRoot(NewOrgsCmd(d), stdout)
	root.SetArgs([]string{"orgs", "use", "acme", "globex"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error con argumentos de más")
	}

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T", err)
	}
	if cliErr.Hint == "" {
		t.Error("el error debería traer un hint con la forma de uso")
	}
	// Comparación exacta, no solo "contiene": esto también protege que
	// usageLine recorte el sufijo " [flags]" que Cobra añadiría de otro modo
	// (el comando hereda los flags globales persistentes de la raíz), que no
	// aporta nada a un hint sobre número de argumentos.
	if want := "Uso: calliope orgs use <organización>"; cliErr.Hint != want {
		t.Errorf("hint = %q, se esperaba %q", cliErr.Hint, want)
	}
	if strings.Contains(cliErr.Message, "accepts") || strings.Contains(cliErr.Message, "arg(s)") {
		t.Errorf("el mensaje no debe ser el de Cobra en inglés: %q", cliErr.Message)
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2 (uso incorrecto)", got)
	}
}
