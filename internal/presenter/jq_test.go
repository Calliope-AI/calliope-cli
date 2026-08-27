package presenter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/calliope/calliope-cli/internal/output"
)

func TestJQExtraeUnCampo(t *testing.T) {
	r := Result{Envelope: output.OKEnvelope(
		[]map[string]any{{"id": "doc-1"}, {"id": "doc-2"}}, "2 documentos")}

	var out bytes.Buffer
	err := Render(r, Options{Mode: ModeJQ, JQExpr: ".data[].id", Out: &out})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	lineas := strings.Fields(out.String())
	if len(lineas) != 2 || lineas[0] != `"doc-1"` || lineas[1] != `"doc-2"` {
		t.Errorf("salida jq inesperada: %q", out.String())
	}
}

func TestJQInvalidoDevuelveErrorDeUso(t *testing.T) {
	r := Result{Envelope: output.OKEnvelope(nil, "")}

	var out bytes.Buffer
	err := Render(r, Options{Mode: ModeJQ, JQExpr: ".data[", Out: &out})
	if err == nil {
		t.Fatal("se esperaba un error por expresión jq inválida")
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2 (uso incorrecto)", got)
	}
}

func TestJQErrorEnEvaluacionDevuelveErrorDeUso(t *testing.T) {
	// Expresión sintácticamente válida pero que falla al evaluar:
	// .data es un array, no tiene propiedades, así que .data.foo es un error de tipo.
	r := Result{Envelope: output.OKEnvelope(
		[]map[string]any{{"id": "doc-1"}}, "1 documento")}

	var out bytes.Buffer
	err := Render(r, Options{Mode: ModeJQ, JQExpr: ".data.foo", Out: &out})
	if err == nil {
		t.Fatal("se esperaba un error por evaluación jq fallida")
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2 (uso incorrecto)", got)
	}
}
