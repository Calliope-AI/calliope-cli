// Package output define el contrato de salida del CLI: un envelope común
// para el éxito y el error, y los códigos de salida del proceso.
package output

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
type Envelope struct {
	OK          bool         `json:"ok"`
	Data        any          `json:"data,omitempty"`
	Summary     string       `json:"summary,omitempty"`
	Breadcrumbs []Breadcrumb `json:"breadcrumbs,omitempty"`
	Error       *Error       `json:"error,omitempty"`
}

// OKEnvelope construye un envelope correcto.
func OKEnvelope(data any, summary string, crumbs ...Breadcrumb) Envelope {
	return Envelope{OK: true, Data: data, Summary: summary, Breadcrumbs: crumbs}
}
