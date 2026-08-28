package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

func TestWriteErrorModoTextoConMensajeYHint(t *testing.T) {
	buf := &bytes.Buffer{}
	err := NewError(CodeNotFound, "Documento no encontrado.", "Lista los documentos con: calliope docs list")
	WriteError(buf, err, false)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Documento no encontrado.")) {
		t.Errorf("output debe incluir mensaje: %q", output)
	}
	if !bytes.Contains([]byte(output), []byte("Lista los documentos con: calliope docs list")) {
		t.Errorf("output debe incluir hint: %q", output)
	}
}

func TestWriteErrorModoTextoConCLIErrorEnvuelto(t *testing.T) {
	buf := &bytes.Buffer{}
	base := NewError(CodeUnauthorized, "No autorizado.", "Ejecuta: calliope auth login")
	envuelto := fmt.Errorf("al consultar documentos: %w", base)
	WriteError(buf, envuelto, false)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("No autorizado.")) {
		t.Errorf("output debe incluir mensaje base: %q", output)
	}
	if !bytes.Contains([]byte(output), []byte("Ejecuta: calliope auth login")) {
		t.Errorf("output debe incluir hint del error envuelto: %q", output)
	}
}

func TestWriteErrorModoJSONConEnvelope(t *testing.T) {
	buf := &bytes.Buffer{}
	err := NewError(CodeNotFound, "Documento no encontrado.", "Lista los documentos con: calliope docs list")
	WriteError(buf, err, true)

	output := buf.Bytes()
	var envelope map[string]any
	if err := json.Unmarshal(output, &envelope); err != nil {
		t.Fatalf("output debe ser JSON válido: %v", err)
	}

	if envelope["ok"] != false {
		t.Errorf("ok debe ser false, fue: %v", envelope["ok"])
	}
	if _, existe := envelope["data"]; existe {
		t.Errorf("JSON no debe incluir 'data' en error: %s", output)
	}

	errObj, ok := envelope["error"].(map[string]any)
	if !ok {
		t.Fatalf("error object missing")
	}
	if errObj["code"] != "NOT_FOUND" {
		t.Errorf("code debe ser NOT_FOUND, fue: %v", errObj["code"])
	}
	if errObj["message"] != "Documento no encontrado." {
		t.Errorf("message incorrecto: %v", errObj["message"])
	}
	if errObj["hint"] != "Lista los documentos con: calliope docs list" {
		t.Errorf("hint incorrecto: %v", errObj["hint"])
	}
}

func TestWriteErrorCLIErrorSinHintNoSerializaHintEnJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	err := NewError(CodeGeneric, "Algo salió mal.", "")
	WriteError(buf, err, true)

	output := buf.Bytes()
	var envelope map[string]any
	if err := json.Unmarshal(output, &envelope); err != nil {
		t.Fatalf("output debe ser JSON válido: %v", err)
	}

	errObj, ok := envelope["error"].(map[string]any)
	if !ok {
		t.Fatalf("error object missing")
	}

	// omitempty debe omitir hint cuando es vacío
	if _, existe := errObj["hint"]; existe {
		t.Errorf("JSON no debe incluir hint vacío: %s", output)
	}
}

func TestWriteErrorCLIErrorSinHintModoTexto(t *testing.T) {
	buf := &bytes.Buffer{}
	err := NewError(CodeGeneric, "Algo salió mal.", "")
	WriteError(buf, err, false)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Algo salió mal.")) {
		t.Errorf("output debe incluir mensaje: %q", output)
	}

	// Contar líneas: debe haber solo una (mensaje), no dos (mensaje + hint vacío)
	lines := bytes.Count([]byte(output), []byte("\n"))
	if lines != 1 {
		t.Errorf("output debe tener una línea, tiene: %d", lines)
	}
}

func TestWriteErrorErrorGenericoModoJSONMapaAEnvelope(t *testing.T) {
	buf := &bytes.Buffer{}
	err := errors.New("unknown command")
	WriteError(buf, err, true)

	output := buf.Bytes()
	var envelope map[string]any
	if err := json.Unmarshal(output, &envelope); err != nil {
		t.Fatalf("output debe ser JSON válido: %v", err)
	}

	if envelope["ok"] != false {
		t.Errorf("ok debe ser false, fue: %v", envelope["ok"])
	}

	errObj, ok := envelope["error"].(map[string]any)
	if !ok {
		t.Fatalf("error object missing")
	}
	if errObj["code"] != "ERROR" {
		t.Errorf("code debe ser ERROR para errores genéricos, fue: %v", errObj["code"])
	}
	if errObj["message"] != "unknown command" {
		t.Errorf("message debe ser el texto del error: %v", errObj["message"])
	}
}

func TestWriteErrorErrorGenericoModoTexto(t *testing.T) {
	buf := &bytes.Buffer{}
	err := errors.New("unknown command")
	WriteError(buf, err, false)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("unknown command")) {
		t.Errorf("output debe incluir mensaje del error: %q", output)
	}
}

// TestWriteErrorModoJSONNoEscapaHTML es el simétrico de
// presenter.TestRenderJSONNoEscapaHTML: un hint con "<" y ">" (el
// placeholder típico de un argumento, p.ej. "calliope orgs use
// <organización>") debe salir literal en el JSON de error, no como
// < y >. Antes de esta ronda, WriteError serializaba con
// json.Marshal a secas -un serializador JSON distinto del que usa
// presenter.writeJSON- así que el mismo contrato de envelope se comportaba
// distinto en JSON según el resultado fuera éxito o error.
func TestWriteErrorModoJSONNoEscapaHTML(t *testing.T) {
	buf := &bytes.Buffer{}
	err := NewError(CodeUsage, "Falta un argumento.", "Uso: calliope orgs use <organización>")
	if err := WriteError(buf, err, true); err != nil {
		t.Fatalf("WriteError: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "<organización>") {
		t.Errorf(`el JSON debe llevar "<organización>" literal, se obtuvo: %q`, output)
	}
	if strings.Contains(output, `\u003c`) || strings.Contains(output, `\u003e`) {
		t.Errorf("el JSON escapó < o > como \\u003c/\\u003e: %q", output)
	}
}
