// Package presenter decide en qué formato sale un resultado. Los comandos
// producen un Result; el modo elegido determina el render.
package presenter

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/calliope/calliope-cli/internal/output"
)

// Mode es el formato de salida solicitado.
type Mode string

const (
	ModeAuto     Mode = "auto" // humano en TTY, JSON en tubería
	ModeJSON     Mode = "json"
	ModeQuiet    Mode = "quiet"
	ModeMarkdown Mode = "md"
	ModeJQ       Mode = "jq"
)

// Options controla el render de una invocación.
type Options struct {
	Mode   Mode
	JQExpr string
	IsTTY  bool
	Out    io.Writer
}

// IsMachineReadable indica si esta invocación pidió -explícita, con un
// flag o la configuración, o implícitamente, por no ser un terminal
// interactivo- un formato pensado para que lo lea un programa, no una
// persona sentada en su terminal. Solo el modo automático en un terminal
// interactivo NO lo es.
//
// main() la usa para decidir cómo sacar un ERROR: como el mismo envelope
// JSON que ya promete el SKILL.md para el éxito, o como texto plano. Antes
// esa decisión se resolvía aparte, con un escaneo de os.Args en busca del
// token literal "--json" -desconectado de este Options, que ya resuelve
// Build/BuildWithoutCredential para el éxito-, así que en tubería, con
// --jq, --quiet, --md, --json=true o CALLIOPE_OUTPUT=json el éxito salía en
// JSON y el error salía en texto plano (C2 de la oleada final). Vive aquí,
// como método de Options, para que no haya una segunda copia de la fórmula
// que pueda desincronizarse otra vez.
func (o Options) IsMachineReadable() bool {
	return o.Mode != ModeAuto || !o.IsTTY
}

// Result es lo que produce un comando: el envelope, más los renders humanos
// opcionales. Si Text o Markdown son nil, se cae a JSON.
type Result struct {
	Envelope output.Envelope
	Text     func(io.Writer) error
	Markdown func(io.Writer) error
}

// Render escribe el resultado en el formato pedido.
func Render(r Result, opts Options) error {
	w := opts.Out
	if w == nil {
		return fmt.Errorf("presenter: Options.Out es obligatorio")
	}

	switch opts.Mode {
	case ModeJQ:
		return renderJQ(w, r.Envelope, opts.JQExpr)
	case ModeQuiet:
		return writeJSON(w, r.Envelope.Data)
	case ModeMarkdown:
		if r.Markdown != nil {
			return r.Markdown(w)
		}
		return writeJSON(w, r.Envelope)
	case ModeJSON:
		return writeJSON(w, r.Envelope)
	default: // ModeAuto
		if opts.IsTTY && r.Text != nil {
			return r.Text(w)
		}
		return writeJSON(w, r.Envelope)
	}
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	// Sin esto, "<" y ">" salen como \u003c y \u003e: JSON válido, pero
	// ilegible en un hint como "calliope orgs use <organización>". El
	// consumidor de este modo es un agente o una persona leyendo la salida
	// cruda, no un navegador, así que no hace falta la cautela de html/template.
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
