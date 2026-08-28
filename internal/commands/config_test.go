package commands

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

	b, err := os.ReadFile(filepath.Join(d.Cwd, ".calliope", "config.json"))
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
