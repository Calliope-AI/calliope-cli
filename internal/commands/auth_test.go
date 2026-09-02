package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/Calliope-AI/calliope-cli/internal/appctx"
	"github.com/Calliope-AI/calliope-cli/internal/auth"
	"github.com/Calliope-AI/calliope-cli/internal/output"
	"github.com/Calliope-AI/calliope-cli/internal/version"
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

// TestAuthLoginMandaElUserAgentConVersion es el test de M1 de la oleada
// final: clientWith (el cliente que usa `login` para validar la credencial
// antes de guardarla) se construía sin UserAgent, así que login mandaba el
// "calliope-cli" por defecto de sdk.New en vez de "calliope-cli/<versión>",
// a diferencia de cualquier comando que pasa por appctx.Build.
func TestAuthLoginMandaElUserAgentConVersion(t *testing.T) {
	var visto string
	d, _, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		visto = r.Header.Get("User-Agent")
		w.Write([]byte(`{"userId":"u-1","email":"a@b.c"}`))
	})

	root := testRoot(NewAuthCmd(d), d.Stdout)
	root.SetArgs([]string{"auth", "login", "--api-key", "cal_live_ok"})
	if err := root.Execute(); err != nil {
		t.Fatalf("login: %v", err)
	}

	quiero := "calliope-cli/" + version.Version
	if visto != quiero {
		t.Errorf("User-Agent = %q, se esperaba %q", visto, quiero)
	}
}

// TestAuthLoginRespetaCalliopeTimeout es la segunda mitad de M1: clientWith
// se construía sin Timeout, así que login ignoraba CALLIOPE_TIMEOUT y se
// quedaba con el timeout por defecto de sdk.New (60s) sin importar lo que
// dijera la configuración. Se fija un timeout muy corto y un backend que
// tarda más: si clientWith siguiera ignorando CALLIOPE_TIMEOUT, este test
// tardaría 60s en fallar (o no fallaría) en vez de los ~200ms que tarda con
// el timeout real aplicado.
func TestAuthLoginRespetaCalliopeTimeout(t *testing.T) {
	d, _, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte(`{"userId":"u-1","email":"a@b.c"}`))
	})
	envBase := d.Env
	d.Env = func(k string) string {
		if k == "CALLIOPE_TIMEOUT" {
			return "20ms"
		}
		return envBase(k)
	}

	root := testRoot(NewAuthCmd(d), d.Stdout)
	root.SetArgs([]string{"auth", "login", "--api-key", "cal_live_ok"})

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

// depsWithServerNoOrg es depsWithServer sin CALLIOPE_ORG, para ejercitar los
// comandos que deben funcionar antes de haber elegido organización.
func depsWithServerNoOrg(t *testing.T, h http.HandlerFunc) (appctx.Deps, *bytes.Buffer, auth.Store) {
	t.Helper()
	d, stdout, st := depsWithServer(t, h)
	anterior := d.Env
	d.Env = func(k string) string {
		if k == "CALLIOPE_ORG" {
			return ""
		}
		return anterior(k)
	}
	return d, stdout, st
}

// El alcance real de una credencial es lo que responde el backend, no lo que
// el usuario haya fijado en local con `orgs use`. Mostrar el valor local bajo
// una etiqueta que parece del servidor hizo creer a un usuario que su clave
// estaba acotada a una organización cuando alcanzaba a siete.
func TestAuthStatusMuestraElAlcanceRealDeLaCredencial(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"id": "u-1", "email": "yo@ejemplo.com",
			"organizations": [
				{"id":"o-1","name":"acme","userRole":"Owner"},
				{"id":"o-2","name":"otra","userRole":"Member"}
			]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "cal_live_x", Org: "acme"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewAuthCmd(d), stdout)
	root.SetArgs([]string{"auth", "status", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("status: %v", err)
	}

	var env struct {
		Data struct {
			OrganizacionActiva string `json:"organizacionActiva"`
			Organizaciones     []struct {
				Name string `json:"name"`
				Rol  string `json:"rol"`
			} `json:"organizaciones"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("json inválido: %v\n%s", err, stdout.String())
	}

	if len(env.Data.Organizaciones) != 2 {
		t.Fatalf("se esperaban las 2 organizaciones del backend, hubo %d:\n%s",
			len(env.Data.Organizaciones), stdout.String())
	}
	if env.Data.Organizaciones[0].Name != "acme" || env.Data.Organizaciones[1].Name != "otra" {
		t.Errorf("nombres inesperados: %+v", env.Data.Organizaciones)
	}
	if env.Data.Organizaciones[0].Rol != "Owner" {
		t.Errorf("el rol debe llegar al usuario: %+v", env.Data.Organizaciones[0])
	}
	if env.Data.OrganizacionActiva != "acme" {
		t.Errorf("organizacionActiva = %q, se esperaba la fijada en local", env.Data.OrganizacionActiva)
	}
}

// El campo "organizacion" a secas se prestaba a leerse como «el alcance de mi
// clave». Ya no debe existir: o es la activa en local, o son las del backend.
func TestAuthStatusNoUsaLaEtiquetaAmbigua(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"u-1","email":"yo@ejemplo.com","organizations":[{"id":"o-1","name":"acme"}]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "cal_live_x", Org: "acme"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewAuthCmd(d), stdout)
	root.SetArgs([]string{"auth", "status", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("status: %v", err)
	}

	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("json inválido: %v", err)
	}
	if _, existe := env.Data["organizacion"]; existe {
		t.Errorf("la clave ambigua \"organizacion\" sigue ahí: %v", env.Data)
	}
}

// `auth status` es el comando que se ejecuta para saber qué organizaciones
// tiene uno. Exigir que ya haya uno elegido lo hace inútil justo cuando más
// se necesita: al empezar, o al no saber qué alcance tiene la clave.
func TestAuthStatusFuncionaSinOrganizacionElegida(t *testing.T) {
	d, stdout, st := depsWithServerNoOrg(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"u-1","email":"yo@ejemplo.com","organizations":[
			{"id":"o-1","name":"acme"},{"id":"o-2","name":"otra"}]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "cal_live_x"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewAuthCmd(d), stdout)
	root.SetArgs([]string{"auth", "status", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("auth status debe funcionar sin organización elegida: %v", err)
	}

	var env struct {
		Data struct {
			OrganizacionActiva string                  `json:"organizacionActiva"`
			Organizaciones     []struct{ Name string } `json:"organizaciones"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("json inválido: %v\n%s", err, stdout.String())
	}
	if len(env.Data.Organizaciones) != 2 {
		t.Errorf("debe listar el alcance aunque no haya organización activa: %s", stdout.String())
	}
	if env.Data.OrganizacionActiva != "" {
		t.Errorf("organizacionActiva debería ir vacía: %q", env.Data.OrganizacionActiva)
	}
}
