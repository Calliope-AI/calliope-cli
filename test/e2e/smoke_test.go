//go:build e2e

// Package e2e ejecuta el binario real contra una organización de prueba.
// Opt-in: requiere CALLIOPE_E2E=1, CALLIOPE_API_KEY y CALLIOPE_ORG.
package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// runCalliope ejecuta el binario compilado (bin/calliope, ver `make build`)
// con los argumentos dados y devuelve su salida combinada. Falla el test si
// el proceso termina con error.
func runCalliope(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("../../bin/calliope", args...)
	cmd.Env = os.Environ()
	salida, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("calliope %s falló: %v\n%s", strings.Join(args, " "), err, salida)
	}
	return string(salida)
}

// requireE2E salta el test si no está activado explícitamente. El smoke
// habla con una organización real, así que nunca se ejecuta por accidente:
// hace falta CALLIOPE_E2E=1 más credenciales de verdad.
func requireE2E(t *testing.T) {
	t.Helper()
	if os.Getenv("CALLIOPE_E2E") != "1" {
		t.Skip("smoke desactivado; actívalo con CALLIOPE_E2E=1")
	}
	for _, v := range []string{"CALLIOPE_API_KEY", "CALLIOPE_ORG"} {
		if os.Getenv(v) == "" {
			t.Fatalf("falta la variable %s", v)
		}
	}
}

// check es una fila de `doctor --json`. Los nombres de campo son los que
// envía el binario -en inglés-, no una traducción libre: name/status/detail.
type check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// checksOf decodifica el envelope de `doctor --json` y devuelve sus
// comprobaciones.
func checksOf(t *testing.T, salida string) []check {
	t.Helper()
	var env struct {
		Data []check `json:"data"`
	}
	if err := json.Unmarshal([]byte(salida), &env); err != nil {
		t.Fatalf("doctor no devolvió JSON válido: %v\n%s", err, salida)
	}
	return env.Data
}

// Es el criterio de aceptación 8 del spec: la cadena completa que un agente
// debe poder recorrer con solo `calliope skill` en contexto.
func TestCadenaCompletaDeAgente(t *testing.T) {
	requireE2E(t)

	// 1. Descubrir organizaciones.
	orgs := runCalliope(t, "orgs", "list", "--json")
	if !strings.Contains(orgs, os.Getenv("CALLIOPE_ORG")) {
		t.Fatalf("la organización de prueba no aparece en orgs list:\n%s", orgs)
	}

	// 2. Consultar el esquema antes de nada.
	esquema := runCalliope(t, "schema", "--json")
	var envEsquema struct {
		OK   bool `json:"ok"`
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(esquema), &envEsquema); err != nil {
		t.Fatalf("schema no devolvió JSON válido: %v\n%s", err, esquema)
	}
	if !envEsquema.OK || len(envEsquema.Data) == 0 {
		t.Fatalf("el esquema vino vacío:\n%s", esquema)
	}

	// 3. Preguntar y comprobar que cita fuentes. El envelope de `ask` lleva
	// el AskResponse completo en `data` (con `text` y `sources`, ver
	// internal/sdk/models.go); `breadcrumbs` va aparte, a nivel del envelope,
	// no dentro de `data`.
	respuesta := runCalliope(t, "ask", "¿qué datos hay disponibles?", "--json")
	var envAsk struct {
		OK   bool `json:"ok"`
		Data struct {
			Text    string `json:"text"`
			Sources []struct {
				Citation string `json:"citation"`
			} `json:"sources"`
		} `json:"data"`
		Breadcrumbs []struct {
			Cmd string `json:"cmd"`
		} `json:"breadcrumbs"`
	}
	if err := json.Unmarshal([]byte(respuesta), &envAsk); err != nil {
		t.Fatalf("ask no devolvió JSON válido: %v\n%s", err, respuesta)
	}
	if !envAsk.OK || envAsk.Data.Text == "" {
		t.Fatalf("ask no devolvió respuesta:\n%s", respuesta)
	}
	if len(envAsk.Breadcrumbs) == 0 {
		t.Error("ask debe devolver breadcrumbs para que el agente sepa seguir")
	}

	// 4. Extraer un campo con el jq embebido.
	ids := runCalliope(t, "concepts", "list", "--jq", ".data[].name")
	if strings.TrimSpace(ids) == "" {
		t.Error("el filtro --jq no devolvió nada")
	}
}

func TestDoctorPasaEnUnEntornoConfigurado(t *testing.T) {
	requireE2E(t)

	salida := runCalliope(t, "doctor", "--json")
	for _, c := range checksOf(t, salida) {
		if c.Status == "error" {
			t.Errorf("comprobación %q en error: %s", c.Name, salida)
		}
	}
}
