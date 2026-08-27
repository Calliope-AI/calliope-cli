package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Code identifica la clase de fallo y determina el código de salida.
type Code string

const (
	CodeGeneric      Code = "ERROR"
	CodeUsage        Code = "USAGE"
	CodeUnauthorized Code = "UNAUTHORIZED"
	CodeNotFound     Code = "NOT_FOUND"
	CodeRateLimited  Code = "RATE_LIMITED"
)

// ExitCode traduce el código de error al código de salida del proceso.
func (c Code) ExitCode() int {
	switch c {
	case CodeUsage:
		return 2
	case CodeUnauthorized:
		return 3
	case CodeNotFound:
		return 4
	case CodeRateLimited:
		return 5
	default:
		return 1
	}
}

// CLIError es un fallo con mensaje para el usuario y, cuando existe una
// acción de recuperación, la pista para llevarla a cabo.
type CLIError struct {
	Code    Code
	Message string
	Hint    string
}

// NewError construye un CLIError. Pasa hint vacío solo si de verdad no hay
// nada que el usuario pueda hacer.
func NewError(code Code, message, hint string) *CLIError {
	return &CLIError{Code: code, Message: message, Hint: hint}
}

func (e *CLIError) Error() string { return e.Message }

// ExitCode es el código de salida que corresponde a este error.
func (e *CLIError) ExitCode() int { return e.Code.ExitCode() }

// Envelope serializa el error con la misma forma que una respuesta correcta.
func (e *CLIError) Envelope() Envelope {
	return Envelope{
		OK:    false,
		Error: &Error{Code: string(e.Code), Message: e.Message, Hint: e.Hint},
	}
}

// ExitCodeFor devuelve el código de salida de cualquier error, desenvolviendo
// las cadenas creadas con %w.
func ExitCodeFor(err error) int {
	if err == nil {
		return 0
	}
	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		return cliErr.ExitCode()
	}
	return 1
}

// WriteError escribe el error en w según el modo (JSON o texto).
// En modo JSON, serializa el error como un Envelope completo.
// En modo texto, imprime el mensaje y el hint (si existe) en líneas separadas.
// Los errores genéricos (no CLIError) se mapean a un Envelope con código ERROR.
func WriteError(w io.Writer, err error, jsonMode bool) error {
	if err == nil {
		return nil
	}

	var cliErr *CLIError
	if !errors.As(err, &cliErr) {
		// Error genérico: mapear a CLIError con código ERROR.
		cliErr = NewError(CodeGeneric, err.Error(), "")
	}

	if jsonMode {
		// En modo JSON, serializar el envelope completo.
		envelope := cliErr.Envelope()
		b, err := json.Marshal(envelope)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, string(b))
		return err
	}

	// En modo texto, imprimir el mensaje y el hint por separado.
	if _, err := fmt.Fprintln(w, cliErr.Message); err != nil {
		return err
	}
	if cliErr.Hint != "" {
		if _, err := fmt.Fprintln(w, cliErr.Hint); err != nil {
			return err
		}
	}
	return nil
}
