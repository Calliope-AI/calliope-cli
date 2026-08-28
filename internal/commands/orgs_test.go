package commands

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/calliope/calliope-cli/internal/auth"
)

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
