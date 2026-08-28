package commands

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/output"
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

// TestAuthEsUnGrupoDeRecursos comprueba el comportamiento de `auth` como
// grupo de recursos (C3 de la oleada final): pelado muestra la ayuda con
// exit 0; con un subcomando que no existe, falla con exit 2. Antes este test
// aseveraba el MECANISMO (RunE == nil) en vez del comportamiento, y ahí se
// coló el bug: cobra decide si el comando es Runnable() -y por tanto si
// muestra la ayuda- antes de validar los argumentos, así que "auth typo"
// mostraba la misma ayuda que "auth" solo, con exit 0.
func TestAuthEsUnGrupoDeRecursos(t *testing.T) {
	cmd := NewAuthCmd(appctx.Deps{})
	if len(cmd.Commands()) == 0 {
		t.Error("auth debe tener subcomandos")
	}

	var out bytes.Buffer
	root := testRoot(NewAuthCmd(appctx.Deps{}), &out)
	root.SetArgs([]string{"auth"})
	if err := root.Execute(); err != nil {
		t.Fatalf("auth pelado debe salir con 0 (ayuda), dio error: %v", err)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Errorf("auth pelado debe imprimir la ayuda, se obtuvo: %q", out.String())
	}

	out.Reset()
	root = testRoot(NewAuthCmd(appctx.Deps{}), &out)
	root.SetArgs([]string{"auth", "esto-no-existe"})
	err := root.Execute()
	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("auth con subcomando desconocido: el error debería ser un *output.CLIError, fue %T (%v)", err, err)
	}
	if cliErr.Hint == "" {
		t.Error("el error debería traer un hint")
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2 (uso incorrecto)", got)
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

// TestAuthStatusEnTTYNoImprimeElToken es la variante con IsTTY:true de la
// prueba anterior. Con IsTTY:false (el cero de bool, lo que deja
// depsWithServer) el modo de salida ModeAuto siempre cae a JSON y el
// renderer Text nunca se llega a ejecutar: una fuga del token ahí pasaría
// inadvertida. Fijando IsTTY:true se ejercita de verdad el camino de Text.
func TestAuthStatusEnTTYNoImprimeElToken(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "cal_live_secreto"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewAuthCmd(d), stdout)
	root.SetArgs([]string{"auth", "status"})
	if err := root.Execute(); err != nil {
		t.Fatalf("status: %v", err)
	}
	if strings.Contains(stdout.String(), "cal_live_secreto") {
		t.Error("auth status no debe imprimir el token completo (renderer de texto, IsTTY)")
	}
}

func TestAuthTokenImprimeElTokenResuelto(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "cal_live_para_script"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewAuthCmd(d), stdout)
	root.SetArgs([]string{"auth", "token"})
	if err := root.Execute(); err != nil {
		t.Fatalf("token: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "cal_live_para_script" {
		t.Errorf("stdout = %q, se esperaba el token resuelto", got)
	}
}

func TestAuthTokenSinCredencialDevuelveErrorConHint(t *testing.T) {
	// depsWithServer no deja ninguna credencial disponible: ni variables
	// CALLIOPE_API_KEY/CALLIOPE_TOKEN en el entorno simulado ni nada guardado
	// en el almacén.
	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	root := testRoot(NewAuthCmd(d), stdout)
	root.SetArgs([]string{"auth", "token"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error sin credencial")
	}

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T", err)
	}
	if cliErr.Hint == "" {
		t.Error("el error debería traer un hint con la acción de recuperación")
	}
	if got := output.ExitCodeFor(err); got != 3 {
		t.Errorf("código de salida = %d, se esperaba 3 (no autorizado)", got)
	}
}
