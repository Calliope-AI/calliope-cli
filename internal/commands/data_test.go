package commands

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Calliope-AI/calliope-cli/internal/auth"
	"github.com/Calliope-AI/calliope-cli/internal/output"
)

// respuestaSchema trae dos tablas cuyo TableName (el identificador real que
// va en el SQL) y Name (el nombre de negocio) son deliberadamente distintos
// entre sí -y distintos entre las dos tablas-, para que un cableado cruzado
// entre ambos campos sea observable en los tests.
const respuestaSchema = `{"organizationId":"acme","tables":[
  {"id":"tbl-1","tableName":"fct_ventas","name":"Ventas",
   "columns":[
     {"id":"col-1","name":"mes","type":"DATE","description":"Mes de la venta"},
     {"id":"col-2","name":"importe","type":"DECIMAL","description":"Importe vendido"}
   ]},
  {"id":"tbl-2","tableName":"dim_clientes","name":"Clientes",
   "columns":[
     {"id":"col-3","name":"id","type":"VARCHAR","description":"Identificador de cliente"}
   ]}
]}`

// respuestaSchemaTableNameIgualQueNombre es una tabla cuyo TableName y Name
// coinciden: sirve para comprobar que el renderer de texto no duplica la
// línea de "nombre de negocio" cuando no aporta información nueva.
const respuestaSchemaTableNameIgualQueNombre = `{"tables":[
  {"tableName":"dim_calendario","name":"dim_calendario",
   "columns":[{"name":"fecha","type":"DATE","description":"Fecha del calendario"}]}
]}`

// TestSchemaYQuerySonAtajosConRunE comprueba que `schema` y `query` son
// atajos, no grupos: definen RunE directamente y no tienen subcomandos.
func TestSchemaYQuerySonAtajosConRunE(t *testing.T) {
	schema := NewSchemaCmd(emptyDeps())
	if schema.RunE == nil {
		t.Error("schema es un atajo: debe definir RunE")
	}
	if len(schema.Commands()) != 0 {
		t.Errorf("schema no debe tener subcomandos, tiene %d", len(schema.Commands()))
	}

	query := NewQueryCmd(emptyDeps())
	if query.RunE == nil {
		t.Error("query es un atajo: debe definir RunE")
	}
	if len(query.Commands()) != 0 {
		t.Errorf("query no debe tener subcomandos, tiene %d", len(query.Commands()))
	}
}

// --- schema ---

func TestSchemaDevuelveTodasLasTablas(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations/acme/database/schema" {
			t.Errorf("ruta = %q", r.URL.Path)
		}
		w.Write([]byte(respuestaSchema))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewSchemaCmd(d), stdout)
	root.SetArgs([]string{"schema", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("schema: %v", err)
	}

	var env struct {
		Summary string `json:"summary"`
		Data    []struct {
			TableName string `json:"tableName"`
			Name      string `json:"name"`
			Columns   []struct {
				Name        string `json:"name"`
				Type        string `json:"type"`
				Description string `json:"description"`
			} `json:"columns"`
		} `json:"data"`
		Breadcrumbs []struct {
			Action string `json:"action"`
			Cmd    string `json:"cmd"`
		} `json:"breadcrumbs"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if len(env.Data) != 2 {
		t.Fatalf("se esperaban 2 tablas, hay %d", len(env.Data))
	}
	// TableName y Name se comprueban por separado: si el código confundiera
	// los dos campos (el error de fondo del brief original), esta aserción
	// lo detectaría porque los valores son distintos entre sí.
	t1, t2 := env.Data[0], env.Data[1]
	if t1.TableName != "fct_ventas" || t1.Name != "Ventas" {
		t.Errorf("tabla 1 inesperada: tableName=%q name=%q", t1.TableName, t1.Name)
	}
	if t2.TableName != "dim_clientes" || t2.Name != "Clientes" {
		t.Errorf("tabla 2 inesperada: tableName=%q name=%q", t2.TableName, t2.Name)
	}
	if len(t1.Columns) != 2 ||
		t1.Columns[0].Name != "mes" || t1.Columns[0].Type != "DATE" || t1.Columns[0].Description != "Mes de la venta" ||
		t1.Columns[1].Name != "importe" || t1.Columns[1].Type != "DECIMAL" || t1.Columns[1].Description != "Importe vendido" {
		t.Errorf("columnas de la tabla 1 inesperadas: %+v", t1.Columns)
	}
	if env.Summary != "2 tablas" {
		t.Errorf("summary = %q, se esperaba %q", env.Summary, "2 tablas")
	}
	quiero := map[string]string{"consultar": `calliope query "SELECT …"`, "preguntar": `calliope ask "<pregunta>"`}
	if len(env.Breadcrumbs) != len(quiero) {
		t.Fatalf("breadcrumbs = %+v, se esperaban %d", env.Breadcrumbs, len(quiero))
	}
	for _, b := range env.Breadcrumbs {
		if quiero[b.Action] != b.Cmd {
			t.Errorf("breadcrumb inesperado: %+v", b)
		}
	}
}

// TestSchemaConTablasNulasDaArrayVacio es el test de I8 de la oleada final
// para `schema`: si el backend manda `"tables":null` (organización recién
// creada, sin tablas conectadas aún), SchemaResponse.Tables queda en un
// slice nil, y antes ese nil llegaba intacto hasta el envelope y
// serializaba como `"data":null` en vez de `"data":[]` -que es lo que
// documenta el §6.1 del spec para una colección vacía.
func TestSchemaConTablasNulasDaArrayVacio(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"organizationId":"acme","tables":null}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewSchemaCmd(d), stdout)
	root.SetArgs([]string{"schema", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("schema: %v", err)
	}

	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if got := strings.TrimSpace(string(env.Data)); got != "[]" {
		t.Errorf(`data = %q, se esperaba "[]" (no "null")`, got)
	}
}

// TestSchemaTableFiltraEnCliente comprueba que --table filtra por TableName
// -el identificador real que va en el SQL-, no por Name -el nombre de
// negocio-: es la corrección central de esta tarea frente al brief original,
// que los confundía.
func TestSchemaTableFiltraEnCliente(t *testing.T) {
	llamadas := 0
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		llamadas++
		if r.URL.RawQuery != "" {
			t.Errorf("el filtro es en cliente: no debe enviarse query, se envió %q", r.URL.RawQuery)
		}
		w.Write([]byte(respuestaSchema))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewSchemaCmd(d), stdout)
	root.SetArgs([]string{"schema", "--table", "fct_ventas", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("schema --table: %v", err)
	}
	if llamadas != 1 {
		t.Errorf("llamadas al backend = %d, se esperaba 1", llamadas)
	}

	var env struct {
		Data []struct {
			TableName string `json:"tableName"`
			Name      string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if len(env.Data) != 1 || env.Data[0].TableName != "fct_ventas" || env.Data[0].Name != "Ventas" {
		t.Errorf("el filtro no se aplicó correctamente: %q", stdout.String())
	}
}

// TestSchemaTableFiltraPorNombreDeNegocioComoComodidad comprueba que filtrar
// por Name ("Ventas") también funciona, como comodidad, aunque no sea el
// identificador SQL. La tabla devuelta debe seguir exponiendo su TableName
// real, no el término de búsqueda.
func TestSchemaTableFiltraPorNombreDeNegocioComoComodidad(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(respuestaSchema))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewSchemaCmd(d), stdout)
	root.SetArgs([]string{"schema", "--table", "Ventas", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("schema --table: %v", err)
	}

	var env struct {
		Data []struct {
			TableName string `json:"tableName"`
			Name      string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if len(env.Data) != 1 || env.Data[0].TableName != "fct_ventas" || env.Data[0].Name != "Ventas" {
		t.Errorf("el filtro por nombre de negocio no se aplicó correctamente: %q", stdout.String())
	}
}

func TestSchemaTableFiltraSinDistinguirMayusculas(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(respuestaSchema))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewSchemaCmd(d), stdout)
	root.SetArgs([]string{"schema", "--table", "FCT_VENTAS", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("schema --table: %v", err)
	}

	var env struct {
		Data []struct {
			TableName string `json:"tableName"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if len(env.Data) != 1 || env.Data[0].TableName != "fct_ventas" {
		t.Errorf("el filtro debía ignorar mayúsculas: %q", stdout.String())
	}
}

func TestSchemaTableInexistenteEsNotFound(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(respuestaSchema))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewSchemaCmd(d), stdout)
	root.SetArgs([]string{"schema", "--table", "no-existe"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error con una tabla inexistente")
	}
	if !strings.Contains(err.Error(), "no-existe") {
		t.Errorf("el error debe nombrar la tabla: %q", err.Error())
	}

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T", err)
	}
	if cliErr.Hint == "" {
		t.Error("el error debería traer un hint con la forma de recuperar")
	}
	if got := output.ExitCodeFor(err); got != 4 {
		t.Errorf("código de salida = %d, se esperaba 4 (no encontrado)", got)
	}
}

// TestSchemaEnTextoMuestraTableNameDeFormaProminente cubre el renderer de
// Text, que depsWithServer (IsTTY:false) nunca ejercita en modo automático.
// Comprueba que TableName se muestra etiquetado sin ambigüedad como el
// identificador SQL, que precede al nombre de negocio (que se muestra
// aparte, también etiquetado), y el orden de columnas de la tabla.
func TestSchemaEnTextoMuestraTableNameDeFormaProminente(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(respuestaSchema))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewSchemaCmd(d), stdout)
	root.SetArgs([]string{"schema", "--table", "fct_ventas"})
	if err := root.Execute(); err != nil {
		t.Fatalf("schema: %v", err)
	}
	salida := stdout.String()

	if !strings.Contains(salida, "fct_ventas") || !strings.Contains(salida, "tabla SQL") {
		t.Errorf("no se muestra el TableName etiquetado como tabla SQL: %q", salida)
	}
	if !strings.Contains(salida, "Ventas") || !strings.Contains(salida, "nombre de negocio") {
		t.Errorf("no se muestra el nombre de negocio etiquetado: %q", salida)
	}
	// El TableName va primero: es la pieza que se escribe en SQL.
	if i, j := strings.Index(salida, "fct_ventas"), strings.Index(salida, "nombre de negocio"); i < 0 || j < 0 || i > j {
		t.Errorf("orden inesperado entre tableName y nombre de negocio: %q", salida)
	}

	for _, cabecera := range []string{"COLUMNA", "TIPO", "DESCRIPCIÓN"} {
		if !strings.Contains(salida, cabecera) {
			t.Errorf("falta la cabecera %q: %q", cabecera, salida)
		}
	}
	var lineaMes string
	for _, l := range strings.Split(salida, "\n") {
		if strings.Contains(l, "mes") && strings.Contains(l, "DATE") {
			lineaMes = l
		}
	}
	if lineaMes == "" {
		t.Fatalf("falta la fila de la columna 'mes': %q", salida)
	}
	// Orden completo de la fila: COLUMNA, TIPO, DESCRIPCIÓN.
	if i, j, k := strings.Index(lineaMes, "mes"), strings.Index(lineaMes, "DATE"), strings.Index(lineaMes, "Mes de la venta"); i < 0 || j < 0 || k < 0 || !(i < j && j < k) {
		t.Errorf("orden de la fila de columna inesperado: %q", lineaMes)
	}
}

// TestSchemaEnTextoNoDuplicaNombreDeNegocioCuandoCoincideConTableName cierra
// la rama contraria a la anterior: si Name coincide con TableName, no debe
// imprimirse una segunda línea redundante.
func TestSchemaEnTextoNoDuplicaNombreDeNegocioCuandoCoincideConTableName(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(respuestaSchemaTableNameIgualQueNombre))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewSchemaCmd(d), stdout)
	root.SetArgs([]string{"schema"})
	if err := root.Execute(); err != nil {
		t.Fatalf("schema: %v", err)
	}
	salida := stdout.String()
	if strings.Contains(salida, "nombre de negocio") {
		t.Errorf("no debe mostrarse la línea de nombre de negocio cuando coincide con TableName: %q", salida)
	}
	if !strings.Contains(salida, "dim_calendario") {
		t.Errorf("falta el TableName: %q", salida)
	}
}

// --- query ---

func TestQueryEnviaElSQLYDevuelveFilas(t *testing.T) {
	var cuerpo map[string]string
	var metodo, ruta string
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		metodo, ruta = r.Method, r.URL.Path
		json.NewDecoder(r.Body).Decode(&cuerpo)
		w.Write([]byte(`{"data":[{"mes":"2026-01","ventas":1200}]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT * FROM ventas", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("query: %v", err)
	}
	if metodo != http.MethodPost || ruta != "/v1/organizations/acme/query" {
		t.Errorf("método/ruta = %s %s, se esperaba POST /v1/organizations/acme/query", metodo, ruta)
	}
	if cuerpo["sql"] != "SELECT * FROM ventas" {
		t.Errorf("sql enviado = %q", cuerpo["sql"])
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
		t.Fatalf("se esperaba 1 fila, hay %d", len(env.Data))
	}
	// Se comprueban los valores de la fila, no solo la cantidad.
	if env.Data[0]["mes"] != "2026-01" || env.Data[0]["ventas"] != float64(1200) {
		t.Errorf("fila inesperada: %+v", env.Data[0])
	}
	// Singular a propósito (Diferidos #12 y #16): con 1 fila, "1 filas"
	// pluraliza mal.
	if env.Summary != "1 fila" {
		t.Errorf("summary = %q, se esperaba %q", env.Summary, "1 fila")
	}
	if len(env.Breadcrumbs) != 1 || env.Breadcrumbs[0].Action != "esquema" || env.Breadcrumbs[0].Cmd != "calliope schema" {
		t.Errorf("breadcrumb inesperado: %+v", env.Breadcrumbs)
	}
}

// TestQueryOutputSeReenviaAlBackend comprueba que --output llega al cuerpo de
// la petición (QueryRequest.output), sin desplazar al sql.
func TestQueryOutputSeReenviaAlBackend(t *testing.T) {
	var cuerpo map[string]string
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&cuerpo)
		w.Write([]byte(`{"data":[]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1", "--output", "arrow", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("query: %v", err)
	}
	if cuerpo["sql"] != "SELECT 1" || cuerpo["output"] != "arrow" {
		t.Errorf("cuerpo = %+v, se esperaba sql=\"SELECT 1\" output=arrow", cuerpo)
	}
}

// TestQueryCSVYOutputSonIndependientes comprueba que --csv y --output no se
// confunden entre sí cuando se usan a la vez: --csv es render local (decide
// qué se escribe en stdout) y --output se reenvía al backend en el cuerpo de
// la petición (decide qué formato pide el CLI); una no debe apagar ni
// sustituir a la otra.
func TestQueryCSVYOutputSonIndependientes(t *testing.T) {
	var cuerpo map[string]string
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&cuerpo)
		w.Write([]byte(`{"data":[{"mes":"2026-01","ventas":1200}]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1", "--output", "arrow", "--csv"})
	if err := root.Execute(); err != nil {
		t.Fatalf("query: %v", err)
	}
	// --output se reenvía al backend igual que sin --csv: --csv no lo apaga.
	if cuerpo["sql"] != "SELECT 1" || cuerpo["output"] != "arrow" {
		t.Errorf("cuerpo = %+v, se esperaba sql=\"SELECT 1\" output=arrow", cuerpo)
	}
	// --csv sigue siendo el render local, con independencia de --output.
	salida := stdout.String()
	if salida != "mes,ventas\n2026-01,1200\n" {
		t.Errorf("CSV inesperado:\n%q", salida)
	}
}

// TestQueryCSVEsRenderLocal comprueba que --csv no toca el cuerpo de la
// petición (a diferencia de --output) y que el CSV escrito en stdout es
// exacto: cabecera y fila, en el orden alfabético que impone columnsOf.
func TestQueryCSVEsRenderLocal(t *testing.T) {
	var cuerpo map[string]string
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&cuerpo)
		w.Write([]byte(`{"data":[{"mes":"2026-01","ventas":1200}]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1", "--csv"})
	if err := root.Execute(); err != nil {
		t.Fatalf("query --csv: %v", err)
	}
	if _, existe := cuerpo["output"]; existe {
		t.Error("--csv es render local: no debe enviarse output al backend")
	}
	if cuerpo["sql"] != "SELECT 1" {
		t.Errorf("sql enviado = %q", cuerpo["sql"])
	}
	salida := stdout.String()
	if salida != "mes,ventas\n2026-01,1200\n" {
		t.Errorf("CSV inesperado:\n%q", salida)
	}
}

// TestQueryEnTextoMuestraTablaConColumnasOrdenadas cubre el renderer de Text
// de `query` (nunca ejercitado en modo automático) y el orden alfabético que
// impone columnsOf, con dos filas de valores distintos entre sí.
func TestQueryEnTextoMuestraTablaConColumnasOrdenadas(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"id":"c-1","monto":300},{"id":"c-2","monto":450}]}`))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT id, monto FROM x"})
	if err := root.Execute(); err != nil {
		t.Fatalf("query: %v", err)
	}
	salida := stdout.String()

	primeraLinea := strings.SplitN(salida, "\n", 2)[0]
	if i, j := strings.Index(primeraLinea, "id"), strings.Index(primeraLinea, "monto"); i < 0 || j < 0 || i > j {
		t.Errorf("orden de cabeceras inesperado: %q", primeraLinea)
	}

	var linea1, linea2 string
	for _, l := range strings.Split(salida, "\n") {
		if strings.Contains(l, "c-1") {
			linea1 = l
		}
		if strings.Contains(l, "c-2") {
			linea2 = l
		}
	}
	if linea1 == "" || linea2 == "" {
		t.Fatalf("faltan filas de datos en la salida:\n%s", salida)
	}
	if i, j := strings.Index(linea1, "c-1"), strings.Index(linea1, "300"); i < 0 || j < 0 || i > j {
		t.Errorf("orden de la fila 1 inesperado: %q", linea1)
	}
	if i, j := strings.Index(linea2, "c-2"), strings.Index(linea2, "450"); i < 0 || j < 0 || i > j {
		t.Errorf("orden de la fila 2 inesperado: %q", linea2)
	}
	if strings.Contains(salida, "{") {
		t.Errorf("en modo texto no debe salir JSON crudo: %s", salida)
	}
}

// TestQueryResultadoNoJSONDevuelveErrorGenerico cubre la rama de error de
// resp.Rows(): un `data` que no es ni array ni cadena JSON no se puede
// interpretar como filas.
func TestQueryResultadoNoJSONDevuelveErrorGenerico(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":42}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error con data no interpretable")
	}

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T", err)
	}
	if cliErr.Code != output.CodeGeneric {
		t.Errorf("code = %q, se esperaba %q", cliErr.Code, output.CodeGeneric)
	}
	if !strings.Contains(cliErr.Hint, "--json") {
		t.Errorf("hint = %q, se esperaba que sugiriera --json", cliErr.Hint)
	}
}

func TestQuerySinSQLEsErrorDeUso(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error sin SQL")
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

// --- Ronda de correcciones 1/5: notación científica en números ---

// TestQueryCSVFormateaNumerosSinNotacionCientifica cubre IMPORTANT 1: los
// números de una fila llegan siempre como float64 tras json.Unmarshal (JSON
// no distingue enteros de decimales), y fmt.Sprintf("%v", ...) usa %g para
// float64, que cambia a notación exponencial con cifras grandes o con
// muchos decimales. Un CSV de cifras financieras en notación científica no
// lo interpreta de un vistazo ni una persona ni una hoja de cálculo.
func TestQueryCSVFormateaNumerosSinNotacionCientifica(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"entero":1200,"decimal":1234567.891234,"grande":123456789012345}]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1", "--csv"})
	if err := root.Execute(); err != nil {
		t.Fatalf("query --csv: %v", err)
	}
	salida := stdout.String()
	if strings.Contains(salida, "e+") || strings.Contains(salida, "E+") {
		t.Errorf("el CSV no debe usar notación científica: %q", salida)
	}
	// Columnas en orden alfabético (columnsOf): decimal, entero, grande. El
	// entero (1200) debe salir sin ceros de más ("1200", no "1200.0000...").
	if salida != "decimal,entero,grande\n1234567.891234,1200,123456789012345\n" {
		t.Errorf("CSV inesperado:\n%q", salida)
	}
}

// TestQueryEnTextoFormateaNumerosSinNotacionCientifica es la variante en el
// renderer de Text (IsTTY:true) del test anterior.
func TestQueryEnTextoFormateaNumerosSinNotacionCientifica(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"entero":1200,"decimal":1234567.891234,"grande":123456789012345}]}`))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("query: %v", err)
	}
	salida := stdout.String()
	if strings.Contains(salida, "e+") || strings.Contains(salida, "E+") {
		t.Errorf("la tabla de texto no debe usar notación científica: %q", salida)
	}
	for _, esperado := range []string{"1234567.891234", "1200", "123456789012345"} {
		if !strings.Contains(salida, esperado) {
			t.Errorf("falta el valor %q en la salida: %q", esperado, salida)
		}
	}
}

// --- Ronda de correcciones 1/5: nulos ---

// TestQueryCSVNuloEsCampoVacio cubre IMPORTANT 2: un NULL del backend debe
// salir como campo vacío en el CSV (RFC 4180), no como la cadena "<nil>" que
// da fmt.Sprintf("%v", nil) por defecto -indistinguible de un valor de
// negocio real con esa cadena-. Se relee el CSV con encoding/csv (no con
// strings.Contains) para comprobar el campo vacío sin ambigüedad de
// comillas o de formato.
func TestQueryCSVNuloEsCampoVacio(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"a":"x","b":null}]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1", "--csv"})
	if err := root.Execute(); err != nil {
		t.Fatalf("query --csv: %v", err)
	}
	if strings.Contains(stdout.String(), "<nil>") {
		t.Errorf("el CSV no debe contener la cadena <nil>: %q", stdout.String())
	}
	registros, err := csv.NewReader(strings.NewReader(stdout.String())).ReadAll()
	if err != nil {
		t.Fatalf("el CSV no se pudo parsear: %v (%q)", err, stdout.String())
	}
	if len(registros) != 2 || len(registros[1]) != 2 || registros[1][0] != "x" || registros[1][1] != "" {
		t.Errorf("fila inesperada: %+v", registros)
	}
}

// TestQueryCSVColumnaAusenteEnFilaHeterogeneaEsCampoVacio cubre el otro
// origen de un nulo: una fila heterogénea (con menos columnas que otras) en
// vez de un NULL explícito del backend. columnsOf calcula la unión de
// columnas de todas las filas, así que la fila que no trae "b" debe salir
// con ese campo vacío igualmente.
func TestQueryCSVColumnaAusenteEnFilaHeterogeneaEsCampoVacio(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"a":"x","b":"y"},{"a":"z"}]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1", "--csv"})
	if err := root.Execute(); err != nil {
		t.Fatalf("query --csv: %v", err)
	}
	registros, err := csv.NewReader(strings.NewReader(stdout.String())).ReadAll()
	if err != nil {
		t.Fatalf("el CSV no se pudo parsear: %v (%q)", err, stdout.String())
	}
	if len(registros) != 3 {
		t.Fatalf("se esperaban 3 líneas (cabecera + 2 filas), hay %d: %+v", len(registros), registros)
	}
	if registros[2][0] != "z" || registros[2][1] != "" {
		t.Errorf("fila heterogénea inesperada: %+v", registros[2])
	}
}

// TestQueryEnTextoNuloNoEsNilLiteral es la variante en el renderer de Text
// (IsTTY:true) de TestQueryCSVNuloEsCampoVacio. El texto usa "NULL" en vez
// de un campo vacío (que en una tabla alineada con espacios pasaría
// desapercibido), pero el requisito es el mismo: nunca "<nil>".
func TestQueryEnTextoNuloNoEsNilLiteral(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"a":"x","b":null}]}`))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("query: %v", err)
	}
	salida := stdout.String()
	if strings.Contains(salida, "<nil>") {
		t.Errorf("la tabla de texto no debe mostrar <nil>: %q", salida)
	}
	if !strings.Contains(salida, "NULL") {
		t.Errorf("la tabla de texto debe marcar el nulo como NULL: %q", salida)
	}
}

// TestQueryEnTextoColumnaAusenteEnFilaHeterogeneaNoEsNilLiteral es la
// variante en Text de TestQueryCSVColumnaAusenteEnFilaHeterogeneaEsCampoVacio.
func TestQueryEnTextoColumnaAusenteEnFilaHeterogeneaNoEsNilLiteral(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"a":"x","b":"y"},{"a":"z"}]}`))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("query: %v", err)
	}
	salida := stdout.String()
	if strings.Contains(salida, "<nil>") {
		t.Errorf("la tabla de texto no debe mostrar <nil>: %q", salida)
	}
	var lineaZ string
	for _, l := range strings.Split(salida, "\n") {
		if strings.Contains(l, "z") {
			lineaZ = l
		}
	}
	if lineaZ == "" || !strings.Contains(lineaZ, "NULL") {
		t.Errorf("la fila de 'z' (sin columna b) debe mostrar NULL: %q", lineaZ)
	}
}

// --- Ronda de correcciones 1/5: --csv frente a los modos de salida ---

// TestQueryCSVConJSONEsErrorDeUso cubre IMPORTANT 3: --csv cortaba en
// silencio antes de llegar a ctx.Render (lo único que mira --json/--quiet
// /--md/--jq), así que combinarlo con --json devolvía CSV sin aviso. Debe
// ser un error de uso -y, por tratarse de un error de uso, sin llegar a
// llamar al backend-.
func TestQueryCSVConJSONEsErrorDeUso(t *testing.T) {
	llamadas := 0
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		llamadas++
		w.Write([]byte(`{"data":[]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1", "--csv", "--json"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error al combinar --csv con --json")
	}
	if llamadas != 0 {
		t.Errorf("no debía llamarse al backend con un error de uso: llamadas = %d", llamadas)
	}

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T", err)
	}
	if cliErr.Code != output.CodeUsage {
		t.Errorf("code = %q, se esperaba %q", cliErr.Code, output.CodeUsage)
	}
	if !strings.Contains(cliErr.Message, "--json") {
		t.Errorf("el mensaje debe nombrar --json: %q", cliErr.Message)
	}
	if cliErr.Hint == "" {
		t.Error("se esperaba un hint que dijera cuál elegir")
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2 (uso incorrecto)", got)
	}
}

func TestQueryCSVConQuietEsErrorDeUso(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1", "--csv", "--quiet"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error al combinar --csv con --quiet")
	}
	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T", err)
	}
	if cliErr.Code != output.CodeUsage || !strings.Contains(cliErr.Message, "--quiet") {
		t.Errorf("error inesperado: %+v", cliErr)
	}
	if cliErr.Hint == "" {
		t.Error("se esperaba un hint que dijera cuál elegir")
	}
}

func TestQueryCSVConMdEsErrorDeUso(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1", "--csv", "--md"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error al combinar --csv con --md")
	}
	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T", err)
	}
	if cliErr.Code != output.CodeUsage || !strings.Contains(cliErr.Message, "--md") {
		t.Errorf("error inesperado: %+v", cliErr)
	}
	if cliErr.Hint == "" {
		t.Error("se esperaba un hint que dijera cuál elegir")
	}
}

func TestQueryCSVConJQEsErrorDeUso(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1", "--csv", "--jq", ".data"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error al combinar --csv con --jq")
	}
	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T", err)
	}
	if cliErr.Code != output.CodeUsage || !strings.Contains(cliErr.Message, "--jq") {
		t.Errorf("error inesperado: %+v", cliErr)
	}
	if cliErr.Hint == "" {
		t.Error("se esperaba un hint que dijera cuál elegir")
	}
}

// --- Ronda de correcciones 1/5: cero filas en modo texto ---

// TestQueryEnTextoConCeroFilasNoQuedaVacio cubre MINOR 4: sin filas, el
// modo texto no debe quedar en blanco -eso no distingue "la consulta
// funcionó y no hay filas" de "algo se rompió en silencio"-, así que debe
// decir explícitamente que no hay filas.
func TestQueryEnTextoConCeroFilasNoQuedaVacio(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[]}`))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("query: %v", err)
	}
	salida := stdout.String()
	if strings.TrimSpace(salida) == "" {
		t.Error("con 0 filas, el modo texto no debe quedar vacío")
	}
	if !strings.Contains(salida, "Sin filas") {
		t.Errorf("se esperaba un mensaje que dijera que no hay filas: %q", salida)
	}
}

// TestQueryConDataNuloEnJSONDaArrayVacio es el test de I8 de la oleada
// final para `query`: el backend puede mandar `"data":null` (no solo
// `"data":[]`) cuando la consulta no devuelve filas, y QueryResponse.Rows
// lo deja en un slice nil (ver TestQueryDataNuloExplicitoDevuelveCero en
// internal/sdk/api_test.go). Antes ese nil llegaba intacto hasta el
// envelope y serializaba como `"data":null`: la receta del propio
// SKILL.md, `calliope query "..." --jq '.data[]'`, daba "cannot iterate
// over: null" con exit 2 en vez de no imprimir nada.
func TestQueryConDataNuloEnJSONDaArrayVacio(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":null}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewQueryCmd(d), stdout)
	root.SetArgs([]string{"query", "SELECT 1 WHERE false", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("query: %v", err)
	}

	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if got := strings.TrimSpace(string(env.Data)); got != "[]" {
		t.Errorf(`data = %q, se esperaba "[]" (no "null")`, got)
	}
}
