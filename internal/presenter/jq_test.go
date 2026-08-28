package presenter

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/Calliope-AI/calliope-cli/internal/output"
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

// TestJQErrorEnEvaluacionDevuelveErrorDeUso cubre I10 de la oleada final:
// era el único CLIError de todo el CLI con el hint vacío -el de parseo,
// justo encima en jq.go, sí apunta al manual- y con el mensaje de gojq
// embebido en inglés sin ninguna frase en español alrededor.
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

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T", err)
	}
	if cliErr.Hint == "" {
		t.Error("el error de evaluación jq debería traer un hint, como ya trae el de parseo")
	}
	if !strings.HasPrefix(cliErr.Message, "Error al evaluar la expresión jq: ") {
		t.Errorf("el mensaje debería ir enmarcado en español, fue: %q", cliErr.Message)
	}
}
