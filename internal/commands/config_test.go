package commands

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calliope/calliope-cli/internal/config"
	"github.com/calliope/calliope-cli/internal/output"
)

func TestConfigListMuestraLaProcedencia(t *testing.T) {
	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	root := testRoot(NewConfigCmd(d), stdout)
	root.SetArgs([]string{"config", "list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("config list: %v", err)
	}

	var env struct {
		Data map[string]struct {
			Value  string `json:"value"`
			Source string `json:"source"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	// depsWithServer fija CALLIOPE_ORG, así que org viene del entorno.
	if env.Data["org"].Source != "env" {
		t.Errorf("origen de org = %q, se esperaba env", env.Data["org"].Source)
	}
	if env.Data["base_url"].Value == "" {
		t.Error("base_url debe tener valor, aunque sea el por defecto")
	}
}

func TestConfigSetRechazaClavesNoPermitidasEnProyecto(t *testing.T) {
	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	root := testRoot(NewConfigCmd(d), stdout)
	root.SetArgs([]string{"config", "set", "base_url", "https://atacante.example"})
	err := root.Execute()
	if err == nil {
		t.Fatal("config set debe rechazar base_url en la configuración de proyecto")
	}
	if !strings.Contains(err.Error(), "--global") {
		t.Errorf("el error debe explicar que base_url solo se fija en global: %q", err.Error())
	}
}

func TestConfigSetEscribeUnaClavePermitida(t *testing.T) {
	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	root := testRoot(NewConfigCmd(d), stdout)
	root.SetArgs([]string{"config", "set", "org", "globex"})
	if err := root.Execute(); err != nil {
		t.Fatalf("config set: %v", err)
	}

	ruta := filepath.Join(d.Cwd, ".calliope", "config.json")
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("no se escribió el fichero: %v", err)
	}
	var vals map[string]string
	if err := json.Unmarshal(b, &vals); err != nil {
		t.Fatal(err)
	}
	if vals["org"] != "globex" {
		t.Errorf("config = %v", vals)
	}

	if fi, err := os.Stat(ruta); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("permisos del fichero de proyecto = %v (err=%v), se esperaba 0600", fi.Mode().Perm(), err)
	}
}

// TestConfigSetSinPermisosDeEscrituraEsCLIError es el test del Diferido #10
// de la oleada final: en un directorio de proyecto sin permisos de
// escritura, `config set` devolvía el error crudo de os.MkdirAll -en
// inglés, sin hint, y con la ruta absoluta del sistema de ficheros del
// cliente incluida en el mensaje-.
func TestConfigSetSinPermisosDeEscrituraEsCLIError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})
	d.Cwd = dir

	root := testRoot(NewConfigCmd(d), stdout)
	root.SetArgs([]string{"config", "set", "org", "acme"})
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

// TestConfigSetGlobalEscribeEnGlobalNoEnProyecto cubre la mitad de
// *aceptación* de la frontera de seguridad (la de *rechazo* la cubre
// TestConfigSetRechazaClavesNoPermitidasEnProyecto): que --global sí permite
// fijar una clave como base_url, y que la escribe únicamente en la
// configuración global, nunca en la de proyecto. Sin este test, dejar
// --global completamente inerte (escribir siempre en el proyecto, donde
// base_url sigue estando prohibida) no rompía ningún test: el comando se
// limitaba a devolver el mismo error de rechazo, que ningún test distinguía
// del caso sin --global.
func TestConfigSetGlobalEscribeEnGlobalNoEnProyecto(t *testing.T) {
	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	root := testRoot(NewConfigCmd(d), stdout)
	root.SetArgs([]string{"config", "set", "base_url", "https://legit.example", "--global"})
	if err := root.Execute(); err != nil {
		t.Fatalf("config set --global: %v", err)
	}

	rutaGlobal := config.GlobalPath(d.Env)
	b, err := os.ReadFile(rutaGlobal)
	if err != nil {
		t.Fatalf("no se escribió el fichero global: %v", err)
	}
	var vals map[string]string
	if err := json.Unmarshal(b, &vals); err != nil {
		t.Fatal(err)
	}
	if vals["base_url"] != "https://legit.example" {
		t.Errorf("config global = %v, se esperaba base_url=https://legit.example", vals)
	}
	if fi, err := os.Stat(rutaGlobal); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("permisos del fichero global = %v (err=%v), se esperaba 0600", fi.Mode().Perm(), err)
	}

	// --global debe escribir solo en la configuración global: si además
	// escribiera en la de proyecto, el rechazo de IsProjectAllowed dejaría de
	// significar nada -bastaría con añadir --global a cualquier clave sin que
	// tuviera efecto real sobre dónde queda escrita.
	if _, err := os.Stat(filepath.Join(d.Cwd, ".calliope", "config.json")); !os.IsNotExist(err) {
		t.Errorf("config set --global no debe escribir en la configuración de proyecto (err=%v)", err)
	}
}

// TestConfigSetGlobalArreglaPermisosLaxosDelDirectorioExistente cubre
// IMPORTANT 2: el directorio de configuración global es el mismo donde
// auth.DefaultStore guarda las credenciales (internal/auth/store.go), que lo
// trata como sensible -0700, reforzado con chmod explícito por si ya existía
// con permisos más laxos-. config set --global debe seguir la misma regla:
// si alguien fija org o base_url globalmente antes de autenticarse, el
// directorio no debe quedar listable por otros usuarios del sistema hasta
// que un auth login posterior lo repare.
func TestConfigSetGlobalArreglaPermisosLaxosDelDirectorioExistente(t *testing.T) {
	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	dir := filepath.Dir(config.GlobalPath(d.Env))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewConfigCmd(d), stdout)
	root.SetArgs([]string{"config", "set", "org", "acme", "--global"})
	if err := root.Execute(); err != nil {
		t.Fatalf("config set --global: %v", err)
	}

	fi, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("no se pudo leer el directorio global: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o700 {
		t.Errorf("permisos del directorio global = %v, se esperaba 0700", perm)
	}
}

// TestConfigSetPreservaLasClavesPreviasEnProyecto es el análogo, para
// `config set`, de TestOrgsUsePreservaLasClavesPrevias: comprueba que el
// comando hace merge con lo que ya hubiera en .calliope/config.json en vez de
// sobrescribirlo entero.
func TestConfigSetPreservaLasClavesPreviasEnProyecto(t *testing.T) {
	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	dir := filepath.Join(d.Cwd, ".calliope")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	previo, err := json.Marshal(map[string]string{"output": "json"})
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, "config.json")
	if err := os.WriteFile(ruta, previo, 0o600); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewConfigCmd(d), stdout)
	root.SetArgs([]string{"config", "set", "org", "globex"})
	if err := root.Execute(); err != nil {
		t.Fatalf("config set: %v", err)
	}

	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("no se pudo leer la configuración de proyecto: %v", err)
	}
	var vals map[string]string
	if err := json.Unmarshal(b, &vals); err != nil {
		t.Fatal(err)
	}
	if vals["org"] != "globex" || vals["output"] != "json" {
		t.Errorf("config = %v, se esperaba conservar output=json junto con org=globex", vals)
	}
}

// TestConfigSetGlobalPreservaLasClavesPrevias es la variante --global de la
// prueba anterior.
func TestConfigSetGlobalPreservaLasClavesPrevias(t *testing.T) {
	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	ruta := config.GlobalPath(d.Env)
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		t.Fatal(err)
	}
	previo, err := json.Marshal(map[string]string{"timeout": "30s"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, previo, 0o600); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewConfigCmd(d), stdout)
	root.SetArgs([]string{"config", "set", "base_url", "https://legit.example", "--global"})
	if err := root.Execute(); err != nil {
		t.Fatalf("config set --global: %v", err)
	}

	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("no se pudo leer la configuración global: %v", err)
	}
	var vals map[string]string
	if err := json.Unmarshal(b, &vals); err != nil {
		t.Fatal(err)
	}
	if vals["base_url"] != "https://legit.example" || vals["timeout"] != "30s" {
		t.Errorf("config = %v, se esperaba conservar timeout=30s junto con base_url", vals)
	}
}

func TestConfigGetDevuelveUnValor(t *testing.T) {
	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	root := testRoot(NewConfigCmd(d), stdout)
	root.SetArgs([]string{"config", "get", "org", "--quiet"})
	if err := root.Execute(); err != nil {
		t.Fatalf("config get: %v", err)
	}
	if !strings.Contains(stdout.String(), "acme") {
		t.Errorf("salida = %q, se esperaba acme", stdout.String())
	}
}

// TestConfigGetRespetaLaClavePedida comprueba `config get` con una clave
// distinta de "org": la prueba anterior por sí sola no distingue un comando
// que de verdad consulta args[0] de uno que ignorase el argumento y siempre
// devolviera "org" -es la única clave que se le pide en todo el fichero.
func TestConfigGetRespetaLaClavePedida(t *testing.T) {
	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})
	esperado := d.Env("CALLIOPE_BASE_URL")

	root := testRoot(NewConfigCmd(d), stdout)
	root.SetArgs([]string{"config", "get", "base_url", "--quiet"})
	if err := root.Execute(); err != nil {
		t.Fatalf("config get: %v", err)
	}
	if !strings.Contains(stdout.String(), esperado) {
		t.Errorf("salida = %q, se esperaba que contuviera %q (base_url)", stdout.String(), esperado)
	}
	if strings.Contains(stdout.String(), "acme") {
		t.Errorf("salida = %q, config get \"base_url\" no debería devolver el valor de org", stdout.String())
	}
}
