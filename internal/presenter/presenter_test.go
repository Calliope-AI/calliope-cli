package presenter

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/calliope/calliope-cli/internal/output"
)

func testResult() Result {
	return Result{
		Envelope: output.OKEnvelope(
			[]map[string]any{{"id": "doc-1", "title": "Informe"}},
			"1 documento",
			output.Breadcrumb{Action: "show", Cmd: "calliope docs show doc-1"},
		),
		Text: func(w io.Writer) error {
			_, err := io.WriteString(w, "TEXTO HUMANO\n")
			return err
		},
		Markdown: func(w io.Writer) error {
			_, err := io.WriteString(w, "# Markdown\n")
			return err
		},
	}
}

func TestAutoEnTTYUsaElRenderHumano(t *testing.T) {
	var out bytes.Buffer
	err := Render(testResult(), Options{Mode: ModeAuto, IsTTY: true, Out: &out})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(out.String(), "TEXTO HUMANO") {
		t.Errorf("se esperaba el render humano, se obtuvo: %q", out.String())
	}
}

func TestAutoEnTuberiaUsaJSON(t *testing.T) {
	var out bytes.Buffer
	err := Render(testResult(), Options{Mode: ModeAuto, IsTTY: false, Out: &out})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	var env map[string]any
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("la salida en tubería debe ser JSON válido: %v (%q)", err, out.String())
	}
	if env["ok"] != true {
		t.Errorf("se esperaba el envelope completo, se obtuvo: %q", out.String())
	}
}

func TestQuietEmiteSoloData(t *testing.T) {
	var out bytes.Buffer
	err := Render(testResult(), Options{Mode: ModeQuiet, Out: &out})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	var data []map[string]any
	if err := json.Unmarshal(out.Bytes(), &data); err != nil {
		t.Fatalf("quiet debe emitir solo el array de data: %v (%q)", err, out.String())
	}
	if len(data) != 1 || data[0]["id"] != "doc-1" {
		t.Errorf("data inesperada: %q", out.String())
	}
}

func TestMarkdownUsaElRenderMarkdown(t *testing.T) {
	var out bytes.Buffer
	if err := Render(testResult(), Options{Mode: ModeMarkdown, Out: &out}); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.HasPrefix(out.String(), "# Markdown") {
		t.Errorf("se esperaba el render markdown, se obtuvo: %q", out.String())
	}
}

func TestSinRenderHumanoCaeAJSON(t *testing.T) {
	r := testResult()
	r.Text = nil

	var out bytes.Buffer
	if err := Render(r, Options{Mode: ModeAuto, IsTTY: true, Out: &out}); err != nil {
		t.Fatalf("render: %v", err)
	}
	var env map[string]any
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("sin Text debe caer a JSON: %v (%q)", err, out.String())
	}
}

func TestSinRenderMarkdownCaeAJSON(t *testing.T) {
	r := testResult()
	r.Markdown = nil

	var out bytes.Buffer
	if err := Render(r, Options{Mode: ModeMarkdown, Out: &out}); err != nil {
		t.Fatalf("render: %v", err)
	}
	var env map[string]any
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("sin Markdown debe caer a JSON: %v (%q)", err, out.String())
	}
	if env["ok"] != true {
		t.Errorf("se esperaba el envelope completo, se obtuvo: %q", out.String())
	}
}
