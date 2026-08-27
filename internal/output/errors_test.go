package output

import (
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
