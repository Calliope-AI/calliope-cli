package output

import (
	"encoding/json"
	"testing"
)

func TestEnvelopeCorrectoOmiteError(t *testing.T) {
	env := Envelope{
		OK:      true,
		Data:    []string{"a"},
		Summary: "1 elemento",
		Breadcrumbs: []Breadcrumb{
			{Action: "show", Cmd: "calliope docs show <id>"},
		},
	}

	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, existe := got["error"]; existe {
		t.Errorf("un envelope correcto no debe incluir 'error': %s", b)
	}
	if got["ok"] != true {
		t.Errorf("ok debería ser true: %s", b)
	}
}

func TestEnvelopeDeErrorOmiteData(t *testing.T) {
	env := NewError(CodeNotFound, "Documento no encontrado.", "Lista los documentos con: calliope docs list").Envelope()

	b, _ := json.Marshal(env)
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, existe := got["data"]; existe {
		t.Errorf("un envelope de error no debe incluir 'data': %s", b)
	}
	e, ok := got["error"].(map[string]any)
	if !ok {
		t.Fatalf("falta el objeto 'error': %s", b)
	}
	if e["code"] != "NOT_FOUND" {
		t.Errorf("code = %v, se esperaba NOT_FOUND", e["code"])
	}
	if e["hint"] == "" || e["hint"] == nil {
		t.Errorf("se esperaba un hint accionable: %s", b)
	}
}
