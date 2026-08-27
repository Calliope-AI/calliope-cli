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
	return enc.Encode(v)
}
