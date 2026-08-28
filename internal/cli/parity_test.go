package cli

import (
	"fmt"
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
// renombra o borra un comando, o le cambia la descripción o los argumentos
// sin avisar.
func TestParidadEntreElCatalogoYElSkill(t *testing.T) {
	enElCLI := map[string]CommandInfo{}
	for _, c := range Catalog(NewRootCmd(appctx.Deps{})) {
		enElCLI[c.Path] = c
	}

	enElSkill := documentedCommands(t, skills.SkillMD)

	var faltan, sobran []string
	for path := range enElCLI {
		if _, ok := enElSkill[path]; !ok {
			faltan = append(faltan, path)
		}
	}
	for path := range enElSkill {
		if _, ok := enElCLI[path]; !ok {
			sobran = append(sobran, path)
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

	// Para los comandos que están a ambos lados, no basta con que existan:
	// la descripción tiene que seguir contando lo mismo que el Short real, y
	// la forma de los argumentos tiene que seguir siendo la misma que la del
	// Use real. Si cualquiera de las dos se desincroniza, el SKILL.md
	// documenta un comando que ya no se comporta -o no se invoca- como dice.
	var descripcionesDistintas, argumentosDistintos []string
	for path, cli := range enElCLI {
		doc, ok := enElSkill[path]
		if !ok {
			continue // ya reportado arriba como "faltan"
		}
		if normalizeDescription(cli.Short) != normalizeDescription(doc.Description) {
			descripcionesDistintas = append(descripcionesDistintas,
				fmt.Sprintf("%s: Short=%q SKILL.md=%q", path, cli.Short, doc.Description))
		}
		if strings.TrimSpace(cli.Args) != strings.TrimSpace(doc.Args) {
			argumentosDistintos = append(argumentosDistintos,
				fmt.Sprintf("%s: Use=%q SKILL.md=%q", path, cli.Args, doc.Args))
		}
	}
	sort.Strings(descripcionesDistintas)
	sort.Strings(argumentosDistintos)

	if len(descripcionesDistintas) > 0 {
		t.Errorf("la descripción del SKILL.md no coincide con el Short real del comando "+
			"(aparte de mayúsculas y espacios): %v\n"+
			"Actualiza el texto en SKILL.md para que diga lo mismo que el Short.", descripcionesDistintas)
	}
	if len(argumentosDistintos) > 0 {
		t.Errorf("la forma de los argumentos documentada en SKILL.md no coincide con el Use real del comando: %v\n"+
			"Actualiza los marcadores <...>/[...] en SKILL.md para que coincidan.", argumentosDistintos)
	}
}

var commandLineRe = regexp.MustCompile("^- `calliope ([^`]+)` — (.+)$")

// documentedCommand es la entrada del SKILL.md para un comando: la forma de
// sus argumentos y su descripción, listas para compararlas contra el
// catálogo real.
type documentedCommand struct {
	Args        string
	Description string
}

// documentedCommands extrae los comandos del bloque delimitado por los
// marcadores de catálogo.
func documentedCommands(t *testing.T, md string) map[string]documentedCommand {
	t.Helper()

	inicio := strings.Index(md, "<!-- catalogo:inicio -->")
	fin := strings.Index(md, "<!-- catalogo:fin -->")
	if inicio < 0 || fin < 0 || fin < inicio {
		t.Fatal("SKILL.md debe contener los marcadores <!-- catalogo:inicio --> y <!-- catalogo:fin -->")
	}

	out := map[string]documentedCommand{}
	for _, linea := range strings.Split(md[inicio:fin], "\n") {
		m := commandLineRe.FindStringSubmatch(strings.TrimSpace(linea))
		if m == nil {
			continue
		}
		out[commandName(m[1])] = documentedCommand{
			Args:        argsUsageOf(m[1]),
			Description: m[2],
		}
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

// argsUsageOf se queda con la forma de los argumentos: "docs show <id>" →
// "<id>". Es el complementario de commandName: entre los dos se reparten
// todos los campos de s.
func argsUsageOf(s string) string {
	var partes []string
	corte := false
	for _, p := range strings.Fields(s) {
		if !corte && (strings.HasPrefix(p, "<") || strings.HasPrefix(p, "[") || strings.HasPrefix(p, "--")) {
			corte = true
		}
		if corte {
			partes = append(partes, p)
		}
	}
	return strings.Join(partes, " ")
}

// normalizeDescription recorta espacios y descarta mayúsculas: dos
// descripciones que solo difieren en eso cuentan como la misma, para que la
// lista del skill pueda empezar en minúscula sin pelearse con el Short.
func normalizeDescription(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
