package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/calliope/calliope-cli/internal/auth"
)

func serverWithFixture(t *testing.T, fixture string, capturar *http.Request) *Client {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if capturar != nil {
			// El cuerpo se lee y se repone como un nuevo lector para que
			// capturar.Body siga siendo legible después de que el handler
			// termine (r.Clone no duplica el flujo subyacente del Body).
			cuerpo, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(cuerpo))
			clon := r.Clone(r.Context())
			clon.Body = io.NopCloser(bytes.NewReader(cuerpo))
			*capturar = *clon
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}))
	t.Cleanup(srv.Close)
	return New(Options{BaseURL: srv.URL, Credential: auth.Credential{Kind: auth.KindAPIKey, Token: "k"}})
}

func TestAskDecodificaLaRespuestaYSusFuentes(t *testing.T) {
	c := serverWithFixture(t, "ask.json", nil)

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
	c := serverWithFixture(t, "ask.json", &visto)

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
	c := serverWithFixture(t, "documents.json", nil)

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
	c := serverWithFixture(t, "documents.json", &visto)

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
	c := serverWithFixture(t, "documents.json", &visto)

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
	c := serverWithFixture(t, "query_data_como_cadena.json", nil)

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

// Los tres tests siguientes cubren Me, Organization y SchemaResponse: sus
// tipos no vienen de calliope-data-mcp (verificado) sino que se corrigieron
// contra los tipos TypeScript de calliope-data-ui (BackendUser, Organization,
// SchemaResponse en types/index.ts), sin confirmación contra el backend en
// vivo. Ver task-10-tipos-corregidos.md y el informe de esta tarea.

func TestMeDecodificaElPerfil(t *testing.T) {
	c := serverWithFixture(t, "me.json", nil)

	me, err := c.Me(context.Background())
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if me.ID != "user-1" {
		t.Errorf("ID = %q — el campo es id, no userId", me.ID)
	}
	if me.Email != "ana@acme.com" {
		t.Errorf("Email = %q", me.Email)
	}
	if me.Username != "ana" || me.FirstName != "Ana" || me.LastName != "Gómez" {
		t.Errorf("perfil mal decodificado: %+v", me)
	}
	if len(me.Organizations) != 1 {
		t.Fatalf("len(Organizations) = %d, se esperaba 1", len(me.Organizations))
	}
	if me.Organizations[0].ID != "acme" || me.Organizations[0].Name != "Acme Corp" || me.Organizations[0].UserRole != "ADMIN" {
		t.Errorf("OrgInfo mal decodificado: %+v", me.Organizations[0])
	}
}

func TestListOrganizationsDecodificaLaLista(t *testing.T) {
	c := serverWithFixture(t, "organizations.json", nil)

	orgs, err := c.ListOrganizations(context.Background())
	if err != nil {
		t.Fatalf("ListOrganizations: %v", err)
	}
	if len(orgs) != 1 {
		t.Fatalf("len(orgs) = %d, se esperaba 1", len(orgs))
	}
	if orgs[0].ID != "acme" || orgs[0].Name != "Acme Corp" {
		t.Errorf("organización mal decodificada: %+v", orgs[0])
	}
	if orgs[0].Status != "ACTIVE" || orgs[0].Description != "Cliente principal" {
		t.Errorf("Status/Description mal decodificados: %+v", orgs[0])
	}
}

func TestSchemaDecodificaTableNameYNamePorSeparado(t *testing.T) {
	c := serverWithFixture(t, "schema.json", nil)

	sch, err := c.Schema(context.Background(), "acme")
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	if sch.OrganizationID != "acme" {
		t.Errorf("OrganizationID = %q, se esperaba acme", sch.OrganizationID)
	}
	if len(sch.Tables) != 1 {
		t.Fatalf("len(Tables) = %d, se esperaba 1", len(sch.Tables))
	}
	tbl := sch.Tables[0]
	if tbl.TableName != "fct_ventas" {
		t.Errorf("TableName = %q, se esperaba fct_ventas", tbl.TableName)
	}
	if tbl.Name != "Ventas" {
		t.Errorf("Name = %q, se esperaba Ventas", tbl.Name)
	}
	if tbl.TableName == tbl.Name {
		t.Error("TableName y Name son campos distintos y deben poder diferir")
	}
	if len(tbl.Columns) != 2 {
		t.Fatalf("len(Columns) = %d, se esperaba 2", len(tbl.Columns))
	}
	if !tbl.Columns[0].IsPrimaryKey || tbl.Columns[0].Nullable {
		t.Errorf("columna 0 mal decodificada: %+v", tbl.Columns[0])
	}
	if tbl.Columns[1].IsPrimaryKey || !tbl.Columns[1].Nullable {
		t.Errorf("columna 1 mal decodificada: %+v", tbl.Columns[1])
	}
}

func TestSchemaEnviaLaRutaConScopeDeOrganizacion(t *testing.T) {
	var visto http.Request
	c := serverWithFixture(t, "schema.json", &visto)

	if _, err := c.Schema(context.Background(), "acme"); err != nil {
		t.Fatalf("Schema: %v", err)
	}
	if visto.URL.Path != "/v1/organizations/acme/database/schema" {
		t.Errorf("ruta = %q", visto.URL.Path)
	}
	if visto.Method != http.MethodGet {
		t.Errorf("método = %q, se esperaba GET", visto.Method)
	}
}

// Los cinco tests siguientes cubren SearchDocuments, GetDocument,
// ListConcepts, GetConcept y ListRules: el brief original solo traía tests
// para Ask, ListDocuments y Query, pero TDD es restricción global del
// proyecto y estos cinco métodos sustentan las Tasks 12-18. Cada uno
// asevera sobre verbo+ruta exacta y sobre los valores decodificados de la
// respuesta (no solo que la petición se hizo), y GetDocument/GetConcept
// comprueban además que un identificador con caracteres especiales llega
// escapado en la ruta.

func TestSearchDocumentsEnviaLaConsultaYDecodificaLosResultados(t *testing.T) {
	var visto http.Request
	c := serverWithFixture(t, "search_documents.json", &visto)

	resultados, err := c.SearchDocuments(context.Background(), "acme", "ventas", 5)
	if err != nil {
		t.Fatalf("SearchDocuments: %v", err)
	}

	if visto.URL.Path != "/v1/organizations/acme/search/documents" {
		t.Errorf("ruta = %q", visto.URL.Path)
	}
	if visto.Method != http.MethodPost {
		t.Errorf("método = %q, se esperaba POST", visto.Method)
	}
	cuerpoBytes, err := io.ReadAll(visto.Body)
	if err != nil {
		t.Fatalf("leyendo el cuerpo capturado: %v", err)
	}
	var cuerpo map[string]any
	if err := json.Unmarshal(cuerpoBytes, &cuerpo); err != nil {
		t.Fatalf("el cuerpo enviado no es JSON válido: %v", err)
	}
	if cuerpo["query"] != "ventas" {
		t.Errorf("cuerpo.query = %v, se esperaba ventas", cuerpo["query"])
	}
	if cuerpo["limit"] != float64(5) {
		t.Errorf("cuerpo.limit = %v, se esperaba 5", cuerpo["limit"])
	}

	if len(resultados) != 1 {
		t.Fatalf("len(resultados) = %d, se esperaba 1", len(resultados))
	}
	r := resultados[0]
	if r.DocumentID != "doc-9" || r.ChunkID != 42 || r.Ordinal != 3 {
		t.Errorf("resultado mal decodificado: %+v", r)
	}
	if r.Score != 0.87 {
		t.Errorf("Score = %v, se esperaba 0.87", r.Score)
	}
}

func TestSearchDocumentsUsaLimitePorDefectoConCeroYNegativo(t *testing.T) {
	for _, limite := range []int{0, -3} {
		var visto http.Request
		c := serverWithFixture(t, "search_documents.json", &visto)

		if _, err := c.SearchDocuments(context.Background(), "acme", "ventas", limite); err != nil {
			t.Fatalf("SearchDocuments(limit=%d): %v", limite, err)
		}
		cuerpoBytes, err := io.ReadAll(visto.Body)
		if err != nil {
			t.Fatalf("leyendo el cuerpo capturado: %v", err)
		}
		var cuerpo map[string]any
		if err := json.Unmarshal(cuerpoBytes, &cuerpo); err != nil {
			t.Fatalf("el cuerpo enviado no es JSON válido: %v", err)
		}
		if cuerpo["limit"] != float64(10) {
			t.Errorf("limit=%d: cuerpo.limit = %v, se esperaba el valor por defecto 10", limite, cuerpo["limit"])
		}
	}
}

func TestGetDocumentDecodificaElDocumentoYEscapaElID(t *testing.T) {
	var visto http.Request
	c := serverWithFixture(t, "document.json", &visto)

	doc, err := c.GetDocument(context.Background(), "acme", "doc/1")
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}

	if visto.RequestURI != "/v1/organizations/acme/documents/doc%2F1" {
		t.Errorf("RequestURI = %q — el id con barra debe llegar escapado (%%2F)", visto.RequestURI)
	}
	if visto.Method != http.MethodGet {
		t.Errorf("método = %q, se esperaba GET", visto.Method)
	}
	if doc.ID != "doc-1" || doc.SizeBytes != 402113 || doc.Status != "READY" {
		t.Errorf("documento mal decodificado: %+v", doc)
	}
}

func TestListConceptsDecodificaElGrafo(t *testing.T) {
	var visto http.Request
	c := serverWithFixture(t, "concepts.json", &visto)

	grafo, err := c.ListConcepts(context.Background(), "acme")
	if err != nil {
		t.Fatalf("ListConcepts: %v", err)
	}

	if visto.URL.Path != "/v1/organizations/acme/knowledge/concepts" {
		t.Errorf("ruta = %q", visto.URL.Path)
	}
	if visto.Method != http.MethodGet {
		t.Errorf("método = %q, se esperaba GET", visto.Method)
	}
	if len(grafo.Concepts) != 1 {
		t.Fatalf("len(Concepts) = %d, se esperaba 1", len(grafo.Concepts))
	}
	n := grafo.Concepts[0]
	if n.ID != "concept-1" || n.Name != "Cliente" || !n.IsActive {
		t.Errorf("nodo mal decodificado: %+v", n)
	}
	if n.RecordCount == nil || *n.RecordCount != 120 {
		t.Errorf("RecordCount mal decodificado: %+v", n.RecordCount)
	}
}

func TestGetConceptDecodificaElDetalleYEscapaElID(t *testing.T) {
	var visto http.Request
	c := serverWithFixture(t, "concept_detail.json", &visto)

	// El id lleva espacio y barra a propósito: net/url escapa un espacio crudo
	// a %20 igual con o sin url.PathEscape (lo comprueba automáticamente al
	// serializar la petición), así que un id de un solo espacio no detecta si
	// el código deja de escapar. La barra sí lo hace: sin escapar, Go la trata
	// como separador de segmento y no la reescapa a %2F.
	detalle, err := c.GetConcept(context.Background(), "acme", "a b/c")
	if err != nil {
		t.Fatalf("GetConcept: %v", err)
	}

	if visto.RequestURI != "/v1/organizations/acme/knowledge/concepts/a%20b%2Fc" {
		t.Errorf("RequestURI = %q — el id con espacio y barra debe llegar escapado (%%20 y %%2F)", visto.RequestURI)
	}
	if visto.Method != http.MethodGet {
		t.Errorf("método = %q, se esperaba GET", visto.Method)
	}
	if detalle.Concept.ID != "concept-1" || detalle.Concept.Name != "Cliente" {
		t.Errorf("Concept mal decodificado: %+v", detalle.Concept)
	}
	if len(detalle.Attributes) != 1 || detalle.Attributes[0].Name != "email" {
		t.Errorf("Attributes mal decodificado: %+v", detalle.Attributes)
	}
}

func TestListRulesDecodificaLasReglas(t *testing.T) {
	var visto http.Request
	c := serverWithFixture(t, "rules.json", &visto)

	reglas, err := c.ListRules(context.Background(), "acme")
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}

	if visto.URL.Path != "/v1/organizations/acme/rules" {
		t.Errorf("ruta = %q", visto.URL.Path)
	}
	if visto.Method != http.MethodGet {
		t.Errorf("método = %q, se esperaba GET", visto.Method)
	}
	if len(reglas) != 1 {
		t.Fatalf("len(reglas) = %d, se esperaba 1", len(reglas))
	}
	if reglas[0].ID != "rule-1" || reglas[0].Details != "No superar el 20%" || reglas[0].Status != "ACTIVE" {
		t.Errorf("regla mal decodificada: %+v", reglas[0])
	}
}

// Los tres tests siguientes cierran los caminos de Rows() que faltaban:
// cadena vacía, cadena con JSON inválido dentro, y `data:null` explícito
// (hasta ahora solo se cubría la clave ausente). Ver ronda de correcciones
// 1/5 en el informe de esta tarea.

func TestQueryDataNuloExplicitoDevuelveCero(t *testing.T) {
	var resp QueryResponse
	if err := json.Unmarshal([]byte(`{"data":null}`), &resp); err != nil {
		t.Fatal(err)
	}
	filas, err := resp.Rows()
	if err != nil {
		t.Fatalf("Rows con data:null no debe fallar: %v", err)
	}
	if len(filas) != 0 {
		t.Errorf("se esperaban 0 filas, se obtuvieron %d", len(filas))
	}
}

func TestQueryDataComoCadenaVaciaDevuelveCero(t *testing.T) {
	var resp QueryResponse
	if err := json.Unmarshal([]byte(`{"data":""}`), &resp); err != nil {
		t.Fatal(err)
	}
	filas, err := resp.Rows()
	if err != nil {
		t.Fatalf("Rows con data:\"\" no debe fallar: %v", err)
	}
	if len(filas) != 0 {
		t.Errorf("se esperaban 0 filas, se obtuvieron %d", len(filas))
	}
}

func TestQueryDataComoCadenaConJSONInvalidoDevuelveError(t *testing.T) {
	var resp QueryResponse
	if err := json.Unmarshal([]byte(`{"data":"esto no es json"}`), &resp); err != nil {
		t.Fatal(err)
	}
	_, err := resp.Rows()
	if err == nil {
		t.Fatal("se esperaba un error al deshacer una cadena con JSON inválido dentro")
	}
	var errSintaxis *json.SyntaxError
	if !errors.As(err, &errSintaxis) {
		t.Errorf("err = %v (%T), se esperaba *json.SyntaxError", err, err)
	}
}
