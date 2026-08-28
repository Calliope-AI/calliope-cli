// Package output define el contrato de salida del CLI: un envelope común
// para el éxito y el error, y los códigos de salida del proceso.
package output

import "reflect"

// Breadcrumb sugiere el siguiente comando. Es la pieza central del diseño
// para agentes: la respuesta enseña cómo continuar, así que el agente navega
// sin recargar documentación en su contexto.
type Breadcrumb struct {
	Action string `json:"action"`
	Cmd    string `json:"cmd"`
}

// Error es la representación serializable de un fallo.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

// Envelope es la forma de toda salida JSON del CLI.
// El campo Data usa omitempty, pero nota: solo omite si el valor de la interfaz es nil.
// Un puntero nil tipado (ej. (*Doc)(nil)) serializa como null, no se omite.
// Un slice vacío serializa como [], no se omite. Esto es intencionado.
// Un slice nil también serializaría como null y tampoco se omitiría -por la
// misma razón que el puntero nil tipado: la interfaz en sí no es nil-, así
// que OKEnvelope lo normaliza a un slice vacío antes de guardarlo aquí (ver
// normalizeData): un `data` de colección siempre es `[]`, nunca `null`.
type Envelope struct {
	OK          bool         `json:"ok"`
	Data        any          `json:"data,omitempty"`
	Summary     string       `json:"summary,omitempty"`
	Breadcrumbs []Breadcrumb `json:"breadcrumbs,omitempty"`
	Error       *Error       `json:"error,omitempty"`
}

// OKEnvelope construye un envelope correcto.
func OKEnvelope(data any, summary string, crumbs ...Breadcrumb) Envelope {
	return Envelope{OK: true, Data: normalizeData(data), Summary: summary, Breadcrumbs: crumbs}
}

// normalizeData sustituye un slice nil por uno vacío del mismo tipo (I8 de
// la oleada final): §6.1 documenta que `data` siempre serializa como `[]`
// para una colección, pero un slice nil dentro de un campo `any` no es un
// valor de interfaz nil -sigue llevando su tipo, así que Data,omitempty no
// lo omite (ver el comentario del campo Data, más abajo)-, y
// encoding/json serializa un slice nil como `null`, no como `[]`.
//
// Afecta a docs list, rules list, concepts list, schema y query: sus
// slices vienen de json.Unmarshal sobre lo que el backend responda, y un
// `"data": null` o una clave ausente en la respuesta del backend deja el
// slice de Go en nil. La propia receta del SKILL.md se rompía con esto:
// `calliope rules list --jq '.data[]'` daba "cannot iterate over: null"
// con exit 2 en vez de no imprimir nada.
//
// Vive aquí -en el único punto por el que pasa todo envelope de éxito- en
// vez de en cada comando, para que valga también para cualquier lista
// futura sin tener que acordarse de normalizarla cada vez.
func normalizeData(data any) any {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Slice && v.IsNil() {
		return reflect.MakeSlice(v.Type(), 0, 0).Interface()
	}
	return data
}
