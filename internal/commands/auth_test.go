package commands

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/auth"
)

// testRoot monta una raíz con los mismos flags globales que la real y le
// cuelga el comando bajo prueba. Sin esto, invocar un subcomando suelto con
// --json falla con "unknown flag": los flags globales son persistentes de la
// raíz. Todos los tests de comandos pasan por aquí.
func testRoot(sub *cobra.Command, out io.Writer) *cobra.Command {
	root := &cobra.Command{Use: "calliope", SilenceUsage: true, SilenceErrors: true}
	appctx.RegisterGlobalFlags(root)
	root.AddCommand(sub)
	root.SetOut(out)
	root.SetErr(out)
	return root
}

func depsWithServer(t *testing.T, h http.HandlerFunc) (appctx.Deps, *bytes.Buffer, auth.Store) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	st := auth.NewFileStore(filepath.Join(t.TempDir(), "c.json"))
	var stdout bytes.Buffer
	home := t.TempDir()
	d := appctx.Deps{
		Cwd: t.TempDir(),
		Env: func(k string) string {
			switch k {
			case "HOME":
				return home
			case "CALLIOPE_BASE_URL":
				return srv.URL
			case "CALLIOPE_ORG":
				return "acme"
			}
			return ""
		},
		Store:  st,
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	return d, &stdout, st
}

func TestAuthEsUnGrupoSinRunE(t *testing.T) {
	cmd := NewAuthCmd(appctx.Deps{})
	if cmd.RunE != nil || cmd.Run != nil {
		t.Error("auth es un grupo: invocarlo pelado debe mostrar la ayuda, no ejecutar nada")
	}
	if len(cmd.Commands()) == 0 {
		t.Error("auth debe tener subcomandos")
	}
}

func TestAuthLoginValidaLaCredencialAntesDeGuardarla(t *testing.T) {
	llamado := false
	d, _, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		llamado = true
		if r.URL.Path != "/v1/auth/me" {
			t.Errorf("login debe validar contra /v1/auth/me, llamó a %q", r.URL.Path)
		}
		w.Write([]byte(`{"userId":"u-1","email":"a@b.c"}`))
	})

	root := testRoot(NewAuthCmd(d), d.Stdout)
	root.SetArgs([]string{"auth", "login", "--api-key", "cal_live_ok"})
	if err := root.Execute(); err != nil {
		t.Fatalf("login: %v", err)
	}

	if !llamado {
		t.Fatal("login no validó la credencial contra el backend")
	}
	c, err := st.Load()
	if err != nil || c == nil || c.Token != "cal_live_ok" {
		t.Errorf("la credencial no se guardó: %+v (%v)", c, err)
	}
}

func TestAuthLoginNoGuardaUnaCredencialRechazada(t *testing.T) {
	d, _, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	root := testRoot(NewAuthCmd(d), d.Stdout)
	root.SetArgs([]string{"auth", "login", "--api-key", "cal_live_mala"})
	if err := root.Execute(); err == nil {
		t.Fatal("se esperaba error con una credencial rechazada")
	}

	if c, _ := st.Load(); c != nil {
		t.Errorf("nunca se debe persistir una credencial no verificada, se guardó: %+v", c)
	}
}

func TestAuthLogoutBorraLaCredencial(t *testing.T) {
	d, _, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewAuthCmd(d), d.Stdout)
	root.SetArgs([]string{"auth", "logout"})
	if err := root.Execute(); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if c, _ := st.Load(); c != nil {
		t.Error("tras logout no debe quedar credencial")
	}
}

func TestAuthStatusNoImprimeElToken(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "cal_live_secreto"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewAuthCmd(d), stdout)
	root.SetArgs([]string{"auth", "status"})
	if err := root.Execute(); err != nil {
		t.Fatalf("status: %v", err)
	}
	if strings.Contains(stdout.String(), "cal_live_secreto") {
		t.Error("auth status no debe imprimir el token completo")
	}
}
