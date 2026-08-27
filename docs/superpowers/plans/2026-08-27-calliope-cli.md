# Calliope CLI — plan de implementación

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Construir `calliope`, un binario Go único que da acceso a Calliope Data desde el terminal y desde agentes de IA, con el skill embebido en el propio binario.

**Architecture:** Cobra sobre cuatro capas aisladas: `config` (capas con procedencia y frontera de confianza) → `auth` (credencial tras una interfaz) → `sdk` (cliente HTTP de `data-0.calliope.so`) → `commands` (un fichero por grupo). Toda salida pasa por un envelope JSON común con breadcrumbs, y un `presenter` decide el formato. El SDK no conoce Cobra; los comandos no construyen URLs.

**Tech Stack:** Go 1.22+, Cobra, `github.com/itchyny/gojq` (filtro `--jq` embebido), `github.com/zalando/go-keyring` (credenciales), `net/http/httptest` (tests de SDK), GoReleaser.

**Spec:** `docs/superpowers/specs/2026-08-27-calliope-cli-design.md`

## Global Constraints

- **Go 1.22 o superior.** El `go.mod` declara `go 1.22`.
- **Backend:** `https://data-0.calliope.so`. Base de organización: `{base_url}/v1/organizations/{org}`.
- **El JSON del backend es camelCase.** Todo struct de respuesta lleva su tag `json:"…"` explícito; nunca confiar en el mapeo por defecto de Go.
- **Los mensajes de error nunca incluyen el cuerpo de la respuesta del backend.** Se mapea por status, siguiendo `mapError` de `calliope-data-mcp`.
- **Todo error de cara al usuario lleva `hint`** siempre que exista una acción de recuperación.
- **Idioma:** mensajes de usuario, ayuda de comandos y comentarios en español. Identificadores en inglés.
- **Códigos de salida:** 0 correcto · 1 error genérico · 2 uso incorrecto · 3 no autorizado · 4 no encontrado · 5 límite superado.
- **Las credenciales nunca se escriben en configuración de proyecto**, solo en keyring o en fichero global con permisos `0600`.
- **TDD sin excepciones:** el test se escribe y se ve fallar antes de la implementación.

## Estructura de ficheros

| Fichero | Responsabilidad |
|---|---|
| `cmd/calliope/main.go` | Punto de entrada; traduce error a código de salida |
| `internal/cli/root.go` | Comando raíz, flags globales, registro de subcomandos |
| `internal/cli/catalog.go` | Catálogo de comandos: fuente de verdad para el test de paridad |
| `internal/output/envelope.go` | `Envelope`, `Breadcrumb` |
| `internal/output/errors.go` | `Code`, `CLIError`, códigos de salida |
| `internal/presenter/presenter.go` | `Result`, `Options`, `Render` |
| `internal/presenter/jq.go` | Filtro `--jq` con gojq |
| `internal/presenter/table.go` | Render de tabla para TTY |
| `internal/config/config.go` | `Value`, `Config`, `Resolve` con procedencia |
| `internal/config/layers.go` | Carga de las seis capas |
| `internal/config/trust.go` | Frontera de confianza de la configuración de proyecto |
| `internal/auth/credential.go` | `Kind`, `Credential`, cabecera HTTP |
| `internal/auth/store.go` | `Store`, keyring con *fallback* a fichero `0600` |
| `internal/auth/resolve.go` | Resolución de credencial y su origen |
| `internal/auth/oauth.go` | Loopback PKCE con PropelAuth (condicional a Task 6) |
| `internal/sdk/client.go` | Transporte, scoping de org, timeout, mapeo de errores |
| `internal/sdk/models.go` | Tipos de respuesta del backend |
| `internal/sdk/api.go` | Un método por endpoint |
| `internal/appctx/appctx.go` | Wiring: config + credencial + cliente + org |
| `internal/commands/*.go` | Un fichero por grupo de comandos |
| `skills/calliope/SKILL.md` | Skill embebido |
| `skills/embed.go` | `go:embed` del skill |
| `.claude-plugin/` | Plugin de Claude Code |

---

### Task 1: Bootstrap del módulo, comando raíz y CI

**Files:**
- Create: `go.mod`, `cmd/calliope/main.go`, `internal/cli/root.go`, `internal/version/version.go`, `.github/workflows/ci.yml`, `Makefile`
- Test: `internal/cli/root_test.go`

**Interfaces:**
- Consumes: nada.
- Produces: `cli.NewRootCmd() *cobra.Command`; `version.Version`, `version.Commit`, `version.Date` (variables de paquete fijadas por `-ldflags`).

- [ ] **Step 1: Escribir el test que falla**

`internal/cli/root_test.go`:

```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootMuestraAyudaSinArgumentos(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if !strings.Contains(out.String(), "calliope") {
		t.Errorf("la ayuda no menciona el binario:\n%s", out.String())
	}
}

func TestVersionImprimeLaVersion(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if !strings.Contains(out.String(), "dev") {
		t.Errorf("se esperaba la versión por defecto 'dev', se obtuvo:\n%s", out.String())
	}
}
```

- [ ] **Step 2: Ejecutar el test y verificar que falla**

Run: `go test ./internal/cli/ -run TestRoot -v`
Expected: FAIL — `undefined: NewRootCmd`

- [ ] **Step 3: Crear el módulo y la implementación mínima**

```bash
go mod init github.com/calliope/calliope-cli
go get github.com/spf13/cobra@latest
```

`internal/version/version.go`:

```go
// Package version expone los datos de compilación, fijados con -ldflags.
package version

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)
```

`internal/cli/root.go`:

```go
// Package cli construye el árbol de comandos de calliope.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/version"
)

// NewRootCmd construye el comando raíz con sus flags globales.
// Se crea uno nuevo por invocación para que los tests no compartan estado.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "calliope",
		Short:         "Interfaz de línea de comandos de Calliope Data",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Flags globales; los consumen appctx y presenter en tareas posteriores.
	f := root.PersistentFlags()
	f.String("org", "", "organización sobre la que operar")
	f.Bool("json", false, "salida JSON con envelope completo")
	f.Bool("quiet", false, "salida solo de datos, sin envelope")
	f.Bool("md", false, "salida en Markdown")
	f.String("jq", "", "filtra la salida con una expresión jq")

	root.AddCommand(newVersionCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Muestra la versión de calliope",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "calliope %s (%s, %s)\n",
				version.Version, version.Commit, version.Date)
			return nil
		},
	}
}
```

`cmd/calliope/main.go`:

```go
// Command calliope es el punto de entrada del CLI de Calliope Data.
package main

import (
	"fmt"
	"os"

	"github.com/calliope/calliope-cli/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1) // Task 2 sustituye esto por el mapeo real de códigos.
	}
}
```

- [ ] **Step 4: Ejecutar los tests y verificar que pasan**

Run: `go test ./... -v`
Expected: PASS (2 tests)

- [ ] **Step 5: Añadir Makefile y CI**

`Makefile`:

```makefile
.PHONY: build test lint
build:
	go build -o bin/calliope ./cmd/calliope
test:
	go test ./... -race
lint:
	go vet ./...
```

`.github/workflows/ci.yml`:

```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go vet ./...
      - run: go test ./... -race
```

- [ ] **Step 6: Verificar la compilación y hacer commit**

Run: `make build && ./bin/calliope version`
Expected: `calliope dev (none, unknown)`

```bash
git add go.mod go.sum cmd internal .github Makefile
git commit -m "feat: bootstrap del módulo, comando raíz y CI"
```

---

### Task 2: Envelope, errores tipados y códigos de salida

**Files:**
- Create: `internal/output/envelope.go`, `internal/output/errors.go`
- Modify: `cmd/calliope/main.go`
- Test: `internal/output/envelope_test.go`, `internal/output/errors_test.go`

**Interfaces:**
- Consumes: nada.
- Produces:
  - `output.Envelope{OK bool, Data any, Summary string, Breadcrumbs []Breadcrumb, Error *Error}`
  - `output.Breadcrumb{Action, Cmd string}`
  - `output.Error{Code, Message, Hint string}`
  - `output.Code` con `CodeGeneric`, `CodeUsage`, `CodeUnauthorized`, `CodeNotFound`, `CodeRateLimited`
  - `output.CLIError` con `Error() string`, `ExitCode() int`, `Envelope() Envelope`
  - `output.NewError(code Code, message, hint string) *CLIError`
  - `output.ExitCodeFor(err error) int`

- [ ] **Step 1: Escribir los tests que fallan**

`internal/output/envelope_test.go`:

```go
package output

import (
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
```

`internal/output/errors_test.go`:

```go
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
```

- [ ] **Step 2: Ejecutar los tests y verificar que fallan**

Run: `go test ./internal/output/ -v`
Expected: FAIL — `undefined: Envelope`, `undefined: NewError`

- [ ] **Step 3: Implementar el envelope**

`internal/output/envelope.go`:

```go
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
```

- [ ] **Step 4: Implementar los errores y los códigos de salida**

`internal/output/errors.go`:

```go
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
```

- [ ] **Step 5: Ejecutar los tests y verificar que pasan**

Run: `go test ./internal/output/ -v`
Expected: PASS (5 tests)

- [ ] **Step 6: Conectar los códigos de salida en main**

Reemplaza el cuerpo de `main()` en `cmd/calliope/main.go`:

```go
func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(output.ExitCodeFor(err))
	}
}
```

Añade `"github.com/calliope/calliope-cli/internal/output"` a los imports.

- [ ] **Step 7: Verificar y hacer commit**

Run: `go test ./... -race && make build`
Expected: PASS y binario compilado

```bash
git add internal/output cmd/calliope/main.go
git commit -m "feat: envelope de salida, errores tipados y códigos de salida"
```

---

### Task 3: Presenter — los seis modos de salida

**Files:**
- Create: `internal/presenter/presenter.go`, `internal/presenter/jq.go`, `internal/presenter/table.go`
- Test: `internal/presenter/presenter_test.go`, `internal/presenter/jq_test.go`

**Interfaces:**
- Consumes: `output.Envelope`, `output.CLIError` (Task 2).
- Produces:
  - `presenter.Mode` con `ModeAuto`, `ModeJSON`, `ModeQuiet`, `ModeMarkdown`, `ModeJQ`
  - `presenter.Options{Mode Mode, JQExpr string, IsTTY bool, Out io.Writer}`
  - `presenter.Result{Envelope output.Envelope, Text func(io.Writer) error, Markdown func(io.Writer) error}`
  - `presenter.Render(r Result, opts Options) error`
  - `presenter.Table(w io.Writer, headers []string, rows [][]string) error`

`Text` y `Markdown` pueden ser `nil`: el presenter cae a JSON cuando faltan. Así un comando nuevo funciona antes de tener render humano.

- [ ] **Step 1: Escribir los tests que fallan**

`internal/presenter/presenter_test.go`:

```go
package presenter

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/calliope/calliope-cli/internal/output"
)

func resultadoDePrueba() Result {
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
	err := Render(resultadoDePrueba(), Options{Mode: ModeAuto, IsTTY: true, Out: &out})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(out.String(), "TEXTO HUMANO") {
		t.Errorf("se esperaba el render humano, se obtuvo: %q", out.String())
	}
}

func TestAutoEnTuberiaUsaJSON(t *testing.T) {
	var out bytes.Buffer
	err := Render(resultadoDePrueba(), Options{Mode: ModeAuto, IsTTY: false, Out: &out})
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
	err := Render(resultadoDePrueba(), Options{Mode: ModeQuiet, Out: &out})
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
	if err := Render(resultadoDePrueba(), Options{Mode: ModeMarkdown, Out: &out}); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.HasPrefix(out.String(), "# Markdown") {
		t.Errorf("se esperaba el render markdown, se obtuvo: %q", out.String())
	}
}

func TestSinRenderHumanoCaeAJSON(t *testing.T) {
	r := resultadoDePrueba()
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
```

`internal/presenter/jq_test.go`:

```go
package presenter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/calliope/calliope-cli/internal/output"
)

func TestJQExtraeUnCampo(t *testing.T) {
	r := Result{Envelope: output.OKEnvelope(
		[]map[string]any{{"id": "doc-1"}, {"id": "doc-2"}}, "2 documentos")}

	var out bytes.Buffer
	err := Render(r, Options{Mode: ModeJQ, JQExpr: ".data[].id", Out: &out})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	lineas := strings.Fields(out.String())
	if len(lineas) != 2 || lineas[0] != `"doc-1"` || lineas[1] != `"doc-2"` {
		t.Errorf("salida jq inesperada: %q", out.String())
	}
}

func TestJQInvalidoDevuelveErrorDeUso(t *testing.T) {
	r := Result{Envelope: output.OKEnvelope(nil, "")}

	var out bytes.Buffer
	err := Render(r, Options{Mode: ModeJQ, JQExpr: ".data[", Out: &out})
	if err == nil {
		t.Fatal("se esperaba un error por expresión jq inválida")
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2 (uso incorrecto)", got)
	}
}
```

- [ ] **Step 2: Ejecutar los tests y verificar que fallan**

Run: `go test ./internal/presenter/ -v`
Expected: FAIL — `undefined: Render`

- [ ] **Step 3: Instalar gojq e implementar el presenter**

```bash
go get github.com/itchyny/gojq@latest
```

`internal/presenter/presenter.go`:

```go
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
		return escribirJSON(w, r.Envelope.Data)
	case ModeMarkdown:
		if r.Markdown != nil {
			return r.Markdown(w)
		}
		return escribirJSON(w, r.Envelope)
	case ModeJSON:
		return escribirJSON(w, r.Envelope)
	default: // ModeAuto
		if opts.IsTTY && r.Text != nil {
			return r.Text(w)
		}
		return escribirJSON(w, r.Envelope)
	}
}

func escribirJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
```

`internal/presenter/jq.go`:

```go
package presenter

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/itchyny/gojq"

	"github.com/calliope/calliope-cli/internal/output"
)

// renderJQ aplica una expresión jq al envelope. El filtro va embebido en el
// binario a propósito: el SKILL.md prohíbe el pipe a un jq externo, que falla
// en las máquinas donde no está instalado.
func renderJQ(w io.Writer, env output.Envelope, expr string) error {
	query, err := gojq.Parse(expr)
	if err != nil {
		return output.NewError(output.CodeUsage,
			fmt.Sprintf("Expresión jq inválida: %v", err),
			"Consulta la sintaxis en https://jqlang.github.io/jq/manual/")
	}

	// gojq opera sobre tipos genéricos, así que damos la vuelta por JSON.
	crudo, err := json.Marshal(env)
	if err != nil {
		return err
	}
	var entrada any
	if err := json.Unmarshal(crudo, &entrada); err != nil {
		return err
	}

	iter := query.Run(entrada)
	for {
		v, ok := iter.Next()
		if !ok {
			return nil
		}
		if err, ok := v.(error); ok {
			return output.NewError(output.CodeUsage,
				fmt.Sprintf("Error al evaluar la expresión jq: %v", err), "")
		}
		if err := escribirJSON(w, v); err != nil {
			return err
		}
	}
}
```

- [ ] **Step 4: Implementar el render de tabla para TTY**

`internal/presenter/table.go`:

```go
package presenter

import (
	"io"
	"text/tabwriter"
)

// Table escribe una tabla alineada. Los comandos la usan desde su Result.Text.
func Table(w io.Writer, headers []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	escribirFila(tw, headers)
	for _, r := range rows {
		escribirFila(tw, r)
	}
	return tw.Flush()
}

func escribirFila(w io.Writer, celdas []string) {
	for i, c := range celdas {
		if i > 0 {
			io.WriteString(w, "\t")
		}
		io.WriteString(w, c)
	}
	io.WriteString(w, "\n")
}
```

- [ ] **Step 5: Ejecutar los tests y verificar que pasan**

Run: `go test ./internal/presenter/ -v`
Expected: PASS (7 tests)

- [ ] **Step 6: Commit**

```bash
git add internal/presenter go.mod go.sum
git commit -m "feat: presenter con los seis modos de salida y jq embebido"
```

---

### Task 4: Configuración en capas con procedencia

**Files:**
- Create: `internal/config/config.go`, `internal/config/layers.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Consumes: nada.
- Produces:
  - `config.Source` con `SourceFlag`, `SourceEnv`, `SourceProject`, `SourceRepo`, `SourceGlobal`, `SourceDefault`
  - `config.Value{Value string, Source Source, Path string}`
  - `config.Layer{Source Source, Path string, Values map[string]string}`
  - `config.Config` con `Get(key string) Value`, `All() map[string]Value`, `BaseURL() string`, `Org() string`, `Output() string`
  - `config.Resolve(layers []Layer) *Config`
  - `config.Load(cwd string, env func(string) string, flags map[string]string) (*Config, []string, error)` — devuelve la config, los avisos de la frontera de confianza y el error
  - Claves reconocidas: `org`, `base_url`, `output`, `timeout`

- [ ] **Step 1: Escribir los tests que fallan**

`internal/config/config_test.go`:

```go
package config

import "testing"

func TestPrecedenciaDeCapas(t *testing.T) {
	cfg := Resolve([]Layer{
		{Source: SourceDefault, Values: map[string]string{"org": "por-defecto", "base_url": "https://data-0.calliope.so"}},
		{Source: SourceGlobal, Path: "/home/u/.config/calliope/config.json", Values: map[string]string{"org": "global"}},
		{Source: SourceRepo, Path: "/repo/.calliope/config.json", Values: map[string]string{"org": "repo"}},
		{Source: SourceProject, Path: "/repo/sub/.calliope/config.json", Values: map[string]string{"org": "proyecto"}},
		{Source: SourceEnv, Values: map[string]string{"org": "entorno"}},
		{Source: SourceFlag, Values: map[string]string{"org": "flag"}},
	})

	if got := cfg.Get("org").Value; got != "flag" {
		t.Errorf("org = %q, el flag debe ganar a todo", got)
	}
	if got := cfg.Get("org").Source; got != SourceFlag {
		t.Errorf("origen = %q, se esperaba flag", got)
	}
	if got := cfg.Get("base_url").Value; got != "https://data-0.calliope.so" {
		t.Errorf("base_url = %q, debe caer al valor por defecto", got)
	}
}

func TestValorRecuerdaSuFicheroDeOrigen(t *testing.T) {
	cfg := Resolve([]Layer{
		{Source: SourceGlobal, Path: "/home/u/.config/calliope/config.json", Values: map[string]string{"org": "acme"}},
	})

	v := cfg.Get("org")
	if v.Path != "/home/u/.config/calliope/config.json" {
		t.Errorf("Path = %q, se esperaba la ruta del fichero global", v.Path)
	}
}

func TestClaveAusenteDevuelveValorVacio(t *testing.T) {
	cfg := Resolve(nil)
	v := cfg.Get("org")
	if v.Value != "" || v.Source != SourceDefault {
		t.Errorf("clave ausente = %+v, se esperaba vacía con origen default", v)
	}
}

func TestLasCapasVaciasNoPisanALasDeMenorPrioridad(t *testing.T) {
	cfg := Resolve([]Layer{
		{Source: SourceGlobal, Values: map[string]string{"org": "acme"}},
		{Source: SourceFlag, Values: map[string]string{"org": ""}}, // flag no informado
	})

	if got := cfg.Get("org").Value; got != "acme" {
		t.Errorf("org = %q, un flag vacío no debe pisar la capa global", got)
	}
}
```

- [ ] **Step 2: Ejecutar los tests y verificar que fallan**

Run: `go test ./internal/config/ -v`
Expected: FAIL — `undefined: Resolve`

- [ ] **Step 3: Implementar la resolución con procedencia**

`internal/config/config.go`:

```go
// Package config resuelve la configuración de calliope a partir de capas
// ordenadas, recordando de dónde salió cada valor. Cuando alguien se pregunta
// por qué el CLI apunta a una organización, la respuesta está en un comando.
package config

// Source identifica de qué capa proviene un valor.
type Source string

const (
	SourceFlag    Source = "flag"
	SourceEnv     Source = "env"
	SourceProject Source = "project"
	SourceRepo    Source = "repo"
	SourceGlobal  Source = "global"
	SourceDefault Source = "default"
)

// prioridad ordena las capas de mayor a menor precedencia.
var prioridad = map[Source]int{
	SourceFlag:    6,
	SourceEnv:     5,
	SourceProject: 4,
	SourceRepo:    3,
	SourceGlobal:  2,
	SourceDefault: 1,
}

// Claves reconocidas por la configuración.
const (
	KeyOrg     = "org"
	KeyBaseURL = "base_url"
	KeyOutput  = "output"
	KeyTimeout = "timeout"
)

// Value es un valor resuelto junto con su procedencia.
type Value struct {
	Value  string `json:"value"`
	Source Source `json:"source"`
	Path   string `json:"path,omitempty"`
}

// Layer es una fuente de valores sin resolver.
type Layer struct {
	Source Source
	Path   string
	Values map[string]string
}

// Config es el resultado de fusionar las capas.
type Config struct {
	values map[string]Value
}

// Resolve fusiona las capas respetando la precedencia. Los valores vacíos no
// cuentan: un flag no informado no debe pisar a la capa de debajo.
func Resolve(layers []Layer) *Config {
	c := &Config{values: map[string]Value{}}
	for _, l := range layers {
		for k, v := range l.Values {
			if v == "" {
				continue
			}
			actual, existe := c.values[k]
			if existe && prioridad[actual.Source] >= prioridad[l.Source] {
				continue
			}
			c.values[k] = Value{Value: v, Source: l.Source, Path: l.Path}
		}
	}
	return c
}

// Get devuelve el valor de una clave; si no existe, uno vacío de origen default.
func (c *Config) Get(key string) Value {
	if v, ok := c.values[key]; ok {
		return v
	}
	return Value{Source: SourceDefault}
}

// All devuelve todos los valores resueltos, para `calliope config list`.
func (c *Config) All() map[string]Value {
	copia := make(map[string]Value, len(c.values))
	for k, v := range c.values {
		copia[k] = v
	}
	return copia
}

func (c *Config) BaseURL() string { return c.Get(KeyBaseURL).Value }
func (c *Config) Org() string     { return c.Get(KeyOrg).Value }
func (c *Config) Output() string  { return c.Get(KeyOutput).Value }
```

- [ ] **Step 4: Implementar la carga de las seis capas**

`internal/config/layers.go`:

```go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// DefaultBaseURL es el backend de Calliope Data.
const DefaultBaseURL = "https://data-0.calliope.so"

// FileName es el fichero de configuración dentro de un directorio .calliope.
const FileName = "config.json"

// Load construye la configuración completa para una invocación. Devuelve
// además los avisos de la frontera de confianza (ver trust.go).
func Load(cwd string, env func(string) string, flags map[string]string) (*Config, []string, error) {
	var avisos []string

	capas := []Layer{
		{Source: SourceDefault, Values: map[string]string{
			KeyBaseURL: DefaultBaseURL,
			KeyTimeout: "60s",
		}},
	}

	if global, err := leerFichero(SourceGlobal, filepath.Join(globalDir(env), FileName)); err != nil {
		return nil, nil, err
	} else if global != nil {
		capas = append(capas, *global)
	}

	// Raíz del repositorio y directorio actual, en ese orden de precedencia.
	raiz := raizDeRepositorio(cwd)
	if raiz != "" && raiz != cwd {
		if l, err := leerFichero(SourceRepo, filepath.Join(raiz, ".calliope", FileName)); err != nil {
			return nil, nil, err
		} else if l != nil {
			saneada, w := sanear(*l)
			avisos = append(avisos, w...)
			capas = append(capas, saneada)
		}
	}
	if l, err := leerFichero(SourceProject, filepath.Join(cwd, ".calliope", FileName)); err != nil {
		return nil, nil, err
	} else if l != nil {
		saneada, w := sanear(*l)
		avisos = append(avisos, w...)
		capas = append(capas, saneada)
	}

	capas = append(capas, Layer{Source: SourceEnv, Values: map[string]string{
		KeyOrg:     env("CALLIOPE_ORG"),
		KeyBaseURL: env("CALLIOPE_BASE_URL"),
		KeyOutput:  env("CALLIOPE_OUTPUT"),
		KeyTimeout: env("CALLIOPE_TIMEOUT"),
	}})

	capas = append(capas, Layer{Source: SourceFlag, Values: flags})

	return Resolve(capas), avisos, nil
}

// GlobalPath es la ruta del fichero de configuración global.
func GlobalPath(env func(string) string) string {
	return filepath.Join(globalDir(env), FileName)
}

func globalDir(env func(string) string) string {
	if x := env("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "calliope")
	}
	return filepath.Join(env("HOME"), ".config", "calliope")
}

func leerFichero(src Source, ruta string) (*Layer, error) {
	b, err := os.ReadFile(ruta)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var vals map[string]string
	if err := json.Unmarshal(b, &vals); err != nil {
		return nil, err
	}
	return &Layer{Source: src, Path: ruta, Values: vals}, nil
}

// raizDeRepositorio sube desde cwd buscando un directorio .git.
func raizDeRepositorio(cwd string) string {
	dir := cwd
	for {
		if fi, err := os.Stat(filepath.Join(dir, ".git")); err == nil && fi.IsDir() {
			return dir
		}
		padre := filepath.Dir(dir)
		if padre == dir {
			return ""
		}
		dir = padre
	}
}
```

- [ ] **Step 5: Ejecutar los tests y verificar que pasan**

`sanear` todavía no existe; la define la Task 5. Para que este paso compile, crea `internal/config/trust.go` con la versión provisional:

```go
package config

// sanear se implementa en la Task 5.
func sanear(l Layer) (Layer, []string) { return l, nil }
```

Run: `go test ./internal/config/ -v`
Expected: PASS (4 tests)

- [ ] **Step 6: Commit**

```bash
git add internal/config
git commit -m "feat: configuración en capas con procedencia"
```

---

### Task 5: Frontera de confianza de la configuración de proyecto

**Files:**
- Modify: `internal/config/trust.go` (sustituye por completo el provisional de la Task 4)
- Test: `internal/config/trust_test.go`

**Interfaces:**
- Consumes: `config.Layer`, `config.Source` (Task 4).
- Produces: `sanear(l Layer) (Layer, []string)`; `config.ProjectAllowed` (conjunto de claves permitidas en capas de proyecto).

**Por qué existe esta tarea.** Un `.calliope/config.json` llega dentro de cualquier repositorio clonado. Si pudiera fijar `base_url`, clonar un repositorio hostil y ejecutar `calliope ask` enviaría el token del usuario a una máquina ajena. Este es el test que más importa del plan.

- [ ] **Step 1: Escribir el test que falla**

`internal/config/trust_test.go`:

```go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositorioHostilNoPuedeCambiarBaseURL(t *testing.T) {
	dir := t.TempDir()
	escribirConfigDeProyecto(t, dir, map[string]string{
		"org":      "acme",
		"base_url": "https://atacante.example.com",
		"timeout":  "1ms",
	})

	env := entornoDePrueba(t.TempDir())
	cfg, avisos, err := Load(dir, env, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got := cfg.BaseURL(); got != DefaultBaseURL {
		t.Fatalf("base_url = %q — un repositorio hostil ha conseguido redirigir el CLI", got)
	}
	if got := cfg.Get(KeyTimeout).Value; got == "1ms" {
		t.Errorf("timeout = %q, la configuración de proyecto no puede fijarlo", got)
	}
	if got := cfg.Org(); got != "acme" {
		t.Errorf("org = %q, la configuración de proyecto sí puede fijar la organización", got)
	}
	if len(avisos) == 0 {
		t.Error("se esperaba un aviso visible sobre los campos ignorados")
	}
	if !strings.Contains(strings.Join(avisos, " "), "base_url") {
		t.Errorf("el aviso debe nombrar el campo ignorado: %v", avisos)
	}
}

func TestLaConfiguracionGlobalSiPuedeFijarBaseURL(t *testing.T) {
	home := t.TempDir()
	dirGlobal := filepath.Join(home, ".config", "calliope")
	if err := os.MkdirAll(dirGlobal, 0o755); err != nil {
		t.Fatal(err)
	}
	escribirJSON(t, filepath.Join(dirGlobal, FileName), map[string]string{
		"base_url": "https://staging.calliope.so",
	})

	cfg, _, err := Load(t.TempDir(), entornoDePrueba(home), nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.BaseURL(); got != "https://staging.calliope.so" {
		t.Errorf("base_url = %q, la capa global sí debe poder fijarlo", got)
	}
}

func TestLaConfiguracionDeProyectoNuncaAportaCredenciales(t *testing.T) {
	dir := t.TempDir()
	escribirConfigDeProyecto(t, dir, map[string]string{"api_key": "cal_live_robada"})

	cfg, avisos, err := Load(dir, entornoDePrueba(t.TempDir()), nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Get("api_key").Value; got != "" {
		t.Fatalf("api_key = %q — la configuración de proyecto no puede aportar credenciales", got)
	}
	if len(avisos) == 0 {
		t.Error("se esperaba un aviso sobre el campo ignorado")
	}
}

// --- ayudantes ---

func escribirConfigDeProyecto(t *testing.T, dir string, vals map[string]string) {
	t.Helper()
	d := filepath.Join(dir, ".calliope")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	escribirJSON(t, filepath.Join(d, FileName), vals)
}

func escribirJSON(t *testing.T, ruta string, vals map[string]string) {
	t.Helper()
	b, err := json.Marshal(vals)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// entornoDePrueba aísla HOME para que los tests no lean la config real.
func entornoDePrueba(home string) func(string) string {
	return func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}
}
```

- [ ] **Step 2: Ejecutar los tests y verificar que fallan**

Run: `go test ./internal/config/ -run Test -v`
Expected: FAIL — `base_url = "https://atacante.example.com"`, porque `sanear` todavía es la identidad

- [ ] **Step 3: Implementar la frontera de confianza**

Sustituye por completo `internal/config/trust.go`:

```go
package config

import "fmt"

// ProjectAllowed son las únicas claves que una capa de proyecto puede fijar.
//
// La regla existe porque un .calliope/config.json viaja dentro de cualquier
// repositorio clonado. Si pudiera fijar base_url, clonar un repositorio hostil
// y ejecutar `calliope ask` enviaría el token del usuario a una máquina ajena.
// Los timeouts quedan fuera porque un timeout absurdo convierte cualquier
// comando en un fallo silencioso.
var ProjectAllowed = map[string]bool{
	KeyOrg:    true,
	KeyOutput: true,
}

// sanear elimina de una capa de proyecto todo campo no permitido y devuelve
// un aviso por cada uno, para que el usuario vea qué se ha ignorado.
func sanear(l Layer) (Layer, []string) {
	if l.Source != SourceProject && l.Source != SourceRepo {
		return l, nil
	}

	limpia := Layer{Source: l.Source, Path: l.Path, Values: map[string]string{}}
	var avisos []string
	for k, v := range l.Values {
		if ProjectAllowed[k] {
			limpia.Values[k] = v
			continue
		}
		avisos = append(avisos, fmt.Sprintf(
			"aviso: se ignora %q de %s; la configuración de proyecto solo puede fijar: org, output",
			k, l.Path))
	}
	return limpia, avisos
}
```

- [ ] **Step 4: Ejecutar los tests y verificar que pasan**

Run: `go test ./internal/config/ -v`
Expected: PASS (7 tests)

- [ ] **Step 5: Emitir los avisos por stderr**

Los avisos que devuelve `Load` se imprimen en `appctx.Build` (Task 11). Deja constancia con un comentario en `Load`:

```go
// Los avisos se devuelven en vez de imprimirse aquí: quien decide dónde
// escribirlos es appctx, que conoce el stderr del comando.
```

- [ ] **Step 6: Commit**

```bash
git add internal/config
git commit -m "feat: frontera de confianza de la configuración de proyecto"
```

---

### Task 6: Spike — ¿acepta el backend el token OAuth de PropelAuth?

**Files:**
- Create: `spike/oauth/main.go` (código desechable), `docs/superpowers/spikes/2026-08-27-oauth-propelauth.md`

**Interfaces:**
- Consumes: nada.
- Produces: una decisión registrada que abre o cierra la Task 8. Ningún código de producción.

**Riesgo que resuelve (R1 del spec).** El README de `calliope-data-mcp` documenta que el token que PropelAuth emite por su flujo OAuth es opaco, y que `data-0` tendría que aceptarlo por introspección. La UI funciona porque usa el SDK de navegador y obtiene un JWT, que no es necesariamente lo mismo. Hasta que esto se compruebe contra el backend real, no se sabe si OAuth entra en la v1.

Esta tarea es un spike: **su salida es una respuesta, no código que se conserve.**

- [ ] **Step 1: Reunir la configuración de PropelAuth**

```bash
grep -i propelauth /Users/j10/repositories/calliope/calliope-data-ui/.env
```

Anota `authUrl` y el `clientId` del entorno de test. Si el fichero no los tiene, están en el dashboard de PropelAuth.

- [ ] **Step 2: Registrar el redirect de loopback**

En el dashboard de PropelAuth, entorno **Test**, añade `http://127.0.0.1:8976/callback` a los redirect URIs permitidos. Es configuración, no cambio de código.

Este paso lo tiene que hacer una persona con acceso al dashboard. Si no lo tienes, para aquí y pídelo antes de continuar.

- [ ] **Step 3: Escribir el probe desechable**

`spike/oauth/main.go` — código de usar y tirar, sin tests:

```go
// Command spike-oauth comprueba si data-0.calliope.so acepta un token OAuth
// de PropelAuth. CÓDIGO DESECHABLE: se borra al cerrar la Task 6.
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

func main() {
	authURL := os.Getenv("PROPELAUTH_AUTH_URL")
	clientID := os.Getenv("PROPELAUTH_CLIENT_ID")
	baseURL := "https://data-0.calliope.so"
	if authURL == "" || clientID == "" {
		fmt.Fprintln(os.Stderr, "faltan PROPELAUTH_AUTH_URL y PROPELAUTH_CLIENT_ID")
		os.Exit(1)
	}

	verifier := aleatorio(64)
	suma := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(suma[:])
	redirect := "http://127.0.0.1:8976/callback"

	autorizar := fmt.Sprintf(
		"%s/oauth/2.1/authorize?response_type=code&client_id=%s&redirect_uri=%s&code_challenge=%s&code_challenge_method=S256&scope=openid+profile+email",
		strings.TrimRight(authURL, "/"), url.QueryEscape(clientID), url.QueryEscape(redirect), challenge)

	codigos := make(chan string, 1)
	srv := &http.Server{Addr: "127.0.0.1:8976"}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Puedes cerrar esta pestaña.")
		codigos <- r.URL.Query().Get("code")
	})
	go srv.ListenAndServe()

	fmt.Println("Abriendo el navegador…")
	exec.Command("open", autorizar).Run()
	codigo := <-codigos

	// Canje del código por el token.
	resp, err := http.PostForm(strings.TrimRight(authURL, "/")+"/oauth/2.1/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {codigo},
		"redirect_uri":  {redirect},
		"client_id":     {clientID},
		"code_verifier": {verifier},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "canje del código:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	cuerpo, _ := io.ReadAll(resp.Body)
	json.Unmarshal(cuerpo, &tok)
	fmt.Printf("token: status=%d, longitud=%d, ¿parece JWT?=%v\n",
		resp.StatusCode, len(tok.AccessToken), strings.Count(tok.AccessToken, ".") == 2)

	// LA PREGUNTA DEL SPIKE: ¿lo acepta el backend?
	req, _ := http.NewRequest("GET", baseURL+"/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	me, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "GET /v1/auth/me:", err)
		os.Exit(1)
	}
	defer me.Body.Close()
	cuerpoMe, _ := io.ReadAll(me.Body)
	fmt.Printf("\nRESULTADO: GET /v1/auth/me → %d\n%s\n", me.StatusCode, cuerpoMe)
}

func aleatorio(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
```

- [ ] **Step 4: Ejecutar el probe**

```bash
PROPELAUTH_AUTH_URL=... PROPELAUTH_CLIENT_ID=... go run ./spike/oauth
```

Expected: una de dos respuestas claras, `200` con el perfil del usuario, o `401`.

- [ ] **Step 5: Registrar la decisión**

Crea `docs/superpowers/spikes/2026-08-27-oauth-propelauth.md` con: el status obtenido, si el token era JWT u opaco, y la decisión.

- **Si `200`:** la Task 8 se ejecuta. OAuth entra en la v1.
- **Si `401`:** la Task 8 **se salta**. La v1 sale solo con API key, que cubre todo el alcance pedido. Anota en el fichero el requisito de backend (aceptar el token por introspección) y actualiza la sección 7 del spec para reflejar que OAuth pasa a v2.

En ambos casos, la interfaz `auth.Source` de la Task 7 no cambia — por eso el aplazamiento no cuesta reescribir ningún comando.

- [ ] **Step 6: Borrar el código desechable y hacer commit**

```bash
rm -rf spike/
git add docs/superpowers/spikes
git commit -m "docs: spike de OAuth con PropelAuth y su decisión"
```

---

### Task 7: Credenciales — interfaz, API key y almacenamiento

**Files:**
- Create: `internal/auth/credential.go`, `internal/auth/store.go`, `internal/auth/resolve.go`
- Test: `internal/auth/credential_test.go`, `internal/auth/store_test.go`, `internal/auth/resolve_test.go`

**Interfaces:**
- Consumes: `output.CLIError`, `output.CodeUnauthorized` (Task 2).
- Produces:
  - `auth.Kind` con `KindAPIKey`, `KindOAuth`
  - `auth.Credential{Kind Kind, Token string, Org string}` con `Header() (string, string)` y `Valid() bool`
  - `auth.Store` con `Save(Credential) error`, `Load() (*Credential, error)`, `Delete() error`
  - `auth.NewFileStore(path string) Store`, `auth.NewKeyringStore(fallback Store) Store`
  - `auth.DefaultStore(globalDir string) Store`
  - `auth.Resolve(env func(string) string, st Store) (Credential, string, error)` — credencial y descripción de su origen

- [ ] **Step 1: Escribir los tests que fallan**

`internal/auth/credential_test.go`:

```go
package auth

import "testing"

func TestCabeceraDeAPIKey(t *testing.T) {
	c := Credential{Kind: KindAPIKey, Token: "cal_live_123"}
	k, v := c.Header()
	if k != "X-API-Key" || v != "cal_live_123" {
		t.Errorf("Header() = (%q, %q), se esperaba (X-API-Key, cal_live_123)", k, v)
	}
}

func TestCabeceraDeOAuth(t *testing.T) {
	c := Credential{Kind: KindOAuth, Token: "tok"}
	k, v := c.Header()
	if k != "Authorization" || v != "Bearer tok" {
		t.Errorf("Header() = (%q, %q), se esperaba (Authorization, Bearer tok)", k, v)
	}
}

func TestCredencialSinTokenNoEsValida(t *testing.T) {
	if (Credential{Kind: KindAPIKey}).Valid() {
		t.Error("una credencial sin token no debe ser válida")
	}
}
```

`internal/auth/store_test.go`:

```go
package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreGuardaYRecupera(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "credentials.json")
	st := NewFileStore(ruta)

	quiero := Credential{Kind: KindAPIKey, Token: "cal_live_abc", Org: "acme"}
	if err := st.Save(quiero); err != nil {
		t.Fatalf("Save: %v", err)
	}

	tengo, err := st.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if tengo == nil || *tengo != quiero {
		t.Errorf("Load() = %+v, se esperaba %+v", tengo, quiero)
	}
}

func TestFileStoreEscribeCon0600(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "credentials.json")
	st := NewFileStore(ruta)
	if err := st.Save(Credential{Kind: KindAPIKey, Token: "secreto"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	fi, err := os.Stat(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("permisos = %o, se esperaba 600 — la credencial quedaría legible por otros", perm)
	}
}

func TestFileStoreSinFicheroDevuelveNil(t *testing.T) {
	st := NewFileStore(filepath.Join(t.TempDir(), "no-existe.json"))
	c, err := st.Load()
	if err != nil {
		t.Fatalf("Load sin fichero no debe fallar: %v", err)
	}
	if c != nil {
		t.Errorf("Load() = %+v, se esperaba nil", c)
	}
}

func TestFileStoreDelete(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "credentials.json")
	st := NewFileStore(ruta)
	if err := st.Save(Credential{Kind: KindAPIKey, Token: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := st.Delete(); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if c, _ := st.Load(); c != nil {
		t.Error("tras Delete, Load debe devolver nil")
	}
}
```

`internal/auth/resolve_test.go`:

```go
package auth

import (
	"path/filepath"
	"testing"

	"github.com/calliope/calliope-cli/internal/output"
)

func TestElEntornoGanaAlAlmacen(t *testing.T) {
	st := NewFileStore(filepath.Join(t.TempDir(), "c.json"))
	if err := st.Save(Credential{Kind: KindAPIKey, Token: "del-almacen"}); err != nil {
		t.Fatal(err)
	}

	env := func(k string) string {
		if k == "CALLIOPE_API_KEY" {
			return "del-entorno"
		}
		return ""
	}

	c, origen, err := Resolve(env, st)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if c.Token != "del-entorno" {
		t.Errorf("token = %q, la variable de entorno debe ganar", c.Token)
	}
	if origen == "" {
		t.Error("se esperaba una descripción del origen")
	}
}

func TestSinCredencialDevuelveErrorConHint(t *testing.T) {
	st := NewFileStore(filepath.Join(t.TempDir(), "c.json"))
	vacio := func(string) string { return "" }

	_, _, err := Resolve(vacio, st)
	if err == nil {
		t.Fatal("se esperaba un error al no haber credencial")
	}
	if got := output.ExitCodeFor(err); got != 3 {
		t.Errorf("código de salida = %d, se esperaba 3 (no autorizado)", got)
	}

	var cliErr *output.CLIError
	if !asCLIError(err, &cliErr) {
		t.Fatal("se esperaba un *output.CLIError")
	}
	if cliErr.Hint == "" {
		t.Error("el error debe decir cómo autenticarse")
	}
}
```

Añade el ayudante al final de `resolve_test.go`:

```go
func asCLIError(err error, target **output.CLIError) bool {
	e, ok := err.(*output.CLIError)
	if ok {
		*target = e
	}
	return ok
}
```

- [ ] **Step 2: Ejecutar los tests y verificar que fallan**

Run: `go test ./internal/auth/ -v`
Expected: FAIL — `undefined: Credential`

- [ ] **Step 3: Implementar la credencial**

`internal/auth/credential.go`:

```go
// Package auth resuelve y almacena la credencial con la que el CLI llama a
// Calliope Data. Las dos formas viven tras la misma interfaz, para que añadir
// una tercera no toque el código de los comandos.
package auth

// Kind distingue las formas de credencial soportadas.
type Kind string

const (
	KindAPIKey Kind = "api_key"
	KindOAuth  Kind = "oauth"
)

// Credential es la credencial resuelta para una invocación. Org es opcional:
// solo lo rellenan las claves limitadas a una organización.
type Credential struct {
	Kind  Kind   `json:"kind"`
	Token string `json:"token"`
	Org   string `json:"org,omitempty"`
}

// Header devuelve la cabecera HTTP que corresponde a esta credencial.
func (c Credential) Header() (string, string) {
	if c.Kind == KindOAuth {
		return "Authorization", "Bearer " + c.Token
	}
	return "X-API-Key", c.Token
}

// Valid indica si la credencial se puede usar.
func (c Credential) Valid() bool { return c.Token != "" }
```

- [ ] **Step 4: Implementar el almacenamiento**

`internal/auth/store.go`:

```go
package auth

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	servicioKeyring = "calliope-cli"
	usuarioKeyring  = "default"
)

// Store guarda la credencial entre invocaciones.
type Store interface {
	Save(c Credential) error
	Load() (*Credential, error)
	Delete() error
}

// fileStore guarda la credencial en un fichero con permisos 0600. Es el
// respaldo para máquinas sin keyring (contenedores, servidores sin sesión).
type fileStore struct{ ruta string }

// NewFileStore crea un almacén respaldado por fichero.
func NewFileStore(ruta string) Store { return &fileStore{ruta: ruta} }

func (s *fileStore) Save(c Credential) error {
	if err := os.MkdirAll(filepath.Dir(s.ruta), 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(s.ruta, b, 0o600)
}

func (s *fileStore) Load() (*Credential, error) {
	b, err := os.ReadFile(s.ruta)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var c Credential
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *fileStore) Delete() error {
	err := os.Remove(s.ruta)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// keyringStore usa el llavero del sistema y cae al respaldo cuando no hay
// ninguno disponible.
type keyringStore struct{ respaldo Store }

// NewKeyringStore crea un almacén sobre el llavero del sistema.
func NewKeyringStore(respaldo Store) Store { return &keyringStore{respaldo: respaldo} }

func (s *keyringStore) Save(c Credential) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := keyring.Set(servicioKeyring, usuarioKeyring, string(b)); err != nil {
		return s.respaldo.Save(c)
	}
	return nil
}

func (s *keyringStore) Load() (*Credential, error) {
	crudo, err := keyring.Get(servicioKeyring, usuarioKeyring)
	if err != nil {
		return s.respaldo.Load()
	}
	var c Credential
	if err := json.Unmarshal([]byte(crudo), &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *keyringStore) Delete() error {
	// Se borra en ambos sitios: da igual dónde acabara al guardarse.
	_ = keyring.Delete(servicioKeyring, usuarioKeyring)
	return s.respaldo.Delete()
}

// DefaultStore es el almacén que usa el CLI: llavero del sistema con respaldo
// en el fichero global. Nunca escribe en configuración de proyecto.
func DefaultStore(dirGlobal string) Store {
	return NewKeyringStore(NewFileStore(filepath.Join(dirGlobal, "credentials.json")))
}
```

```bash
go get github.com/zalando/go-keyring@latest
```

- [ ] **Step 5: Implementar la resolución de la credencial**

`internal/auth/resolve.go`:

```go
package auth

import "github.com/calliope/calliope-cli/internal/output"

// Resolve devuelve la credencial de esta invocación y una descripción legible
// de su origen, que `auth status` y `doctor` muestran tal cual.
//
// Precedencia: variables de entorno antes que el almacén, para que un entorno
// de CI pueda inyectar la credencial sin tocar el llavero del usuario.
func Resolve(env func(string) string, st Store) (Credential, string, error) {
	if k := env("CALLIOPE_API_KEY"); k != "" {
		return Credential{Kind: KindAPIKey, Token: k, Org: env("CALLIOPE_ORG")},
			"variable de entorno CALLIOPE_API_KEY", nil
	}
	if t := env("CALLIOPE_TOKEN"); t != "" {
		return Credential{Kind: KindOAuth, Token: t, Org: env("CALLIOPE_ORG")},
			"variable de entorno CALLIOPE_TOKEN", nil
	}

	c, err := st.Load()
	if err != nil {
		return Credential{}, "", err
	}
	if c != nil && c.Valid() {
		return *c, "almacén local de credenciales", nil
	}

	return Credential{}, "", output.NewError(output.CodeUnauthorized,
		"No hay ninguna credencial de Calliope configurada.",
		"Ejecuta: calliope auth login --api-key <clave>  (créala en el UI, en Observabilidad → Claves API)")
}
```

- [ ] **Step 6: Ejecutar los tests y verificar que pasan**

Run: `go test ./internal/auth/ -v`
Expected: PASS (9 tests)

- [ ] **Step 7: Commit**

```bash
git add internal/auth go.mod go.sum
git commit -m "feat: credenciales con almacén en llavero y respaldo en fichero"
```

---

### Task 8: OAuth con PropelAuth (CONDICIONAL — solo si la Task 6 dio 200)

**Si el spike de la Task 6 devolvió 401, salta esta tarea por completo** y pasa a la Task 9. La v1 sale solo con API key.

**Files:**
- Create: `internal/auth/oauth.go`
- Test: `internal/auth/oauth_test.go`

**Interfaces:**
- Consumes: `auth.Credential`, `auth.KindOAuth` (Task 7).
- Produces: `auth.PKCE{Verifier, Challenge string}`, `auth.NewPKCE() (PKCE, error)`, `auth.AuthorizeURL(authURL, clientID, redirect string, p PKCE) string`, `auth.LoginOAuth(ctx context.Context, authURL, clientID string) (Credential, error)`

- [ ] **Step 1: Escribir el test que falla**

`internal/auth/oauth_test.go`:

```go
package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
)

func TestPKCEChallengeEsElSHA256DelVerifier(t *testing.T) {
	p, err := NewPKCE()
	if err != nil {
		t.Fatalf("NewPKCE: %v", err)
	}
	suma := sha256.Sum256([]byte(p.Verifier))
	quiero := base64.RawURLEncoding.EncodeToString(suma[:])
	if p.Challenge != quiero {
		t.Errorf("Challenge = %q, se esperaba %q", p.Challenge, quiero)
	}
}

func TestPKCEGeneraVerifiersDistintos(t *testing.T) {
	a, _ := NewPKCE()
	b, _ := NewPKCE()
	if a.Verifier == b.Verifier {
		t.Error("dos verifiers consecutivos no pueden coincidir")
	}
}

func TestAuthorizeURLLlevaS256YElRedirect(t *testing.T) {
	p := PKCE{Verifier: "v", Challenge: "c"}
	crudo := AuthorizeURL("https://acme.propelauthtest.com/", "cli-id", "http://127.0.0.1:8976/callback", p)

	u, err := url.Parse(crudo)
	if err != nil {
		t.Fatalf("URL inválida: %v", err)
	}
	q := u.Query()
	if q.Get("code_challenge_method") != "S256" {
		t.Errorf("code_challenge_method = %q, se esperaba S256", q.Get("code_challenge_method"))
	}
	if q.Get("code_challenge") != "c" {
		t.Errorf("code_challenge = %q", q.Get("code_challenge"))
	}
	if q.Get("redirect_uri") != "http://127.0.0.1:8976/callback" {
		t.Errorf("redirect_uri = %q", q.Get("redirect_uri"))
	}
	if q.Get("response_type") != "code" {
		t.Errorf("response_type = %q, se esperaba code", q.Get("response_type"))
	}
	if strings.Contains(crudo, "//oauth") {
		t.Error("la barra final del authURL debe recortarse")
	}
}
```

- [ ] **Step 2: Ejecutar el test y verificar que falla**

Run: `go test ./internal/auth/ -run PKCE -v`
Expected: FAIL — `undefined: NewPKCE`

- [ ] **Step 3: Implementar PKCE y la URL de autorización**

`internal/auth/oauth.go`:

```go
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

// PKCE es el par verifier/challenge del flujo de código de autorización.
type PKCE struct {
	Verifier  string
	Challenge string
}

// NewPKCE genera un verifier aleatorio y su challenge S256.
func NewPKCE() (PKCE, error) {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return PKCE{}, err
	}
	verifier := base64.RawURLEncoding.EncodeToString(b)
	suma := sha256.Sum256([]byte(verifier))
	return PKCE{
		Verifier:  verifier,
		Challenge: base64.RawURLEncoding.EncodeToString(suma[:]),
	}, nil
}

// AuthorizeURL construye la URL a la que se envía el navegador.
func AuthorizeURL(authURL, clientID, redirect string, p PKCE) string {
	base := strings.TrimRight(authURL, "/") + "/oauth/2.1/authorize"
	q := url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirect},
		"code_challenge":        {p.Challenge},
		"code_challenge_method": {"S256"},
		"scope":                 {"openid profile email"},
	}
	return fmt.Sprintf("%s?%s", base, q.Encode())
}
```

- [ ] **Step 4: Ejecutar los tests y verificar que pasan**

Run: `go test ./internal/auth/ -v`
Expected: PASS (12 tests)

- [ ] **Step 5: Implementar el flujo de loopback**

Añade a `internal/auth/oauth.go`:

```go
// RedirectAddr es la dirección de loopback registrada en PropelAuth.
const RedirectAddr = "127.0.0.1:8976"

// RedirectURL es el redirect_uri completo.
const RedirectURL = "http://" + RedirectAddr + "/callback"

// LoginOAuth abre el navegador, recoge el código en el loopback y lo canjea
// por un token. El servidor se levanta con un mux propio, no con el
// DefaultServeMux, para que dos invocaciones no se pisen.
func LoginOAuth(ctx context.Context, authURL, clientID string) (Credential, error) {
	p, err := NewPKCE()
	if err != nil {
		return Credential{}, err
	}

	codigos := make(chan string, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, paginaDeCallback)
		codigos <- r.URL.Query().Get("code")
	})
	ln, err := net.Listen("tcp", RedirectAddr)
	if err != nil {
		return Credential{}, output.NewError(output.CodeGeneric,
			fmt.Sprintf("No se pudo abrir el puerto de retorno %s.", RedirectAddr),
			"Cierra el proceso que lo ocupe, o autentícate con: calliope auth login --api-key <clave>")
	}
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Shutdown(context.Background())

	abrirNavegador(AuthorizeURL(authURL, clientID, RedirectURL, p))

	var codigo string
	select {
	case codigo = <-codigos:
	case <-ctx.Done():
		return Credential{}, ctx.Err()
	}

	tok, err := canjear(ctx, authURL, clientID, codigo, p.Verifier)
	if err != nil {
		return Credential{}, err
	}
	return Credential{Kind: KindOAuth, Token: tok}, nil
}

const paginaDeCallback = `<!doctype html><meta charset="utf-8">
<title>Calliope</title>
<body style="font:16px system-ui;padding:3rem;text-align:center">
<h1>Sesión iniciada</h1><p>Ya puedes cerrar esta pestaña y volver al terminal.</p>`

func canjear(ctx context.Context, authURL, clientID, codigo, verifier string) (string, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {codigo},
		"redirect_uri":  {RedirectURL},
		"client_id":     {clientID},
		"code_verifier": {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(authURL, "/")+"/oauth/2.1/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", output.NewError(output.CodeUnauthorized,
			"PropelAuth rechazó el código de autorización.",
			"Vuelve a intentarlo con: calliope auth login")
	}

	var cuerpo struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cuerpo); err != nil {
		return "", err
	}
	return cuerpo.AccessToken, nil
}

func abrirNavegador(u string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler"}
	default:
		cmd = "xdg-open"
	}
	_ = exec.Command(cmd, append(args, u)...).Start()
}
```

Imports que hay que añadir al fichero: `context`, `encoding/json`, `io`, `net`, `net/http`, `os/exec`, `runtime`, y `github.com/calliope/calliope-cli/internal/output`.

- [ ] **Step 6: Verificar y hacer commit**

Run: `go test ./internal/auth/ -race && go vet ./...`
Expected: PASS

```bash
git add internal/auth
git commit -m "feat: login OAuth con PropelAuth por loopback y PKCE"
```

---

### Task 9: SDK — transporte, scoping de organización y mapeo de errores

**Files:**
- Create: `internal/sdk/client.go`
- Test: `internal/sdk/client_test.go`

**Interfaces:**
- Consumes: `auth.Credential` (Task 7), `output.CLIError` y sus códigos (Task 2).
- Produces:
  - `sdk.Options{BaseURL string, Credential auth.Credential, Timeout time.Duration, UserAgent string, HTTPClient *http.Client}`
  - `sdk.New(opts Options) *Client`
  - `(*Client).Do(ctx context.Context, method, path string, body, out any) error`
  - `(*Client).OrgPath(org, suffix string) string`

El SDK no conoce Cobra, ni flags, ni formatos de salida. Traduce status HTTP a `*output.CLIError` y nada más.

- [ ] **Step 1: Escribir los tests que fallan**

`internal/sdk/client_test.go`:

```go
package sdk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/output"
)

func clienteDePrueba(t *testing.T, h http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New(Options{
		BaseURL:    srv.URL,
		Credential: auth.Credential{Kind: auth.KindAPIKey, Token: "cal_live_test"},
		Timeout:    5 * time.Second,
	}), srv
}

func TestLaRutaLlevaElScopeDeOrganizacion(t *testing.T) {
	var visto string
	c, _ := clienteDePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		visto = r.URL.Path
		w.Write([]byte(`{}`))
	})

	var out map[string]any
	if err := c.Do(context.Background(), http.MethodGet, c.OrgPath("acme", "/rules"), nil, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if visto != "/v1/organizations/acme/rules" {
		t.Errorf("ruta = %q, se esperaba /v1/organizations/acme/rules", visto)
	}
}

func TestElNombreDeOrganizacionSeEscapa(t *testing.T) {
	c, _ := clienteDePrueba(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) })

	got := c.OrgPath("acme corp/../otra", "/rules")
	if strings.Contains(got, "..") || strings.Contains(got, " ") {
		t.Errorf("OrgPath = %q — el nombre de organización debe ir escapado", got)
	}
}

func TestSeEnviaLaCabeceraDeAutenticacion(t *testing.T) {
	var visto string
	c, _ := clienteDePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		visto = r.Header.Get("X-API-Key")
		w.Write([]byte(`{}`))
	})

	var out map[string]any
	if err := c.Do(context.Background(), http.MethodGet, "/v1/auth/me", nil, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if visto != "cal_live_test" {
		t.Errorf("X-API-Key = %q", visto)
	}
}

func TestMapeoDeStatusACodigosDeSalida(t *testing.T) {
	casos := []struct {
		status   int
		codigo   output.Code
		salida   int
	}{
		{http.StatusUnauthorized, output.CodeUnauthorized, 3},
		{http.StatusForbidden, output.CodeUnauthorized, 3},
		{http.StatusNotFound, output.CodeNotFound, 4},
		{http.StatusTooManyRequests, output.CodeRateLimited, 5},
		{http.StatusInternalServerError, output.CodeGeneric, 1},
	}

	for _, caso := range casos {
		c, _ := clienteDePrueba(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(caso.status)
			w.Write([]byte(`{"detail":"traza interna del backend"}`))
		})

		var out map[string]any
		err := c.Do(context.Background(), http.MethodGet, "/x", nil, &out)
		if err == nil {
			t.Fatalf("status %d: se esperaba error", caso.status)
		}
		if got := output.ExitCodeFor(err); got != caso.salida {
			t.Errorf("status %d: código de salida = %d, se esperaba %d", caso.status, got, caso.salida)
		}
	}
}

func TestElErrorNoFiltraElCuerpoDelBackend(t *testing.T) {
	c, _ := clienteDePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"detail":"panic en /srv/app/handlers/ask.py línea 412"}`))
	})

	var out map[string]any
	err := c.Do(context.Background(), http.MethodGet, "/x", nil, &out)
	if err == nil {
		t.Fatal("se esperaba error")
	}
	if strings.Contains(err.Error(), "ask.py") || strings.Contains(err.Error(), "panic") {
		t.Errorf("el mensaje filtra internals del backend: %q", err.Error())
	}
}

func TestElTimeoutProduceUnErrorAccionable(t *testing.T) {
	c, _ := clienteDePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte(`{}`))
	})
	c.http.Timeout = 20 * time.Millisecond

	var out map[string]any
	err := c.Do(context.Background(), http.MethodGet, "/x", nil, &out)
	if err == nil {
		t.Fatal("se esperaba un error de timeout")
	}
	var cliErr *output.CLIError
	if !errorComoCLI(err, &cliErr) {
		t.Fatalf("se esperaba *output.CLIError, se obtuvo %T", err)
	}
	if cliErr.Hint == "" {
		t.Error("el error de timeout debe sugerir qué hacer")
	}
}

func TestRespuestaVaciaNoRompe(t *testing.T) {
	c, _ := clienteDePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	var out map[string]any
	if err := c.Do(context.Background(), http.MethodGet, "/x", nil, &out); err != nil {
		t.Errorf("una respuesta vacía no debe fallar: %v", err)
	}
}

func errorComoCLI(err error, target **output.CLIError) bool {
	e, ok := err.(*output.CLIError)
	if ok {
		*target = e
	}
	return ok
}
```

- [ ] **Step 2: Ejecutar los tests y verificar que fallan**

Run: `go test ./internal/sdk/ -v`
Expected: FAIL — `undefined: New`

- [ ] **Step 3: Implementar el transporte**

`internal/sdk/client.go`:

```go
// Package sdk es el cliente HTTP de Calliope Data. Construye la URL con el
// scope de organización, aplica la credencial, impone el timeout y traduce el
// status HTTP a un error de dominio. No conoce Cobra ni formatos de salida.
package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/output"
)

// Options configura el cliente.
type Options struct {
	BaseURL    string
	Credential auth.Credential
	Timeout    time.Duration
	UserAgent  string
	HTTPClient *http.Client
}

// Client habla con Calliope Data.
type Client struct {
	baseURL string
	cred    auth.Credential
	http    *http.Client
	ua      string
}

// New construye un cliente. Si no se da HTTPClient, se crea uno con el timeout.
func New(opts Options) *Client {
	hc := opts.HTTPClient
	if hc == nil {
		t := opts.Timeout
		if t == 0 {
			t = 60 * time.Second
		}
		hc = &http.Client{Timeout: t}
	}
	ua := opts.UserAgent
	if ua == "" {
		ua = "calliope-cli"
	}
	return &Client{
		baseURL: strings.TrimRight(opts.BaseURL, "/"),
		cred:    opts.Credential,
		http:    hc,
		ua:      ua,
	}
}

// OrgPath construye una ruta con el scope de organización. El nombre se escapa:
// llega de configuración de proyecto, que es una entrada no confiable.
func (c *Client) OrgPath(org, suffix string) string {
	return "/v1/organizations/" + url.PathEscape(org) + suffix
}

// Do ejecuta una petición y decodifica la respuesta en out. Si out es nil, el
// cuerpo se descarta.
func (c *Client) Do(ctx context.Context, method, path string, body, out any) error {
	var cuerpo io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		cuerpo = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, cuerpo)
	if err != nil {
		return err
	}
	k, v := c.cred.Header()
	req.Header.Set(k, v)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.ua)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return errorDeTransporte(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// El cuerpo se descarta a propósito: no filtramos internals del backend.
		io.Copy(io.Discard, resp.Body)
		return mapearStatus(resp.StatusCode)
	}

	if out == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}

	crudo, err := io.ReadAll(resp.Body)
	if err != nil {
		return errorDeTransporte(err)
	}
	if len(bytes.TrimSpace(crudo)) == 0 {
		return nil
	}
	if err := json.Unmarshal(crudo, out); err != nil {
		return output.NewError(output.CodeGeneric,
			"La respuesta de Calliope Data no tiene el formato esperado.",
			"Comprueba la conectividad y la versión del backend con: calliope doctor")
	}
	return nil
}

// mapearStatus traduce el status a un error limpio, sin cuerpo del backend.
// Sigue el criterio de mapError en calliope-data-mcp.
func mapearStatus(status int) error {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return output.NewError(output.CodeUnauthorized,
			"No autorizado para acceder a estos datos.",
			"Comprueba tu credencial con: calliope auth status")
	case status == http.StatusNotFound:
		return output.NewError(output.CodeNotFound,
			"Recurso no encontrado.",
			"Comprueba el identificador y la organización activa con: calliope config list")
	case status == http.StatusTooManyRequests:
		return output.NewError(output.CodeRateLimited,
			"Se ha superado el límite de solicitudes.",
			"Espera unos segundos y reinténtalo.")
	default:
		return output.NewError(output.CodeGeneric,
			"Error al consultar Calliope Data.",
			"Diagnostica la conexión con: calliope doctor")
	}
}

func errorDeTransporte(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || esTimeout(err) {
		return output.NewError(output.CodeGeneric,
			"La solicitud a Calliope Data superó el tiempo límite.",
			"Sube el límite con CALLIOPE_TIMEOUT=120s, o comprueba la red con: calliope doctor")
	}
	return output.NewError(output.CodeGeneric,
		"No se pudo contactar con Calliope Data.",
		"Comprueba tu conexión y la URL del backend con: calliope doctor")
}

func esTimeout(err error) bool {
	var t interface{ Timeout() bool }
	return errors.As(err, &t) && t.Timeout()
}
```

- [ ] **Step 4: Ejecutar los tests y verificar que pasan**

Run: `go test ./internal/sdk/ -v`
Expected: PASS (7 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/sdk
git commit -m "feat: transporte del SDK con scoping de organización y mapeo de errores"
```

---

### Task 10: SDK — modelos y un método por endpoint

**Files:**
- Create: `internal/sdk/models.go`, `internal/sdk/api.go`
- Test: `internal/sdk/api_test.go`, `internal/sdk/testdata/*.json`

**Interfaces:**
- Consumes: `(*Client).Do`, `(*Client).OrgPath` (Task 9).
- Produces (todos los métodos toman `ctx context.Context` como primer parámetro):
  - `Ask(ctx, org, question, action string) (*AskResponse, error)`
  - `SearchDocuments(ctx, org, query string, limit int) ([]DocumentSearchResult, error)`
  - `ListDocuments(ctx, org string, p ListDocumentsParams) (*DocumentPage, error)`
  - `GetDocument(ctx, org, id string) (*DocumentResponse, error)`
  - `ListConcepts(ctx, org string) (*ConceptGraphResponse, error)`
  - `GetConcept(ctx, org, id string) (*ConceptDetailResponse, error)`
  - `ListRules(ctx, org string) ([]Rule, error)`
  - `Schema(ctx, org string) (*SchemaResponse, error)`
  - `Query(ctx, org, sql, formato string) (*QueryResponse, error)`
  - `Me(ctx) (*Me, error)`
  - `ListOrganizations(ctx) ([]Organization, error)`
  - `ListDocumentsParams{Status, Tag, Q string, Page, Size int}`
  - `(*QueryResponse).Rows() ([]map[string]any, error)`

Los tipos replican el contrato real, verificado en `calliope-data-mcp/src/calliope/types.ts`. **El JSON es camelCase**, de ahí los tags explícitos.

- [ ] **Step 1: Grabar los fixtures**

`internal/sdk/testdata/ask.json`:

```json
{
  "success": true,
  "text": "Las ventas crecieron un 12% interanual.",
  "rowCount": 2,
  "data": [{"mes": "2026-01", "ventas": 1200}, {"mes": "2026-02", "ventas": 1344}],
  "analysisType": "trend",
  "sources": [
    {
      "citation": "Informe anual, p. 12",
      "chunkId": 88,
      "documentId": "doc-1",
      "documentTitle": "Informe anual",
      "filename": "informe.pdf",
      "page": 12,
      "headingPath": "Resultados > Ventas",
      "excerpt": "Las ventas del ejercicio…"
    }
  ]
}
```

`internal/sdk/testdata/documents.json`:

```json
{
  "content": [
    {
      "id": "doc-1",
      "filename": "informe.pdf",
      "title": "Informe anual",
      "mimeType": "application/pdf",
      "declaredMime": "application/pdf",
      "sizeBytes": 402113,
      "status": "READY",
      "pageCount": 24,
      "charCount": 51200,
      "language": "es",
      "tags": ["finanzas"],
      "createdAt": "2026-01-04T10:00:00Z",
      "updatedAt": "2026-01-04T10:05:00Z",
      "readyAt": "2026-01-04T10:05:00Z"
    }
  ],
  "totalSize": 1
}
```

`internal/sdk/testdata/query_data_como_cadena.json`:

```json
{
  "metadata": {"rows": 2},
  "data": "[{\"mes\":\"2026-01\",\"ventas\":1200},{\"mes\":\"2026-02\",\"ventas\":1344}]"
}
```

- [ ] **Step 2: Escribir los tests que fallan**

`internal/sdk/api_test.go`:

```go
package sdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/calliope/calliope-cli/internal/auth"
)

func servidorConFixture(t *testing.T, fixture string, capturar *http.Request) *Client {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if capturar != nil {
			*capturar = *r.Clone(r.Context())
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}))
	t.Cleanup(srv.Close)
	return New(Options{BaseURL: srv.URL, Credential: auth.Credential{Kind: auth.KindAPIKey, Token: "k"}})
}

func TestAskDecodificaLaRespuestaYSusFuentes(t *testing.T) {
	c := servidorConFixture(t, "ask.json", nil)

	resp, err := c.Ask(context.Background(), "acme", "¿cómo van las ventas?", "")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !resp.Success {
		t.Error("Success debería ser true")
	}
	if resp.Text == "" {
		t.Error("Text vacío")
	}
	if len(resp.Sources) != 1 {
		t.Fatalf("len(Sources) = %d, se esperaba 1", len(resp.Sources))
	}
	if resp.Sources[0].DocumentID != "doc-1" {
		t.Errorf("DocumentID = %q — comprueba el tag json camelCase", resp.Sources[0].DocumentID)
	}
	if resp.Sources[0].ChunkID != 88 {
		t.Errorf("ChunkID = %d, se esperaba 88", resp.Sources[0].ChunkID)
	}
}

func TestAskEnviaLaPreguntaEnElCuerpo(t *testing.T) {
	var visto http.Request
	c := servidorConFixture(t, "ask.json", &visto)

	if _, err := c.Ask(context.Background(), "acme", "¿ventas?", "trend"); err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if visto.URL.Path != "/v1/organizations/acme/ask" {
		t.Errorf("ruta = %q", visto.URL.Path)
	}
	if visto.Method != http.MethodPost {
		t.Errorf("método = %q, se esperaba POST", visto.Method)
	}
}

func TestListDocumentsDecodificaLaPagina(t *testing.T) {
	c := servidorConFixture(t, "documents.json", nil)

	page, err := c.ListDocuments(context.Background(), "acme", ListDocumentsParams{Size: 10})
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if page.TotalSize != 1 || len(page.Content) != 1 {
		t.Fatalf("página inesperada: %+v", page)
	}
	d := page.Content[0]
	if d.SizeBytes != 402113 || d.Status != "READY" || d.PageCount == nil || *d.PageCount != 24 {
		t.Errorf("documento mal decodificado: %+v", d)
	}
}

func TestListDocumentsPasaLosFiltrosPorQuery(t *testing.T) {
	var visto http.Request
	c := servidorConFixture(t, "documents.json", &visto)

	_, err := c.ListDocuments(context.Background(), "acme",
		ListDocumentsParams{Status: "READY", Tag: "finanzas", Q: "ventas", Page: 2, Size: 50})
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}

	q := visto.URL.Query()
	for clave, quiero := range map[string]string{
		"status": "READY", "tag": "finanzas", "q": "ventas", "page": "2", "size": "50",
	} {
		if got := q.Get(clave); got != quiero {
			t.Errorf("query %s = %q, se esperaba %q", clave, got, quiero)
		}
	}
}

func TestListDocumentsOmiteLosFiltrosVacios(t *testing.T) {
	var visto http.Request
	c := servidorConFixture(t, "documents.json", &visto)

	if _, err := c.ListDocuments(context.Background(), "acme", ListDocumentsParams{}); err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if visto.URL.RawQuery != "" {
		t.Errorf("query = %q, sin filtros no debe enviarse ninguno", visto.URL.RawQuery)
	}
}

// El backend devuelve a veces `data` como una cadena JSON en vez de como
// array. Lo hace la UI en composables/useApi.ts, así que el SDK debe cubrirlo.
func TestQueryAceptaDataComoCadenaJSON(t *testing.T) {
	c := servidorConFixture(t, "query_data_como_cadena.json", nil)

	resp, err := c.Query(context.Background(), "acme", "SELECT 1", "")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	filas, err := resp.Rows()
	if err != nil {
		t.Fatalf("Rows: %v", err)
	}
	if len(filas) != 2 {
		t.Fatalf("len(filas) = %d, se esperaba 2", len(filas))
	}
	if filas[0]["mes"] != "2026-01" {
		t.Errorf("fila[0] = %+v", filas[0])
	}
}

func TestQueryAceptaDataComoArray(t *testing.T) {
	crudo := `{"data":[{"mes":"2026-03","ventas":900}]}`
	var resp QueryResponse
	if err := json.Unmarshal([]byte(crudo), &resp); err != nil {
		t.Fatal(err)
	}
	filas, err := resp.Rows()
	if err != nil {
		t.Fatalf("Rows: %v", err)
	}
	if len(filas) != 1 || filas[0]["mes"] != "2026-03" {
		t.Errorf("filas = %+v", filas)
	}
}

func TestQuerySinDataDevuelveCero(t *testing.T) {
	var resp QueryResponse
	if err := json.Unmarshal([]byte(`{}`), &resp); err != nil {
		t.Fatal(err)
	}
	filas, err := resp.Rows()
	if err != nil {
		t.Fatalf("Rows sin data no debe fallar: %v", err)
	}
	if len(filas) != 0 {
		t.Errorf("se esperaban 0 filas, se obtuvieron %d", len(filas))
	}
}
```

- [ ] **Step 3: Ejecutar los tests y verificar que fallan**

Run: `go test ./internal/sdk/ -run "Ask|ListDocuments|Query" -v`
Expected: FAIL — `undefined: AskResponse`

- [ ] **Step 4: Implementar los modelos**

`internal/sdk/models.go`:

```go
package sdk

import "encoding/json"

// Los tipos replican el contrato de Calliope Data verificado en
// calliope-data-mcp/src/calliope/types.ts. El JSON del backend es camelCase.
// Solo se declaran los campos que el CLI usa; el resto se ignora.

// --- /ask ---

// AskDocumentSource es una cita devuelta por /ask.
type AskDocumentSource struct {
	Citation      string  `json:"citation"`
	ChunkID       int     `json:"chunkId"`
	DocumentID    string  `json:"documentId"`
	DocumentTitle *string `json:"documentTitle"`
	Filename      string  `json:"filename"`
	Page          *int    `json:"page"`
	HeadingPath   *string `json:"headingPath"`
	Excerpt       string  `json:"excerpt"`
}

// AskQueryDataset es uno de los conjuntos de datos que respaldan la respuesta.
type AskQueryDataset struct {
	ID       string           `json:"id"`
	Purpose  *string          `json:"purpose"`
	Data     []map[string]any `json:"data"`
	RowCount *int             `json:"rowCount"`
	Error    *string          `json:"error"`
}

// AskResponse es la respuesta de /ask.
type AskResponse struct {
	Success      bool                `json:"success"`
	Text         string              `json:"text"`
	Data         []map[string]any    `json:"data"`
	RowCount     *int                `json:"rowCount"`
	Queries      []AskQueryDataset   `json:"queries"`
	AnalysisType *string             `json:"analysisType"`
	Sources      []AskDocumentSource `json:"sources"`
	Error        *string             `json:"error"`
}

// --- /search/documents ---

// DocumentSearchResult es un fragmento devuelto por la búsqueda semántica.
type DocumentSearchResult struct {
	ChunkID       int     `json:"chunkId"`
	DocumentID    string  `json:"documentId"`
	DocumentTitle *string `json:"documentTitle"`
	Filename      string  `json:"filename"`
	Ordinal       int     `json:"ordinal"`
	PageFrom      *int    `json:"pageFrom"`
	PageTo        *int    `json:"pageTo"`
	HeadingPath   *string `json:"headingPath"`
	Excerpt       string  `json:"excerpt"`
	Score         float64 `json:"score"`
}

// --- /documents ---

// DocumentResponse son los metadatos de un documento.
type DocumentResponse struct {
	ID           string   `json:"id"`
	Filename     string   `json:"filename"`
	Title        *string  `json:"title"`
	MimeType     *string  `json:"mimeType"`
	DeclaredMime string   `json:"declaredMime"`
	SizeBytes    int64    `json:"sizeBytes"`
	Status       string   `json:"status"`
	PageCount    *int     `json:"pageCount"`
	CharCount    *int     `json:"charCount"`
	Language     *string  `json:"language"`
	Tags         []string `json:"tags"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
	ReadyAt      *string  `json:"readyAt"`
}

// DocumentPage es una página de documentos.
type DocumentPage struct {
	Content   []DocumentResponse `json:"content"`
	TotalSize int                `json:"totalSize"`
}

// --- /knowledge/concepts ---

// GraphConceptNode es un concepto en el grafo de la ontología.
type GraphConceptNode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	IsActive    bool   `json:"isActive"`
	RecordCount *int   `json:"recordCount"`
	SourceCount *int   `json:"sourceCount"`
}

// ConceptGraphResponse es el grafo completo de conceptos.
type ConceptGraphResponse struct {
	Concepts []GraphConceptNode `json:"concepts"`
}

// ConceptResponse es la cabecera de un concepto.
type ConceptResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IsActive    bool    `json:"isActive"`
}

// AttributeResponse es un atributo de un concepto.
type AttributeResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IsActive    bool    `json:"isActive"`
}

// ConceptDetailResponse es el detalle de un concepto con sus atributos.
type ConceptDetailResponse struct {
	Concept    ConceptResponse     `json:"concept"`
	Attributes []AttributeResponse `json:"attributes"`
}

// --- /rules ---

// Rule es una regla de negocio compartida de la organización.
type Rule struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Details  string  `json:"details"`
	Category *string `json:"category"`
	Status   string  `json:"status"`
}

// --- /database/schema ---

// SchemaColumn es una columna de una tabla.
type SchemaColumn struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// SchemaTable es una tabla del esquema.
type SchemaTable struct {
	Name    string         `json:"name"`
	Columns []SchemaColumn `json:"columns"`
}

// SchemaResponse es el esquema de la base de datos de la organización.
type SchemaResponse struct {
	Tables []SchemaTable `json:"tables"`
}

// --- /query ---

// QueryResponse es el resultado de una consulta SQL. El backend devuelve
// `data` unas veces como array y otras como cadena JSON; Rows normaliza ambas.
type QueryResponse struct {
	Data     json.RawMessage `json:"data"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// Rows devuelve las filas del resultado, venga `data` como array o como cadena.
func (r *QueryResponse) Rows() ([]map[string]any, error) {
	if len(r.Data) == 0 || string(r.Data) == "null" {
		return nil, nil
	}

	var filas []map[string]any
	if err := json.Unmarshal(r.Data, &filas); err == nil {
		return filas, nil
	}

	// El backend la ha enviado como cadena JSON: se deshace una capa.
	var comoCadena string
	if err := json.Unmarshal(r.Data, &comoCadena); err != nil {
		return nil, err
	}
	if comoCadena == "" {
		return nil, nil
	}
	if err := json.Unmarshal([]byte(comoCadena), &filas); err != nil {
		return nil, err
	}
	return filas, nil
}

// --- auth y organizaciones ---

// Me es el perfil del titular de la credencial.
type Me struct {
	UserID        string         `json:"userId"`
	Email         string         `json:"email"`
	Organizations []Organization `json:"organizations,omitempty"`
}

// Organization es una organización accesible con la credencial actual.
type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
```

- [ ] **Step 5: Implementar los métodos de la API**

`internal/sdk/api.go`:

```go
package sdk

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// Ask hace una pregunta en lenguaje natural sobre los datos y la documentación.
func (c *Client) Ask(ctx context.Context, org, question, action string) (*AskResponse, error) {
	cuerpo := map[string]string{"question": question}
	if action != "" {
		cuerpo["action"] = action
	}
	var out AskResponse
	if err := c.Do(ctx, http.MethodPost, c.OrgPath(org, "/ask"), cuerpo, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SearchDocuments busca semánticamente en la documentación.
func (c *Client) SearchDocuments(ctx context.Context, org, query string, limit int) ([]DocumentSearchResult, error) {
	if limit <= 0 {
		limit = 10
	}
	cuerpo := map[string]any{"query": query, "limit": limit}
	var out []DocumentSearchResult
	if err := c.Do(ctx, http.MethodPost, c.OrgPath(org, "/search/documents"), cuerpo, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListDocumentsParams son los filtros de listado de documentos.
type ListDocumentsParams struct {
	Status string
	Tag    string
	Q      string
	Page   int
	Size   int
}

func (p ListDocumentsParams) query() string {
	q := url.Values{}
	if p.Status != "" {
		q.Set("status", p.Status)
	}
	if p.Tag != "" {
		q.Set("tag", p.Tag)
	}
	if p.Q != "" {
		q.Set("q", p.Q)
	}
	if p.Page > 0 {
		q.Set("page", strconv.Itoa(p.Page))
	}
	if p.Size > 0 {
		q.Set("size", strconv.Itoa(p.Size))
	}
	if len(q) == 0 {
		return ""
	}
	return "?" + q.Encode()
}

// ListDocuments lista los documentos de la organización.
func (c *Client) ListDocuments(ctx context.Context, org string, p ListDocumentsParams) (*DocumentPage, error) {
	var out DocumentPage
	if err := c.Do(ctx, http.MethodGet, c.OrgPath(org, "/documents"+p.query()), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDocument devuelve los metadatos de un documento.
func (c *Client) GetDocument(ctx context.Context, org, id string) (*DocumentResponse, error) {
	var out DocumentResponse
	if err := c.Do(ctx, http.MethodGet, c.OrgPath(org, "/documents/"+url.PathEscape(id)), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListConcepts devuelve el grafo de conceptos de la ontología.
func (c *Client) ListConcepts(ctx context.Context, org string) (*ConceptGraphResponse, error) {
	var out ConceptGraphResponse
	if err := c.Do(ctx, http.MethodGet, c.OrgPath(org, "/knowledge/concepts"), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetConcept devuelve un concepto con sus atributos.
func (c *Client) GetConcept(ctx context.Context, org, id string) (*ConceptDetailResponse, error) {
	var out ConceptDetailResponse
	ruta := c.OrgPath(org, "/knowledge/concepts/"+url.PathEscape(id))
	if err := c.Do(ctx, http.MethodGet, ruta, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListRules devuelve las reglas de negocio compartidas.
func (c *Client) ListRules(ctx context.Context, org string) ([]Rule, error) {
	var out []Rule
	if err := c.Do(ctx, http.MethodGet, c.OrgPath(org, "/rules"), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Schema devuelve el esquema de la base de datos de la organización.
func (c *Client) Schema(ctx context.Context, org string) (*SchemaResponse, error) {
	var out SchemaResponse
	if err := c.Do(ctx, http.MethodGet, c.OrgPath(org, "/database/schema"), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Query ejecuta SQL crudo. El parámetro formato se reenvía tal cual al backend
// en QueryRequest.output; no confundir con el flag --csv, que es render local.
func (c *Client) Query(ctx context.Context, org, sql, formato string) (*QueryResponse, error) {
	cuerpo := map[string]string{"sql": sql}
	if formato != "" {
		cuerpo["output"] = formato
	}
	var out QueryResponse
	if err := c.Do(ctx, http.MethodPost, c.OrgPath(org, "/query"), cuerpo, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Me devuelve el perfil del titular de la credencial.
func (c *Client) Me(ctx context.Context) (*Me, error) {
	var out Me
	if err := c.Do(ctx, http.MethodGet, "/v1/auth/me", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListOrganizations devuelve las organizaciones accesibles con la credencial.
func (c *Client) ListOrganizations(ctx context.Context) ([]Organization, error) {
	var out []Organization
	if err := c.Do(ctx, http.MethodGet, "/v1/organizations", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
```

- [ ] **Step 6: Ejecutar los tests y verificar que pasan**

Run: `go test ./internal/sdk/ -v`
Expected: PASS (15 tests)

- [ ] **Step 7: Verificar las formas que no vienen del MCP contra el backend real**

`SchemaResponse`, `Me` y `Organization` se han modelado a partir de las llamadas de la UI, no del MCP, así que sus campos exactos no están verificados. Con una API key real:

```bash
curl -s -H "X-API-Key: $CALLIOPE_API_KEY" https://data-0.calliope.so/v1/auth/me | head -40
curl -s -H "X-API-Key: $CALLIOPE_API_KEY" https://data-0.calliope.so/v1/organizations | head -40
curl -s -H "X-API-Key: $CALLIOPE_API_KEY" "https://data-0.calliope.so/v1/organizations/$ORG/database/schema" | head -60
```

Ajusta los tags `json` de esos tres tipos a lo que devuelva el backend y guarda las respuestas recortadas como `testdata/me.json`, `testdata/organizations.json` y `testdata/schema.json`. Añade un test por cada uno siguiendo el patrón de `TestListDocumentsDecodificaLaPagina`.

Si no tienes API key todavía, para aquí: los comandos de las tareas 12 a 18 dependen de que estos tipos sean correctos.

- [ ] **Step 8: Commit**

```bash
git add internal/sdk
git commit -m "feat: modelos y métodos del SDK de Calliope Data"
```

---

### Task 11: appctx — el wiring de una invocación

**Files:**
- Create: `internal/appctx/appctx.go`
- Test: `internal/appctx/appctx_test.go`

**Interfaces:**
- Consumes: `config.Load` (Tasks 4-5), `auth.Resolve` y `auth.DefaultStore` (Task 7), `sdk.New` (Task 9), `presenter.Options` y `presenter.Render` (Task 3).
- Produces:
  - `appctx.Deps{Cwd string, Env func(string) string, Store auth.Store, Stdout, Stderr io.Writer, IsTTY bool}`
  - `appctx.DefaultDeps() Deps`
  - `appctx.Context{Cfg *config.Config, Cred auth.Credential, CredSource string, Client *sdk.Client, Org string, Present presenter.Options}`
  - `appctx.RegisterGlobalFlags(cmd *cobra.Command)` — declara `--org`, `--json`, `--quiet`, `--md`, `--jq` como flags persistentes
  - `appctx.Build(cmd *cobra.Command, d Deps) (*Context, error)`
  - `appctx.BuildSinCredencial(cmd *cobra.Command, d Deps) (*Context, error)` — para `config`, `version` y `doctor`, que deben funcionar sin estar autenticado
  - `(*Context).Render(r presenter.Result) error`

Este es el único sitio donde se juntan las cuatro capas. Los comandos reciben un `*Context` ya montado y no vuelven a hablar de configuración ni de credenciales.

- [ ] **Step 1: Escribir los tests que fallan**

`internal/appctx/appctx_test.go`:

```go
package appctx

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/output"
)

func comandoConFlags(flags map[string]string) *cobra.Command {
	cmd := &cobra.Command{Use: "x"}
	RegisterGlobalFlags(cmd)
	for k, v := range flags {
		_ = cmd.Flags().Set(k, v)
	}
	return cmd
}

func depsDePrueba(t *testing.T, cwd string) (Deps, *bytes.Buffer) {
	t.Helper()
	st := auth.NewFileStore(filepath.Join(t.TempDir(), "c.json"))
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	home := t.TempDir() // fuera del closure: si no, cada llamada crearía otro
	return Deps{
		Cwd: cwd,
		Env: func(k string) string {
			if k == "HOME" {
				return home
			}
			return ""
		},
		Store:  st,
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
	}, &stderr
}

func escribirConfigProyecto(t *testing.T, dir string, vals map[string]string) {
	t.Helper()
	d := filepath.Join(dir, ".calliope")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(vals)
	if err := os.WriteFile(filepath.Join(d, "config.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLaOrganizacionSaleDeLaConfiguracionDeProyecto(t *testing.T) {
	dir := t.TempDir()
	escribirConfigProyecto(t, dir, map[string]string{"org": "acme"})
	deps, _ := depsDePrueba(t, dir)

	ctx, err := Build(comandoConFlags(nil), deps)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if ctx.Org != "acme" {
		t.Errorf("Org = %q, se esperaba acme", ctx.Org)
	}
}

func TestElFlagOrgGanaALaConfiguracion(t *testing.T) {
	dir := t.TempDir()
	escribirConfigProyecto(t, dir, map[string]string{"org": "acme"})
	deps, _ := depsDePrueba(t, dir)

	ctx, err := Build(comandoConFlags(map[string]string{"org": "otra"}), deps)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if ctx.Org != "otra" {
		t.Errorf("Org = %q, el flag debe ganar", ctx.Org)
	}
}

func TestSinOrganizacionElErrorDiceComoElegirla(t *testing.T) {
	deps, _ := depsDePrueba(t, t.TempDir())

	_, err := Build(comandoConFlags(nil), deps)
	if err == nil {
		t.Fatal("se esperaba error al no haber organización")
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2 (uso incorrecto)", got)
	}
	if !strings.Contains(err.Error(), "organización") {
		t.Errorf("mensaje poco claro: %q", err.Error())
	}
}

func TestLosAvisosDeConfiguracionVanAStderr(t *testing.T) {
	dir := t.TempDir()
	escribirConfigProyecto(t, dir, map[string]string{"org": "acme", "base_url": "https://atacante.example"})
	deps, stderr := depsDePrueba(t, dir)

	if _, err := Build(comandoConFlags(nil), deps); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(stderr.String(), "base_url") {
		t.Errorf("el aviso debe llegar a stderr, se obtuvo: %q", stderr.String())
	}
}

func TestLosFlagsDeterminanElModoDeSalida(t *testing.T) {
	dir := t.TempDir()
	escribirConfigProyecto(t, dir, map[string]string{"org": "acme"})

	casos := []struct {
		flags map[string]string
		quiero string
	}{
		{map[string]string{"json": "true"}, "json"},
		{map[string]string{"quiet": "true"}, "quiet"},
		{map[string]string{"md": "true"}, "md"},
		{map[string]string{"jq": ".data"}, "jq"},
		{nil, "auto"},
	}

	for _, caso := range casos {
		deps, _ := depsDePrueba(t, dir)
		ctx, err := Build(comandoConFlags(caso.flags), deps)
		if err != nil {
			t.Fatalf("Build(%v): %v", caso.flags, err)
		}
		if string(ctx.Present.Mode) != caso.quiero {
			t.Errorf("flags %v → modo %q, se esperaba %q", caso.flags, ctx.Present.Mode, caso.quiero)
		}
	}
}

func TestBuildSinCredencialNoFallaSinAutenticacion(t *testing.T) {
	dir := t.TempDir()
	escribirConfigProyecto(t, dir, map[string]string{"org": "acme"})
	deps, _ := depsDePrueba(t, dir)
	deps.Store = auth.NewFileStore(filepath.Join(t.TempDir(), "vacio.json"))

	ctx, err := BuildSinCredencial(comandoConFlags(nil), deps)
	if err != nil {
		t.Fatalf("BuildSinCredencial no debe exigir credencial: %v", err)
	}
	if ctx.Cred.Valid() {
		t.Error("no debería haber credencial")
	}
}
```

- [ ] **Step 2: Ejecutar los tests y verificar que fallan**

Run: `go test ./internal/appctx/ -v`
Expected: FAIL — `undefined: Build`

- [ ] **Step 3: Implementar el wiring**

`internal/appctx/appctx.go`:

```go
// Package appctx monta el contexto de una invocación: configuración,
// credencial, cliente y modo de salida. Es el único punto donde se juntan las
// cuatro capas; los comandos reciben un *Context ya montado.
package appctx

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/config"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/presenter"
	"github.com/calliope/calliope-cli/internal/sdk"
	"github.com/calliope/calliope-cli/internal/version"
)

// Deps son las dependencias externas de una invocación, inyectables para poder
// probar el wiring sin tocar el entorno real.
type Deps struct {
	Cwd    string
	Env    func(string) string
	Store  auth.Store
	Stdout io.Writer
	Stderr io.Writer
	IsTTY  bool
}

// DefaultDeps son las dependencias reales del proceso.
func DefaultDeps() Deps {
	cwd, _ := os.Getwd()
	env := os.Getenv
	return Deps{
		Cwd:    cwd,
		Env:    env,
		Store:  auth.DefaultStore(filepath.Dir(config.GlobalPath(env))),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		IsTTY:  term.IsTerminal(int(os.Stdout.Fd())),
	}
}

// RegisterGlobalFlags declara los flags que toda invocación entiende. Vive
// aquí y no en cli para que los tests de comandos puedan montar una raíz
// idéntica a la real sin duplicar la lista, que si no se desviaría.
func RegisterGlobalFlags(cmd *cobra.Command) {
	f := cmd.PersistentFlags()
	f.String("org", "", "organización sobre la que operar")
	f.Bool("json", false, "salida JSON con envelope completo")
	f.Bool("quiet", false, "salida solo de datos, sin envelope")
	f.Bool("md", false, "salida en Markdown")
	f.String("jq", "", "filtra la salida con una expresión jq")
}

// Context es todo lo que un comando necesita para hacer su trabajo.
type Context struct {
	Cfg        *config.Config
	Cred       auth.Credential
	CredSource string
	Client     *sdk.Client
	Org        string
	Present    presenter.Options
	Deps       Deps
}

// Build monta el contexto y exige credencial y organización. Es lo que usan
// todos los comandos que hablan con el backend.
func Build(cmd *cobra.Command, d Deps) (*Context, error) {
	ctx, err := BuildSinCredencial(cmd, d)
	if err != nil {
		return nil, err
	}

	cred, origen, err := auth.Resolve(d.Env, d.Store)
	if err != nil {
		return nil, err
	}
	ctx.Cred = cred
	ctx.CredSource = origen

	if ctx.Org == "" {
		ctx.Org = cred.Org
	}
	if ctx.Org == "" {
		return nil, output.NewError(output.CodeUsage,
			"No hay ninguna organización seleccionada.",
			"Elige una con: calliope orgs use <nombre>   (lista las disponibles con: calliope orgs list)")
	}

	ctx.Client = sdk.New(sdk.Options{
		BaseURL:    ctx.Cfg.BaseURL(),
		Credential: cred,
		Timeout:    timeoutDe(ctx.Cfg),
		UserAgent:  "calliope-cli/" + version.Version,
	})
	return ctx, nil
}

// BuildSinCredencial monta lo que se puede montar sin estar autenticado. Lo
// usan config, version y doctor, que tienen que funcionar precisamente cuando
// la autenticación es el problema.
func BuildSinCredencial(cmd *cobra.Command, d Deps) (*Context, error) {
	flags := map[string]string{}
	if v, _ := cmd.Flags().GetString("org"); v != "" {
		flags[config.KeyOrg] = v
	}

	cfg, avisos, err := config.Load(d.Cwd, d.Env, flags)
	if err != nil {
		return nil, err
	}
	for _, a := range avisos {
		fmt.Fprintln(d.Stderr, a)
	}

	return &Context{
		Cfg:     cfg,
		Org:     cfg.Org(),
		Present: modoDeSalida(cmd, cfg, d),
		Deps:    d,
	}, nil
}

// Render escribe un resultado con el modo de salida de esta invocación.
func (c *Context) Render(r presenter.Result) error {
	return presenter.Render(r, c.Present)
}

func modoDeSalida(cmd *cobra.Command, cfg *config.Config, d Deps) presenter.Options {
	opts := presenter.Options{Mode: presenter.ModeAuto, IsTTY: d.IsTTY, Out: d.Stdout}

	// El fichero de configuración puede fijar el modo por defecto; los flags
	// mandan sobre él.
	switch cfg.Output() {
	case "json":
		opts.Mode = presenter.ModeJSON
	case "quiet":
		opts.Mode = presenter.ModeQuiet
	case "md":
		opts.Mode = presenter.ModeMarkdown
	}

	if v, _ := cmd.Flags().GetString("jq"); v != "" {
		opts.Mode, opts.JQExpr = presenter.ModeJQ, v
		return opts
	}
	if v, _ := cmd.Flags().GetBool("json"); v {
		opts.Mode = presenter.ModeJSON
	}
	if v, _ := cmd.Flags().GetBool("quiet"); v {
		opts.Mode = presenter.ModeQuiet
	}
	if v, _ := cmd.Flags().GetBool("md"); v {
		opts.Mode = presenter.ModeMarkdown
	}
	return opts
}

func timeoutDe(cfg *config.Config) time.Duration {
	d, err := time.ParseDuration(cfg.Get(config.KeyTimeout).Value)
	if err != nil || d <= 0 {
		return 60 * time.Second
	}
	return d
}
```

```bash
go get golang.org/x/term@latest
```

- [ ] **Step 4: Ejecutar los tests y verificar que pasan**

Run: `go test ./internal/appctx/ -v`
Expected: PASS (6 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/appctx go.mod go.sum
git commit -m "feat: appctx monta configuración, credencial, cliente y salida"
```

---

### Task 12: Comandos `auth` y `orgs`

**Files:**
- Create: `internal/commands/auth.go`, `internal/commands/orgs.go`
- Modify: `internal/cli/root.go`
- Test: `internal/commands/auth_test.go`, `internal/commands/orgs_test.go`

**Interfaces:**
- Consumes: `appctx.Build`, `appctx.BuildSinCredencial`, `appctx.Deps` (Task 11); `sdk.Me`, `sdk.Organization` (Task 10); `auth.Store` (Task 7).
- Produces: `commands.NewAuthCmd(d appctx.Deps) *cobra.Command`, `commands.NewOrgsCmd(d appctx.Deps) *cobra.Command`.

`auth` y `orgs` son grupos: no definen `RunE`, así que invocarlos pelados muestra la ayuda.

- [ ] **Step 1: Escribir los tests que fallan**

`internal/commands/auth_test.go`:

```go
package commands

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/auth"
)

// raizDePrueba monta una raíz con los mismos flags globales que la real y le
// cuelga el comando bajo prueba. Sin esto, invocar un subcomando suelto con
// --json falla con "unknown flag": los flags globales son persistentes de la
// raíz. Todos los tests de comandos pasan por aquí.
func raizDePrueba(sub *cobra.Command, out io.Writer) *cobra.Command {
	root := &cobra.Command{Use: "calliope", SilenceUsage: true, SilenceErrors: true}
	appctx.RegisterGlobalFlags(root)
	root.AddCommand(sub)
	root.SetOut(out)
	root.SetErr(out)
	return root
}

func depsConServidor(t *testing.T, h http.HandlerFunc) (appctx.Deps, *bytes.Buffer, auth.Store) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	st := auth.NewFileStore(filepath.Join(t.TempDir(), "c.json"))
	var stdout bytes.Buffer
	home := t.TempDir()
	d := appctx.Deps{
		Cwd: t.TempDir(),
		Env: func(k string) string {
			switch k {
			case "HOME":
				return home
			case "CALLIOPE_BASE_URL":
				return srv.URL
			case "CALLIOPE_ORG":
				return "acme"
			}
			return ""
		},
		Store:  st,
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	return d, &stdout, st
}

func TestAuthEsUnGrupoSinRunE(t *testing.T) {
	cmd := NewAuthCmd(appctx.Deps{})
	if cmd.RunE != nil || cmd.Run != nil {
		t.Error("auth es un grupo: invocarlo pelado debe mostrar la ayuda, no ejecutar nada")
	}
	if len(cmd.Commands()) == 0 {
		t.Error("auth debe tener subcomandos")
	}
}

func TestAuthLoginValidaLaCredencialAntesDeGuardarla(t *testing.T) {
	llamado := false
	d, _, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		llamado = true
		if r.URL.Path != "/v1/auth/me" {
			t.Errorf("login debe validar contra /v1/auth/me, llamó a %q", r.URL.Path)
		}
		w.Write([]byte(`{"userId":"u-1","email":"a@b.c"}`))
	})

	root := raizDePrueba(NewAuthCmd(d), d.Stdout)
	root.SetArgs([]string{"auth", "login", "--api-key", "cal_live_ok"})
	if err := root.Execute(); err != nil {
		t.Fatalf("login: %v", err)
	}

	if !llamado {
		t.Fatal("login no validó la credencial contra el backend")
	}
	c, err := st.Load()
	if err != nil || c == nil || c.Token != "cal_live_ok" {
		t.Errorf("la credencial no se guardó: %+v (%v)", c, err)
	}
}

func TestAuthLoginNoGuardaUnaCredencialRechazada(t *testing.T) {
	d, _, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	root := raizDePrueba(NewAuthCmd(d), d.Stdout)
	root.SetArgs([]string{"auth", "login", "--api-key", "cal_live_mala"})
	if err := root.Execute(); err == nil {
		t.Fatal("se esperaba error con una credencial rechazada")
	}

	if c, _ := st.Load(); c != nil {
		t.Errorf("nunca se debe persistir una credencial no verificada, se guardó: %+v", c)
	}
}

func TestAuthLogoutBorraLaCredencial(t *testing.T) {
	d, _, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewAuthCmd(d), d.Stdout)
	root.SetArgs([]string{"auth", "logout"})
	if err := root.Execute(); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if c, _ := st.Load(); c != nil {
		t.Error("tras logout no debe quedar credencial")
	}
}

func TestAuthStatusNoImprimeElToken(t *testing.T) {
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "cal_live_secreto"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewAuthCmd(d), stdout)
	root.SetArgs([]string{"auth", "status"})
	if err := root.Execute(); err != nil {
		t.Fatalf("status: %v", err)
	}
	if strings.Contains(stdout.String(), "cal_live_secreto") {
		t.Error("auth status no debe imprimir el token completo")
	}
}
```

`internal/commands/orgs_test.go`:

```go
package commands

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/calliope/calliope-cli/internal/auth"
)

func TestOrgsListDevuelveLasOrganizaciones(t *testing.T) {
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations" {
			t.Errorf("ruta = %q", r.URL.Path)
		}
		w.Write([]byte(`[{"id":"o-1","name":"acme"},{"id":"o-2","name":"globex"}]`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewOrgsCmd(d), stdout)
	root.SetArgs([]string{"orgs", "list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("orgs list: %v", err)
	}

	var env struct {
		OK   bool `json:"ok"`
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if len(env.Data) != 2 || env.Data[0].Name != "acme" {
		t.Errorf("data inesperada: %q", stdout.String())
	}
}

func TestOrgsUseEscribeLaConfiguracionDeProyecto(t *testing.T) {
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewOrgsCmd(d), stdout)
	root.SetArgs([]string{"orgs", "use", "globex"})
	if err := root.Execute(); err != nil {
		t.Fatalf("orgs use: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(d.Cwd, ".calliope", "config.json"))
	if err != nil {
		t.Fatalf("no se escribió la configuración de proyecto: %v", err)
	}
	var vals map[string]string
	if err := json.Unmarshal(b, &vals); err != nil {
		t.Fatal(err)
	}
	if vals["org"] != "globex" {
		t.Errorf("config = %v, se esperaba org=globex", vals)
	}
}
```

- [ ] **Step 2: Ejecutar los tests y verificar que fallan**

Run: `go test ./internal/commands/ -v`
Expected: FAIL — `undefined: NewAuthCmd`

- [ ] **Step 3: Implementar `auth`**

`internal/commands/auth.go`:

```go
// Package commands define los comandos de calliope, uno por fichero de grupo.
package commands

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/presenter"
	"github.com/calliope/calliope-cli/internal/sdk"
)

// NewAuthCmd construye el grupo `auth`. Sin RunE: invocarlo pelado muestra la
// ayuda, que es lo que hace Cobra con un comando que solo tiene subcomandos.
func NewAuthCmd(d appctx.Deps) *cobra.Command {
	grupo := &cobra.Command{
		Use:   "auth",
		Short: "Gestiona la autenticación con Calliope Data",
	}
	grupo.AddCommand(newAuthLoginCmd(d), newAuthLogoutCmd(d), newAuthStatusCmd(d), newAuthTokenCmd(d))
	return grupo
}

func newAuthLoginCmd(d appctx.Deps) *cobra.Command {
	var apiKey string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Guarda y verifica una credencial de Calliope",
		RunE: func(cmd *cobra.Command, args []string) error {
			if apiKey == "" {
				return output.NewError(output.CodeUsage,
					"Falta la credencial.",
					"Ejecuta: calliope auth login --api-key <clave>  (créala en el UI, en Observabilidad → Claves API)")
			}

			cred := auth.Credential{Kind: auth.KindAPIKey, Token: apiKey}

			// Se valida ANTES de guardar: nunca se persiste una credencial
			// no verificada.
			ctx, err := appctx.BuildSinCredencial(cmd, d)
			if err != nil {
				return err
			}
			cliente := sdk.New(sdk.Options{BaseURL: ctx.Cfg.BaseURL(), Credential: cred})
			me, err := cliente.Me(cmd.Context())
			if err != nil {
				return err
			}

			if err := d.Store.Save(cred); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Sesión iniciada como %s.\n", me.Email)
			return nil
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "", "clave de API de Calliope")
	return cmd
}

func newAuthLogoutCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Borra la credencial almacenada",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.Store.Delete(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Credencial borrada.")
			return nil
		},
	}
}

func newAuthStatusCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Muestra quién eres y de dónde sale la credencial",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}
			me, err := ctx.Client.Me(cmd.Context())
			if err != nil {
				return err
			}

			datos := map[string]string{
				"email":       me.Email,
				"userId":      me.UserID,
				"credencial":  string(ctx.Cred.Kind),
				"origen":      ctx.CredSource,
				"organizacion": ctx.Org,
			}
			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(datos, "autenticado como "+me.Email,
					output.Breadcrumb{Action: "organizaciones", Cmd: "calliope orgs list"}),
				Text: func(w io.Writer) error {
					// El token nunca se imprime aquí; para eso está `auth token`.
					_, err := fmt.Fprintf(w, "Autenticado como %s\nCredencial: %s (%s)\nOrganización: %s\n",
						me.Email, ctx.Cred.Kind, ctx.CredSource, ctx.Org)
					return err
				},
			})
		},
	}
}

func newAuthTokenCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "token",
		Short: "Imprime la credencial almacenada (para scripts)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cred, _, err := auth.Resolve(d.Env, d.Store)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), cred.Token)
			return nil
		},
	}
}
```

- [ ] **Step 4: Implementar `orgs`**

`internal/commands/orgs.go`:

```go
package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/config"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/presenter"
)

// NewOrgsCmd construye el grupo `orgs`.
func NewOrgsCmd(d appctx.Deps) *cobra.Command {
	grupo := &cobra.Command{
		Use:   "orgs",
		Short: "Lista y selecciona la organización activa",
	}
	grupo.AddCommand(newOrgsListCmd(d), newOrgsUseCmd(d))
	return grupo
}

func newOrgsListCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lista las organizaciones accesibles con tu credencial",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Listar organizaciones no requiere tener una elegida.
			ctx, err := appctx.BuildSinCredencial(cmd, d)
			if err != nil {
				return err
			}
			cred, _, err := authResolve(d)
			if err != nil {
				return err
			}
			cliente := clienteCon(ctx, cred)

			orgs, err := cliente.ListOrganizations(cmd.Context())
			if err != nil {
				return err
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(orgs, fmt.Sprintf("%d organizaciones", len(orgs)),
					output.Breadcrumb{Action: "usar", Cmd: "calliope orgs use <nombre>"}),
				Text: func(w io.Writer) error {
					filas := make([][]string, 0, len(orgs))
					for _, o := range orgs {
						filas = append(filas, []string{o.Name, o.ID})
					}
					return presenter.Table(w, []string{"NOMBRE", "ID"}, filas)
				},
			})
		},
	}
}

func newOrgsUseCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "use <organización>",
		Short: "Fija la organización activa en este directorio",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := filepath.Join(d.Cwd, ".calliope")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			ruta := filepath.Join(dir, config.FileName)

			// Se conservan los valores que ya hubiera en el fichero.
			vals := map[string]string{}
			if b, err := os.ReadFile(ruta); err == nil {
				_ = json.Unmarshal(b, &vals)
			}
			vals[config.KeyOrg] = args[0]

			b, err := json.MarshalIndent(vals, "", "  ")
			if err != nil {
				return err
			}
			if err := os.WriteFile(ruta, b, 0o600); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Organización activa: %s (%s)\n", args[0], ruta)
			return nil
		},
	}
}
```

Añade los dos ayudantes al final de `internal/commands/auth.go`:

```go
// authResolve y clienteCon evitan repetir el mismo wiring en los comandos que
// necesitan cliente sin exigir organización.
func authResolve(d appctx.Deps) (auth.Credential, string, error) {
	return auth.Resolve(d.Env, d.Store)
}

func clienteCon(ctx *appctx.Context, cred auth.Credential) *sdk.Client {
	return sdk.New(sdk.Options{BaseURL: ctx.Cfg.BaseURL(), Credential: cred})
}
```

- [ ] **Step 5: Registrar los comandos en la raíz**

En `internal/cli/root.go`, cambia `NewRootCmd` para que acepte las dependencias y registre los grupos:

```go
// NewRootCmd construye el comando raíz con sus flags globales.
func NewRootCmd(d appctx.Deps) *cobra.Command {
	root := &cobra.Command{
		Use:           "calliope",
		Short:         "Interfaz de línea de comandos de Calliope Data",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	// La lista de flags vive en appctx, no aquí: así los tests de comandos
	// montan una raíz idéntica a la real sin duplicarla.
	appctx.RegisterGlobalFlags(root)

	root.AddCommand(
		newVersionCmd(),
		commands.NewAuthCmd(d),
		commands.NewOrgsCmd(d),
	)
	return root
}
```

Borra de `root.go` el bloque de `f := root.PersistentFlags()` de la Task 1: ahora
lo declara `appctx.RegisterGlobalFlags`.

Actualiza `cmd/calliope/main.go` a `cli.NewRootCmd(appctx.DefaultDeps())` y los tests de la Task 1 a `NewRootCmd(appctx.Deps{})`.

- [ ] **Step 6: Ejecutar los tests y verificar que pasan**

Run: `go test ./... -race`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/commands internal/cli cmd
git commit -m "feat: comandos auth y orgs"
```

---

### Task 13: Comando `config`

**Files:**
- Create: `internal/commands/config.go`
- Modify: `internal/cli/root.go` (registrar el grupo)
- Test: `internal/commands/config_test.go`

**Interfaces:**
- Consumes: `appctx.BuildSinCredencial`, `config.Config.All`, `config.GlobalPath`, `config.ProjectAllowed` (Tasks 4, 5, 11).
- Produces: `commands.NewConfigCmd(d appctx.Deps) *cobra.Command`.

`config list` es lo que responde a «¿por qué el CLI apunta a esta organización?»: muestra cada valor **con su procedencia**.

- [ ] **Step 1: Escribir los tests que fallan**

`internal/commands/config_test.go`:

```go
package commands

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigListMuestraLaProcedencia(t *testing.T) {
	d, stdout, _ := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {})

	root := raizDePrueba(NewConfigCmd(d), stdout)
	root.SetArgs([]string{"config", "list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("config list: %v", err)
	}

	var env struct {
		Data map[string]struct {
			Value  string `json:"value"`
			Source string `json:"source"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	// depsConServidor fija CALLIOPE_ORG, así que org viene del entorno.
	if env.Data["org"].Source != "env" {
		t.Errorf("origen de org = %q, se esperaba env", env.Data["org"].Source)
	}
	if env.Data["base_url"].Value == "" {
		t.Error("base_url debe tener valor, aunque sea el por defecto")
	}
}

func TestConfigSetRechazaClavesNoPermitidasEnProyecto(t *testing.T) {
	d, stdout, _ := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {})

	root := raizDePrueba(NewConfigCmd(d), stdout)
	root.SetArgs([]string{"config", "set", "base_url", "https://atacante.example"})
	err := root.Execute()
	if err == nil {
		t.Fatal("config set debe rechazar base_url en la configuración de proyecto")
	}
	if !strings.Contains(err.Error(), "--global") {
		t.Errorf("el error debe explicar que base_url solo se fija en global: %q", err.Error())
	}
}

func TestConfigSetEscribeUnaClavePermitida(t *testing.T) {
	d, stdout, _ := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {})

	root := raizDePrueba(NewConfigCmd(d), stdout)
	root.SetArgs([]string{"config", "set", "org", "globex"})
	if err := root.Execute(); err != nil {
		t.Fatalf("config set: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(d.Cwd, ".calliope", "config.json"))
	if err != nil {
		t.Fatalf("no se escribió el fichero: %v", err)
	}
	var vals map[string]string
	if err := json.Unmarshal(b, &vals); err != nil {
		t.Fatal(err)
	}
	if vals["org"] != "globex" {
		t.Errorf("config = %v", vals)
	}
}

func TestConfigGetDevuelveUnValor(t *testing.T) {
	d, stdout, _ := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {})

	root := raizDePrueba(NewConfigCmd(d), stdout)
	root.SetArgs([]string{"config", "get", "org", "--quiet"})
	if err := root.Execute(); err != nil {
		t.Fatalf("config get: %v", err)
	}
	if !strings.Contains(stdout.String(), "acme") {
		t.Errorf("salida = %q, se esperaba acme", stdout.String())
	}
}
```

- [ ] **Step 2: Ejecutar los tests y verificar que fallan**

Run: `go test ./internal/commands/ -run Config -v`
Expected: FAIL — `undefined: NewConfigCmd`

- [ ] **Step 3: Implementar el comando**

`internal/commands/config.go`:

```go
package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/config"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/presenter"
)

// NewConfigCmd construye el grupo `config`.
func NewConfigCmd(d appctx.Deps) *cobra.Command {
	grupo := &cobra.Command{
		Use:   "config",
		Short: "Consulta y modifica la configuración de calliope",
	}
	grupo.AddCommand(newConfigListCmd(d), newConfigGetCmd(d), newConfigSetCmd(d), newConfigPathCmd(d))
	return grupo
}

func newConfigListCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Muestra cada valor con la capa de la que proviene",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.BuildSinCredencial(cmd, d)
			if err != nil {
				return err
			}
			todos := ctx.Cfg.All()

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(todos, fmt.Sprintf("%d valores", len(todos)),
					output.Breadcrumb{Action: "cambiar", Cmd: "calliope config set <clave> <valor>"}),
				Text: func(w io.Writer) error {
					claves := make([]string, 0, len(todos))
					for k := range todos {
						claves = append(claves, k)
					}
					sort.Strings(claves)

					filas := make([][]string, 0, len(claves))
					for _, k := range claves {
						v := todos[k]
						filas = append(filas, []string{k, v.Value, string(v.Source), v.Path})
					}
					return presenter.Table(w, []string{"CLAVE", "VALOR", "ORIGEN", "FICHERO"}, filas)
				},
			})
		},
	}
}

func newConfigGetCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "get <clave>",
		Short: "Muestra el valor de una clave",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.BuildSinCredencial(cmd, d)
			if err != nil {
				return err
			}
			v := ctx.Cfg.Get(args[0])
			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(v.Value, fmt.Sprintf("%s (%s)", args[0], v.Source)),
				Text: func(w io.Writer) error {
					_, err := fmt.Fprintln(w, v.Value)
					return err
				},
			})
		},
	}
}

func newConfigSetCmd(d appctx.Deps) *cobra.Command {
	var global bool
	cmd := &cobra.Command{
		Use:   "set <clave> <valor>",
		Short: "Fija una clave en la configuración de proyecto (o global con --global)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clave, valor := args[0], args[1]

			ruta := filepath.Join(d.Cwd, ".calliope", config.FileName)
			if global {
				ruta = config.GlobalPath(d.Env)
			} else if !config.ProjectAllowed[clave] {
				// La misma regla que aplica al leer se aplica al escribir: si
				// no se pudiera leer, escribirlo solo confunde.
				return output.NewError(output.CodeUsage,
					fmt.Sprintf("La clave %q no se puede fijar en la configuración de proyecto.", clave),
					fmt.Sprintf("Fíjala en la global con: calliope config set %s %s --global", clave, valor))
			}

			if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
				return err
			}
			vals := map[string]string{}
			if b, err := os.ReadFile(ruta); err == nil {
				_ = json.Unmarshal(b, &vals)
			}
			vals[clave] = valor

			b, err := json.MarshalIndent(vals, "", "  ")
			if err != nil {
				return err
			}
			if err := os.WriteFile(ruta, b, 0o600); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s = %s (%s)\n", clave, valor, ruta)
			return nil
		},
	}
	cmd.Flags().BoolVar(&global, "global", false, "escribe en la configuración global en vez de la del proyecto")
	return cmd
}

func newConfigPathCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Imprime la ruta del fichero de configuración global",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), config.GlobalPath(d.Env))
			return nil
		},
	}
}
```

- [ ] **Step 4: Registrar y verificar**

Añade `commands.NewConfigCmd(d)` a `root.AddCommand(...)` en `internal/cli/root.go`.

Run: `go test ./... -race`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/commands internal/cli
git commit -m "feat: comando config con procedencia de cada valor"
```

---

### Task 14: Comando `ask`

**Files:**
- Create: `internal/commands/ask.go`
- Modify: `internal/cli/root.go`
- Test: `internal/commands/ask_test.go`

**Interfaces:**
- Consumes: `appctx.Build` (Task 11), `(*sdk.Client).Ask`, `sdk.AskResponse` (Task 10).
- Produces: `commands.NewAskCmd(d appctx.Deps) *cobra.Command`.

`ask` es el comando central: es la vía por defecto que el SKILL.md impone a los agentes frente a `query`.

- [ ] **Step 1: Escribir los tests que fallan**

`internal/commands/ask_test.go`:

```go
package commands

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/calliope/calliope-cli/internal/auth"
)

const respuestaAsk = `{
  "success": true,
  "text": "Las ventas crecieron un 12%.",
  "rowCount": 1,
  "data": [{"mes":"2026-01","ventas":1200}],
  "sources": [{"citation":"Informe anual, p. 12","chunkId":88,"documentId":"doc-1",
               "documentTitle":"Informe anual","filename":"informe.pdf","page":12,
               "headingPath":"Resultados","excerpt":"Las ventas…"}]
}`

func TestAskDevuelveTextoYFuentes(t *testing.T) {
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations/acme/ask" {
			t.Errorf("ruta = %q", r.URL.Path)
		}
		w.Write([]byte(respuestaAsk))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewAskCmd(d), stdout)
	root.SetArgs([]string{"ask", "¿cómo van las ventas?", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("ask: %v", err)
	}

	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			Text    string `json:"text"`
			Sources []struct {
				Citation string `json:"citation"`
			} `json:"sources"`
		} `json:"data"`
		Breadcrumbs []struct {
			Cmd string `json:"cmd"`
		} `json:"breadcrumbs"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if !env.OK || env.Data.Text == "" {
		t.Errorf("respuesta inesperada: %q", stdout.String())
	}
	if len(env.Data.Sources) != 1 {
		t.Errorf("se esperaba 1 fuente citada, hay %d", len(env.Data.Sources))
	}
	if len(env.Breadcrumbs) == 0 {
		t.Error("ask debe sugerir el siguiente comando")
	}
}

func TestAskEnMarkdownCitaLasFuentes(t *testing.T) {
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(respuestaAsk))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewAskCmd(d), stdout)
	root.SetArgs([]string{"ask", "¿ventas?", "--md"})
	if err := root.Execute(); err != nil {
		t.Fatalf("ask: %v", err)
	}
	salida := stdout.String()
	if !strings.Contains(salida, "Informe anual, p. 12") {
		t.Errorf("el markdown debe citar las fuentes, se obtuvo:\n%s", salida)
	}
}

func TestAskSinPreguntaEsErrorDeUso(t *testing.T) {
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewAskCmd(d), stdout)
	root.SetArgs([]string{"ask"})
	if err := root.Execute(); err == nil {
		t.Fatal("se esperaba error sin pregunta")
	}
}
```

- [ ] **Step 2: Ejecutar los tests y verificar que fallan**

Run: `go test ./internal/commands/ -run Ask -v`
Expected: FAIL — `undefined: NewAskCmd`

- [ ] **Step 3: Implementar el comando**

`internal/commands/ask.go`:

```go
package commands

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/presenter"
	"github.com/calliope/calliope-cli/internal/sdk"
)

// NewAskCmd construye `ask`, la vía por defecto para consultar los datos.
func NewAskCmd(d appctx.Deps) *cobra.Command {
	var accion string
	cmd := &cobra.Command{
		Use:   "ask <pregunta>",
		Short: "Pregunta en lenguaje natural sobre tus datos y tu documentación",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}

			resp, err := ctx.Client.Ask(cmd.Context(), ctx.Org, args[0], accion)
			if err != nil {
				return err
			}
			if !resp.Success {
				mensaje := "Calliope no pudo responder a la pregunta."
				if resp.Error != nil && *resp.Error != "" {
					mensaje = *resp.Error
				}
				return output.NewError(output.CodeGeneric, mensaje,
					"Reformula la pregunta, o mira qué datos existen con: calliope concepts list")
			}

			resumen := fmt.Sprintf("%d fuentes citadas", len(resp.Sources))
			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(resp, resumen,
					output.Breadcrumb{Action: "documento", Cmd: "calliope docs show <id>"},
					output.Breadcrumb{Action: "conceptos", Cmd: "calliope concepts list"}),
				Text:     func(w io.Writer) error { return escribirAskTexto(w, resp) },
				Markdown: func(w io.Writer) error { return escribirAskMarkdown(w, resp) },
			})
		},
	}
	cmd.Flags().StringVar(&accion, "action", "", "tipo de análisis a solicitar")
	return cmd
}

func escribirAskTexto(w io.Writer, r *sdk.AskResponse) error {
	if _, err := fmt.Fprintln(w, r.Text); err != nil {
		return err
	}
	if len(r.Sources) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, "\nFuentes:"); err != nil {
		return err
	}
	for _, s := range r.Sources {
		if _, err := fmt.Fprintf(w, "  · %s (%s)\n", s.Citation, s.DocumentID); err != nil {
			return err
		}
	}
	return nil
}

func escribirAskMarkdown(w io.Writer, r *sdk.AskResponse) error {
	if _, err := fmt.Fprintf(w, "%s\n", r.Text); err != nil {
		return err
	}
	if len(r.Sources) == 0 {
		return nil
	}
	// Las fuentes se citan siempre: es la invariante 5 del SKILL.md.
	if _, err := fmt.Fprintln(w, "\n### Fuentes\n"); err != nil {
		return err
	}
	for _, s := range r.Sources {
		titulo := s.Filename
		if s.DocumentTitle != nil && *s.DocumentTitle != "" {
			titulo = *s.DocumentTitle
		}
		if _, err := fmt.Fprintf(w, "- **%s** — %s (`%s`)\n", titulo, s.Citation, s.DocumentID); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 4: Registrar y verificar**

Añade `commands.NewAskCmd(d)` a `root.AddCommand(...)`.

Run: `go test ./internal/commands/ -race -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/commands internal/cli
git commit -m "feat: comando ask con citación de fuentes"
```

---

### Task 15: Comando `docs`

**Files:**
- Create: `internal/commands/docs.go`
- Modify: `internal/cli/root.go`
- Test: `internal/commands/docs_test.go`

**Interfaces:**
- Consumes: `appctx.Build`; `ListDocuments`, `GetDocument`, `SearchDocuments`, `ListDocumentsParams` (Task 10).
- Produces: `commands.NewDocsCmd(d appctx.Deps) *cobra.Command`.

- [ ] **Step 1: Escribir los tests que fallan**

`internal/commands/docs_test.go`:

```go
package commands

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/calliope/calliope-cli/internal/auth"
)

func TestDocsListPasaLosFiltros(t *testing.T) {
	var vistaQuery string
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		vistaQuery = r.URL.RawQuery
		w.Write([]byte(`{"content":[{"id":"doc-1","filename":"a.pdf","status":"READY","declaredMime":"application/pdf","sizeBytes":1,"createdAt":"","updatedAt":""}],"totalSize":1}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewDocsCmd(d), stdout)
	root.SetArgs([]string{"docs", "list", "--status", "READY", "--tag", "finanzas", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("docs list: %v", err)
	}
	if vistaQuery == "" {
		t.Error("los filtros deben viajar en la query string")
	}

	var env struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if len(env.Data) != 1 {
		t.Errorf("se esperaba 1 documento en data, hay %d", len(env.Data))
	}
}

func TestDocsShowLlamaAlEndpointDelDocumento(t *testing.T) {
	var ruta string
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		ruta = r.URL.Path
		w.Write([]byte(`{"id":"doc-1","filename":"a.pdf","status":"READY","declaredMime":"application/pdf","sizeBytes":1,"createdAt":"","updatedAt":""}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewDocsCmd(d), stdout)
	root.SetArgs([]string{"docs", "show", "doc-1", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("docs show: %v", err)
	}
	if ruta != "/v1/organizations/acme/documents/doc-1" {
		t.Errorf("ruta = %q", ruta)
	}
}

func TestDocsSearchUsaPOST(t *testing.T) {
	var metodo, ruta string
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		metodo, ruta = r.Method, r.URL.Path
		w.Write([]byte(`[{"chunkId":1,"documentId":"doc-1","filename":"a.pdf","ordinal":0,"excerpt":"…","score":0.9}]`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewDocsCmd(d), stdout)
	root.SetArgs([]string{"docs", "search", "ventas", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("docs search: %v", err)
	}
	if metodo != http.MethodPost || ruta != "/v1/organizations/acme/search/documents" {
		t.Errorf("método/ruta = %s %s", metodo, ruta)
	}
}
```

- [ ] **Step 2: Ejecutar los tests y verificar que fallan**

Run: `go test ./internal/commands/ -run Docs -v`
Expected: FAIL — `undefined: NewDocsCmd`

- [ ] **Step 3: Implementar el comando**

`internal/commands/docs.go`:

```go
package commands

import (
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/presenter"
	"github.com/calliope/calliope-cli/internal/sdk"
)

// NewDocsCmd construye el grupo `docs`.
func NewDocsCmd(d appctx.Deps) *cobra.Command {
	grupo := &cobra.Command{
		Use:   "docs",
		Short: "Consulta la documentación de la organización",
	}
	grupo.AddCommand(newDocsListCmd(d), newDocsShowCmd(d), newDocsSearchCmd(d))
	return grupo
}

func newDocsListCmd(d appctx.Deps) *cobra.Command {
	var p sdk.ListDocumentsParams
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lista los documentos disponibles",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}
			page, err := ctx.Client.ListDocuments(cmd.Context(), ctx.Org, p)
			if err != nil {
				return err
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(page.Content,
					fmt.Sprintf("%d de %d documentos", len(page.Content), page.TotalSize),
					output.Breadcrumb{Action: "detalle", Cmd: "calliope docs show <id>"},
					output.Breadcrumb{Action: "buscar", Cmd: "calliope docs search \"<consulta>\""}),
				Text: func(w io.Writer) error {
					filas := make([][]string, 0, len(page.Content))
					for _, doc := range page.Content {
						filas = append(filas, []string{doc.ID, tituloDe(doc), doc.Status, strconv.FormatInt(doc.SizeBytes, 10)})
					}
					return presenter.Table(w, []string{"ID", "TÍTULO", "ESTADO", "BYTES"}, filas)
				},
			})
		},
	}
	f := cmd.Flags()
	f.StringVar(&p.Status, "status", "", "filtra por estado (READY, PROCESSING, FAILED…)")
	f.StringVar(&p.Tag, "tag", "", "filtra por etiqueta")
	f.StringVar(&p.Q, "q", "", "filtra por texto")
	f.IntVar(&p.Page, "page", 0, "página (base 1)")
	f.IntVar(&p.Size, "size", 0, "tamaño de página")
	return cmd
}

func newDocsShowCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Muestra los metadatos de un documento",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}
			doc, err := ctx.Client.GetDocument(cmd.Context(), ctx.Org, args[0])
			if err != nil {
				return err
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(doc, tituloDe(*doc),
					output.Breadcrumb{Action: "buscar dentro", Cmd: "calliope docs search \"<consulta>\""}),
				Text: func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "%s\nFichero: %s\nEstado: %s\nCreado: %s\n",
						tituloDe(*doc), doc.Filename, doc.Status, doc.CreatedAt)
					return err
				},
			})
		},
	}
}

func newDocsSearchCmd(d appctx.Deps) *cobra.Command {
	var limite int
	cmd := &cobra.Command{
		Use:   "search <consulta>",
		Short: "Búsqueda semántica en la documentación",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}
			res, err := ctx.Client.SearchDocuments(cmd.Context(), ctx.Org, args[0], limite)
			if err != nil {
				return err
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(res, fmt.Sprintf("%d fragmentos", len(res)),
					output.Breadcrumb{Action: "documento", Cmd: "calliope docs show <id>"}),
				Text: func(w io.Writer) error {
					filas := make([][]string, 0, len(res))
					for _, r := range res {
						filas = append(filas, []string{
							r.DocumentID,
							strconv.FormatFloat(r.Score, 'f', 3, 64),
							recortar(r.Excerpt, 70),
						})
					}
					return presenter.Table(w, []string{"DOCUMENTO", "SCORE", "FRAGMENTO"}, filas)
				},
			})
		},
	}
	cmd.Flags().IntVar(&limite, "limit", 10, "número máximo de fragmentos")
	return cmd
}

func tituloDe(doc sdk.DocumentResponse) string {
	if doc.Title != nil && *doc.Title != "" {
		return *doc.Title
	}
	return doc.Filename
}

func recortar(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
```

- [ ] **Step 4: Registrar y verificar**

Añade `commands.NewDocsCmd(d)` a `root.AddCommand(...)`.

Run: `go test ./internal/commands/ -race`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/commands internal/cli
git commit -m "feat: comando docs (list, show, search)"
```

---

### Task 16: Comandos `concepts` y `rules`

**Files:**
- Create: `internal/commands/knowledge.go`
- Modify: `internal/cli/root.go`
- Test: `internal/commands/knowledge_test.go`

**Interfaces:**
- Consumes: `appctx.Build`; `ListConcepts`, `GetConcept`, `ListRules` (Task 10).
- Produces: `commands.NewConceptsCmd(d appctx.Deps) *cobra.Command`, `commands.NewRulesCmd(d appctx.Deps) *cobra.Command`.

Van juntos porque comparten el dominio del conocimiento y ninguno tiene entidad para una revisión propia.

- [ ] **Step 1: Escribir los tests que fallan**

`internal/commands/knowledge_test.go`:

```go
package commands

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/calliope/calliope-cli/internal/auth"
)

func TestConceptsListDevuelveElGrafo(t *testing.T) {
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations/acme/knowledge/concepts" {
			t.Errorf("ruta = %q", r.URL.Path)
		}
		w.Write([]byte(`{"concepts":[{"id":"c-1","name":"Cliente","isActive":true,"recordCount":42}]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewConceptsCmd(d), stdout)
	root.SetArgs([]string{"concepts", "list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("concepts list: %v", err)
	}

	var env struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
		Breadcrumbs []struct {
			Cmd string `json:"cmd"`
		} `json:"breadcrumbs"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if len(env.Data) != 1 || env.Data[0].Name != "Cliente" {
		t.Errorf("data inesperada: %q", stdout.String())
	}
	if len(env.Breadcrumbs) == 0 {
		t.Error("concepts list debe sugerir cómo ver el detalle")
	}
}

func TestConceptsShowDevuelveAtributos(t *testing.T) {
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations/acme/knowledge/concepts/c-1" {
			t.Errorf("ruta = %q", r.URL.Path)
		}
		w.Write([]byte(`{"concept":{"id":"c-1","name":"Cliente","isActive":true},
		                "attributes":[{"id":"a-1","name":"email","isActive":true}]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewConceptsCmd(d), stdout)
	root.SetArgs([]string{"concepts", "show", "c-1", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("concepts show: %v", err)
	}

	var env struct {
		Data struct {
			Attributes []struct {
				Name string `json:"name"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if len(env.Data.Attributes) != 1 || env.Data.Attributes[0].Name != "email" {
		t.Errorf("atributos inesperados: %q", stdout.String())
	}
}

func TestRulesListDevuelveLasReglas(t *testing.T) {
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations/acme/rules" {
			t.Errorf("ruta = %q", r.URL.Path)
		}
		w.Write([]byte(`[{"id":"r-1","name":"Cliente activo","details":"Compró en 90 días","status":"ACTIVE"}]`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewRulesCmd(d), stdout)
	root.SetArgs([]string{"rules", "list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("rules list: %v", err)
	}

	var env struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if len(env.Data) != 1 || env.Data[0].Name != "Cliente activo" {
		t.Errorf("data inesperada: %q", stdout.String())
	}
}

func TestConceptsYRulesSonGruposSinRunE(t *testing.T) {
	if c := NewConceptsCmd(d0()); c.RunE != nil || c.Run != nil {
		t.Error("concepts es un grupo: no debe definir RunE")
	}
	if c := NewRulesCmd(d0()); c.RunE != nil || c.Run != nil {
		t.Error("rules es un grupo: no debe definir RunE")
	}
}
```

Añade el ayudante al final del fichero:

```go
func d0() appctx.Deps { return appctx.Deps{} }
```

y el import `"github.com/calliope/calliope-cli/internal/appctx"`.

- [ ] **Step 2: Ejecutar los tests y verificar que fallan**

Run: `go test ./internal/commands/ -run "Concepts|Rules" -v`
Expected: FAIL — `undefined: NewConceptsCmd`

- [ ] **Step 3: Implementar los comandos**

`internal/commands/knowledge.go`:

```go
package commands

import (
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/presenter"
)

// NewConceptsCmd construye el grupo `concepts`: qué datos existen, en lenguaje
// de negocio. Es lo primero que debe mirar un agente antes de preguntar.
func NewConceptsCmd(d appctx.Deps) *cobra.Command {
	grupo := &cobra.Command{
		Use:   "concepts",
		Short: "Explora los conceptos de negocio de la ontología",
	}
	grupo.AddCommand(newConceptsListCmd(d), newConceptsShowCmd(d))
	return grupo
}

func newConceptsListCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lista los conceptos de negocio",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}
			grafo, err := ctx.Client.ListConcepts(cmd.Context(), ctx.Org)
			if err != nil {
				return err
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(grafo.Concepts,
					fmt.Sprintf("%d conceptos", len(grafo.Concepts)),
					output.Breadcrumb{Action: "detalle", Cmd: "calliope concepts show <id>"},
					output.Breadcrumb{Action: "preguntar", Cmd: "calliope ask \"<pregunta>\""}),
				Text: func(w io.Writer) error {
					filas := make([][]string, 0, len(grafo.Concepts))
					for _, c := range grafo.Concepts {
						registros := "—"
						if c.RecordCount != nil {
							registros = strconv.Itoa(*c.RecordCount)
						}
						filas = append(filas, []string{c.ID, c.Name, registros, activo(c.IsActive)})
					}
					return presenter.Table(w, []string{"ID", "CONCEPTO", "REGISTROS", "ACTIVO"}, filas)
				},
			})
		},
	}
}

func newConceptsShowCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Muestra un concepto y sus atributos",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}
			det, err := ctx.Client.GetConcept(cmd.Context(), ctx.Org, args[0])
			if err != nil {
				return err
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(det,
					fmt.Sprintf("%s · %d atributos", det.Concept.Name, len(det.Attributes)),
					output.Breadcrumb{Action: "preguntar", Cmd: "calliope ask \"<pregunta sobre " + det.Concept.Name + ">\""}),
				Text: func(w io.Writer) error {
					if _, err := fmt.Fprintf(w, "%s\n", det.Concept.Name); err != nil {
						return err
					}
					if det.Concept.Description != nil && *det.Concept.Description != "" {
						if _, err := fmt.Fprintf(w, "%s\n", *det.Concept.Description); err != nil {
							return err
						}
					}
					fmt.Fprintln(w)
					filas := make([][]string, 0, len(det.Attributes))
					for _, a := range det.Attributes {
						desc := ""
						if a.Description != nil {
							desc = *a.Description
						}
						filas = append(filas, []string{a.Name, desc, activo(a.IsActive)})
					}
					return presenter.Table(w, []string{"ATRIBUTO", "DESCRIPCIÓN", "ACTIVO"}, filas)
				},
			})
		},
	}
}

// NewRulesCmd construye el grupo `rules`.
func NewRulesCmd(d appctx.Deps) *cobra.Command {
	grupo := &cobra.Command{
		Use:   "rules",
		Short: "Consulta las reglas de negocio compartidas",
	}
	grupo.AddCommand(newRulesListCmd(d))
	return grupo
}

func newRulesListCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lista las reglas de negocio de la organización",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}
			reglas, err := ctx.Client.ListRules(cmd.Context(), ctx.Org)
			if err != nil {
				return err
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(reglas, fmt.Sprintf("%d reglas", len(reglas)),
					output.Breadcrumb{Action: "conceptos", Cmd: "calliope concepts list"}),
				Text: func(w io.Writer) error {
					filas := make([][]string, 0, len(reglas))
					for _, r := range reglas {
						cat := ""
						if r.Category != nil {
							cat = *r.Category
						}
						filas = append(filas, []string{r.Name, cat, r.Status, recortar(r.Details, 60)})
					}
					return presenter.Table(w, []string{"REGLA", "CATEGORÍA", "ESTADO", "DETALLE"}, filas)
				},
			})
		},
	}
}

func activo(b bool) string {
	if b {
		return "sí"
	}
	return "no"
}
```

- [ ] **Step 4: Registrar y verificar**

Añade `commands.NewConceptsCmd(d)` y `commands.NewRulesCmd(d)` a `root.AddCommand(...)`.

Run: `go test ./internal/commands/ -race`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/commands internal/cli
git commit -m "feat: comandos concepts y rules"
```

---

### Task 17: Comandos `schema` y `query`

**Files:**
- Create: `internal/commands/data.go`
- Modify: `internal/cli/root.go`
- Test: `internal/commands/data_test.go`

**Interfaces:**
- Consumes: `appctx.Build`; `Schema`, `Query`, `(*QueryResponse).Rows` (Task 10).
- Produces: `commands.NewSchemaCmd(d appctx.Deps) *cobra.Command`, `commands.NewQueryCmd(d appctx.Deps) *cobra.Command`.

Ambos son atajos, no grupos: **sí** definen `RunE`. `--table` de `schema` filtra en cliente; `--output` de `query` se reenvía al backend y `--csv` es render local.

- [ ] **Step 1: Escribir los tests que fallan**

`internal/commands/data_test.go`:

```go
package commands

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/calliope/calliope-cli/internal/auth"
)

const respuestaSchema = `{"tables":[
  {"name":"ventas","columns":[{"name":"mes","type":"DATE"},{"name":"importe","type":"DECIMAL"}]},
  {"name":"clientes","columns":[{"name":"id","type":"VARCHAR"}]}
]}`

func TestSchemaDevuelveTodasLasTablas(t *testing.T) {
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations/acme/database/schema" {
			t.Errorf("ruta = %q", r.URL.Path)
		}
		w.Write([]byte(respuestaSchema))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewSchemaCmd(d), stdout)
	root.SetArgs([]string{"schema", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("schema: %v", err)
	}

	var env struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if len(env.Data) != 2 {
		t.Errorf("se esperaban 2 tablas, hay %d", len(env.Data))
	}
}

func TestSchemaTableFiltraEnCliente(t *testing.T) {
	llamadas := 0
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		llamadas++
		if r.URL.RawQuery != "" {
			t.Errorf("el filtro es en cliente: no debe enviarse query, se envió %q", r.URL.RawQuery)
		}
		w.Write([]byte(respuestaSchema))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewSchemaCmd(d), stdout)
	root.SetArgs([]string{"schema", "--table", "ventas", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("schema --table: %v", err)
	}

	var env struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if len(env.Data) != 1 || env.Data[0].Name != "ventas" {
		t.Errorf("el filtro no se aplicó: %q", stdout.String())
	}
}

func TestSchemaTableInexistenteEsNotFound(t *testing.T) {
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(respuestaSchema))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewSchemaCmd(d), stdout)
	root.SetArgs([]string{"schema", "--table", "no-existe"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error con una tabla inexistente")
	}
	if !strings.Contains(err.Error(), "no-existe") {
		t.Errorf("el error debe nombrar la tabla: %q", err.Error())
	}
}

func TestQueryEnviaElSQLYDevuelveFilas(t *testing.T) {
	var cuerpo map[string]string
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&cuerpo)
		w.Write([]byte(`{"data":[{"mes":"2026-01","ventas":1200}]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT * FROM ventas", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("query: %v", err)
	}
	if cuerpo["sql"] != "SELECT * FROM ventas" {
		t.Errorf("sql enviado = %q", cuerpo["sql"])
	}

	var env struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if len(env.Data) != 1 {
		t.Errorf("se esperaba 1 fila, hay %d", len(env.Data))
	}
}

func TestQueryOutputSeReenviaAlBackend(t *testing.T) {
	var cuerpo map[string]string
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&cuerpo)
		w.Write([]byte(`{"data":[]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1", "--output", "arrow", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("query: %v", err)
	}
	if cuerpo["output"] != "arrow" {
		t.Errorf("output = %q, --output se reenvía al backend", cuerpo["output"])
	}
}

func TestQueryCSVEsRenderLocal(t *testing.T) {
	var cuerpo map[string]string
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&cuerpo)
		w.Write([]byte(`{"data":[{"mes":"2026-01","ventas":1200}]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1", "--csv"})
	if err := root.Execute(); err != nil {
		t.Fatalf("query --csv: %v", err)
	}
	if _, existe := cuerpo["output"]; existe {
		t.Error("--csv es render local: no debe enviarse output al backend")
	}
	salida := stdout.String()
	if !strings.Contains(salida, "mes,ventas") && !strings.Contains(salida, "ventas,mes") {
		t.Errorf("se esperaba cabecera CSV, se obtuvo:\n%s", salida)
	}
}
```

- [ ] **Step 2: Ejecutar los tests y verificar que fallan**

Run: `go test ./internal/commands/ -run "Schema|Query" -v`
Expected: FAIL — `undefined: NewSchemaCmd`

- [ ] **Step 3: Implementar los comandos**

`internal/commands/data.go`:

```go
package commands

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/presenter"
	"github.com/calliope/calliope-cli/internal/sdk"
)

// NewSchemaCmd construye `schema`. Es un atajo, no un grupo, así que define
// RunE. El SKILL.md obliga a los agentes a ejecutarlo antes de escribir SQL.
func NewSchemaCmd(d appctx.Deps) *cobra.Command {
	var tabla string
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Muestra el esquema de la base de datos de la organización",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}
			esquema, err := ctx.Client.Schema(cmd.Context(), ctx.Org)
			if err != nil {
				return err
			}

			tablas := esquema.Tables
			if tabla != "" {
				// El filtro es en cliente: el backend no expone filtrado por
				// tabla. Si el esquema crece mucho, ese endpoint sería el
				// siguiente paso (ver la sección 15 del spec).
				tablas = filtrarTablas(tablas, tabla)
				if len(tablas) == 0 {
					return output.NewError(output.CodeNotFound,
						fmt.Sprintf("No existe la tabla %q en el esquema.", tabla),
						"Lista todas las tablas con: calliope schema")
				}
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(tablas, fmt.Sprintf("%d tablas", len(tablas)),
					output.Breadcrumb{Action: "consultar", Cmd: "calliope query \"SELECT …\""},
					output.Breadcrumb{Action: "preguntar", Cmd: "calliope ask \"<pregunta>\""}),
				Text: func(w io.Writer) error {
					for _, t := range tablas {
						if _, err := fmt.Fprintf(w, "\n%s\n", t.Name); err != nil {
							return err
						}
						filas := make([][]string, 0, len(t.Columns))
						for _, c := range t.Columns {
							filas = append(filas, []string{c.Name, c.Type, c.Description})
						}
						if err := presenter.Table(w, []string{"COLUMNA", "TIPO", "DESCRIPCIÓN"}, filas); err != nil {
							return err
						}
					}
					return nil
				},
			})
		},
	}
	cmd.Flags().StringVar(&tabla, "table", "", "muestra solo esta tabla")
	return cmd
}

func filtrarTablas(tablas []sdk.SchemaTable, nombre string) []sdk.SchemaTable {
	var out []sdk.SchemaTable
	for _, t := range tablas {
		if strings.EqualFold(t.Name, nombre) {
			out = append(out, t)
		}
	}
	return out
}

// NewQueryCmd construye `query`, el SQL crudo. El SKILL.md establece `ask`
// como vía por defecto; esto es el escape para cuando `ask` no basta.
func NewQueryCmd(d appctx.Deps) *cobra.Command {
	var formato string
	var comoCSV bool

	cmd := &cobra.Command{
		Use:   "query <SQL>",
		Short: "Ejecuta SQL contra los datos de la organización",
		Long: "Ejecuta SQL contra los datos de la organización.\n\n" +
			"Ejecuta antes `calliope schema` para conocer las tablas reales.\n" +
			"Para preguntas de negocio, prefiere `calliope ask`.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.Build(cmd, d)
			if err != nil {
				return err
			}

			resp, err := ctx.Client.Query(cmd.Context(), ctx.Org, args[0], formato)
			if err != nil {
				return err
			}
			filas, err := resp.Rows()
			if err != nil {
				return output.NewError(output.CodeGeneric,
					"No se pudo interpretar el resultado de la consulta.",
					"Prueba con --json para ver la respuesta cruda del backend")
			}

			columnas := columnasDe(filas)

			if comoCSV {
				// --csv es render local y no toca el cuerpo de la petición.
				return escribirCSV(ctx.Deps.Stdout, columnas, filas)
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(filas, fmt.Sprintf("%d filas", len(filas)),
					output.Breadcrumb{Action: "esquema", Cmd: "calliope schema"}),
				Text: func(w io.Writer) error {
					tabla := make([][]string, 0, len(filas))
					for _, f := range filas {
						fila := make([]string, 0, len(columnas))
						for _, c := range columnas {
							fila = append(fila, fmt.Sprintf("%v", f[c]))
						}
						tabla = append(tabla, fila)
					}
					return presenter.Table(w, columnas, tabla)
				},
			})
		},
	}
	cmd.Flags().StringVar(&formato, "output", "", "formato que se pide al backend (se reenvía en QueryRequest.output)")
	cmd.Flags().BoolVar(&comoCSV, "csv", false, "render local del resultado en CSV")
	return cmd
}

// columnasDe deduce las columnas de las filas, en orden estable.
func columnasDe(filas []map[string]any) []string {
	vistas := map[string]bool{}
	var cols []string
	for _, f := range filas {
		for k := range f {
			if !vistas[k] {
				vistas[k] = true
				cols = append(cols, k)
			}
		}
	}
	sort.Strings(cols)
	return cols
}

func escribirCSV(w io.Writer, columnas []string, filas []map[string]any) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(columnas); err != nil {
		return err
	}
	for _, f := range filas {
		registro := make([]string, 0, len(columnas))
		for _, c := range columnas {
			registro = append(registro, fmt.Sprintf("%v", f[c]))
		}
		if err := cw.Write(registro); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
```

- [ ] **Step 4: Registrar y verificar**

Añade `commands.NewSchemaCmd(d)` y `commands.NewQueryCmd(d)` a `root.AddCommand(...)`.

Run: `go test ./internal/commands/ -race -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/commands internal/cli
git commit -m "feat: comandos schema y query"
```

---

### Task 18: Comando `doctor`

**Files:**
- Create: `internal/commands/doctor.go`
- Modify: `internal/cli/root.go`
- Test: `internal/commands/doctor_test.go`

**Interfaces:**
- Consumes: `appctx.BuildSinCredencial`, `auth.Resolve`, `(*sdk.Client).Me` (Tasks 7, 10, 11).
- Produces: `commands.NewDoctorCmd(d appctx.Deps) *cobra.Command`; `commands.Chequeo{Nombre, Estado, Detalle string}` con `Estado` en `ok`, `aviso`, `error`.

`doctor` tiene que funcionar **precisamente cuando la autenticación está rota**, así que nunca usa `appctx.Build`. Nunca devuelve error: informa. Lo consumen el hook de sesión del plugin, el soporte a clientes y el agente cuando algo falla.

- [ ] **Step 1: Escribir los tests que fallan**

`internal/commands/doctor_test.go`:

```go
package commands

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/calliope/calliope-cli/internal/auth"
)

func chequeosDe(t *testing.T, salida []byte) map[string]string {
	t.Helper()
	var env struct {
		Data []struct {
			Nombre string `json:"nombre"`
			Estado string `json:"estado"`
		} `json:"data"`
	}
	if err := json.Unmarshal(salida, &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, salida)
	}
	m := map[string]string{}
	for _, c := range env.Data {
		m[c.Nombre] = c.Estado
	}
	return m
}

func TestDoctorTodoCorrecto(t *testing.T) {
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"userId":"u-1","email":"a@b.c"}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewDoctorCmd(d), stdout)
	root.SetArgs([]string{"doctor", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doctor: %v", err)
	}

	c := chequeosDe(t, stdout.Bytes())
	for _, nombre := range []string{"credencial", "organización", "conectividad"} {
		if c[nombre] != "ok" {
			t.Errorf("chequeo %q = %q, se esperaba ok", nombre, c[nombre])
		}
	}
}

func TestDoctorSinCredencialInformaEnVezDeFallar(t *testing.T) {
	d, stdout, _ := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {})
	d.Store = auth.NewFileStore(filepath.Join(t.TempDir(), "vacio.json"))

	root := raizDePrueba(NewDoctorCmd(d), stdout)
	root.SetArgs([]string{"doctor", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doctor nunca debe fallar, debe informar: %v", err)
	}

	c := chequeosDe(t, stdout.Bytes())
	if c["credencial"] != "error" {
		t.Errorf("chequeo credencial = %q, se esperaba error", c["credencial"])
	}
}

func TestDoctorConBackendCaidoLoDetecta(t *testing.T) {
	d, stdout, st := depsConServidor(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := raizDePrueba(NewDoctorCmd(d), stdout)
	root.SetArgs([]string{"doctor", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doctor: %v", err)
	}

	c := chequeosDe(t, stdout.Bytes())
	if c["conectividad"] != "error" {
		t.Errorf("chequeo conectividad = %q, se esperaba error", c["conectividad"])
	}
}
```

- [ ] **Step 2: Ejecutar los tests y verificar que fallan**

Run: `go test ./internal/commands/ -run Doctor -v`
Expected: FAIL — `undefined: NewDoctorCmd`

- [ ] **Step 3: Implementar el comando**

`internal/commands/doctor.go`:

```go
package commands

import (
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/presenter"
	"github.com/calliope/calliope-cli/internal/sdk"
	"github.com/calliope/calliope-cli/internal/version"
)

// Chequeo es el resultado de una comprobación de doctor.
type Chequeo struct {
	Nombre  string `json:"nombre"`
	Estado  string `json:"estado"` // ok | aviso | error
	Detalle string `json:"detalle"`
}

// NewDoctorCmd construye `doctor`. Nunca devuelve error: informa. Tiene que
// funcionar precisamente cuando la autenticación está rota, así que no usa
// appctx.Build.
func NewDoctorCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnostica la instalación, la credencial y la conectividad",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := appctx.BuildSinCredencial(cmd, d)
			if err != nil {
				return err
			}

			chequeos := []Chequeo{{
				Nombre:  "versión",
				Estado:  "ok",
				Detalle: fmt.Sprintf("calliope %s (%s)", version.Version, version.Commit),
			}, {
				Nombre:  "backend",
				Estado:  "ok",
				Detalle: ctx.Cfg.BaseURL() + " (" + string(ctx.Cfg.Get("base_url").Source) + ")",
			}}

			cred, origen, errCred := auth.Resolve(d.Env, d.Store)
			if errCred != nil {
				chequeos = append(chequeos, Chequeo{
					Nombre:  "credencial",
					Estado:  "error",
					Detalle: "no hay ninguna configurada; ejecuta: calliope auth login --api-key <clave>",
				})
			} else {
				chequeos = append(chequeos, Chequeo{
					Nombre:  "credencial",
					Estado:  "ok",
					Detalle: fmt.Sprintf("%s desde %s", cred.Kind, origen),
				})
			}

			org := ctx.Org
			if org == "" {
				org = cred.Org
			}
			if org == "" {
				chequeos = append(chequeos, Chequeo{
					Nombre:  "organización",
					Estado:  "error",
					Detalle: "ninguna seleccionada; ejecuta: calliope orgs use <nombre>",
				})
			} else {
				chequeos = append(chequeos, Chequeo{
					Nombre:  "organización",
					Estado:  "ok",
					Detalle: org + " (" + string(ctx.Cfg.Get("org").Source) + ")",
				})
			}

			if errCred == nil {
				chequeos = append(chequeos, chequearConectividad(cmd, ctx, cred))
			}

			return ctx.Render(presenter.Result{
				Envelope: output.OKEnvelope(chequeos, resumenDe(chequeos)),
				Text: func(w io.Writer) error {
					filas := make([][]string, 0, len(chequeos))
					for _, c := range chequeos {
						filas = append(filas, []string{simbolo(c.Estado), c.Nombre, c.Detalle})
					}
					return presenter.Table(w, []string{"", "COMPROBACIÓN", "DETALLE"}, filas)
				},
			})
		},
	}
}

func chequearConectividad(cmd *cobra.Command, ctx *appctx.Context, cred auth.Credential) Chequeo {
	cliente := sdk.New(sdk.Options{
		BaseURL:    ctx.Cfg.BaseURL(),
		Credential: cred,
		Timeout:    10 * time.Second,
		UserAgent:  "calliope-cli/" + version.Version,
	})

	inicio := time.Now()
	me, err := cliente.Me(cmd.Context())
	transcurrido := time.Since(inicio).Round(time.Millisecond)

	if err != nil {
		return Chequeo{Nombre: "conectividad", Estado: "error", Detalle: err.Error()}
	}
	return Chequeo{
		Nombre:  "conectividad",
		Estado:  "ok",
		Detalle: fmt.Sprintf("%s en %s", me.Email, transcurrido),
	}
}

func resumenDe(cs []Chequeo) string {
	fallos := 0
	for _, c := range cs {
		if c.Estado == "error" {
			fallos++
		}
	}
	if fallos == 0 {
		return "todo correcto"
	}
	return fmt.Sprintf("%d de %d comprobaciones fallan", fallos, len(cs))
}

func simbolo(estado string) string {
	switch estado {
	case "ok":
		return "✓"
	case "aviso":
		return "!"
	default:
		return "✗"
	}
}
```

- [ ] **Step 4: Registrar y verificar**

Añade `commands.NewDoctorCmd(d)` a `root.AddCommand(...)`.

Run: `go test ./... -race`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/commands internal/cli
git commit -m "feat: comando doctor"
```

---

### Task 19: Skill embebido, catálogo y test de paridad

**Files:**
- Create: `skills/calliope/SKILL.md`, `skills/embed.go`, `internal/cli/catalog.go`, `internal/commands/skill.go`
- Modify: `internal/cli/root.go`
- Test: `internal/cli/catalog_test.go`, `internal/cli/paridad_test.go`

**Interfaces:**
- Consumes: el árbol de comandos completo (Tasks 12-18).
- Produces:
  - `skills.SkillMD` (string embebido con `go:embed`)
  - `cli.CommandInfo{Path, Short string}`
  - `cli.Catalog(root *cobra.Command) []CommandInfo` — hojas del árbol, en orden alfabético
  - `commands.NewSkillCmd() *cobra.Command`

**Por qué esta tarea es la clave del proyecto.** El skill embebido no puede desincronizarse de la versión instalada: si el agente tiene el binario, tiene su documentación exacta. El test de paridad es lo que hace que esa promesa sea cierta — sin él, el SKILL.md miente en cuanto alguien añade un comando.

- [ ] **Step 1: Escribir el SKILL.md**

`skills/calliope/SKILL.md`:

```markdown
---
name: calliope
description: Consulta los datos, la documentación, la ontología y las reglas de negocio de una organización en Calliope Data mediante el CLI `calliope`. Úsalo para CUALQUIER pregunta o acción sobre Calliope Data.
---

# Calliope Data

`calliope` da acceso gobernado a los datos de una organización: preguntas en
lenguaje natural, documentación, ontología, reglas de negocio y SQL.

Comprueba que está listo con `calliope doctor`. Si falta el binario o la
credencial, ese comando dice exactamente qué ejecutar.

## Invariantes — CUMPLE estas reglas

1. **Elige el modo de salida.** `--jq '<expr>'` para extraer un campo, `--json`
   para el envelope completo, `--md` para presentar a una persona, `--quiet`
   para scripting. **Nunca** hagas pipe a un `jq` externo: el filtro va dentro
   del binario y el externo no existe en muchas máquinas.
2. **`ask` antes que `query`.** La pregunta en lenguaje natural es la vía por
   defecto. Recurre al SQL crudo solo cuando `ask` no baste o necesites una
   columna concreta.
3. **Ejecuta `calliope schema` antes de escribir SQL.** Nunca inventes nombres
   de tabla ni de columna.
4. **El scope de organización es obligatorio.** Usa `--org <nombre>` o fija una
   con `calliope orgs use <nombre>`. Si no sabes cuál, `calliope orgs list`.
5. **Cita las fuentes.** `ask` devuelve `sources`; cuando presentes el
   resultado a una persona, inclúyelas.
6. **Sigue los `breadcrumbs`.** Cada respuesta trae el siguiente comando en el
   campo `breadcrumbs`. Úsalo en vez de adivinar.

## Comandos

<!-- catalogo:inicio -->
- `calliope ask <pregunta>` — pregunta en lenguaje natural sobre datos y documentación
- `calliope auth login` — guarda y verifica una credencial
- `calliope auth logout` — borra la credencial almacenada
- `calliope auth status` — muestra quién eres y de dónde sale la credencial
- `calliope auth token` — imprime la credencial almacenada
- `calliope concepts list` — lista los conceptos de negocio
- `calliope concepts show <id>` — muestra un concepto y sus atributos
- `calliope config get <clave>` — muestra el valor de una clave
- `calliope config list` — muestra cada valor con su procedencia
- `calliope config path` — imprime la ruta de la configuración global
- `calliope config set <clave> <valor>` — fija una clave
- `calliope doctor` — diagnostica instalación, credencial y conectividad
- `calliope docs list` — lista los documentos disponibles
- `calliope docs search <consulta>` — búsqueda semántica en la documentación
- `calliope docs show <id>` — metadatos de un documento
- `calliope orgs list` — lista las organizaciones accesibles
- `calliope orgs use <organización>` — fija la organización activa
- `calliope query <SQL>` — ejecuta SQL contra los datos
- `calliope rules list` — lista las reglas de negocio
- `calliope schema` — esquema de la base de datos
- `calliope skill` — vuelca este documento
- `calliope version` — versión del binario
<!-- catalogo:fin -->

## Recetas

**Responder una pregunta de negocio y citar las fuentes:**

    calliope ask "¿cómo evolucionaron las ventas este trimestre?" --md

**Saber qué datos existen antes de preguntar:**

    calliope concepts list --jq '.data[].name'

**SQL, siempre después de mirar el esquema:**

    calliope schema --table ventas
    calliope query "SELECT mes, SUM(importe) FROM ventas GROUP BY mes" --json

**Extraer un solo campo:**

    calliope docs list --status READY --jq '.data[].id'

## Errores

Los fallos salen con esta forma, y `hint` dice qué hacer:

    {"ok": false, "error": {"code": "UNAUTHORIZED", "message": "…", "hint": "Ejecuta: calliope auth login"}}

Códigos de salida: `0` correcto · `1` error · `2` uso incorrecto ·
`3` no autorizado · `4` no encontrado · `5` límite superado.
```

- [ ] **Step 2: Escribir los tests que fallan**

`internal/cli/catalog_test.go`:

```go
package cli

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
)

func TestCatalogDevuelveSoloLasHojas(t *testing.T) {
	root := &cobra.Command{Use: "calliope"}
	grupo := &cobra.Command{Use: "docs"}
	grupo.AddCommand(&cobra.Command{Use: "list", Short: "lista", RunE: noop})
	grupo.AddCommand(&cobra.Command{Use: "show <id>", Short: "muestra", RunE: noop})
	root.AddCommand(grupo)
	root.AddCommand(&cobra.Command{Use: "ask <pregunta>", Short: "pregunta", RunE: noop})

	cat := Catalog(root)
	quiero := []string{"ask", "docs list", "docs show"}
	if len(cat) != len(quiero) {
		t.Fatalf("Catalog devolvió %d entradas, se esperaban %d: %+v", len(cat), len(quiero), cat)
	}
	for i, q := range quiero {
		if cat[i].Path != q {
			t.Errorf("cat[%d].Path = %q, se esperaba %q", i, cat[i].Path, q)
		}
	}
}

func TestNingunGrupoDeRecursosDefineRunE(t *testing.T) {
	root := NewRootCmd(appctx.Deps{})

	for _, c := range root.Commands() {
		if len(c.Commands()) == 0 {
			continue // es un atajo, no un grupo
		}
		if c.RunE != nil || c.Run != nil {
			t.Errorf("%q tiene subcomandos y además define RunE; invocarlo pelado debe mostrar la ayuda", c.Name())
		}
	}
}

func noop(cmd *cobra.Command, args []string) error { return nil }
```

`internal/cli/paridad_test.go`:

```go
package cli

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/skills"
)

// El SKILL.md va embebido en el binario, así que un agente que tenga el
// binario tiene su documentación exacta. Este test es lo que hace que esa
// promesa sea cierta: sin él, el skill miente en cuanto alguien añade,
// renombra o borra un comando.
func TestParidadEntreElCatalogoYElSkill(t *testing.T) {
	enElCLI := map[string]bool{}
	for _, c := range Catalog(NewRootCmd(appctx.Deps{})) {
		enElCLI[c.Path] = true
	}

	enElSkill := comandosDocumentados(t, skills.SkillMD)

	var faltan, sobran []string
	for cmd := range enElCLI {
		if !enElSkill[cmd] {
			faltan = append(faltan, cmd)
		}
	}
	for cmd := range enElSkill {
		if !enElCLI[cmd] {
			sobran = append(sobran, cmd)
		}
	}
	sort.Strings(faltan)
	sort.Strings(sobran)

	if len(faltan) > 0 {
		t.Errorf("comandos del CLI sin documentar en SKILL.md: %v\n"+
			"Añádelos entre los marcadores <!-- catalogo:inicio --> y <!-- catalogo:fin -->.", faltan)
	}
	if len(sobran) > 0 {
		t.Errorf("comandos documentados en SKILL.md que ya no existen: %v\n"+
			"El skill estaría mintiendo a los agentes; quítalos.", sobran)
	}
}

var lineaDeComando = regexp.MustCompile("^- `calliope ([^`]+)`")

// comandosDocumentados extrae los comandos del bloque delimitado por los
// marcadores de catálogo, quedándose con el nombre sin argumentos.
func comandosDocumentados(t *testing.T, md string) map[string]bool {
	t.Helper()

	inicio := strings.Index(md, "<!-- catalogo:inicio -->")
	fin := strings.Index(md, "<!-- catalogo:fin -->")
	if inicio < 0 || fin < 0 || fin < inicio {
		t.Fatal("SKILL.md debe contener los marcadores <!-- catalogo:inicio --> y <!-- catalogo:fin -->")
	}

	out := map[string]bool{}
	for _, linea := range strings.Split(md[inicio:fin], "\n") {
		m := lineaDeComando.FindStringSubmatch(strings.TrimSpace(linea))
		if m == nil {
			continue
		}
		out[soloNombre(m[1])] = true
	}
	return out
}

// soloNombre recorta los argumentos: "docs show <id>" → "docs show".
func soloNombre(s string) string {
	var partes []string
	for _, p := range strings.Fields(s) {
		if strings.HasPrefix(p, "<") || strings.HasPrefix(p, "[") || strings.HasPrefix(p, "--") {
			break
		}
		partes = append(partes, p)
	}
	return strings.Join(partes, " ")
}
```

- [ ] **Step 3: Ejecutar los tests y verificar que fallan**

Run: `go test ./internal/cli/ -v`
Expected: FAIL — `undefined: Catalog`, `no required module provides package .../skills`

- [ ] **Step 4: Implementar el embebido y el catálogo**

`skills/embed.go`:

```go
// Package skills embebe la documentación que el CLI publica para agentes.
// Va dentro del binario a propósito: así el skill no puede desincronizarse de
// la versión instalada.
package skills

import _ "embed"

//go:embed calliope/SKILL.md
var SkillMD string
```

`internal/cli/catalog.go`:

```go
package cli

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// CommandInfo es una hoja del árbol de comandos.
type CommandInfo struct {
	Path  string `json:"path"`
	Short string `json:"short"`
}

// Catalog recorre el árbol y devuelve sus hojas, ordenadas. Se deriva del
// árbol real en vez de mantenerse a mano: así no puede desviarse de lo que el
// binario registra.
func Catalog(root *cobra.Command) []CommandInfo {
	var out []CommandInfo
	var recorrer func(c *cobra.Command, prefijo []string)

	recorrer = func(c *cobra.Command, prefijo []string) {
		for _, hijo := range c.Commands() {
			if hijo.Hidden || hijo.Name() == "help" || hijo.Name() == "completion" {
				continue
			}
			ruta := append(append([]string{}, prefijo...), hijo.Name())
			if len(hijo.Commands()) > 0 {
				recorrer(hijo, ruta)
				continue
			}
			out = append(out, CommandInfo{Path: strings.Join(ruta, " "), Short: hijo.Short})
		}
	}
	recorrer(root, nil)

	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
```

`internal/commands/skill.go`:

```go
package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/skills"
)

// NewSkillCmd vuelca el skill embebido. Es como un agente aprende a usar este
// binario, sin depender de un repositorio de skills que puede estar desfasado.
func NewSkillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "skill",
		Short: "Vuelca la documentación para agentes de esta versión del CLI",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(cmd.OutOrStdout(), skills.SkillMD)
			return nil
		},
	}
}
```

- [ ] **Step 5: Registrar `skill` y ejecutar los tests**

Añade `commands.NewSkillCmd()` a `root.AddCommand(...)`.

Run: `go test ./internal/cli/ -v`
Expected: PASS. Si el test de paridad falla, **ajusta el SKILL.md, no el test** — el test tiene razón por construcción.

- [ ] **Step 6: Verificar de extremo a extremo**

```bash
make build && ./bin/calliope skill | head -20
```
Expected: la cabecera del SKILL.md

- [ ] **Step 7: Commit**

```bash
git add skills internal/cli internal/commands
git commit -m "feat: skill embebido en el binario con test de paridad del catálogo"
```

---

### Task 20: Plugin de Claude Code

**Files:**
- Create: `.claude-plugin/plugin.json`, `.claude-plugin/commands/calliope.md`, `.claude-plugin/hooks/session-start.sh`, `.claude-plugin/skills` (enlace simbólico a `../skills`)
- Test: `internal/cli/plugin_test.go`

**Interfaces:**
- Consumes: `skills/calliope/SKILL.md` (Task 19), `calliope doctor` (Task 18).
- Produces: el directorio del plugin, instalable en Claude Code.

- [ ] **Step 1: Escribir el test que falla**

`internal/cli/plugin_test.go`:

```go
package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// El plugin apunta al mismo SKILL.md que embebe el binario. Si alguien
// duplicara el fichero en vez de enlazarlo, las dos copias divergirían.
func TestElPluginUsaElMismoSkillQueElBinario(t *testing.T) {
	raiz := raizDelRepo(t)

	delPlugin, err := os.ReadFile(filepath.Join(raiz, ".claude-plugin", "skills", "calliope", "SKILL.md"))
	if err != nil {
		t.Fatalf("el plugin debe exponer el skill: %v", err)
	}
	delRepo, err := os.ReadFile(filepath.Join(raiz, "skills", "calliope", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(delPlugin) != string(delRepo) {
		t.Error("el SKILL.md del plugin y el del binario han divergido; el plugin debe ser un enlace simbólico")
	}
}

func TestElManifiestoDelPluginEsValido(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(raizDelRepo(t), ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("plugin.json no es JSON válido: %v", err)
	}
	if m.Name == "" || m.Version == "" || m.Description == "" {
		t.Errorf("plugin.json incompleto: %+v", m)
	}
}

func raizDelRepo(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		padre := filepath.Dir(dir)
		if padre == dir {
			t.Fatal("no se encontró la raíz del repositorio")
		}
		dir = padre
	}
}
```

- [ ] **Step 2: Ejecutar el test y verificar que falla**

Run: `go test ./internal/cli/ -run Plugin -v`
Expected: FAIL — no existe `.claude-plugin`

- [ ] **Step 3: Crear el manifiesto y el enlace del skill**

`.claude-plugin/plugin.json`:

```json
{
  "name": "calliope",
  "version": "0.1.0",
  "description": "Acceso a los datos, la documentación y las reglas de negocio de Calliope Data desde Claude Code.",
  "author": "Calliope Data"
}
```

```bash
cd .claude-plugin && ln -s ../skills skills && cd ..
```

- [ ] **Step 4: Escribir el comando y el hook de sesión**

`.claude-plugin/commands/calliope.md`:

```markdown
---
description: Consulta Calliope Data (datos, documentación, ontología y reglas de negocio)
---

Usa el CLI `calliope` para responder a lo siguiente: $ARGUMENTS

Antes de nada, ejecuta `calliope skill` y sigue sus invariantes al pie de la
letra. Si `calliope` no está instalado o no hay credencial, `calliope doctor`
dice exactamente qué hacer.
```

`.claude-plugin/hooks/session-start.sh`:

```bash
#!/usr/bin/env bash
# Avisa al empezar la sesión si calliope no está listo, para que el agente no
# descubra el problema a mitad de una tarea.
set -uo pipefail

if ! command -v calliope >/dev/null 2>&1; then
  echo "calliope no está instalado. Instálalo con: brew install calliope/tap/calliope"
  exit 0
fi

if ! calliope doctor --quiet >/dev/null 2>&1; then
  echo "calliope está instalado pero no listo. Diagnostica con: calliope doctor"
  exit 0
fi

echo "calliope listo: $(calliope config get org --quiet 2>/dev/null || echo 'sin organización activa')"
```

```bash
chmod +x .claude-plugin/hooks/session-start.sh
```

Registra el hook en `.claude-plugin/plugin.json` añadiendo:

```json
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/hooks/session-start.sh" }
        ]
      }
    ]
  }
```

- [ ] **Step 5: Ejecutar los tests y verificar que pasan**

Run: `go test ./internal/cli/ -race`
Expected: PASS

- [ ] **Step 6: Probar el hook a mano**

```bash
./.claude-plugin/hooks/session-start.sh
```
Expected: una línea legible, sin traza de error, tanto con `calliope` instalado como sin él.

- [ ] **Step 7: Commit**

```bash
git add .claude-plugin internal/cli
git commit -m "feat: plugin de Claude Code con skill y hook de sesión"
```

---

### Task 21: Distribución con GoReleaser

**Files:**
- Create: `.goreleaser.yaml`, `.github/workflows/release.yml`, `install.sh`
- Modify: `Makefile`

**Interfaces:**
- Consumes: `version.Version`, `version.Commit`, `version.Date` (Task 1).
- Produces: binarios para seis combinaciones de plataforma y arquitectura, tap de Homebrew, paquetes y checksums.

La distribución es requisito de producto, no un extra: la audiencia incluye clientes finales, y por eso se eligió Go.

- [ ] **Step 1: Escribir la configuración de GoReleaser**

`.goreleaser.yaml`:

```yaml
version: 2
project_name: calliope

builds:
  - id: calliope
    main: ./cmd/calliope
    binary: calliope
    env: [CGO_ENABLED=0]
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w
      - -X github.com/calliope/calliope-cli/internal/version.Version={{.Version}}
      - -X github.com/calliope/calliope-cli/internal/version.Commit={{.ShortCommit}}
      - -X github.com/calliope/calliope-cli/internal/version.Date={{.Date}}

archives:
  - formats: [tar.gz]
    format_overrides:
      - goos: windows
        formats: [zip]

checksum:
  name_template: checksums.txt

nfpms:
  - formats: [deb, rpm, apk]
    maintainer: Calliope Data
    description: CLI de Calliope Data
    license: Proprietary

brews:
  - repository:
      owner: calliope
      name: homebrew-tap
    description: CLI de Calliope Data
    install: |
      bin.install "calliope"

scoops:
  - repository:
      owner: calliope
      name: scoop-bucket
    description: CLI de Calliope Data

release:
  draft: false
```

- [ ] **Step 2: Verificar la configuración en local**

```bash
brew install goreleaser
goreleaser check
goreleaser build --snapshot --clean --single-target
./dist/calliope_*/calliope version
```
Expected: `goreleaser check` sin errores, y `version` imprimiendo la versión inyectada por `-ldflags` en vez de `dev`.

- [ ] **Step 3: Escribir el workflow de release**

`.github/workflows/release.yml`:

```yaml
name: Release
on:
  push:
    tags: ['v*']

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go test ./... -race
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          TAP_GITHUB_TOKEN: ${{ secrets.TAP_GITHUB_TOKEN }}
```

`TAP_GITHUB_TOKEN` es un token con permiso de escritura sobre los repositorios `calliope/homebrew-tap` y `calliope/scoop-bucket`, que hay que crear antes del primer release.

- [ ] **Step 4: Escribir el script de instalación**

`install.sh`:

```bash
#!/usr/bin/env bash
# Instalador de calliope. Uso:
#   curl -fsSL https://raw.githubusercontent.com/calliope/calliope-cli/main/install.sh | bash
set -euo pipefail

REPO="calliope/calliope-cli"
DESTINO="${CALLIOPE_INSTALL_DIR:-/usr/local/bin}"

so=$(uname -s | tr '[:upper:]' '[:lower:]')
arco=$(uname -m)
case "$arco" in
  x86_64) arco=amd64 ;;
  aarch64|arm64) arco=arm64 ;;
  *) echo "Arquitectura no soportada: $arco" >&2; exit 1 ;;
esac

version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | head -1 | cut -d'"' -f4)
if [ -z "$version" ]; then
  echo "No se pudo determinar la última versión." >&2
  exit 1
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

archivo="calliope_${version#v}_${so}_${arco}.tar.gz"
url="https://github.com/${REPO}/releases/download/${version}/${archivo}"

echo "Descargando calliope ${version} (${so}/${arco})…"
curl -fsSL "$url" -o "$tmp/$archivo"
curl -fsSL "https://github.com/${REPO}/releases/download/${version}/checksums.txt" -o "$tmp/checksums.txt"

# Se verifica el checksum antes de instalar nada.
(cd "$tmp" && grep " ${archivo}\$" checksums.txt | shasum -a 256 -c -)

tar -xzf "$tmp/$archivo" -C "$tmp"
install -m 0755 "$tmp/calliope" "$DESTINO/calliope"

echo "calliope instalado en $DESTINO/calliope"
"$DESTINO/calliope" version
```

```bash
chmod +x install.sh
```

- [ ] **Step 5: Añadir el objetivo de release al Makefile**

```makefile
.PHONY: snapshot
snapshot:
	goreleaser build --snapshot --clean
```

- [ ] **Step 6: Escribir el test del aviso de nueva versión**

La sección 13 del spec pide que `calliope version` avise cuando hay una versión
más reciente. `internal/version/check_test.go`:

```go
package version

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDetectaUnaVersionMasReciente(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v1.4.0"}`))
	}))
	defer srv.Close()

	got := UltimaVersion(srv.URL, "v1.2.0", time.Second)
	if got != "v1.4.0" {
		t.Errorf("UltimaVersion = %q, se esperaba v1.4.0", got)
	}
}

func TestNoAvisaSiYaEstaAlDia(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v1.2.0"}`))
	}))
	defer srv.Close()

	if got := UltimaVersion(srv.URL, "v1.2.0", time.Second); got != "" {
		t.Errorf("UltimaVersion = %q, no debe avisar estando al día", got)
	}
}

// Sin red, el aviso se calla: nunca debe romper `calliope version`.
func TestSinRedNoRompe(t *testing.T) {
	if got := UltimaVersion("http://127.0.0.1:1", "v1.2.0", 50*time.Millisecond); got != "" {
		t.Errorf("UltimaVersion = %q, sin red debe callarse", got)
	}
}

func TestEnCompilacionDeDesarrolloNoConsulta(t *testing.T) {
	llamado := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llamado = true
	}))
	defer srv.Close()

	if got := UltimaVersion(srv.URL, "dev", time.Second); got != "" {
		t.Errorf("UltimaVersion = %q, una compilación dev no debe avisar", got)
	}
	if llamado {
		t.Error("una compilación dev no debe ni consultar la red")
	}
}
```

Run: `go test ./internal/version/ -v`
Expected: FAIL — `undefined: UltimaVersion`

- [ ] **Step 7: Implementar el aviso de nueva versión**

`internal/version/check.go`:

```go
package version

import (
	"encoding/json"
	"net/http"
	"time"
)

// URLReleases es el endpoint que se consulta para saber la última versión.
const URLReleases = "https://api.github.com/repos/calliope/calliope-cli/releases/latest"

// UltimaVersion devuelve la última versión publicada si es distinta de la
// actual, o cadena vacía. Nunca devuelve error: un aviso de cortesía jamás
// debe hacer fallar `calliope version`.
func UltimaVersion(url, actual string, timeout time.Duration) string {
	// Una compilación de desarrollo no tiene con qué comparar.
	if actual == "" || actual == "dev" {
		return ""
	}

	cliente := &http.Client{Timeout: timeout}
	resp, err := cliente.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var cuerpo struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cuerpo); err != nil {
		return ""
	}
	if cuerpo.TagName == "" || cuerpo.TagName == actual {
		return ""
	}
	return cuerpo.TagName
}
```

En `internal/cli/root.go`, amplía `newVersionCmd` para que avise **solo en
terminal interactivo**: en una tubería la línea extra rompería a quien parsee
la salida.

```go
func newVersionCmd(d appctx.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Muestra la versión de calliope",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "calliope %s (%s, %s)\n",
				version.Version, version.Commit, version.Date)

			if d.IsTTY {
				if nueva := version.UltimaVersion(version.URLReleases, version.Version, 2*time.Second); nueva != "" {
					fmt.Fprintf(cmd.OutOrStdout(),
						"\nHay una versión más reciente: %s. Actualiza con: brew upgrade calliope\n", nueva)
				}
			}
			return nil
		},
	}
}
```

Actualiza la llamada en `NewRootCmd` a `newVersionCmd(d)`, y los tests de la
Task 1 a `NewRootCmd(appctx.Deps{})` (con `IsTTY` falso, así no consultan la red).

Run: `go test ./... -race`
Expected: PASS

- [ ] **Step 8: Verificar y hacer commit**

Run: `goreleaser check && go test ./... -race`
Expected: PASS

```bash
git add .goreleaser.yaml .github install.sh Makefile internal/version internal/cli
git commit -m "feat: distribución con GoReleaser, instalador y aviso de nueva versión"
```

---

### Task 22: Smoke de extremo a extremo y README

**Files:**
- Create: `test/e2e/smoke_test.go`, `README.md`
- Modify: `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: el binario compilado y todos los comandos.
- Produces: el criterio de aceptación 8 del spec, comprobado de forma automática.

El smoke se ejecuta contra una organización real, así que va **opt-in** por variable de entorno y queda fuera del CI por defecto.

- [ ] **Step 1: Escribir el smoke**

`test/e2e/smoke_test.go`:

```go
//go:build e2e

// Package e2e ejecuta el binario real contra una organización de prueba.
// Opt-in: requiere CALLIOPE_E2E=1, CALLIOPE_API_KEY y CALLIOPE_ORG.
package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func calliope(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("../../bin/calliope", args...)
	cmd.Env = os.Environ()
	salida, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("calliope %s falló: %v\n%s", strings.Join(args, " "), err, salida)
	}
	return string(salida)
}

func requiereE2E(t *testing.T) {
	t.Helper()
	if os.Getenv("CALLIOPE_E2E") != "1" {
		t.Skip("smoke desactivado; actívalo con CALLIOPE_E2E=1")
	}
	for _, v := range []string{"CALLIOPE_API_KEY", "CALLIOPE_ORG"} {
		if os.Getenv(v) == "" {
			t.Fatalf("falta la variable %s", v)
		}
	}
}

// Es el criterio de aceptación 8 del spec: la cadena completa que un agente
// debe poder recorrer con solo `calliope skill` en contexto.
func TestCadenaCompletaDeAgente(t *testing.T) {
	requiereE2E(t)

	// 1. Descubrir organizaciones.
	orgs := calliope(t, "orgs", "list", "--json")
	if !strings.Contains(orgs, os.Getenv("CALLIOPE_ORG")) {
		t.Fatalf("la organización de prueba no aparece en orgs list:\n%s", orgs)
	}

	// 2. Consultar el esquema antes de nada.
	esquema := calliope(t, "schema", "--json")
	var envEsquema struct {
		OK   bool `json:"ok"`
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(esquema), &envEsquema); err != nil {
		t.Fatalf("schema no devolvió JSON válido: %v\n%s", err, esquema)
	}
	if !envEsquema.OK || len(envEsquema.Data) == 0 {
		t.Fatalf("el esquema vino vacío:\n%s", esquema)
	}

	// 3. Preguntar y comprobar que cita fuentes.
	respuesta := calliope(t, "ask", "¿qué datos hay disponibles?", "--json")
	var envAsk struct {
		OK   bool `json:"ok"`
		Data struct {
			Text string `json:"text"`
		} `json:"data"`
		Breadcrumbs []struct {
			Cmd string `json:"cmd"`
		} `json:"breadcrumbs"`
	}
	if err := json.Unmarshal([]byte(respuesta), &envAsk); err != nil {
		t.Fatalf("ask no devolvió JSON válido: %v\n%s", err, respuesta)
	}
	if !envAsk.OK || envAsk.Data.Text == "" {
		t.Fatalf("ask no devolvió respuesta:\n%s", respuesta)
	}
	if len(envAsk.Breadcrumbs) == 0 {
		t.Error("ask debe devolver breadcrumbs para que el agente sepa seguir")
	}

	// 4. Extraer un campo con el jq embebido.
	ids := calliope(t, "concepts", "list", "--jq", ".data[].name")
	if strings.TrimSpace(ids) == "" {
		t.Error("el filtro --jq no devolvió nada")
	}
}

func TestDoctorPasaEnUnEntornoConfigurado(t *testing.T) {
	requiereE2E(t)

	salida := calliope(t, "doctor", "--json")
	var env struct {
		Data []struct {
			Nombre string `json:"nombre"`
			Estado string `json:"estado"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(salida), &env); err != nil {
		t.Fatalf("doctor no devolvió JSON válido: %v\n%s", err, salida)
	}
	for _, c := range env.Data {
		if c.Estado == "error" {
			t.Errorf("comprobación %q en error: %s", c.Nombre, salida)
		}
	}
}
```

- [ ] **Step 2: Ejecutar el smoke con credenciales reales**

```bash
make build
CALLIOPE_E2E=1 CALLIOPE_API_KEY=... CALLIOPE_ORG=... go test -tags=e2e ./test/e2e/ -v
```
Expected: PASS. Sin las variables, los tests se saltan.

- [ ] **Step 3: Comprobar que el CI no ejecuta el smoke**

Run: `go test ./... -race`
Expected: PASS, sin compilar `test/e2e` (lo excluye la etiqueta `e2e`).

- [ ] **Step 4: Escribir el README**

`README.md`:

```markdown
# calliope

CLI de [Calliope Data](https://data-0.calliope.so): pregunta en lenguaje
natural sobre tus datos, consulta la documentación, la ontología y las reglas
de negocio de tu organización, desde el terminal y desde agentes de IA.

## Instalación

    brew install calliope/tap/calliope

    # o bien
    curl -fsSL https://raw.githubusercontent.com/calliope/calliope-cli/main/install.sh | bash

## Primeros pasos

    calliope auth login --api-key <clave>   # créala en el UI, en Observabilidad → Claves API
    calliope orgs use <organización>
    calliope ask "¿cómo van las ventas este trimestre?"

`calliope doctor` diagnostica instalación, credencial y conectividad.

## Uso con agentes

El skill va **embebido en el binario**, así que nunca se desincroniza de la
versión instalada:

    calliope skill

En Claude Code, instala el plugin de este repositorio y el agente queda
equipado: obtiene el skill, un comando `/calliope` y un aviso al empezar la
sesión si falta el binario o la credencial.

Sobre esa base, las skills de dominio se escriben en Markdown componiendo
comandos, sin tocar Go.

## Salida

Toda salida usa un envelope común, y los `breadcrumbs` indican el siguiente
comando:

    {"ok": true, "data": [...], "summary": "12 documentos",
     "breadcrumbs": [{"action": "show", "cmd": "calliope docs show <id>"}]}

Modos: humano en terminal · JSON en tubería · `--json` · `--quiet` · `--md` ·
`--jq '<expr>'` (el filtro va dentro del binario; no hace falta `jq`).

Códigos de salida: `0` correcto · `1` error · `2` uso incorrecto ·
`3` no autorizado · `4` no encontrado · `5` límite superado.

## Configuración

Por capas, de mayor a menor precedencia: flags · variables `CALLIOPE_*` ·
`.calliope/config.json` del directorio · el de la raíz del repositorio ·
`~/.config/calliope/config.json` · valores por defecto.

`calliope config list` muestra cada valor **con la capa de la que proviene**.

Por seguridad, la configuración de proyecto solo puede fijar `org` y `output`:
un `.calliope/config.json` viaja dentro de cualquier repositorio clonado, y si
pudiera fijar `base_url` un repositorio hostil redirigiría tu token.

## Desarrollo

    make test     # go test ./... -race
    make build    # bin/calliope
    make snapshot # binarios de todas las plataformas con GoReleaser

Los tests de extremo a extremo van aparte y necesitan credenciales reales:

    CALLIOPE_E2E=1 CALLIOPE_API_KEY=... CALLIOPE_ORG=... go test -tags=e2e ./test/e2e/
```

- [ ] **Step 5: Commit**

```bash
git add test README.md .github
git commit -m "feat: smoke de extremo a extremo y README"
```

---

## Orden de ejecución y dependencias

```
Task 1  bootstrap
  ├─ Task 2  envelope y errores
  │    └─ Task 3  presenter
  ├─ Task 4  configuración en capas
  │    └─ Task 5  frontera de confianza
  ├─ Task 6  spike de OAuth  ──decide──▶ Task 8 (condicional)
  │    └─ Task 7  credenciales
  └─ Task 9  transporte del SDK
       └─ Task 10 modelos y métodos

Tasks 2,3,5,7,10  ──▶  Task 11 appctx
                          ├─ Task 12 auth y orgs
                          ├─ Task 13 config
                          ├─ Task 14 ask
                          ├─ Task 15 docs
                          ├─ Task 16 concepts y rules
                          ├─ Task 17 schema y query
                          └─ Task 18 doctor
                                └─ Task 19 skill y paridad
                                     ├─ Task 20 plugin
                                     ├─ Task 21 distribución
                                     └─ Task 22 smoke y README
```

Las tareas 2-3, 4-5 y 9-10 son independientes entre sí y pueden ir en paralelo.
Las tareas 12-18 también, una vez cerrada la 11.

La **Task 6** conviene lanzarla pronto aunque su resultado no haga falta hasta
la 8: necesita acceso al dashboard de PropelAuth, que puede tardar en llegar.
