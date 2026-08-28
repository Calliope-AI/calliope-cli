package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// El plugin apunta al mismo SKILL.md que embebe el binario. Si alguien
// duplicara el fichero en vez de enlazarlo, las dos copias divergirían.
func TestElPluginUsaElMismoSkillQueElBinario(t *testing.T) {
	raiz := repoRoot(t)

	delPlugin, err := os.ReadFile(filepath.Join(raiz, ".claude-plugin", "skills", "calliope", "SKILL.md"))
	if err != nil {
		t.Fatalf("el plugin debe exponer el skill: %v", err)
	}
	delRepo, err := os.ReadFile(filepath.Join(raiz, "skills", "calliope", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(delPlugin) != string(delRepo) {
		t.Error("el SKILL.md del plugin y el del binario han divergido; el plugin debe ser un enlace simbólico")
	}
}

func TestElManifiestoDelPluginEsValido(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("plugin.json no es JSON válido: %v", err)
	}
	if m.Name == "" || m.Version == "" || m.Description == "" {
		t.Errorf("plugin.json incompleto: %+v", m)
	}
}

// Ronda 1/5: verificado por mutación que, sin este test, se puede quitar el
// bit ejecutable del hook (o borrar el fichero entero) sin que la suite se
// entere -- el plugin quedaría con un hook que Claude Code no puede invocar.
func TestElHookDeSesionEsEjecutable(t *testing.T) {
	ruta := filepath.Join(repoRoot(t), ".claude-plugin", "hooks", "session-start.sh")
	info, err := os.Stat(ruta)
	if err != nil {
		t.Fatalf("el hook de sesión debe existir: %v", err)
	}
	if info.IsDir() {
		t.Fatalf("%s es un directorio, se esperaba el script", ruta)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("%s no tiene el bit ejecutable (modo %v)", ruta, info.Mode())
	}
}

// Ronda 1/5: verificado por mutación que, sin este test, se puede borrar
// commands/calliope.md sin que la suite se entere -- el plugin se instalaría
// sin el comando /calliope.
func TestElComandoCalliopeNoEstaVacio(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), ".claude-plugin", "commands", "calliope.md"))
	if err != nil {
		t.Fatalf("commands/calliope.md debe existir: %v", err)
	}
	if strings.TrimSpace(string(b)) == "" {
		t.Error("commands/calliope.md está vacío")
	}
}

// Ronda 1/5: verificado por mutación que, sin este test, se puede borrar el
// bloque hooks.SessionStart de plugin.json sin que la suite se entere -- el
// plugin se instalaría sin avisar nunca al empezar la sesión.
func TestElManifiestoDeclaraElHookDeSesion(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Hooks struct {
			SessionStart []struct {
				Hooks []struct {
					Type    string `json:"type"`
					Command string `json:"command"`
				} `json:"hooks"`
			} `json:"SessionStart"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("plugin.json no es JSON válido: %v", err)
	}
	if len(m.Hooks.SessionStart) == 0 || len(m.Hooks.SessionStart[0].Hooks) == 0 {
		t.Fatal("plugin.json no declara el hook SessionStart")
	}
	h := m.Hooks.SessionStart[0].Hooks[0]
	if h.Type != "command" {
		t.Errorf(`el hook SessionStart debe ser de tipo "command", tiene %q`, h.Type)
	}
	if !strings.Contains(h.Command, "${CLAUDE_PLUGIN_ROOT}/hooks/session-start.sh") {
		t.Errorf("el hook SessionStart no apunta a hooks/session-start.sh: %q", h.Command)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		padre := filepath.Dir(dir)
		if padre == dir {
			t.Fatal("no se encontró la raíz del repositorio")
		}
		dir = padre
	}
}
