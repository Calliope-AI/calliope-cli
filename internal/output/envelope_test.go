package output

import (
	"bytes"
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

func TestEnvelopeDataPunteroNilTipadoSerializaNull(t *testing.T) {
	type Doc struct {
		ID string `json:"id"`
	}

	// Un puntero nil tipado se serializa como null, no se omite.
	env := Envelope{
		OK:   true,
		Data: (*Doc)(nil),
	}

	b, _ := json.Marshal(env)
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// omitempty no aplica a punteros nil tipados, solo a interfaces nil.
	if _, existe := got["data"]; !existe {
		t.Errorf("puntero nil tipado debería serializar como null, no omitirse: %s", b)
	}
	if got["data"] != nil {
		t.Errorf("puntero nil tipado debería ser null, fue: %v", got["data"])
	}
}

// TestOKEnvelopeNormalizaSliceNilAVacio es el test de I8 de la oleada
// final: §6.1 documenta que `data` siempre serializa como `[]` para una
// colección, pero un slice nil que llega directo a Envelope.Data serializa
// como `null` -TestEnvelopeDataPunteroNilTipadoSerializaNull, más arriba,
// documenta el porqué: la interfaz en sí no es nil-. Afectaba a docs list,
// rules list, concepts list, schema y query: sus slices vienen de
// json.Unmarshal sobre la respuesta del backend, y un `"data": null` (o la
// clave ausente) deja el slice de Go en nil. La propia receta del
// SKILL.md se rompía con esto: `calliope rules list --jq '.data[]'` daba
// "cannot iterate over: null" con exit 2 en vez de no imprimir nada.
//
// OKEnvelope -no Envelope{} directamente, que es lo que usan los dos tests
// de arriba a propósito para documentar el comportamiento crudo de
// encoding/json- es el único punto por el que pasa todo comando de éxito,
// así que normalizar ahí vale para cualquier lista sin tener que
// acordarse en cada comando.
func TestOKEnvelopeNormalizaSliceNilAVacio(t *testing.T) {
	type Rule struct {
		ID string `json:"id"`
	}

	casos := []struct {
		nombre string
		data   any
	}{
		{"[]string nil", []string(nil)},
		{"[]map[string]any nil (filas de query)", []map[string]any(nil)},
		{"[]Rule nil (struct con json tags)", []Rule(nil)},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			env := OKEnvelope(caso.data, "0 elementos")

			b, err := json.Marshal(env)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			var got map[string]any
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			raw, existe := got["data"]
			if !existe {
				t.Fatalf("data no debería omitirse: %s", b)
			}
			if raw == nil {
				t.Fatalf("data debería ser [], no null: %s", b)
			}
			arr, ok := raw.([]any)
			if !ok {
				t.Fatalf("data debería ser un array, fue %T: %s", raw, b)
			}
			if len(arr) != 0 {
				t.Errorf("data debería estar vacío, tuvo %d elementos: %s", len(arr), b)
			}

			// El propio marshal no basta: json.RawMessage de "null" también
			// pasaría un chequeo laxo. Se comprueba también el texto crudo,
			// que es justo lo que --jq '.data[]' recibe.
			if !bytes.Contains(b, []byte(`"data":[]`)) {
				t.Errorf("el JSON crudo debería contener \"data\":[], se obtuvo: %s", b)
			}
		})
	}
}

// TestOKEnvelopeConservaUnSliceConElementos comprueba que normalizeData no
// toca un slice que ya trae datos: solo actúa sobre slices nil.
func TestOKEnvelopeConservaUnSliceConElementos(t *testing.T) {
	env := OKEnvelope([]string{"a", "b"}, "2 elementos")
	arr, ok := env.Data.([]string)
	if !ok || len(arr) != 2 {
		t.Errorf("Data = %#v, se esperaba []string{\"a\", \"b\"} intacto", env.Data)
	}
}

// TestOKEnvelopeConservaValoresNoSlice comprueba que normalizeData deja
// intactos los valores que no son un slice (structs, punteros, mapas,
// escalares): la normalización de I8 es específica de colecciones.
func TestOKEnvelopeConservaValoresNoSlice(t *testing.T) {
	type Doc struct {
		ID string `json:"id"`
	}

	if env := OKEnvelope(nil, "sin datos"); env.Data != nil {
		t.Errorf("Data de un nil sin tipo debería seguir siendo nil, fue %#v", env.Data)
	}
	if env := OKEnvelope((*Doc)(nil), "puntero nil"); env.Data != (*Doc)(nil) {
		t.Errorf("Data de un puntero nil tipado no debería tocarse, fue %#v", env.Data)
	}
	env := OKEnvelope(map[string]string(nil), "mapa nil")
	m, ok := env.Data.(map[string]string)
	if !ok || m != nil {
		t.Errorf("Data de un mapa nil no lo toca normalizeData -solo slices-, se esperaba map[string]string(nil) intacto, fue %#v", env.Data)
	}
}

func TestEnvelopeDataSliceVacioSerializaArray(t *testing.T) {
	// Un slice vacío se serializa como array, no se omite.
	env := Envelope{
		OK:   true,
		Data: []string{},
	}

	b, _ := json.Marshal(env)
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Los slices vacíos se serializa como [].
	if _, existe := got["data"]; !existe {
		t.Errorf("slice vacío debería serializar como [], no omitirse: %s", b)
	}
	arr, ok := got["data"].([]any)
	if !ok {
		t.Errorf("slice vacío debería ser array, fue: %T", got["data"])
	}
	if len(arr) != 0 {
		t.Errorf("slice vacío debería tener longitud 0, fue: %d", len(arr))
	}
}
