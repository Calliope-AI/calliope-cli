package presenter

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestTableAlinea(t *testing.T) {
	var out bytes.Buffer
	headers := []string{"ID", "Nombre"}
	rows := [][]string{
		{"1", "Documento A"},
		{"2", "Muy Largo"},
	}

	err := Table(&out, headers, rows)
	if err != nil {
		t.Fatalf("Table: %v", err)
	}

	output := out.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Errorf("se esperaban 3 líneas (cabecera + 2 datos), se obtuvieron %d", len(lines))
	}

	// Las columnas deben estar alineadas: espacios constantes entre ID y Nombre
	if !strings.Contains(lines[0], "ID") || !strings.Contains(lines[0], "Nombre") {
		t.Errorf("cabecera mal formada: %q", lines[0])
	}
	if !strings.Contains(lines[1], "1") || !strings.Contains(lines[1], "Documento A") {
		t.Errorf("primera fila mal formada: %q", lines[1])
	}
}

func TestTablePropagaErrorDeWriter(t *testing.T) {
	mockErr := errors.New("mock write error")
	failingWriter := &failWriter{err: mockErr}

	headers := []string{"A", "B"}
	rows := [][]string{{"1", "2"}}

	err := Table(failingWriter, headers, rows)
	if err != mockErr {
		t.Errorf("se esperaba que el error se propagara, se obtuvo: %v", err)
	}
}

type failWriter struct {
	err error
}

func (fw *failWriter) Write(b []byte) (int, error) {
	return 0, fw.err
}
