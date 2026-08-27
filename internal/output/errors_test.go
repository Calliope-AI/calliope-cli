package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func TestCodigosDeSalidaPorCodigo(t *testing.T) {
	casos := []struct {
		code Code
		want int
	}{
		{CodeGeneric, 1},
		{CodeUsage, 2},
		{CodeUnauthorized, 3},
		{CodeNotFound, 4},
		{CodeRateLimited, 5},
	}
	for _, c := range casos {
		if got := c.code.ExitCode(); got != c.want {
			t.Errorf("%s.ExitCode() = %d, se esperaba %d", c.code, got, c.want)
		}
	}
}

func TestExitCodeForNilEsCero(t *testing.T) {
	if got := ExitCodeFor(nil); got != 0 {
		t.Errorf("ExitCodeFor(nil) = %d, se esperaba 0", got)
	}
}

func TestExitCodeForErrorGenericoEsUno(t *testing.T) {
	if got := ExitCodeFor(errors.New("cualquier cosa")); got != 1 {
		t.Errorf("ExitCodeFor(genérico) = %d, se esperaba 1", got)
	}
}

func TestExitCodeForDesenvuelveCLIError(t *testing.T) {
	base := NewError(CodeUnauthorized, "No autorizado.", "Ejecuta: calliope auth login")
	envuelto := fmt.Errorf("al consultar documentos: %w", base)

	if got := ExitCodeFor(envuelto); got != 3 {
		t.Errorf("ExitCodeFor(envuelto) = %d, se esperaba 3", got)
	}
}

func TestCLIErrorErrorDevolverSoloMensaje(t *testing.T) {
	err := NewError(CodeNotFound, "Documento no encontrado.", "Lista los documentos con: calliope docs list")
	if got := err.Error(); got != "Documento no encontrado." {
		t.Errorf("Error() = %q, se esperaba %q", got, "Documento no encontrado.")
	}
}

func TestCLIErrorEnvelopeLlevahint(t *testing.T) {
	err := NewError(CodeNotFound, "Documento no encontrado.", "Lista los documentos con: calliope docs list")
	env := err.Envelope()

	if env.Error == nil || env.Error.Hint == "" {
		t.Errorf("Envelope debe incluir hint: %+v", env.Error)
	}
	if env.Error.Hint != "Lista los documentos con: calliope docs list" {
		t.Errorf("Hint = %q, se esperaba %q", env.Error.Hint, "Lista los documentic con: calliope docs list")
	}
}

func TestCLIErrorEnvelopeJSONIncluirHint(t *testing.T) {
	err := NewError(CodeNotFound, "Documento no encontrado.", "Lista los documentos con: calliope docs list")
	env := err.Envelope()

	b, _ := json.Marshal(env)
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	errObj, ok := got["error"].(map[string]any)
	if !ok {
		t.Fatalf("error object missing")
	}

	if hint, ok := errObj["hint"]; !ok || hint == "" {
		t.Errorf("JSON del error debe incluir hint: %s", b)
	}
}
