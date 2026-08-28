package cli

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/skills"
)

// El SKILL.md va embebido en el binario, así que un agente que tenga el
// binario tiene su documentación exacta. Este test es lo que hace que esa
// promesa sea cierta: sin él, el skill miente en cuanto alguien añade,
// renombra o borra un comando.
func TestParidadEntreElCatalogoYElSkill(t *testing.T) {
	enElCLI := map[string]bool{}
	for _, c := range Catalog(NewRootCmd(appctx.Deps{})) {
		enElCLI[c.Path] = true
	}

	enElSkill := documentedCommands(t, skills.SkillMD)

	var faltan, sobran []string
	for cmd := range enElCLI {
		if !enElSkill[cmd] {
			faltan = append(faltan, cmd)
		}
	}
	for cmd := range enElSkill {
		if !enElCLI[cmd] {
			sobran = append(sobran, cmd)
		}
	}
	sort.Strings(faltan)
	sort.Strings(sobran)

	if len(faltan) > 0 {
		t.Errorf("comandos del CLI sin documentar en SKILL.md: %v\n"+
			"Añádelos entre los marcadores <!-- catalogo:inicio --> y <!-- catalogo:fin -->.", faltan)
	}
	if len(sobran) > 0 {
		t.Errorf("comandos documentados en SKILL.md que ya no existen: %v\n"+
			"El skill estaría mintiendo a los agentes; quítalos.", sobran)
	}
}

var commandLineRe = regexp.MustCompile("^- `calliope ([^`]+)`")

// documentedCommands extrae los comandos del bloque delimitado por los
// marcadores de catálogo, quedándose con el nombre sin argumentos.
func documentedCommands(t *testing.T, md string) map[string]bool {
	t.Helper()

	inicio := strings.Index(md, "<!-- catalogo:inicio -->")
	fin := strings.Index(md, "<!-- catalogo:fin -->")
	if inicio < 0 || fin < 0 || fin < inicio {
		t.Fatal("SKILL.md debe contener los marcadores <!-- catalogo:inicio --> y <!-- catalogo:fin -->")
	}

	out := map[string]bool{}
	for _, linea := range strings.Split(md[inicio:fin], "\n") {
		m := commandLineRe.FindStringSubmatch(strings.TrimSpace(linea))
		if m == nil {
			continue
		}
		out[commandName(m[1])] = true
	}
	return out
}

// commandName recorta los argumentos: "docs show <id>" → "docs show".
func commandName(s string) string {
	var partes []string
	for _, p := range strings.Fields(s) {
		if strings.HasPrefix(p, "<") || strings.HasPrefix(p, "[") || strings.HasPrefix(p, "--") {
			break
		}
		partes = append(partes, p)
	}
	return strings.Join(partes, " ")
}
