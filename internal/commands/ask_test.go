package commands

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/output"
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
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations/acme/ask" {
			t.Errorf("ruta = %q", r.URL.Path)
		}
		w.Write([]byte(respuestaAsk))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewAskCmd(d), stdout)
	root.SetArgs([]string{"ask", "¿cómo van las ventas?", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("ask: %v", err)
	}

	var env struct {
		OK      bool   `json:"ok"`
		Summary string `json:"summary"`
		Data    struct {
			Text    string `json:"text"`
			Sources []struct {
				Citation   string `json:"citation"`
				DocumentID string `json:"documentId"`
			} `json:"sources"`
		} `json:"data"`
		Breadcrumbs []struct {
			Action string `json:"action"`
			Cmd    string `json:"cmd"`
		} `json:"breadcrumbs"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if !env.OK || env.Data.Text != "Las ventas crecieron un 12%." {
		t.Errorf("respuesta inesperada: %q", stdout.String())
	}
	if len(env.Data.Sources) != 1 || env.Data.Sources[0].Citation != "Informe anual, p. 12" ||
		env.Data.Sources[0].DocumentID != "doc-1" {
		t.Errorf("fuente inesperada: %+v", env.Data.Sources)
	}
	// El resumen cuenta las fuentes citadas: es lo primero que lee un
	// agente. Singular a propósito (Diferidos #12 y #16): con 1 fuente,
	// "1 fuentes citadas" pluraliza mal.
	if env.Summary != "1 fuente citada" {
		t.Errorf("summary = %q, se esperaba %q", env.Summary, "1 fuente citada")
	}
	// Breadcrumbs concretos, no solo "hay alguno": son la navegación que el
	// SKILL.md espera que un agente siga tras `ask`.
	quiero := map[string]string{"documento": "calliope docs show <id>", "conceptos": "calliope concepts list"}
	if len(env.Breadcrumbs) != len(quiero) {
		t.Fatalf("breadcrumbs = %+v, se esperaban %d", env.Breadcrumbs, len(quiero))
	}
	for _, b := range env.Breadcrumbs {
		if quiero[b.Action] != b.Cmd {
			t.Errorf("breadcrumb inesperado: %+v", b)
		}
	}
}

func TestAskEnMarkdownCitaLasFuentes(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(respuestaAsk))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewAskCmd(d), stdout)
	root.SetArgs([]string{"ask", "¿ventas?", "--md"})
	if err := root.Execute(); err != nil {
		t.Fatalf("ask: %v", err)
	}
	salida := stdout.String()
	if !strings.Contains(salida, "Las ventas crecieron un 12%.") {
		t.Errorf("el markdown debe incluir la respuesta, se obtuvo:\n%s", salida)
	}
	// Comprobación específica del formato Markdown (cabecera de sección y
	// enlace al documento entre backticks), no solo que la cita aparezca en
	// algún sitio: eso también lo produciría por error el renderer de texto.
	if !strings.Contains(salida, "### Fuentes") {
		t.Errorf("el markdown debe traer la sección de fuentes, se obtuvo:\n%s", salida)
	}
	if !strings.Contains(salida, "**Informe anual** — Informe anual, p. 12 (`doc-1`)") {
		t.Errorf("el markdown debe citar la fuente con su título y documento, se obtuvo:\n%s", salida)
	}
}

// TestAskEnTextoCitaLasFuentes es la variante con IsTTY:true de las dos
// anteriores. depsWithServer deja IsTTY:false, así que en modo automático la
// salida siempre cae a JSON y el renderer de Text (escribirAskTexto en el
// brief, writeAskText aquí) nunca se llega a ejercitar: una regresión ahí
// pasaría inadvertida. Fijando IsTTY:true y sin --json/--md se fuerza el
// camino de Text y se comprueba que también cita las fuentes, con el formato
// propio de Text (no el de Markdown).
func TestAskEnTextoCitaLasFuentes(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(respuestaAsk))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewAskCmd(d), stdout)
	root.SetArgs([]string{"ask", "¿ventas?"})
	if err := root.Execute(); err != nil {
		t.Fatalf("ask: %v", err)
	}
	salida := stdout.String()
	if !strings.Contains(salida, "Las ventas crecieron un 12%.") {
		t.Errorf("el texto debe incluir la respuesta, se obtuvo:\n%s", salida)
	}
	if !strings.Contains(salida, "Fuentes:") {
		t.Errorf("el texto debe traer la cabecera de fuentes, se obtuvo:\n%s", salida)
	}
	if !strings.Contains(salida, "  · Informe anual, p. 12 (doc-1)") {
		t.Errorf("el texto debe citar la fuente con su documento, se obtuvo:\n%s", salida)
	}
	// El formato Markdown no debe colarse en el renderer de texto.
	if strings.Contains(salida, "### Fuentes") || strings.Contains(salida, "**Informe anual**") {
		t.Errorf("el texto no debe usar el formato Markdown, se obtuvo:\n%s", salida)
	}
}

// TestAskConRespuestaSinExitoDevuelveError comprueba que un `success:false`
// del backend se traduce en un error de CLI con un mensaje fijo en español y
// el hint de recuperación, no en una respuesta "correcta" vacía.
//
// El mensaje es fijo -no el resp.Error del backend- a propósito (C1 de la
// oleada final): antes se reenviaba *resp.Error verbatim, y ese campo es
// cuerpo de respuesta del backend tanto como el de un 4xx, así que podía
// sacar nombres de tabla internos, hostnames o puertos por el stdout de un
// CLI que corre en la máquina del cliente.
func TestAskConRespuestaSinExitoDevuelveError(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":false,"error":"La pregunta es ambigua."}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewAskCmd(d), stdout)
	root.SetArgs([]string{"ask", "¿?"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error con success:false")
	}

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T", err)
	}
	if cliErr.Message != "Calliope no pudo responder a la pregunta." {
		t.Errorf("message = %q, se esperaba el mensaje fijo en español", cliErr.Message)
	}
	if !strings.Contains(cliErr.Hint, "calliope concepts list") {
		t.Errorf("hint = %q, se esperaba que sugiriera calliope concepts list", cliErr.Hint)
	}
	if got := output.ExitCodeFor(err); got != 1 {
		t.Errorf("código de salida = %d, se esperaba 1 (genérico)", got)
	}
}

// TestAskConRespuestaSinExitoNuncaFiltraElCuerpoDelBackend es el test de
// no-regresión de C1: sea lo que sea que el backend mande en resp.Error -aquí
// deliberadamente algo con pinta de detalle interno (nombre de tabla,
// hostname, puerto)-, ese texto no debe aparecer en ningún sitio de la salida
// del comando, ni en el mensaje ni en el hint. Antes de C1, este test habría
// fallado: *resp.Error se usaba verbatim como mensaje.
func TestAskConRespuestaSinExitoNuncaFiltraElCuerpoDelBackend(t *testing.T) {
	cuerpoInterno := "falló fct_ventas_internal en db-shard-07.internal.calliope.so:5432"
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":false,"error":"` + cuerpoInterno + `"}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewAskCmd(d), stdout)
	root.SetArgs([]string{"ask", "¿?"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error con success:false")
	}

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T", err)
	}
	if strings.Contains(cliErr.Message, cuerpoInterno) || strings.Contains(cliErr.Hint, cuerpoInterno) {
		t.Fatalf("el cuerpo de error del backend se filtró: message=%q hint=%q", cliErr.Message, cliErr.Hint)
	}
	if strings.Contains(cliErr.Message, "internal.calliope.so") || strings.Contains(cliErr.Message, "5432") {
		t.Fatalf("detalle interno del backend filtrado en el mensaje: %q", cliErr.Message)
	}
}

// TestAskEnviaLaAccionSolicitada comprueba que el flag --action llega de
// verdad al backend: sin esta aserción, olvidar pasar `accion` a Client.Ask
// (o pasar siempre "") dejaría la suite en verde igualmente.
func TestAskEnviaLaAccionSolicitada(t *testing.T) {
	var cuerpo map[string]string
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&cuerpo); err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(respuestaAsk))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewAskCmd(d), stdout)
	root.SetArgs([]string{"ask", "¿ventas?", "--action", "resumen", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("ask: %v", err)
	}
	if cuerpo["question"] != "¿ventas?" || cuerpo["action"] != "resumen" {
		t.Errorf("cuerpo enviado = %+v, se esperaba question=¿ventas? action=resumen", cuerpo)
	}
}

func TestAskSinPreguntaEsErrorDeUso(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewAskCmd(d), stdout)
	root.SetArgs([]string{"ask"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error sin pregunta")
	}

	// La validación de argumentos debe pasar por exactArgs, no por
	// cobra.ExactArgs: sin él, el error sale en inglés, sin hint y con
	// código de salida 1 en vez de 2 (uso incorrecto).
	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T", err)
	}
	if cliErr.Hint == "" {
		t.Error("el error debería traer un hint con la forma de uso")
	}
	if strings.Contains(cliErr.Message, "accepts") || strings.Contains(cliErr.Message, "arg(s)") {
		t.Errorf("el mensaje no debe ser el de Cobra en inglés: %q", cliErr.Message)
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2 (uso incorrecto)", got)
	}
}
