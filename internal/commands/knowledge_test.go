package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/output"
)

// TestConceptsYRulesSonGruposDeRecursos comprueba el comportamiento de
// `concepts` y `rules` como grupos de recursos (C3 de la oleada final):
// pelados muestran la ayuda con exit 0; con un subcomando que no existe,
// fallan con exit 2. También se comprueba el número de subcomandos: sin
// esto, olvidar un AddCommand pasaría inadvertido. Ver el comentario
// equivalente en auth_test.go sobre por qué esto ya no se asevera
// comprobando que RunE sea nil.
func TestConceptsYRulesSonGruposDeRecursos(t *testing.T) {
	concepts := NewConceptsCmd(emptyDeps())
	if len(concepts.Commands()) != 2 {
		t.Errorf("concepts debe tener 2 subcomandos (list, show), tiene %d", len(concepts.Commands()))
	}
	rules := NewRulesCmd(emptyDeps())
	if len(rules.Commands()) != 1 {
		t.Errorf("rules debe tener 1 subcomando (list), tiene %d", len(rules.Commands()))
	}

	for _, caso := range []struct {
		nombre string
		nuevo  func() *cobra.Command
	}{
		{"concepts", func() *cobra.Command { return NewConceptsCmd(emptyDeps()) }},
		{"rules", func() *cobra.Command { return NewRulesCmd(emptyDeps()) }},
	} {
		var out bytes.Buffer
		root := testRoot(caso.nuevo(), &out)
		root.SetArgs([]string{caso.nombre})
		if err := root.Execute(); err != nil {
			t.Fatalf("%s pelado debe salir con 0 (ayuda), dio error: %v", caso.nombre, err)
		}
		if !strings.Contains(out.String(), "Usage:") {
			t.Errorf("%s pelado debe imprimir la ayuda, se obtuvo: %q", caso.nombre, out.String())
		}

		out.Reset()
		root = testRoot(caso.nuevo(), &out)
		root.SetArgs([]string{caso.nombre, "esto-no-existe"})
		err := root.Execute()
		var cliErr *output.CLIError
		if !errors.As(err, &cliErr) {
			t.Fatalf("%s con subcomando desconocido: el error debería ser un *output.CLIError, fue %T (%v)", caso.nombre, err, err)
		}
		if cliErr.Hint == "" {
			t.Errorf("%s: el error debería traer un hint", caso.nombre)
		}
		if got := output.ExitCodeFor(err); got != 2 {
			t.Errorf("%s: código de salida = %d, se esperaba 2 (uso incorrecto)", caso.nombre, got)
		}
	}
}

// --- concepts list ---

func TestConceptsListDevuelveElGrafo(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations/acme/knowledge/concepts" {
			t.Errorf("ruta = %q", r.URL.Path)
		}
		w.Write([]byte(`{"concepts":[
			{"id":"c-1","name":"Cliente","isActive":true,"recordCount":42,"sourceCount":7},
			{"id":"c-2","name":"Pedido","isActive":false}
		]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewConceptsCmd(d), stdout)
	root.SetArgs([]string{"concepts", "list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("concepts list: %v", err)
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
	if len(env.Data) != 2 {
		t.Fatalf("se esperaban 2 conceptos en data, hay %d", len(env.Data))
	}
	// Se comprueban los valores de cada concepto, no solo la cantidad: esto
	// detecta un tag json mal escrito o un campo que no llega a la salida.
	// Los dos conceptos tienen valores distintos entre sí (nombre, isActive,
	// recordCount presente/ausente) para que un cableado cruzado sea
	// observable.
	c1, c2 := env.Data[0], env.Data[1]
	if c1["id"] != "c-1" || c1["name"] != "Cliente" || c1["isActive"] != true {
		t.Errorf("concepto 1 inesperado: %+v", c1)
	}
	if c1["recordCount"] != float64(42) || c1["sourceCount"] != float64(7) {
		t.Errorf("concepto 1 debe traer recordCount/sourceCount: %+v", c1)
	}
	if c2["id"] != "c-2" || c2["name"] != "Pedido" || c2["isActive"] != false {
		t.Errorf("concepto 2 inesperado: %+v", c2)
	}
	if c2["recordCount"] != nil {
		t.Errorf("concepto 2 no debe traer recordCount: %+v", c2)
	}
	if env.Summary != "2 conceptos" {
		t.Errorf("summary = %q, se esperaba %q", env.Summary, "2 conceptos")
	}
	quiero := map[string]string{
		"detalle":   "calliope concepts show <id>",
		"preguntar": `calliope ask "<pregunta>"`,
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

// TestConceptsListConConceptosNulosDaArrayVacio es el test de I8 de la
// oleada final para `concepts list`: si el backend manda `"concepts":null`
// (una ontología recién creada, por ejemplo), ConceptGraphResponse.Concepts
// queda en un slice nil, y antes ese nil llegaba intacto hasta el envelope
// y serializaba como `"data":null` en vez de `"data":[]` -lo que documenta
// el §6.1 del spec para una colección vacía.
func TestConceptsListConConceptosNulosDaArrayVacio(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"concepts":null}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewConceptsCmd(d), stdout)
	root.SetArgs([]string{"concepts", "list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("concepts list: %v", err)
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

// TestConceptsListEnTextoMuestraTabla cubre el renderer de Text de `concepts
// list`, que depsWithServer (IsTTY:false) nunca ejercita en modo automático.
// Se fija IsTTY:true a propósito para forzar ese camino, y se comprueba tanto
// el caso con recordCount (número) como sin él ("—"), y activo/no activo.
func TestConceptsListEnTextoMuestraTabla(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"concepts":[
			{"id":"c-1","name":"Cliente","isActive":true,"recordCount":42},
			{"id":"c-2","name":"Pedido","isActive":false}
		]}`))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewConceptsCmd(d), stdout)
	root.SetArgs([]string{"concepts", "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("concepts list: %v", err)
	}
	salida := stdout.String()
	for _, cabecera := range []string{"ID", "CONCEPTO", "REGISTROS", "ACTIVO"} {
		if !strings.Contains(salida, cabecera) {
			t.Errorf("la tabla debe traer la cabecera %q, se obtuvo:\n%s", cabecera, salida)
		}
	}
	primeraLinea := strings.SplitN(salida, "\n", 2)[0]
	if i, j, k, l := strings.Index(primeraLinea, "ID"), strings.Index(primeraLinea, "CONCEPTO"), strings.Index(primeraLinea, "REGISTROS"), strings.Index(primeraLinea, "ACTIVO"); !(i < j && j < k && k < l) {
		t.Errorf("orden de cabeceras inesperado: %q", primeraLinea)
	}

	var lineaC1, lineaC2 string
	for _, l := range strings.Split(salida, "\n") {
		if strings.Contains(l, "c-1") {
			lineaC1 = l
		}
		if strings.Contains(l, "c-2") {
			lineaC2 = l
		}
	}
	if lineaC1 == "" || lineaC2 == "" {
		t.Fatalf("faltan filas de concepto en la salida:\n%s", salida)
	}
	// c-1 tiene recordCount: debe mostrarse el número, no "—", y "sí" porque
	// está activo.
	if !strings.Contains(lineaC1, "Cliente") || !strings.Contains(lineaC1, "42") || !strings.Contains(lineaC1, "sí") {
		t.Errorf("fila de c-1 inesperada: %q", lineaC1)
	}
	// Orden completo de la fila: ID, CONCEPTO, REGISTROS, ACTIVO. Se incluye
	// la posición relativa de ID frente a CONCEPTO (no solo frente a
	// REGISTROS/ACTIVO): sin ella, una fila construida como {c.Name, c.ID,
	// registros, activo} -ID y CONCEPTO intercambiados entre sí- pasaría
	// inadvertida, porque "c-1" seguiría apareciendo antes que "42" y "sí"
	// aunque fuera en segunda posición en vez de la primera.
	if g, h, i, j := strings.Index(lineaC1, "c-1"), strings.Index(lineaC1, "Cliente"), strings.Index(lineaC1, "42"), strings.Index(lineaC1, "sí"); g < 0 || h < 0 || i < 0 || j < 0 || !(g < h && h < i && i < j) {
		t.Errorf("orden de la fila c-1 inesperado: %q", lineaC1)
	}
	// c-2 no tiene recordCount: debe caer a "—", y "no" porque no está activo.
	if !strings.Contains(lineaC2, "Pedido") || !strings.Contains(lineaC2, "—") || !strings.Contains(lineaC2, "no") {
		t.Errorf("fila de c-2 inesperada: %q", lineaC2)
	}
	if strings.Contains(salida, "{") {
		t.Errorf("en modo texto no debe salir JSON crudo: %s", salida)
	}
}

// --- concepts show ---

func TestConceptsShowDevuelveAtributos(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations/acme/knowledge/concepts/c-1" {
			t.Errorf("ruta = %q", r.URL.Path)
		}
		w.Write([]byte(`{"concept":{"id":"c-1","name":"Cliente","description":"Persona u organización que compra","isActive":true},
		                "attributes":[
		                  {"id":"a-1","name":"email","description":"Correo de contacto","isActive":true},
		                  {"id":"a-2","name":"telefono","isActive":false}
		                ]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewConceptsCmd(d), stdout)
	root.SetArgs([]string{"concepts", "show", "c-1", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("concepts show: %v", err)
	}

	var env struct {
		Summary string `json:"summary"`
		Data    struct {
			Concept struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
				IsActive    bool   `json:"isActive"`
			} `json:"concept"`
			Attributes []struct {
				ID          string  `json:"id"`
				Name        string  `json:"name"`
				Description *string `json:"description"`
				IsActive    bool    `json:"isActive"`
			} `json:"attributes"`
		} `json:"data"`
		Breadcrumbs []struct {
			Action string `json:"action"`
			Cmd    string `json:"cmd"`
		} `json:"breadcrumbs"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, stdout.String())
	}
	if env.Data.Concept.ID != "c-1" || env.Data.Concept.Name != "Cliente" ||
		env.Data.Concept.Description != "Persona u organización que compra" || !env.Data.Concept.IsActive {
		t.Errorf("concepto inesperado: %+v", env.Data.Concept)
	}
	if len(env.Data.Attributes) != 2 {
		t.Fatalf("se esperaban 2 atributos, hay %d", len(env.Data.Attributes))
	}
	// Los dos atributos tienen valores distintos entre sí (nombre, isActive,
	// descripción presente/ausente) para que un cableado cruzado sea
	// observable.
	a1, a2 := env.Data.Attributes[0], env.Data.Attributes[1]
	if a1.ID != "a-1" || a1.Name != "email" || a1.Description == nil || *a1.Description != "Correo de contacto" || !a1.IsActive {
		t.Errorf("atributo 1 inesperado: %+v", a1)
	}
	if a2.ID != "a-2" || a2.Name != "telefono" || a2.Description != nil || a2.IsActive {
		t.Errorf("atributo 2 inesperado: %+v", a2)
	}
	if env.Summary != "Cliente · 2 atributos" {
		t.Errorf("summary = %q, se esperaba %q", env.Summary, "Cliente · 2 atributos")
	}
	if len(env.Breadcrumbs) != 1 || env.Breadcrumbs[0].Action != "preguntar" ||
		env.Breadcrumbs[0].Cmd != `calliope ask "<pregunta sobre Cliente>"` {
		t.Errorf("breadcrumbs inesperados: %+v", env.Breadcrumbs)
	}
}

// TestConceptsShowConDescripcionLaImprimeAntesDeLaTabla cubre el renderer de
// Text de `concepts show` cuando el concepto sí tiene descripción: debe
// aparecer en su propia línea, seguida de una línea en blanco, antes de la
// tabla de atributos.
func TestConceptsShowConDescripcionLaImprimeAntesDeLaTabla(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"concept":{"id":"c-1","name":"Cliente","description":"Persona u organización que compra","isActive":true},"attributes":[]}`))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewConceptsCmd(d), stdout)
	root.SetArgs([]string{"concepts", "show", "c-1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("concepts show: %v", err)
	}
	lineas := strings.Split(stdout.String(), "\n")
	if len(lineas) < 4 {
		t.Fatalf("salida con muy pocas líneas: %q", stdout.String())
	}
	if lineas[0] != "Cliente" {
		t.Errorf("primera línea = %q, se esperaba el nombre del concepto", lineas[0])
	}
	if lineas[1] != "Persona u organización que compra" {
		t.Errorf("segunda línea = %q, se esperaba la descripción", lineas[1])
	}
	if lineas[2] != "" {
		t.Errorf("tercera línea = %q, se esperaba vacía tras la descripción", lineas[2])
	}
	if !strings.Contains(lineas[3], "ATRIBUTO") {
		t.Errorf("cuarta línea debe ser la cabecera de la tabla, se obtuvo %q", lineas[3])
	}
}

// TestConceptsShowSinDescripcionNoImprimeLineaDeDescripcion es el reverso de
// la anterior: sin descripción (ni presente ni vacía) no debe imprimirse una
// línea de más antes de la tabla.
func TestConceptsShowSinDescripcionNoImprimeLineaDeDescripcion(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"concept":{"id":"c-1","name":"Cliente","isActive":true},"attributes":[{"id":"a-1","name":"email","isActive":true}]}`))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewConceptsCmd(d), stdout)
	root.SetArgs([]string{"concepts", "show", "c-1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("concepts show: %v", err)
	}
	lineas := strings.Split(stdout.String(), "\n")
	if len(lineas) < 3 {
		t.Fatalf("salida con muy pocas líneas: %q", stdout.String())
	}
	if lineas[0] != "Cliente" {
		t.Errorf("primera línea = %q, se esperaba el nombre del concepto", lineas[0])
	}
	if lineas[1] != "" {
		t.Errorf("segunda línea = %q, se esperaba vacía al no haber descripción", lineas[1])
	}
	if !strings.Contains(lineas[2], "ATRIBUTO") {
		t.Errorf("tercera línea debe ser la cabecera de la tabla, se obtuvo %q", lineas[2])
	}
	// El atributo debe mostrarse con su ACTIVO tras la DESCRIPCIÓN en la fila.
	var lineaAtributo string
	for _, l := range lineas[3:] {
		if strings.Contains(l, "email") {
			lineaAtributo = l
			break
		}
	}
	if !strings.Contains(lineaAtributo, "sí") {
		t.Errorf("fila del atributo debe mostrar activo=sí, se obtuvo %q", lineaAtributo)
	}
}

// TestConceptsShowEnTextoOrdenaColumnasDeAtributos comprueba el orden de las
// columnas dentro de la fila de un atributo: ATRIBUTO, DESCRIPCIÓN, ACTIVO.
// Se usa una descripción no vacía a propósito: sin un valor de descripción
// observable, una fila construida como {desc, a.Name, activo} -ATRIBUTO y
// DESCRIPCIÓN intercambiadas entre sí- pasaría inadvertida, porque las dos
// cadenas seguirían apareciendo en la salida igual.
func TestConceptsShowEnTextoOrdenaColumnasDeAtributos(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"concept":{"id":"c-1","name":"Cliente","isActive":true},"attributes":[{"id":"a-1","name":"email","description":"Correo de contacto","isActive":true}]}`))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewConceptsCmd(d), stdout)
	root.SetArgs([]string{"concepts", "show", "c-1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("concepts show: %v", err)
	}
	var lineaAtributo string
	for _, l := range strings.Split(stdout.String(), "\n") {
		if strings.Contains(l, "email") {
			lineaAtributo = l
			break
		}
	}
	if lineaAtributo == "" {
		t.Fatalf("falta la fila del atributo en la salida:\n%s", stdout.String())
	}
	if i, j, k := strings.Index(lineaAtributo, "email"), strings.Index(lineaAtributo, "Correo de contacto"), strings.Index(lineaAtributo, "sí"); i < 0 || j < 0 || k < 0 || !(i < j && j < k) {
		t.Errorf("orden de la fila de atributo inesperado: %q", lineaAtributo)
	}
}

// TestConceptsShowConDescripcionVaciaNoImprimeLineaDeDescripcion es el caso
// borde entre las dos anteriores: una descripción presente pero vacía (""),
// no ausente (nil). Sin comprobar `!= ""` además de `!= nil`, se imprimiría
// una línea en blanco de más antes de la línea separadora.
func TestConceptsShowConDescripcionVaciaNoImprimeLineaDeDescripcion(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"concept":{"id":"c-1","name":"Cliente","description":"","isActive":true},"attributes":[]}`))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewConceptsCmd(d), stdout)
	root.SetArgs([]string{"concepts", "show", "c-1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("concepts show: %v", err)
	}
	lineas := strings.Split(stdout.String(), "\n")
	if len(lineas) < 3 {
		t.Fatalf("salida con muy pocas líneas: %q", stdout.String())
	}
	if lineas[0] != "Cliente" {
		t.Errorf("primera línea = %q, se esperaba el nombre del concepto", lineas[0])
	}
	if lineas[1] != "" {
		t.Errorf("segunda línea = %q, se esperaba vacía con descripción vacía", lineas[1])
	}
	if !strings.Contains(lineas[2], "ATRIBUTO") {
		t.Errorf("tercera línea debe ser la cabecera de la tabla, no una línea en blanco de más: %q", lineas[2])
	}
}

// TestConceptsShowEscapaLaBarraDelIDEnLaRuta comprueba que un identificador
// con una barra viaja escapado en la ruta HTTP. No se prueba con un espacio:
// un espacio se reescribe igual en la URL tanto si el CLI lo escapa como si
// no, así que no discrimina nada. Una barra sí: sin escapar, partiría la ruta
// en un segmento de más.
func TestConceptsShowEscapaLaBarraDelIDEnLaRuta(t *testing.T) {
	var pathEscapado string
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		pathEscapado = r.URL.EscapedPath()
		w.Write([]byte(`{"concept":{"id":"carpeta/c-1","name":"Cliente","isActive":true},"attributes":[]}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewConceptsCmd(d), stdout)
	root.SetArgs([]string{"concepts", "show", "carpeta/c-1", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("concepts show: %v", err)
	}
	if pathEscapado != "/v1/organizations/acme/knowledge/concepts/carpeta%2Fc-1" {
		t.Errorf("ruta escapada = %q, se esperaba la barra del id escapada como %%2F", pathEscapado)
	}
}

// TestConceptsShowSinArgumentoEsErrorDeUso comprueba que la validación de
// argumentos pasa por exactArgs (corrección al brief), no por
// cobra.ExactArgs: sin él, el error sale en inglés, sin hint y con código de
// salida 1 en vez de 2 (uso incorrecto).
func TestConceptsShowSinArgumentoEsErrorDeUso(t *testing.T) {
	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	root := testRoot(NewConceptsCmd(d), stdout)
	root.SetArgs([]string{"concepts", "show"})
	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error sin argumento")
	}

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T", err)
	}
	if want := "Uso: calliope concepts show <id>"; cliErr.Hint != want {
		t.Errorf("hint = %q, se esperaba %q", cliErr.Hint, want)
	}
	if strings.Contains(cliErr.Message, "accepts") || strings.Contains(cliErr.Message, "arg(s)") {
		t.Errorf("el mensaje no debe ser el de Cobra en inglés: %q", cliErr.Message)
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2 (uso incorrecto)", got)
	}
}

// --- rules list ---

func TestRulesListDevuelveLasReglas(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations/acme/rules" {
			t.Errorf("ruta = %q", r.URL.Path)
		}
		w.Write([]byte(`[
			{"id":"r-1","name":"Cliente activo","details":"Compró en 90 días","category":"Ventas","status":"ACTIVE"},
			{"id":"r-2","name":"Pedido cancelado","details":"Cancelado antes de enviar","status":"DRAFT"}
		]`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewRulesCmd(d), stdout)
	root.SetArgs([]string{"rules", "list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("rules list: %v", err)
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
	if len(env.Data) != 2 {
		t.Fatalf("se esperaban 2 reglas en data, hay %d", len(env.Data))
	}
	// Las dos reglas tienen todos sus campos distintos entre sí (nombre,
	// details, category presente/ausente, status) para que un cableado
	// cruzado entre campos sea observable.
	r1, r2 := env.Data[0], env.Data[1]
	if r1["id"] != "r-1" || r1["name"] != "Cliente activo" || r1["details"] != "Compró en 90 días" ||
		r1["category"] != "Ventas" || r1["status"] != "ACTIVE" {
		t.Errorf("regla 1 inesperada: %+v", r1)
	}
	if r2["id"] != "r-2" || r2["name"] != "Pedido cancelado" || r2["details"] != "Cancelado antes de enviar" ||
		r2["category"] != nil || r2["status"] != "DRAFT" {
		t.Errorf("regla 2 inesperada: %+v", r2)
	}
	if env.Summary != "2 reglas" {
		t.Errorf("summary = %q, se esperaba %q", env.Summary, "2 reglas")
	}
	if len(env.Breadcrumbs) != 1 || env.Breadcrumbs[0].Action != "conceptos" ||
		env.Breadcrumbs[0].Cmd != "calliope concepts list" {
		t.Errorf("breadcrumbs inesperados: %+v", env.Breadcrumbs)
	}
}

// TestRulesListConRespuestaNulaDaArrayVacio es el test de I8 de la oleada
// final para `rules list`: el backend responde con el array directamente
// (no envuelto en un objeto, a diferencia de docs/concepts), así que una
// organización sin reglas puede mandar el cuerpo `null` a secas. ListRules
// decodifica esa respuesta en un slice de Go que queda en nil, y antes ese
// nil llegaba intacto hasta el envelope y serializaba como `"data":null`
// en vez de `"data":[]` -lo que documenta el §6.1 del spec para una
// colección vacía.
func TestRulesListConRespuestaNulaDaArrayVacio(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`null`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewRulesCmd(d), stdout)
	root.SetArgs([]string{"rules", "list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("rules list: %v", err)
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

// TestRulesListEnTextoMuestraTabla cubre el renderer de Text de `rules list`,
// forzando IsTTY:true, y comprueba tanto el caso con categoría como sin ella
// (cadena vacía).
func TestRulesListEnTextoMuestraTabla(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[
			{"id":"r-1","name":"Cliente activo","details":"Compró en 90 días","category":"Ventas","status":"ACTIVE"},
			{"id":"r-2","name":"Pedido cancelado","details":"Cancelado antes de enviar","status":"DRAFT"}
		]`))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewRulesCmd(d), stdout)
	root.SetArgs([]string{"rules", "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("rules list: %v", err)
	}
	salida := stdout.String()
	for _, cabecera := range []string{"REGLA", "CATEGORÍA", "ESTADO", "DETALLE"} {
		if !strings.Contains(salida, cabecera) {
			t.Errorf("la tabla debe traer la cabecera %q, se obtuvo:\n%s", cabecera, salida)
		}
	}
	primeraLinea := strings.SplitN(salida, "\n", 2)[0]
	if i, j, k, l := strings.Index(primeraLinea, "REGLA"), strings.Index(primeraLinea, "CATEGORÍA"), strings.Index(primeraLinea, "ESTADO"), strings.Index(primeraLinea, "DETALLE"); !(i < j && j < k && k < l) {
		t.Errorf("orden de cabeceras inesperado: %q", primeraLinea)
	}

	var lineaR1, lineaR2 string
	for _, l := range strings.Split(salida, "\n") {
		if strings.Contains(l, "Cliente activo") {
			lineaR1 = l
		}
		if strings.Contains(l, "Pedido cancelado") {
			lineaR2 = l
		}
	}
	if lineaR1 == "" || lineaR2 == "" {
		t.Fatalf("faltan filas de regla en la salida:\n%s", salida)
	}
	if !strings.Contains(lineaR1, "Ventas") || !strings.Contains(lineaR1, "ACTIVE") || !strings.Contains(lineaR1, "Compró en 90 días") {
		t.Errorf("fila de r-1 inesperada: %q", lineaR1)
	}
	// Orden dentro de la fila: REGLA, CATEGORÍA, ESTADO, DETALLE. Se incluye
	// la posición del nombre de la regla (no solo categoría/estado/detalle):
	// sin ella, una fila construida como {cat, r.Name, r.Status, detalle} —
	// REGLA y CATEGORÍA intercambiadas entre sí— pasaría inadvertida, porque
	// "Ventas" seguiría apareciendo antes que "ACTIVE" y "Compró".
	if h, i, j, k := strings.Index(lineaR1, "Cliente activo"), strings.Index(lineaR1, "Ventas"), strings.Index(lineaR1, "ACTIVE"), strings.Index(lineaR1, "Compró"); h < 0 || i < 0 || j < 0 || k < 0 || !(h < i && i < j && j < k) {
		t.Errorf("orden de la fila r-1 inesperado: %q", lineaR1)
	}
	// r-2 no tiene categoría: la celda debe quedar vacía, no un valor
	// inventado, y el resto de la fila sigue en su sitio.
	if !strings.Contains(lineaR2, "DRAFT") || !strings.Contains(lineaR2, "Cancelado antes de enviar") {
		t.Errorf("fila de r-2 inesperada: %q", lineaR2)
	}
	if strings.Contains(salida, "{") {
		t.Errorf("en modo texto no debe salir JSON crudo: %s", salida)
	}
}

// TestRulesListRecortaDetallesLargos comprueba que un detail largo se recorta
// con truncate (60 runas) en el render de texto.
func TestRulesListRecortaDetallesLargos(t *testing.T) {
	largo := strings.Repeat("x", 90)
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":"r-1","name":"Regla larga","details":"` + largo + `","status":"ACTIVE"}]`))
	})
	d.IsTTY = true
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewRulesCmd(d), stdout)
	root.SetArgs([]string{"rules", "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("rules list: %v", err)
	}
	salida := stdout.String()
	esperado := strings.Repeat("x", 60) + "…"
	if !strings.Contains(salida, esperado) {
		t.Errorf("el detalle debe recortarse a 60 runas más puntos suspensivos, se obtuvo:\n%s", salida)
	}
	if strings.Contains(salida, largo) {
		t.Errorf("el detalle largo no debe salir completo sin recortar: %s", salida)
	}
}

// --- yesNo ---

func TestActivoDevuelveSiParaVerdadero(t *testing.T) {
	if got := yesNo(true); got != "sí" {
		t.Errorf("yesNo(true) = %q, se esperaba %q", got, "sí")
	}
}

func TestActivoDevuelveNoParaFalso(t *testing.T) {
	if got := yesNo(false); got != "no" {
		t.Errorf("yesNo(false) = %q, se esperaba %q", got, "no")
	}
}

func emptyDeps() appctx.Deps { return appctx.Deps{} }
