package commands

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calliope/calliope-cli/internal/auth"
)

func checksOf(t *testing.T, salida []byte) map[string]string {
	t.Helper()
	var env struct {
		Data []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(salida, &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, salida)
	}
	m := map[string]string{}
	for _, c := range env.Data {
		m[c.Name] = c.Status
	}
	return m
}

func TestDoctorTodoCorrecto(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"u-1","email":"a@b.c"}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewDoctorCmd(d), stdout)
	root.SetArgs([]string{"doctor", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doctor: %v", err)
	}

	c := checksOf(t, stdout.Bytes())
	for _, nombre := range []string{"credencial", "organización", "conectividad"} {
		if c[nombre] != "ok" {
			t.Errorf("chequeo %q = %q, se esperaba ok", nombre, c[nombre])
		}
	}
}

func TestDoctorSinCredencialInformaEnVezDeFallar(t *testing.T) {
	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})
	d.Store = auth.NewFileStore(filepath.Join(t.TempDir(), "vacio.json"))

	root := testRoot(NewDoctorCmd(d), stdout)
	root.SetArgs([]string{"doctor", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doctor nunca debe fallar, debe informar: %v", err)
	}

	c := checksOf(t, stdout.Bytes())
	if c["credencial"] != "error" {
		t.Errorf("chequeo credencial = %q, se esperaba error", c["credencial"])
	}
}

func TestDoctorConBackendCaidoLoDetecta(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewDoctorCmd(d), stdout)
	root.SetArgs([]string{"doctor", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doctor: %v", err)
	}

	c := checksOf(t, stdout.Bytes())
	if c["conectividad"] != "error" {
		t.Errorf("chequeo conectividad = %q, se esperaba error", c["conectividad"])
	}
}

// TestDoctorNuncaImprimeElToken cubre el aviso más delicado de esta tarea: el
// token de la credencial no debe aparecer en la salida de `doctor` en ningún
// modo, ni siquiera cuando la conectividad falla y el error del backend
// podría, por descuido, arrastrar detalles de la petición. Se comprueban los
// dos modos de salida (JSON, el único que depsWithServer ejercita sin TTY, y
// el renderer de texto con IsTTY:true) para que un futuro cambio en
// checkConnectivity que interpole ctx.Cred en vez del cred resuelto no pase
// inadvertido.
func TestDoctorNuncaImprimeElToken(t *testing.T) {
	const tokenSecreto = "cal_live_no_debe_salir_nunca"

	casos := []struct {
		nombre  string
		handler http.HandlerFunc
		isTTY   bool
		args    []string
	}{
		{
			nombre: "json/todo-correcto",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"id":"u-1","email":"a@b.c"}`))
			},
			args: []string{"doctor", "--json"},
		},
		{
			nombre: "json/backend-caido",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			args: []string{"doctor", "--json"},
		},
		{
			nombre: "texto/tty",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"id":"u-1","email":"a@b.c"}`))
			},
			isTTY: true,
			args:  []string{"doctor"},
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			d, stdout, st := depsWithServer(t, caso.handler)
			d.IsTTY = caso.isTTY
			if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: tokenSecreto}); err != nil {
				t.Fatal(err)
			}

			root := testRoot(NewDoctorCmd(d), stdout)
			root.SetArgs(caso.args)
			if err := root.Execute(); err != nil {
				t.Fatalf("doctor: %v", err)
			}

			if strings.Contains(stdout.String(), tokenSecreto) {
				t.Errorf("doctor imprimió el token en salida %q: %s", caso.nombre, stdout.String())
			}
		})
	}
}
