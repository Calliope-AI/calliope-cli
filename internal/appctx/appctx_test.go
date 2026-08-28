package appctx

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/presenter"
	"github.com/calliope/calliope-cli/internal/version"
)

func commandWithFlags(flags map[string]string) *cobra.Command {
	cmd := &cobra.Command{Use: "x"}
	RegisterGlobalFlags(cmd)
	for k, v := range flags {
		_ = cmd.Flags().Set(k, v)
	}
	return cmd
}

func testDeps(t *testing.T, cwd string) (Deps, *bytes.Buffer) {
	t.Helper()
	st := auth.NewFileStore(filepath.Join(t.TempDir(), "c.json"))
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	home := t.TempDir() // fuera del closure: si no, cada llamada crearía otro
	return Deps{
		Cwd: cwd,
		Env: func(k string) string {
			if k == "HOME" {
				return home
			}
			return ""
		},
		Store:  st,
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
	}, &stderr
}

// testDepsWithoutCredential monta unas Deps sin ninguna credencial disponible:
// ni variables de entorno (CALLIOPE_API_KEY, CALLIOPE_TOKEN) ni almacén con
// nada guardado. Es el reverso de testDeps, que siempre deja una credencial
// válida y por eso nunca ejercita el camino en el que Build debe fallar.
func testDepsWithoutCredential(t *testing.T, cwd string) Deps {
	t.Helper()
	home := t.TempDir() // fuera del closure: si no, cada llamada crearía otro
	return Deps{
		Cwd: cwd,
		Env: func(k string) string {
			if k == "HOME" {
				return home
			}
			return ""
		},
		Store:  auth.NewFileStore(filepath.Join(t.TempDir(), "vacio.json")),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}
}

func writeProjectConfig(t *testing.T, dir string, vals map[string]string) {
	t.Helper()
	d := filepath.Join(dir, ".calliope")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(vals)
	if err := os.WriteFile(filepath.Join(d, "config.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLaOrganizacionSaleDeLaConfiguracionDeProyecto(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, map[string]string{"org": "acme"})
	deps, _ := testDeps(t, dir)

	ctx, err := Build(commandWithFlags(nil), deps)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if ctx.Org != "acme" {
		t.Errorf("Org = %q, se esperaba acme", ctx.Org)
	}
}

func TestElFlagOrgGanaALaConfiguracion(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, map[string]string{"org": "acme"})
	deps, _ := testDeps(t, dir)

	ctx, err := Build(commandWithFlags(map[string]string{"org": "otra"}), deps)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if ctx.Org != "otra" {
		t.Errorf("Org = %q, el flag debe ganar", ctx.Org)
	}
}

func TestSinOrganizacionElErrorDiceComoElegirla(t *testing.T) {
	deps, _ := testDeps(t, t.TempDir())

	_, err := Build(commandWithFlags(nil), deps)
	if err == nil {
		t.Fatal("se esperaba error al no haber organización")
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2 (uso incorrecto)", got)
	}
	if !strings.Contains(err.Error(), "organización") {
		t.Errorf("mensaje poco claro: %q", err.Error())
	}
}

func TestLosAvisosDeConfiguracionVanAStderr(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, map[string]string{"org": "acme", "base_url": "https://atacante.example"})
	deps, stderr := testDeps(t, dir)

	if _, err := Build(commandWithFlags(nil), deps); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(stderr.String(), "base_url") {
		t.Errorf("el aviso debe llegar a stderr, se obtuvo: %q", stderr.String())
	}
}

func TestLosFlagsDeterminanElModoDeSalida(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, map[string]string{"org": "acme"})

	casos := []struct {
		flags  map[string]string
		quiero string
	}{
		{map[string]string{"json": "true"}, "json"},
		{map[string]string{"quiet": "true"}, "quiet"},
		{map[string]string{"md": "true"}, "md"},
		{map[string]string{"jq": ".data"}, "jq"},
		{nil, "auto"},
	}

	for _, caso := range casos {
		deps, _ := testDeps(t, dir)
		ctx, err := Build(commandWithFlags(caso.flags), deps)
		if err != nil {
			t.Fatalf("Build(%v): %v", caso.flags, err)
		}
		if string(ctx.Present.Mode) != caso.quiero {
			t.Errorf("flags %v → modo %q, se esperaba %q", caso.flags, ctx.Present.Mode, caso.quiero)
		}
	}
}

func TestBuildSinCredencialNoFallaSinAutenticacion(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, map[string]string{"org": "acme"})
	deps, _ := testDeps(t, dir)
	deps.Store = auth.NewFileStore(filepath.Join(t.TempDir(), "vacio.json"))

	ctx, err := BuildSinCredencial(commandWithFlags(nil), deps)
	if err != nil {
		t.Fatalf("BuildSinCredencial no debe exigir credencial: %v", err)
	}
	if ctx.Cred.Valid() {
		t.Error("no debería haber credencial")
	}
}

func TestBuildExigeCredencialYElErrorTraeCodigoYHint(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, map[string]string{"org": "acme"})
	deps := testDepsWithoutCredential(t, dir)

	_, err := Build(commandWithFlags(nil), deps)
	if err == nil {
		t.Fatal("se esperaba error al no haber credencial")
	}
	if got := output.ExitCodeFor(err); got != 3 {
		t.Errorf("código de salida = %d, se esperaba 3 (no autorizado)", got)
	}
	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T", err)
	}
	if cliErr.Hint == "" {
		t.Error("el error debería traer un hint con la acción de recuperación")
	}
}

// TestResolveOutputModeCubreLosSeisModos es el test de C2 de la oleada
// final: antes, main() decidía si un ERROR salía en JSON escaneando
// os.Args en busca del token literal "--json", desconectado de esta
// resolución. En tubería, con --jq, --quiet, --md, --json=true o
// CALLIOPE_OUTPUT=json el éxito salía en JSON pero el error salía en
// texto plano. Ahora ResolveOutputMode es la misma función que usan
// Build/BuildWithoutCredential para el éxito, así que IsMachineReadable()
// tiene que dar el mismo resultado para los seis modos de la tabla 6.2 del
// spec (más CALLIOPE_OUTPUT=json, otra vía hacia el mismo modo --json).
func TestResolveOutputModeCubreLosSeisModos(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, map[string]string{"org": "acme"})

	casos := []struct {
		nombre        string
		flags         map[string]string
		env           map[string]string
		isTTY         bool
		quiereMachine bool
	}{
		{nombre: "por defecto, TTY", isTTY: true, quiereMachine: false},
		{nombre: "por defecto, tubería", isTTY: false, quiereMachine: true},
		{nombre: "--json", flags: map[string]string{"json": "true"}, isTTY: true, quiereMachine: true},
		{nombre: "--quiet", flags: map[string]string{"quiet": "true"}, isTTY: true, quiereMachine: true},
		{nombre: "--md", flags: map[string]string{"md": "true"}, isTTY: true, quiereMachine: true},
		{nombre: "--jq", flags: map[string]string{"jq": ".data"}, isTTY: true, quiereMachine: true},
		{nombre: "CALLIOPE_OUTPUT=json", env: map[string]string{"CALLIOPE_OUTPUT": "json"}, isTTY: true, quiereMachine: true},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			deps, _ := testDeps(t, dir)
			deps.IsTTY = caso.isTTY
			if caso.env != nil {
				base := deps.Env
				deps.Env = func(k string) string {
					if v, ok := caso.env[k]; ok {
						return v
					}
					return base(k)
				}
			}

			opts := ResolveOutputMode(commandWithFlags(caso.flags), deps)
			if got := opts.IsMachineReadable(); got != caso.quiereMachine {
				t.Errorf("IsMachineReadable() = %v, se esperaba %v (Mode=%q IsTTY=%v)",
					got, caso.quiereMachine, opts.Mode, opts.IsTTY)
			}
		})
	}
}

// TestResolveOutputModeConConfiguracionRotaNoFalla comprueba el segundo
// invariante de C2: si config.Load falla (p. ej. un config.json corrupto,
// que es precisamente el tipo de fallo que main() podría estar informando
// en ese momento), ResolveOutputMode no debe propagar ese error -solo
// decide el FORMATO del error que ya se está informando- y debe seguir
// resolviendo el modo a partir de los flags de la invocación.
func TestResolveOutputModeConConfiguracionRotaNoFalla(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".calliope"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".calliope", "config.json"), []byte("{no es json"), 0o600); err != nil {
		t.Fatal(err)
	}
	deps, _ := testDeps(t, dir)

	opts := ResolveOutputMode(commandWithFlags(map[string]string{"json": "true"}), deps)
	if opts.Mode != presenter.ModeJSON {
		t.Errorf("Mode = %q, se esperaba json incluso con configuración rota", opts.Mode)
	}
}

// DefaultDeps es lo único que conecta el proceso real con version.ReleasesURL
// (Deps.ReleasesURL es inyectable justo para que los tests de cli no
// dependan de esa constante). Si alguien olvida cablearlo aquí, el binario
// real seguiría avisando de nuevas versiones porque un string vacío también
// falla la petición HTTP -> LatestVersion se calla igual, pero calladamente
// mal: dejaría de consultar el endpoint real sin que ningún test lo notara.
func TestDefaultDepsUsaElEndpointRealDeVersion(t *testing.T) {
	d := DefaultDeps()
	if d.ReleasesURL != version.ReleasesURL {
		t.Errorf("DefaultDeps().ReleasesURL = %q, se esperaba %q", d.ReleasesURL, version.ReleasesURL)
	}
}
