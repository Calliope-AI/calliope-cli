package output

import "errors"

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
