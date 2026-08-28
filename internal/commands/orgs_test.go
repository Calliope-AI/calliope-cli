package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/version"
)

// TestOrgsEsUnGrupoDeRecursos comprueba el comportamiento de `orgs` como
// grupo de recursos (C3 de la oleada final): pelado muestra la ayuda con
// exit 0; con un subcomando que no existe, falla con exit 2. Ver el
// comentario equivalente en auth_test.go sobre por qué esto ya no se
// asevera comprobando que RunE sea nil.
func TestOrgsEsUnGrupoDeRecursos(t *testing.T) {
	cmd := NewOrgsCmd(appctx.Deps{})
	if len(cmd.Commands()) == 0 {
		t.Error("orgs debe tener subcomandos")
	}

	var out bytes.Buffer
	root := testRoot(NewOrgsCmd(appctx.Deps{}), &out)
	root.SetArgs([]string{"orgs"})
	if err := root.Execute(); err != nil {
		t.Fatalf("orgs pelado debe salir con 0 (ayuda), dio error: %v", err)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Errorf("orgs pelado debe imprimir la ayuda, se obtuvo: %q", out.String())
	}

	out.Reset()
	root = testRoot(NewOrgsCmd(appctx.Deps{}), &out)
	root.SetArgs([]string{"orgs", "esto-no-existe"})
	err := root.Execute()
	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("orgs con subcomando desconocido: el error debería ser un *output.CLIError, fue %T (%v)", err, err)
	}
	if cliErr.Hint == "" {
		t.Error("el error debería traer un hint")
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2 (uso incorrecto)", got)
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

// TestOrgsListMandaElUserAgentConVersionYRespetaElTimeout es el test de M1
// de la oleada final: clientWith (el cliente que usa `orgs list`, que no
// pasa por appctx.Build porque listar organizaciones no exige tener una ya
// elegida) se construía sin Timeout ni UserAgent, así que orgs list
// ignoraba CALLIOPE_TIMEOUT y mandaba el "calliope-cli" por defecto de
// sdk.New en vez de "calliope-cli/<versión>".
func TestOrgsListMandaElUserAgentConVersionYRespetaElTimeout(t *testing.T) {
	var visto string
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		visto = r.Header.Get("User-Agent")
		w.Write([]byte(`[{"id":"o-1","name":"acme"}]`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewOrgsCmd(d), stdout)
	root.SetArgs([]string{"orgs", "list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("orgs list: %v", err)
	}

	quiero := "calliope-cli/" + version.Version
	if visto != quiero {
		t.Errorf("User-Agent = %q, se esperaba %q", visto, quiero)
	}
}

// TestOrgsListRespetaCalliopeTimeout es la segunda mitad de M1 para orgs
// list: sin Timeout en clientWith, un backend lento se habría quedado con
// el timeout por defecto de sdk.New (60s) sin importar CALLIOPE_TIMEOUT. Se
// fija un timeout muy corto y un backend que tarda más: si clientWith
// siguiera ignorando CALLIOPE_TIMEOUT, este test tardaría 60s en fallar (o
// no fallaría) en vez de los ~200ms que tarda con el timeout real aplicado.
func TestOrgsListRespetaCalliopeTimeout(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte(`[]`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}
	envBase := d.Env
	d.Env = func(k string) string {
		if k == "CALLIOPE_TIMEOUT" {
			return "20ms"
		}
		return envBase(k)
	}

	root := testRoot(NewOrgsCmd(d), stdout)
	root.SetArgs([]string{"orgs", "list"})

	inicio := time.Now()
	err := root.Execute()
	transcurrido := time.Since(inicio)

	if err == nil {
		t.Fatal("se esperaba un error de timeout")
	}
	if transcurrido > 2*time.Second {
		t.Fatalf("tardó %s: CALLIOPE_TIMEOUT no se está aplicando (se habría quedado con el timeout por defecto de 60s)", transcurrido)
	}
	if !strings.Contains(err.Error(), "tiempo límite") {
		t.Errorf("mensaje = %q, se esperaba un error de timeout", err.Error())
	}
}

// TestOrgsUseSinPermisosDeEscrituraEsCLIError es el test del Diferido #10
// de la oleada final: en un directorio sin permisos de escritura, `orgs use`
// devolvía el error crudo de os.MkdirAll -en inglés, sin hint, y con la ruta
// absoluta del sistema de ficheros del cliente incluida en el mensaje (p.
// ej. "mkdir /Users/alguien/proyecto/.calliope: permission denied")-.
func TestOrgsUseSinPermisosDeEscrituraEsCLIError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	// TempDir necesita poder borrar dir al terminar el test; se restaura el
	// permiso de escritura antes de que el cleanup de t.TempDir() lo intente.
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})
	d.Cwd = dir
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewOrgsCmd(d), stdout)
	root.SetArgs([]string{"orgs", "use", "acme"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba un error de E/S")
	}

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T (%v)", err, err)
	}
	if cliErr.Hint == "" {
		t.Error("el error debería traer un hint")
	}
	if strings.Contains(cliErr.Message, dir) {
		t.Errorf("el mensaje filtra la ruta absoluta del sistema: %q", cliErr.Message)
	}
	if strings.Contains(cliErr.Message, "permission denied") {
		t.Errorf("el mensaje no debe ser el de Go en inglés: %q", cliErr.Message)
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
