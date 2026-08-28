package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositorioHostilNoPuedeCambiarBaseURL(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, map[string]string{
		"org":      "acme",
		"base_url": "https://atacante.example.com",
		"timeout":  "1ms",
	})

	env := testEnv(t.TempDir())
	cfg, avisos, err := Load(dir, env, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got := cfg.BaseURL(); got != DefaultBaseURL {
		t.Fatalf("base_url = %q — un repositorio hostil ha conseguido redirigir el CLI", got)
	}
	if got := cfg.Get(KeyTimeout).Value; got == "1ms" {
		t.Errorf("timeout = %q, la configuración de proyecto no puede fijarlo", got)
	}
	if got := cfg.Org(); got != "acme" {
		t.Errorf("org = %q, la configuración de proyecto sí puede fijar la organización", got)
	}
	if len(avisos) == 0 {
		t.Error("se esperaba un aviso visible sobre los campos ignorados")
	}
	if !strings.Contains(strings.Join(avisos, " "), "base_url") {
		t.Errorf("el aviso debe nombrar el campo ignorado: %v", avisos)
	}
}

// TestRepositorioHostilEnLaRaizNoPuedeCambiarBaseURLDesdeUnSubdirectorio cubre
// el caso en que la víctima no ejecuta calliope desde la raíz del repositorio
// clonado sino desde un subdirectorio (p. ej. un monorepo). Ahí la capa que
// entra en juego es SourceRepo, no SourceProject, y también debe sanearse:
// de lo contrario un repositorio hostil con un subdirectorio de trabajo
// habitual seguiría pudiendo fijar base_url sin que ningún test lo detectase.
func TestRepositorioHostilEnLaRaizNoPuedeCambiarBaseURLDesdeUnSubdirectorio(t *testing.T) {
	raiz := t.TempDir()
	if err := os.MkdirAll(filepath.Join(raiz, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeProjectConfig(t, raiz, map[string]string{
		"org":      "acme",
		"base_url": "https://atacante.example.com",
	})

	sub := filepath.Join(raiz, "paquetes", "app")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, avisos, err := Load(sub, testEnv(t.TempDir()), nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got := cfg.BaseURL(); got != DefaultBaseURL {
		t.Fatalf("base_url = %q — un repositorio hostil ha conseguido redirigir el CLI desde un subdirectorio", got)
	}
	if len(avisos) == 0 {
		t.Error("se esperaba un aviso visible sobre los campos ignorados")
	}
	if !strings.Contains(strings.Join(avisos, " "), "base_url") {
		t.Errorf("el aviso debe nombrar el campo ignorado: %v", avisos)
	}
}

// TestRepositorioHostilEnUnGitWorktreeNoPuedeCambiarBaseURL cubre el
// Diferido #5 de la oleada final: en un git worktree o un submódulo, .git
// no es un directorio sino un fichero de una línea ("gitdir: <ruta>") que
// apunta al git real en otro sitio. repoRoot exigía fi.IsDir(), así que en
// un worktree la raíz del repositorio nunca se detectaba y la capa
// SourceRepo desaparecía en silencio -tanto para una configuración
// legítima como, más grave, para una hostil que debía sanearse.
func TestRepositorioHostilEnUnGitWorktreeNoPuedeCambiarBaseURL(t *testing.T) {
	raiz := t.TempDir()
	// Un .git de worktree real es un fichero de una línea apuntando al
	// git commondir real; el contenido exacto no importa aquí, solo que
	// sea un fichero regular y no un directorio.
	if err := os.WriteFile(filepath.Join(raiz, ".git"),
		[]byte("gitdir: /otra/ruta/.git/worktrees/rama-de-feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeProjectConfig(t, raiz, map[string]string{
		"org":      "acme",
		"base_url": "https://atacante.example.com",
	})

	sub := filepath.Join(raiz, "paquetes", "app")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, avisos, err := Load(sub, testEnv(t.TempDir()), nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got := cfg.BaseURL(); got != DefaultBaseURL {
		t.Fatalf("base_url = %q — un repositorio hostil en un worktree ha conseguido redirigir el CLI", got)
	}
	if got := cfg.Org(); got != "acme" {
		t.Errorf("org = %q, la configuración de la raíz del worktree sí debe aplicarse (campo permitido)", got)
	}
	if len(avisos) == 0 {
		t.Error("se esperaba un aviso visible sobre los campos ignorados")
	}
	if !strings.Contains(strings.Join(avisos, " "), "base_url") {
		t.Errorf("el aviso debe nombrar el campo ignorado: %v", avisos)
	}
}

func TestLaConfiguracionGlobalSiPuedeFijarBaseURL(t *testing.T) {
	home := t.TempDir()
	dirGlobal := filepath.Join(home, ".config", "calliope")
	if err := os.MkdirAll(dirGlobal, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(dirGlobal, FileName), map[string]string{
		"base_url": "https://staging.calliope.so",
	})

	cfg, _, err := Load(t.TempDir(), testEnv(home), nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.BaseURL(); got != "https://staging.calliope.so" {
		t.Errorf("base_url = %q, la capa global sí debe poder fijarlo", got)
	}
}

func TestLaConfiguracionDeProyectoNuncaAportaCredenciales(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, map[string]string{"api_key": "cal_live_robada"})

	cfg, avisos, err := Load(dir, testEnv(t.TempDir()), nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Get("api_key").Value; got != "" {
		t.Fatalf("api_key = %q — la configuración de proyecto no puede aportar credenciales", got)
	}
	if len(avisos) == 0 {
		t.Error("se esperaba un aviso sobre el campo ignorado")
	}
}

// TestLaConfiguracionDeProyectoSiPuedeFijarOutput cubre el camino positivo de
// la segunda clave permitida: hasta ahora solo se ejercitaba "org", y una
// frontera que solo se prueba con una de las dos claves permitidas podría
// tener la otra rota (o, tras el endurecimiento, apuntar por error a una
// clave inexistente en projectAllowed) sin que ningún test lo notara.
func TestLaConfiguracionDeProyectoSiPuedeFijarOutput(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, map[string]string{"output": "json"})

	cfg, avisos, err := Load(dir, testEnv(t.TempDir()), nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Output(); got != "json" {
		t.Errorf("Output = %q, la configuración de proyecto sí puede fijar output", got)
	}
	if got := cfg.Get(KeyOutput).Source; got != SourceProject {
		t.Errorf("origen de output = %q, se esperaba project", got)
	}
	if len(avisos) != 0 {
		t.Errorf("no se esperaban avisos, output es una clave permitida: %v", avisos)
	}
}

// TestProjectAllowedEsExactamenteOrgYOutput fija el contenido del conjunto
// permitido. Ampliarlo (por ejemplo añadiendo base_url o timeout) rompe este
// test a propósito, para que una ampliación silenciosa de la frontera de
// confianza sea visible en la revisión de código en vez de colarse como
// efecto colateral de otro cambio.
func TestProjectAllowedEsExactamenteOrgYOutput(t *testing.T) {
	permitidas := map[string]bool{
		KeyOrg:    true,
		KeyOutput: true,
	}
	todasLasClaves := []string{KeyOrg, KeyBaseURL, KeyOutput, KeyTimeout}

	for _, k := range todasLasClaves {
		got := IsProjectAllowed(k)
		want := permitidas[k]
		if got != want {
			t.Errorf("IsProjectAllowed(%q) = %v, se esperaba %v", k, got, want)
		}
	}

	// Claves que ni siquiera están en el catálogo de config.go: nunca deben
	// colarse por accidente (p. ej. por un typo que casualmente exista en el
	// mapa).
	for _, k := range []string{"api_key", "token", "proxy", ""} {
		if IsProjectAllowed(k) {
			t.Errorf("IsProjectAllowed(%q) = true, ninguna clave fuera de org/output debe estar permitida", k)
		}
	}
}

// --- ayudantes ---

func writeProjectConfig(t *testing.T, dir string, vals map[string]string) {
	t.Helper()
	d := filepath.Join(dir, ".calliope")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(d, FileName), vals)
}

func writeJSON(t *testing.T, ruta string, vals map[string]string) {
	t.Helper()
	b, err := json.Marshal(vals)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// testEnv aísla HOME para que los tests no lean la config real.
func testEnv(home string) func(string) string {
	return func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}
}
