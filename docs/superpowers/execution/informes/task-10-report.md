# Task 10 — Informe

## Estado

DONE

## Qué se implementó

- `internal/sdk/models.go` — tipos de respuesta para `/ask`, `/search/documents`,
  `/documents`, `/knowledge/concepts`, `/rules`, `/database/schema`, `/query`,
  `/v1/auth/me` y `/v1/organizations`.
- `internal/sdk/api.go` — un método por endpoint sobre `*Client`, usando
  `c.Do` y `c.OrgPath` de la Task 9 sin modificarlos.
- `internal/sdk/api_test.go` — 12 tests nuevos (los del brief + 3 de la
  corrección de tipos).
- Fixtures en `internal/sdk/testdata/`: `ask.json`, `documents.json`,
  `query_data_como_cadena.json` (literales del brief), más `me.json`,
  `organizations.json` y `schema.json` (nuevos, para los tipos corregidos).

## Tipos corregidos — sin confirmar contra el backend en vivo

**Advertencia explícita, tal como pide el encargo:** `Me`, `OrgInfo`,
`Organization` y `SchemaResponse`/`SchemaTable`/`SchemaColumn` **no** están
verificados contra el backend real de Calliope Data — no hay API key
disponible. Se sustituyeron por las definiciones de
`task-10-tipos-corregidos.md`, que a su vez se derivaron de
`calliope-data-ui/types/index.ts` (`BackendUser`, `BackendOrgInfo`,
`Organization`, `SchemaResponse`, `SchemaTable`, `SchemaColumn`) — el cliente
que hoy consume esos mismos endpoints en producción, pero no el backend
mismo. Verifiqué independientemente esa correspondencia leyendo el fichero
TypeScript directamente (líneas ~15-23 para `Organization`, ~884-895 para
`BackendUser`/`BackendOrgInfo`, ~114-145 para `SchemaResponse`/`SchemaTable`/
`SchemaColumn`) antes de escribir `models.go`, y coincide exactamente con lo
que documenta la corrección.

No se ejecutó el Step 7 del brief (curl contra `data-0.calliope.so`) por
instrucción explícita del controlador. Las tareas 12–18, que consumen estos
tipos, heredan esta misma reserva: si aparece una API key, hay que repetir el
Step 7 y ajustar `Me`, `Organization` y `SchemaResponse`/`SchemaTable`/
`SchemaColumn` a lo que el backend devuelva realmente.

Los demás tipos (`AskResponse` y afines, `DocumentSearchResult`,
`DocumentResponse`/`DocumentPage`, `ConceptGraphResponse`/
`ConceptDetailResponse`, `Rule`) se copiaron literalmente del brief: están
verificados contra `calliope-data-mcp/src/calliope/types.ts`, que sí llama a
esos endpoints en producción.

## Desviaciones respecto al brief

1. **Tipos de `Me`, `Organization`, `SchemaColumn`, `SchemaTable`,
   `SchemaResponse`** sustituidos por los de `task-10-tipos-corregidos.md`
   (vinculante, manda sobre el brief). Se añadió el tipo nuevo `OrgInfo`
   (no estaba en el brief original) porque `Me.Organizations` en la UI es
   `BackendOrgInfo[]`, con un campo `userRole` que `Organization` no tiene.
2. **Glosario:** `servidorConFixture` → `serverWithFixture` (entrada ya
   fijada en `glosario.md`). Es el único identificador de nivel de paquete
   en español que traía el brief para esta tarea. El resto de nombres de
   paquete (tipos, métodos, campos) ya estaban en inglés en el brief; las
   variables locales dentro de los cuerpos de función se dejaron tal como
   las escribe el brief (`cuerpo`, `out`, `ruta`, `visto`, `filas`,
   `comoCadena`, `q`, `quiero`, `clave`), conforme al alcance acotado del
   glosario ("variables locales... se quedan como las escribe el brief").
   Los nombres de test se dejaron en español, también por indicación del
   glosario.
3. **3 tests añadidos** más allá del brief (`TestMeDecodificaElPerfil`,
   `TestListOrganizationsDecodificaLaLista`,
   `TestSchemaDecodificaTableNameYNamePorSeparado`) y uno adicional de ruta
   (`TestSchemaEnviaLaRutaConScopeDeOrganizacion`), pedidos explícitamente
   por `task-10-tipos-corregidos.md` ("Añade también testdata/me.json y
   testdata/organizations.json... y un test de decodificación para cada
   uno... En el test del esquema, comprueba explícitamente que TableName y
   Name se decodifican por separado y pueden diferir"). El test del esquema
   hace exactamente esa comprobación (`tbl.TableName == tbl.Name` sería un
   error) usando un fixture donde `tableName: "fct_ventas"` y
   `name: "Ventas"` difieren a propósito.
4. Todo lo demás (rutas, verbos HTTP, forma de `AskResponse` y compañía,
   filtros de `ListDocumentsParams`, normalización de `QueryResponse.Rows()`)
   es copia literal del brief, incluidos sus comentarios en español.

## Resumen de tests

`go test ./internal/sdk/ -v`: **20/20 PASS** (12 nuevos de `api_test.go` +
8 existentes de `client_test.go`, Task 9, sin tocar).

`go test ./... -race`, `go vet ./...`, `gofmt -l .`: limpios.

## Mutaciones (verificación de costura), sobre copia fuera del repo

Copia hecha en `/private/tmp/.../scratchpad/mutation-copy-task10` (fuera de
`/Users/j10/repositories/calliope/calliope-cli`), compilada y con el
paquete `internal/sdk` en verde antes de mutar. Cada mutación se aplicó
sobre `models.go` o `api.go`, se corrió *solo* el test que debía
protegerla, se confirmó el fallo, y se revirtió desde una copia de
respaldo del fichero original antes de la siguiente. Al terminar, diff de
ambos ficheros contra los del repo real = idéntico; repo real sin tocar
(`git status` solo mostraba los ficheros nuevos, no comiteados en ese
momento). La copia y los backups se borraron después.

| # | Test protegido | Mutación | Resultado |
|---|---|---|---|
| 1 | `TestAskDecodificaLaRespuestaYSusFuentes` | `AskDocumentSource.DocumentID`: `json:"documentId"` → `json:"docId"` | FAIL: `DocumentID = "" — comprueba el tag json camelCase` |
| 2 | `TestAskEnviaLaPreguntaEnElCuerpo` | En `Ask`, `c.OrgPath(org, "/ask")` → `"/ask2"` | FAIL: `ruta = "/v1/organizations/acme/ask2"` |
| 3 | `TestListDocumentsDecodificaLaPagina` | `DocumentResponse.SizeBytes`: `json:"sizeBytes"` → `json:"byteSize"` | FAIL: `documento mal decodificado: {... SizeBytes:0 ...}` |
| 4 | `TestListDocumentsPasaLosFiltrosPorQuery` | En `query()`, se elimina el bloque `if p.Tag != "" { q.Set("tag", p.Tag) }` | FAIL: `query tag = "", se esperaba "finanzas"` |
| 5 | `TestListDocumentsOmiteLosFiltrosVacios` | En `query()`, `if p.Page > 0` → `if p.Page >= 0` (se cuela `page=0` sin pedirlo) | FAIL: `query = "page=0", sin filtros no debe enviarse ninguno` |
| 6 | `TestQueryAceptaDataComoCadenaJSON` | En `Rows()`, se elimina la rama que deshace `data` cuando viene como cadena JSON (solo intenta el array) | FAIL: `Rows: json: cannot unmarshal string into Go value of type []map[string]interface {}` |
| 6b | *(control)* `TestQueryAceptaDataComoArray` | misma mutación que #6 | PASS — confirma que la mutación 6 es específica de la rama "cadena", no rompe la rama "array" |
| 7 | `TestQuerySinDataDevuelveCero` | En `Rows()`, el atajo `if len(r.Data) == 0 \|\| string(r.Data) == "null"` → `if false` | FAIL: `Rows sin data no debe fallar: unexpected end of JSON input` |
| 8 | `TestMeDecodificaElPerfil` | `Me.ID`: `json:"id"` → `json:"userId"` (reproduce el bug exacto del brief original que corrigió `task-10-tipos-corregidos.md`) | FAIL: `ID = "" — el campo es id, no userId` |
| 9 | `TestListOrganizationsDecodificaLaLista` | `Organization.Status`: `json:"status"` → `json:"state"` | FAIL: `Status/Description mal decodificados: {... Status: ...}` |
| 10 | `TestSchemaDecodificaTableNameYNamePorSeparado` | `SchemaTable.TableName`: `json:"tableName"` → `json:"table_name"` | FAIL: `TableName = "", se esperaba fct_ventas` |
| 11 | `TestSchemaEnviaLaRutaConScopeDeOrganizacion` | En `Schema`, `c.OrgPath(org, "/database/schema")` → `"/database/schemas"` | FAIL: `ruta = "/v1/organizations/acme/database/schemas"` |
| 12 | `TestSchemaDecodificaTableNameYNamePorSeparado` | `SchemaColumn.IsPrimaryKey`: `json:"isPrimaryKey"` → `json:"primaryKey"` | FAIL: `columna 0 mal decodificada: {... IsPrimaryKey:false ...}` |

Las 12 mutaciones (más el control 6b) fallaron exactamente en el test que
debían proteger; ninguna quedó sin detectar.

**Hallazgo notable durante la mutación 1:** la primera versión de esa
mutación cambió solo la capitalización del tag (`documentId` →
`documentID`) y el test **siguió pasando** — `encoding/json` empareja
claves JSON con tags de struct de forma insensible a mayúsculas como
mecanismo de *fallback* cuando no hay coincidencia exacta. Tuve que
cambiar la mutación a una clave genuinamente distinta (`docId`) para que
representara una costura real. Esto es relevante para el resto del CLI:
un tag con la capitalización equivocada pero las mismas letras **no** se
detecta por decodificación fallida ni, en general, por un test que solo
compare valores — porque el valor sí llega, por la vía del fallback. La
única red de seguridad real contra ese tipo concreto de error es la
revisión visual del tag contra el contrato documentado (aquí,
`calliope-data-ui/types/index.ts`), que es justo lo que se hizo para los
tres tipos corregidos.

## Dudas / puntos para quien revise

- Ninguna duda sobre la implementación: los 3 tipos corregidos se
  verificaron independientemente contra `calliope-data-ui/types/index.ts`
  antes de escribir código, y coinciden con lo que documenta
  `task-10-tipos-corregidos.md`.
- Reitero la reserva que pide el encargo: `Me`, `Organization`,
  `SchemaResponse` (y sus tipos anidados) **no están confirmados contra el
  backend en vivo**. Si en algún momento hay una API key disponible, hace
  falta repetir el Step 7 del brief original antes de dar por buenas las
  Tasks 12–18 que dependen de `Schema`, `Me` o `ListOrganizations`.
- No hubo que añadir ninguna entrada nueva al glosario: `servidorConFixture`
  ya estaba fijada, y el resto de identificadores de nivel de paquete del
  brief ya venían en inglés.

---

## Ronda de correcciones 1/5

### Adjudicación del coordinador

Dos huecos Important:

1. **Cinco métodos sin test:** `SearchDocuments`, `GetDocument`, `ListConcepts`, `GetConcept`, `ListRules`. El brief dejó de traer tests a partir del tercer método, pero TDD es restricción global y me correspondía cerrarlo por iniciativa propia — igual que hice con los tipos corregidos. El coordinador verificó por mutación que 7 roturas distintas en estos cinco métodos pasaban el suite de 20 tests sin fallar nada.
2. **`Rows()` con dos caminos sin cubrir:** cadena vacía (`{"data":""}`) y cadena con JSON inválido dentro. Además, `{"data":null}` explícito solo se cubría indirectamente vía la clave ausente.
3. **Minor:** el guard `string(r.Data) == "null"` es redundante porque `json.Unmarshal` de `null` en un slice ya produce `nil` sin error.

Acepto los tres puntos. Son exactamente el tipo de hueco que "TDD sin excepciones" debía cerrar sin necesidad de que el brief lo pidiera explícitamente método por método.

### Cambios aplicados

1. **`internal/sdk/models.go`:** guard de `Rows()` simplificado a `if len(r.Data) == 0 { return nil, nil }`, con comentario explicando por qué `data:null` no necesita caso aparte (Go ya lo maneja) y remitiendo al test que lo fija.
2. **Fixtures nuevos** en `internal/sdk/testdata/`: `search_documents.json`, `document.json`, `concepts.json`, `concept_detail.json`, `rules.json`.
3. **`internal/sdk/api_test.go`:**
   - `serverWithFixture` extendido para capturar también el cuerpo de la petición de forma fiable: el handler lee `r.Body`, lo repone como un nuevo `io.NopCloser` tanto en `r` como en el clon, porque `(*http.Request).Clone` no duplica el flujo subyacente del `Body` — sin esto, leer `capturar.Body` después de que el handler termina habría sido no determinista.
   - **6 tests nuevos** para los 5 métodos sin cobertura (`SearchDocuments` se probó con dos: decodificación+cuerpo, y límite por defecto con `0` y negativo), cada uno aseverando sobre verbo, ruta exacta, y valores decodificados de la respuesta — no solo que la petición se hizo.
   - `GetDocument` y `GetConcept` comprueban además que un id con caracteres especiales llega escapado, usando `visto.RequestURI` (el request-target crudo tal como llega al servidor) en vez de `visto.URL.Path`, porque `URL.Path` decodifica `%2F` de vuelta a `/`, indistinguible de un segmento adicional.
   - **3 tests nuevos** para `Rows()`: `data:null` explícito, `data:""`, y `data` como cadena con JSON inválido dentro (con `errors.As` comprobando que el error es un `*json.SyntaxError`, no cualquier error).

### Desviación encontrada durante la propia verificación por mutación

El primer intento de `TestGetConceptDecodificaElDetalleYEscapaElID` usaba `id = "a b"` (solo espacio) y comprobaba `visto.RequestURI`. La mutación que quita `url.PathEscape(id)` en `GetConcept` **no rompió el test**. Investigué con un programa mínimo (`http.NewRequest` con un espacio crudo en el path) y confirmé que `net/url` reescapa automáticamente un espacio crudo a `%20` en `EscapedPath()`/`RequestURI()` **exista o no** una llamada a `url.PathEscape` en el código de la aplicación — el espacio no es un carácter estructural de la URL, así que Go lo neutraliza igual al serializar la petición. La barra (`/`) es distinta: si llega cruda, Go la trata como separador de segmento real y no la reescapa a `%2F`. Cambié el id de prueba a `"a b/c"` (espacio y barra) y `EscapedPath` esperado a `.../a%20b%2Fc`; con eso, la misma mutación sí rompe el test (ver mutación 13 más abajo). El test corregido ya está en `internal/sdk/api_test.go`.

Esto es la tercera vez en esta tarea que una propiedad "obvia" de una librería estándar de Go rompe la intuición de qué constituye una aserción discriminante: primero el emparejamiento insensible a mayúsculas de tags JSON (ronda inicial), y ahora el re-escapado automático de espacios en `net/url`. Documento ambos por si son relevantes para el resto del proyecto: **un test de "esto debe llegar escapado" solo es válido si usa un carácter que la capa de transporte no neutraliza por su cuenta** — en URLs, eso descarta el espacio y exige algo como `/`, `?` o `#`.

### Mutaciones (ronda 1), sobre copia fresca fuera del repo

Copia en `/private/tmp/.../scratchpad/mutation-copy-task10-r1`, compilada y con el paquete en verde antes de mutar. Cada mutación se aplicó, se corrió *solo* el test que debía protegerla, se confirmó el resultado, y se revirtió desde backup del fichero original antes de la siguiente. Al terminar, `api.go` y `models.go` de la copia son idénticos a los del repo real (verificado por diff programático); la copia y los backups se borraron después.

| # | Test protegido | Mutación | Resultado |
|---|---|---|---|
| 1 | `TestSearchDocumentsEnviaLaConsultaYDecodificaLosResultados` | Ruta `"/search/documents"` → `"/search/documents2"` | FAIL: `ruta = ".../search/documents2"` |
| 2 | `TestSearchDocumentsEnviaLaConsultaYDecodificaLosResultados` | Verbo `POST` → `GET` | FAIL: `método = "GET", se esperaba POST` |
| 3 | `TestSearchDocumentsEnviaLaConsultaYDecodificaLosResultados` | `DocumentSearchResult.DocumentID`: `json:"documentId"` → `json:"docId"` | FAIL: `DocumentID:` vacío en el resultado decodificado |
| 4 | `TestSearchDocumentsUsaLimitePorDefectoConCeroYNegativo` | `if limit <= 0` → `if limit < 0` (se cuela `limit=0`) | FAIL: `limit=0: cuerpo.limit = 0, se esperaba 10` |
| 5 | `TestGetDocumentDecodificaElDocumentoYEscapaElID` | Ruta `"/documents/"` → `"/document/"` | FAIL: `RequestURI = ".../document/doc%2F1"` |
| 6 | `TestGetDocumentDecodificaElDocumentoYEscapaElID` | Se quita `url.PathEscape(id)` | FAIL: `RequestURI = ".../documents/doc/1"` (sin `%2F`) |
| 7 | `TestGetDocumentDecodificaElDocumentoYEscapaElID` | `DocumentResponse.Status`: `json:"status"` → `json:"state"` | FAIL: `Status:` vacío |
| 8 | `TestGetDocumentDecodificaElDocumentoYEscapaElID` | Verbo `GET` → `POST` | FAIL: `método = "POST", se esperaba GET` |
| 9 | `TestListConceptsDecodificaElGrafo` | Ruta `"/knowledge/concepts"` → `"/knowledge/concept"` | FAIL: `ruta = ".../knowledge/concept"` |
| 10 | `TestListConceptsDecodificaElGrafo` | `GraphConceptNode.IsActive`: `json:"isActive"` → `json:"active"` | FAIL: `IsActive:false` |
| 11 | `TestListConceptsDecodificaElGrafo` | Verbo `GET` → `POST` | FAIL: `método = "POST", se esperaba GET` |
| 12 | `TestGetConceptDecodificaElDetalleYEscapaElID` | Ruta `"/knowledge/concepts/"` → `"/knowledge/concept/"` | FAIL: `RequestURI` con `concept/` |
| 13 | `TestGetConceptDecodificaElDetalleYEscapaElID` | Se quita `url.PathEscape(id)` (test ya corregido, id `"a b/c"`) | FAIL: `RequestURI = ".../a%20b/c"` (la barra sin escapar) |
| 14 | `TestGetConceptDecodificaElDetalleYEscapaElID` | `ConceptResponse.Name`: `json:"name"` → `json:"title"` | FAIL: `Name:` vacío |
| 15 | `TestGetConceptDecodificaElDetalleYEscapaElID` | Verbo `GET` → `POST` | FAIL: `método = "POST", se esperaba GET` |
| 16 | `TestListRulesDecodificaLasReglas` | Ruta `"/rules"` → `"/rule"` | FAIL: `ruta = ".../rule"` |
| 17 | `TestListRulesDecodificaLasReglas` | `Rule.Status`: `json:"status"` → `json:"state"` | FAIL: `Status:` vacío |
| 18 | `TestListRulesDecodificaLasReglas` | Verbo `GET` → `POST` | FAIL: `método = "POST", se esperaba GET` |
| 19 | `TestQueryDataComoCadenaVaciaDevuelveCero` | Se quita `if comoCadena == "" { return nil, nil }` | FAIL: `unexpected end of JSON input` (el error exacto que citó el coordinador) |
| 20 | `TestQueryDataComoCadenaConJSONInvalidoDevuelveError` | Se ignora el error del `json.Unmarshal` final | FAIL: `se esperaba un error...` |
| 21 | `TestQueryDataNuloExplicitoDevuelveCero` | El camino "array" trata `data:null` como error explícito (único tipo de bug real que este test puede atrapar; ver nota abajo) | FAIL: `Rows con data:null no debe fallar: data nulo no soportado` |

Las 21 mutaciones fallaron exactamente en el test que debían proteger (incluida la 13, tras corregir el id de prueba).

**Nota sobre la mutación 21 y la redundancia del guard `"null"`:** dado que `json.Unmarshal` de `null` no produce error ni en un slice (`filas` queda `nil`) ni en un `string` (`comoCadena` queda como estaba, `""`), *cualquier* variante razonable de `Rows()` que no rechace `null` explícitamente produce el mismo resultado `(nil, nil)` sin importar por qué rama pase. Antes de la mutación 21 probé variantes de control-de-flujo más "naturales" (p. ej. exigir `len(filas) > 0` en la rama array, o distinguir por el primer byte de `r.Data`) y **ninguna** rompía este test — no porque el test esté mal calibrado, sino porque de verdad no hay diferencia observable entre esas rutas para `null`. La única mutación que lo rompe es una que decide explícitamente tratar `null` como error, que es justo el único tipo de regresión real contra el que este test protege.

**Comprobación confirmatoria del Remate (no es una mutación adversarial, es la prueba de que la simplificación es segura):** reintroduje el guard eliminado (`string(r.Data) == "null"`) sobre la copia externa y corrí el suite completo de `internal/sdk` (30 tests) — **los 30 siguen en verde**, confirmando empíricamente que la comprobación era redundante y que quitarla (Remate) no cambia el comportamiento observable en ningún test existente.

### Verificación final

`go test ./... -race -v`: **30/30 PASS** en `internal/sdk` (11 nuevos de esta ronda + 19 de la entrega anterior), resto del módulo sin cambios. `go vet ./...` y `gofmt -l .` limpios.

### Dudas

Ninguna. Los dos Important y el Minor quedaron cerrados y verificados por mutación; la única sorpresa (mutación 13 no detectada en su primera versión) se investigó, se explicó con una prueba mínima reproducible, y se corrigió el test antes de seguir — documentado arriba para que quede como precedente del proyecto.
