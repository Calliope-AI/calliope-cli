package commands

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/sdk"
)

// TestDocsEsUnGrupoSinRunE comprueba que `docs` es un grupo de recursos: sin
// RunE ni Run, invocarlo pelado debe mostrar la ayuda en vez de ejecutar nada.
func TestDocsEsUnGrupoSinRunE(t *testing.T) {
	cmd := NewDocsCmd(appctx.Deps{})
	if cmd.RunE != nil || cmd.Run != nil {
		t.Error("docs es un grupo: invocarlo pelado debe mostrar la ayuda, no ejecutar nada")
	}
	if len(cmd.Commands()) != 3 {
		t.Errorf("docs debe tener 3 subcomandos (list, show, search), tiene %d", len(cmd.Commands()))
	}
}

func TestDocsListPasaLosFiltros(t *testing.T) {
	var vistaQuery string
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		vistaQuery = r.URL.RawQuery
		w.Write([]byte(`{"content":[{"id":"doc-1","filename":"a.pdf","status":"READY","declaredMime":"application/pdf","sizeBytes":1,"createdAt":"","updatedAt":""}],"totalSize":1}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewDocsCmd(d), stdout)
	root.SetArgs([]string{"docs", "list", "--status", "READY", "--tag", "finanzas", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("docs list: %v", err)
	}
	if vistaQuery == "" {
		t.Error("los filtros deben viajar en la query string")
	}
	// Se comprueba el valor exacto de cada filtro, no solo que la query no esté
	// vacía: eso no detectaría un flag cableado al campo equivocado (p. ej.
	// --status escribiendo en p.Tag).
	q, err := url.ParseQuery(vistaQuery)
	if err != nil {
		t.Fatalf("query inválida: %v", err)
	}
	if q.Get("status") != "READY" || q.Get("tag") != "finanzas" {
		t.Errorf("query = %q, se esperaba status=READY y tag=finanzas", vistaQuery)
	}
	if q.Get("q") != "" || q.Get("page") != "" || q.Get("size") != "" {
		t.Errorf("query = %q, no se pasaron --q/--page/--size y no deberían aparecer", vistaQuery)
	}

	var env struct {
		Summary     string           `json:"summary"`
		Data        []map[string]any `json:"data"`
		Breadcrumbs []struct {
			Action string `json:"action"`
			Cmd    string `json:"cmd"`
		} `json:"breadcrumbs"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if len(env.Data) != 1 {
		t.Fatalf("se esperaba 1 documento en data, hay %d", len(env.Data))
	}
	// Se comprueban los valores del documento devuelto, no solo su cantidad:
	// esto detecta un tag json mal escrito o un campo que no llega a la salida.
	doc := env.Data[0]
	if doc["id"] != "doc-1" || doc["filename"] != "a.pdf" || doc["status"] != "READY" {
		t.Errorf("documento inesperado: %+v", doc)
	}
	if doc["sizeBytes"] != float64(1) {
		t.Errorf("sizeBytes = %v, se esperaba 1", doc["sizeBytes"])
	}
	if env.Summary != "1 de 1 documentos" {
		t.Errorf("summary = %q, se esperaba %q", env.Summary, "1 de 1 documentos")
	}
	quiero := map[string]string{
		"detalle": "calliope docs show <id>",
		"buscar":  `calliope docs search "<consulta>"`,
	}
	if len(env.Breadcrumbs) != len(quiero) {
		t.Fatalf("breadcrumbs = %+v, se esperaban %d", env.Breadcrumbs, len(quiero))
	}
	for _, b := range env.Breadcrumbs {
		if quiero[b.Action] != b.Cmd {
			t.Errorf("breadcrumb inesperado: %+v", b)
		}
	}
}

// TestDocsListSinFiltrosNoEnviaQueryString comprueba el reverso de la
// anterior: sin flags, ListDocumentsParams no debe generar ningún parámetro
// (page/size en 0 se omiten, no se envían como "0").
func TestDocsListSinFiltrosNoEnviaQueryString(t *testing.T) {
	var vistaQuery string
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		vistaQuery = r.URL.RawQuery
		w.Write([]byte(`{"content":[],"totalSize":0}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewDocsCmd(d), stdout)
	root.SetArgs([]string{"docs", "list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("docs list: %v", err)
	}
	if vistaQuery != "" {
		t.Errorf("query = %q, se esperaba vacía sin flags", vistaQuery)
	}
}

// TestDocsListEnTextoMuestraTabla cubre el renderer de Text de `docs list`,
// que depsWithServer (IsTTY:false) nunca ejercita en modo automático. Se fija
// IsTTY:true a propósito para forzar ese camino.
func TestDocsListEnTextoMuestraTabla(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"content":[
			{"id":"doc-1","filename":"a.pdf","title":"Informe anual","status":"READY","declaredMime":"application/pdf","sizeBytes":2048,"createdAt":"","updatedAt":""},
			{"id":"doc-2","filename":"b.pdf","status":"PROCESSING","declaredMime":"application/pdf","sizeBytes":99,"createdAt":"","updatedAt":""}
		],"totalSize":2}`))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewDocsCmd(d), stdout)
	root.SetArgs([]string{"docs", "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("docs list: %v", err)
	}
	salida := stdout.String()
	for _, cabecera := range []string{"ID", "TÍTULO", "ESTADO", "BYTES"} {
		if !strings.Contains(salida, cabecera) {
			t.Errorf("la tabla debe traer la cabecera %q, se obtuvo:\n%s", cabecera, salida)
		}
	}
	// Orden de columnas, no solo presencia: un Contains por cabecera no
	// detectaría una permutación de las columnas.
	primeraLinea := strings.SplitN(salida, "\n", 2)[0]
	if i, j, k := strings.Index(primeraLinea, "ID"), strings.Index(primeraLinea, "TÍTULO"), strings.Index(primeraLinea, "ESTADO"); !(i < j && j < k) {
		t.Errorf("orden de cabeceras inesperado: %q", primeraLinea)
	}
	if j, k, l := strings.Index(primeraLinea, "TÍTULO"), strings.Index(primeraLinea, "ESTADO"), strings.Index(primeraLinea, "BYTES"); !(j < k && k < l) {
		t.Errorf("orden de cabeceras inesperado: %q", primeraLinea)
	}
	// doc-1 tiene título propio: debe mostrarse en vez del nombre de fichero.
	if !strings.Contains(salida, "doc-1") || !strings.Contains(salida, "Informe anual") {
		t.Errorf("la tabla debe mostrar doc-1 con su título, se obtuvo:\n%s", salida)
	}
	if strings.Contains(salida, "a.pdf") {
		t.Errorf("doc-1 tiene título propio, no debe caer al nombre de fichero: %s", salida)
	}
	// doc-2 no tiene título: debe caer al nombre de fichero.
	if !strings.Contains(salida, "doc-2") || !strings.Contains(salida, "b.pdf") || !strings.Contains(salida, "PROCESSING") {
		t.Errorf("la tabla debe mostrar doc-2 con su fichero como título, se obtuvo:\n%s", salida)
	}
	if !strings.Contains(salida, "2048") || !strings.Contains(salida, "99") {
		t.Errorf("la tabla debe mostrar los bytes de cada documento, se obtuvo:\n%s", salida)
	}
	// Orden de cada fila, no solo presencia: un Contains global no detectaría
	// que ESTADO y BYTES viajaran intercambiados dentro de la misma fila.
	var lineaDoc1, lineaDoc2 string
	for _, l := range strings.Split(salida, "\n") {
		if strings.Contains(l, "doc-1") {
			lineaDoc1 = l
		}
		if strings.Contains(l, "doc-2") {
			lineaDoc2 = l
		}
	}
	if i, j := strings.Index(lineaDoc1, "READY"), strings.Index(lineaDoc1, "2048"); i < 0 || j < 0 || i > j {
		t.Errorf("en la fila de doc-1, ESTADO (READY) debe ir antes que BYTES (2048): %q", lineaDoc1)
	}
	if i, j := strings.Index(lineaDoc2, "PROCESSING"), strings.Index(lineaDoc2, "99"); i < 0 || j < 0 || i > j {
		t.Errorf("en la fila de doc-2, ESTADO (PROCESSING) debe ir antes que BYTES (99): %q", lineaDoc2)
	}
	if strings.Contains(salida, "{") {
		t.Errorf("en modo texto no debe salir JSON crudo: %s", salida)
	}
}

func TestDocsShowLlamaAlEndpointDelDocumento(t *testing.T) {
	var ruta string
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		ruta = r.URL.Path
		w.Write([]byte(`{"id":"doc-1","filename":"a.pdf","status":"READY","declaredMime":"application/pdf","sizeBytes":1,"createdAt":"","updatedAt":""}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewDocsCmd(d), stdout)
	root.SetArgs([]string{"docs", "show", "doc-1", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("docs show: %v", err)
	}
	if ruta != "/v1/organizations/acme/documents/doc-1" {
		t.Errorf("ruta = %q", ruta)
	}

	var env struct {
		Summary string `json:"summary"`
		Data    struct {
			ID       string `json:"id"`
			Filename string `json:"filename"`
			Status   string `json:"status"`
		} `json:"data"`
		Breadcrumbs []struct {
			Action string `json:"action"`
			Cmd    string `json:"cmd"`
		} `json:"breadcrumbs"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if env.Data.ID != "doc-1" || env.Data.Filename != "a.pdf" || env.Data.Status != "READY" {
		t.Errorf("documento inesperado: %+v", env.Data)
	}
	// Sin título propio, el resumen cae al nombre de fichero.
	if env.Summary != "a.pdf" {
		t.Errorf("summary = %q, se esperaba %q", env.Summary, "a.pdf")
	}
	if len(env.Breadcrumbs) != 1 || env.Breadcrumbs[0].Action != "buscar dentro" ||
		env.Breadcrumbs[0].Cmd != `calliope docs search "<consulta>"` {
		t.Errorf("breadcrumbs inesperados: %+v", env.Breadcrumbs)
	}
}

// TestDocsShowUsaElTituloDelDocumentoComoResumenSiExiste comprueba que,
// cuando el documento sí tiene título propio, el resumen lo usa en vez del
// nombre de fichero: sin esta aserción, un titleOf que siempre devolviera
// Filename pasaría inadvertido.
func TestDocsShowUsaElTituloDelDocumentoComoResumenSiExiste(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"doc-1","filename":"a.pdf","title":"Informe anual","status":"READY","declaredMime":"application/pdf","sizeBytes":1,"createdAt":"","updatedAt":""}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewDocsCmd(d), stdout)
	root.SetArgs([]string{"docs", "show", "doc-1", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("docs show: %v", err)
	}

	var env struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if env.Summary != "Informe anual" {
		t.Errorf("summary = %q, se esperaba el título del documento", env.Summary)
	}
}

// TestDocsShowEnTextoMuestraLosMetadatos cubre el renderer de Text de `docs
// show`, forzando IsTTY:true por la misma razón que en docs list.
func TestDocsShowEnTextoMuestraLosMetadatos(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"doc-1","filename":"informe.pdf","title":"Informe anual","status":"READY","declaredMime":"application/pdf","sizeBytes":1,"createdAt":"2026-01-15T10:00:00Z","updatedAt":""}`))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewDocsCmd(d), stdout)
	root.SetArgs([]string{"docs", "show", "doc-1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("docs show: %v", err)
	}
	salida := stdout.String()
	if !strings.Contains(salida, "Informe anual") {
		t.Errorf("el texto debe mostrar el título, se obtuvo:\n%s", salida)
	}
	if !strings.Contains(salida, "informe.pdf") {
		t.Errorf("el texto debe mostrar el fichero, se obtuvo:\n%s", salida)
	}
	if !strings.Contains(salida, "READY") {
		t.Errorf("el texto debe mostrar el estado, se obtuvo:\n%s", salida)
	}
	if !strings.Contains(salida, "2026-01-15T10:00:00Z") {
		t.Errorf("el texto debe mostrar la fecha de creación, se obtuvo:\n%s", salida)
	}
	if strings.Contains(salida, "{") {
		t.Errorf("en modo texto no debe salir JSON crudo: %s", salida)
	}
}

// TestDocsShowEscapaLaBarraDelIDEnLaRuta comprueba que un identificador con
// una barra viaja escapado en la ruta HTTP. No se prueba con un espacio: un
// espacio se reescribe igual en la URL tanto si el CLI lo escapa como si no,
// así que no discrimina nada. Una barra sí: sin escapar, partiría la ruta en
// un segmento de más; hay que mirar la forma escapada de la petición
// (r.URL.EscapedPath / RequestURI), porque r.URL.Path ya viene desescapado
// por net/http y no distingue los dos casos.
func TestDocsShowEscapaLaBarraDelIDEnLaRuta(t *testing.T) {
	var pathEscapado string
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		pathEscapado = r.URL.EscapedPath()
		w.Write([]byte(`{"id":"carpeta/doc-1","filename":"a.pdf","status":"READY","declaredMime":"application/pdf","sizeBytes":1,"createdAt":"","updatedAt":""}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewDocsCmd(d), stdout)
	root.SetArgs([]string{"docs", "show", "carpeta/doc-1", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("docs show: %v", err)
	}
	if pathEscapado != "/v1/organizations/acme/documents/carpeta%2Fdoc-1" {
		t.Errorf("ruta escapada = %q, se esperaba la barra del id escapada como %%2F", pathEscapado)
	}
}

// TestDocsShowSinArgumentoEsErrorDeUso comprueba que la validación de
// argumentos pasa por exactArgs (corrección del brief), no por
// cobra.ExactArgs: sin él, el error sale en inglés, sin hint y con código de
// salida 1 en vez de 2 (uso incorrecto).
func TestDocsShowSinArgumentoEsErrorDeUso(t *testing.T) {
	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	root := testRoot(NewDocsCmd(d), stdout)
	root.SetArgs([]string{"docs", "show"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error sin argumento")
	}

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T", err)
	}
	if want := "Uso: calliope docs show <id>"; cliErr.Hint != want {
		t.Errorf("hint = %q, se esperaba %q", cliErr.Hint, want)
	}
	if strings.Contains(cliErr.Message, "accepts") || strings.Contains(cliErr.Message, "arg(s)") {
		t.Errorf("el mensaje no debe ser el de Cobra en inglés: %q", cliErr.Message)
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2 (uso incorrecto)", got)
	}
}

func TestDocsSearchUsaPOST(t *testing.T) {
	var metodo, ruta string
	var cuerpo map[string]any
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		metodo, ruta = r.Method, r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&cuerpo); err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(`[{"chunkId":1,"documentId":"doc-1","filename":"a.pdf","ordinal":0,"excerpt":"…","score":0.9}]`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewDocsCmd(d), stdout)
	root.SetArgs([]string{"docs", "search", "ventas", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("docs search: %v", err)
	}
	if metodo != http.MethodPost || ruta != "/v1/organizations/acme/search/documents" {
		t.Errorf("método/ruta = %s %s", metodo, ruta)
	}
	// El flag --limit no se pasó: debe usarse el valor por defecto (10), no 0.
	if cuerpo["query"] != "ventas" || cuerpo["limit"] != float64(10) {
		t.Errorf("cuerpo enviado = %+v, se esperaba query=ventas limit=10", cuerpo)
	}

	var env struct {
		Summary     string `json:"summary"`
		Data        []map[string]any
		Breadcrumbs []struct {
			Action string `json:"action"`
			Cmd    string `json:"cmd"`
		} `json:"breadcrumbs"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if len(env.Data) != 1 || env.Data[0]["documentId"] != "doc-1" || env.Data[0]["score"] != 0.9 {
		t.Errorf("fragmento inesperado: %+v", env.Data)
	}
	if env.Summary != "1 fragmentos" {
		t.Errorf("summary = %q, se esperaba %q", env.Summary, "1 fragmentos")
	}
	if len(env.Breadcrumbs) != 1 || env.Breadcrumbs[0].Action != "documento" ||
		env.Breadcrumbs[0].Cmd != "calliope docs show <id>" {
		t.Errorf("breadcrumbs inesperados: %+v", env.Breadcrumbs)
	}
}

// TestDocsSearchRespetaElLimite comprueba que --limit llega de verdad al
// backend: sin esta aserción, olvidar cablear el flag (o ignorarlo) dejaría
// la suite en verde igual, porque el valor por defecto (10) ya se comprueba
// en TestDocsSearchUsaPOST.
func TestDocsSearchRespetaElLimite(t *testing.T) {
	var cuerpo map[string]any
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&cuerpo); err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(`[]`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewDocsCmd(d), stdout)
	root.SetArgs([]string{"docs", "search", "ventas", "--limit", "3", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("docs search: %v", err)
	}
	if cuerpo["limit"] != float64(3) {
		t.Errorf("cuerpo enviado = %+v, se esperaba limit=3", cuerpo)
	}
}

// TestDocsSearchEnTextoMuestraTablaConFragmentoRecortado cubre el renderer de
// Text de `docs search`, forzando IsTTY:true, y comprueba que un excerpt
// largo se recorta con truncate.
func TestDocsSearchEnTextoMuestraTablaConFragmentoRecortado(t *testing.T) {
	largo := strings.Repeat("x", 90)
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"chunkId":1,"documentId":"doc-1","filename":"a.pdf","ordinal":0,"excerpt":"` + largo + `","score":0.876}]`))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewDocsCmd(d), stdout)
	root.SetArgs([]string{"docs", "search", "ventas"})
	if err := root.Execute(); err != nil {
		t.Fatalf("docs search: %v", err)
	}
	salida := stdout.String()
	for _, cabecera := range []string{"DOCUMENTO", "SCORE", "FRAGMENTO"} {
		if !strings.Contains(salida, cabecera) {
			t.Errorf("la tabla debe traer la cabecera %q, se obtuvo:\n%s", cabecera, salida)
		}
	}
	primeraLinea := strings.SplitN(salida, "\n", 2)[0]
	if i, j, k := strings.Index(primeraLinea, "DOCUMENTO"), strings.Index(primeraLinea, "SCORE"), strings.Index(primeraLinea, "FRAGMENTO"); !(i < j && j < k) {
		t.Errorf("orden de cabeceras inesperado: %q", primeraLinea)
	}
	if !strings.Contains(salida, "doc-1") || !strings.Contains(salida, "0.876") {
		t.Errorf("la tabla debe mostrar el documento y el score, se obtuvo:\n%s", salida)
	}
	// Orden dentro de la fila de datos: DOCUMENTO, luego SCORE, luego el
	// fragmento recortado. Un Contains global no detectaría que SCORE y
	// FRAGMENTO viajaran intercambiados dentro de la misma fila.
	var lineaDatos string
	for _, l := range strings.Split(salida, "\n") {
		if strings.Contains(l, "doc-1") {
			lineaDatos = l
			break
		}
	}
	if i, j, k := strings.Index(lineaDatos, "doc-1"), strings.Index(lineaDatos, "0.876"), strings.Index(lineaDatos, "xxx"); !(i < j && j < k) {
		t.Errorf("orden de la fila inesperado: %q", lineaDatos)
	}
	esperado := strings.Repeat("x", 70) + "…"
	if !strings.Contains(salida, esperado) {
		t.Errorf("el fragmento debe recortarse a 70 runas más puntos suspensivos, se obtuvo:\n%s", salida)
	}
	if strings.Contains(salida, largo) {
		t.Errorf("el fragmento largo no debe salir completo sin recortar: %s", salida)
	}
}

// TestDocsSearchSinArgumentoEsErrorDeUso es el equivalente para `docs
// search` de TestDocsShowSinArgumentoEsErrorDeUso.
func TestDocsSearchSinArgumentoEsErrorDeUso(t *testing.T) {
	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	root := testRoot(NewDocsCmd(d), stdout)
	root.SetArgs([]string{"docs", "search"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error sin argumento")
	}

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T", err)
	}
	if want := "Uso: calliope docs search <consulta>"; cliErr.Hint != want {
		t.Errorf("hint = %q, se esperaba %q", cliErr.Hint, want)
	}
	if strings.Contains(cliErr.Message, "accepts") || strings.Contains(cliErr.Message, "arg(s)") {
		t.Errorf("el mensaje no debe ser el de Cobra en inglés: %q", cliErr.Message)
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2 (uso incorrecto)", got)
	}
}

// --- Tests unitarios de los ayudantes de paquete ---

func TestTituloDeUsaElTituloSiExiste(t *testing.T) {
	titulo := "Informe anual"
	doc := sdk.DocumentResponse{Filename: "a.pdf", Title: &titulo}
	if got := titleOf(doc); got != "Informe anual" {
		t.Errorf("titleOf = %q, se esperaba %q", got, "Informe anual")
	}
}

func TestTituloDeCaeAlNombreDeFicheroSinTitulo(t *testing.T) {
	doc := sdk.DocumentResponse{Filename: "a.pdf"}
	if got := titleOf(doc); got != "a.pdf" {
		t.Errorf("titleOf = %q, se esperaba %q", got, "a.pdf")
	}
}

func TestTituloDeCaeAlNombreDeFicheroConTituloVacio(t *testing.T) {
	vacio := ""
	doc := sdk.DocumentResponse{Filename: "a.pdf", Title: &vacio}
	if got := titleOf(doc); got != "a.pdf" {
		t.Errorf("titleOf = %q, se esperaba %q", got, "a.pdf")
	}
}

// TestRecortaPorRunesNoPorBytes comprueba que truncate cuenta runas y no
// bytes: un carácter multibyte cortado por bytes produciría una cadena
// truncada en un punto distinto (o UTF-8 inválido).
func TestRecortaPorRunesNoPorBytes(t *testing.T) {
	original := strings.Repeat("ñ", 80) // cada ñ ocupa 2 bytes en UTF-8
	got := truncate(original, 70)
	r := []rune(got)
	if len(r) != 71 { // 70 runas + "…"
		t.Fatalf("truncate produjo %d runas, se esperaban 71: %q", len(r), got)
	}
	if r[len(r)-1] != '…' {
		t.Errorf("truncate debe terminar en …, obtuvo %q", got)
	}
	if string(r[:70]) != strings.Repeat("ñ", 70) {
		t.Errorf("truncate cortó por bytes en vez de por runas: %q", got)
	}
}

func TestRecortaNoTocaCadenasCortas(t *testing.T) {
	if got := truncate("corto", 70); got != "corto" {
		t.Errorf("truncate = %q, se esperaba %q sin cambios", got, "corto")
	}
}

// TestRecortaCadenaExactamenteEnElLimiteNoAñadePuntosSuspensivos comprueba el
// caso borde len(r) == n: la condición debe ser <=, no <.
func TestRecortaCadenaExactamenteEnElLimiteNoAñadePuntosSuspensivos(t *testing.T) {
	s := strings.Repeat("a", 70)
	if got := truncate(s, 70); got != s {
		t.Errorf("truncate = %q, se esperaba la cadena intacta en el límite exacto", got)
	}
}
