# Informe — Task 17: Comandos `schema` y `query`

## Estado

DONE

## Commit

`583cde7` — feat: comandos schema y query
(3 ficheros: `internal/commands/data.go`, `internal/commands/data_test.go`, `internal/cli/root.go`)

## Resumen de tests

`go test ./... -race` → PASS en los 9 paquetes (7 con tests); 15/15 tests propios de `data_test.go` en verde; `go vet ./...` limpio; `gofmt -l .` sin salida; `go mod tidy` sin cambios. 22 mutaciones dirigidas verificadas sobre una copia completa del repo fuera del árbol de trabajo: las 22 hicieron fallar al menos un test o el build (una mutación sobrevivió a la primera pasada y quedó cerrada con un test nuevo — ver abajo).

## Qué se hizo

- **Step 1-2 (TDD)**: escribí `internal/commands/data_test.go` y confirmé el fallo esperado: `go test ./internal/commands/ -run "Schema|Query" -v` → `undefined: NewSchemaCmd` / `undefined: NewQueryCmd`.
- **Step 3**: `internal/commands/data.go` — `NewSchemaCmd`, `filterTables`, `NewQueryCmd`, `columnsOf`, `writeCSV` — copiado del brief con las tres correcciones del controlador y el ajuste de identificadores del glosario (`filtrarTablas`→`filterTables`, `columnasDe`→`columnsOf`, `escribirCSV`→`writeCSV`; variables locales y comentarios en español, tal como el brief los escribe).
- **Step 4**: `internal/cli/root.go` — añadidos `commands.NewSchemaCmd(d)` y `commands.NewQueryCmd(d)` a `root.AddCommand(...)`.
- Verificado contra el binario real (`go build ./cmd/calliope`): `calliope --help` lista `schema` y `query`; `calliope schema --help` muestra `--table`; `calliope query --help` muestra `--output`/`--csv` y el `Long` con la referencia a `calliope schema` y `calliope ask`.

## Corrección 1 — TableName vs Name

El brief filtraba y mostraba tablas por `t.Name` (el nombre de negocio), cuando el identificador real que va en el SQL es `t.TableName`. Esto es peligroso: un agente que lea el esquema y use el nombre de negocio en su SQL produciría una consulta inválida contra una tabla que no existe con ese nombre.

Aplicado como indicaba el encargo:

- `filterTables(tablas []sdk.SchemaTable, nombre string)` casa con `strings.EqualFold` contra **`TableName`** primero, y contra `Name` como comodidad (`||`), para que filtrar por el nombre de negocio también funcione.
- El renderer de `Text` de `schema` imprime primero `t.TableName` etiquetado explícitamente como `[tabla SQL]` (línea propia, en cabecera de cada tabla). Si `Name` no está vacío y difiere de `TableName`, se imprime una segunda línea `nombre de negocio: <Name>` — nunca cuando coinciden, para no duplicar información redundante.
- El mensaje de "tabla inexistente" (`output.CodeNotFound`) sigue nombrando literalmente el término buscado (`tabla`, el flag `--table` tal cual lo escribió el usuario), sin tocar.
- El envelope JSON expone la tabla completa (`sdk.SchemaTable`, con `TableName` y `Name` como campos separados con sus tags `tableName`/`name`), así que un consumidor JSON nunca ve los dos campos fundidos.

Las fixtures de test usan `TableName`/`Name` deliberadamente distintos entre sí (`fct_ventas`/`Ventas`, `dim_clientes`/`Clientes`) y entre las dos tablas del fixture, precisamente para que un cableado cruzado entre ambos campos sea observable — fue lo que la mutación M1 (reintroducir el bug del brief: filtrar solo por `Name`) confirmó al fallar tres tests distintos.

## Corrección 2 — `exactArgs` en `query`

`Args: exactArgs(1)` en vez de `cobra.ExactArgs(1)`, igual que en `ask`/`docs show`/`docs search`/`concepts show`. Cubierto por `TestQuerySinSQLEsErrorDeUso` (mensaje en español, hint con la forma de uso, código de salida 2) y verificado por mutación (M11): revertir a `cobra.ExactArgs(1)` hace que el error deje de ser un `*output.CLIError`.

## Corrección 3 — `--output` vs `--csv`

`--output` se pasa tal cual a `ctx.Client.Query(..., formato)`, que ya lo reenvía en el cuerpo de la petición (`QueryRequest.output`, código de Task 10, no tocado). `--csv` es puramente local: cuando está activo, la función corta antes de construir el `presenter.Result` y llama a `writeCSV(ctx.Deps.Stdout, columnas, filas)` directamente, sin pasar por `ctx.Render`.

Fijado con tests que comprueban el cuerpo de la petición HTTP real recibido por el servidor de prueba (no solo que la llamada se hizo):

- `TestQueryOutputSeReenviaAlBackend`: `--output arrow` sin `--csv` → `cuerpo["output"] == "arrow"`.
- `TestQueryCSVEsRenderLocal`: `--csv` sin `--output` → `"output"` **ausente** del cuerpo, y el CSV en stdout es exacto (`"mes,ventas\n2026-01,1200\n"`).
- `TestQueryCSVYOutputSonIndependientes` (añadido durante la verificación por mutación, ver abajo): **ambos flags a la vez** → `--output` se sigue reenviando (`cuerpo["output"] == "arrow"`) y el CSV se sigue escribiendo igual, sin que uno apague al otro.

## Tests añadidos más allá del brief, y por qué

El brief trae 5 tests (`TestSchemaDevuelveTodasLasTablas`, `TestSchemaTableFiltraEnCliente`, `TestSchemaTableInexistenteEsNotFound`, `TestQueryEnviaElSQLYDevuelveFilas`, `TestQueryOutputSeReenviaAlBackend`, `TestQueryCSVEsRenderLocal` — 6 en realidad). Siguiendo los cuatro avisos del encargo añadí:

1. **Aviso 1 (Text nunca se ejercita en modo automático)**: `TestSchemaEnTextoMuestraTableNameDeFormaProminente`, `TestSchemaEnTextoNoDuplicaNombreDeNegocioCuandoCoincideConTableName`, `TestQueryEnTextoMuestraTablaConColumnasOrdenadas`, todos con `d.IsTTY = true` explícito.
2. **Aviso 2 (aseverar valores, no solo que la petición se hizo)**: reforcé las aserciones de todos los tests del brief para comparar campos concretos (`TableName`/`Name`/columnas de `schema`; `mes`/`ventas` de `query`), no solo longitudes; y añadí `Summary`/`Breadcrumbs` exactos donde el brief no los comprobaba.
3. **Aviso 3 (valores distintos entre sí en las fixtures)**: `respuestaSchema` tiene `TableName != Name` en ambas tablas, y las dos tablas entre sí también difieren en todos los campos comprobados. Esto es justo lo que M1 confirmó que hacía falta: sin ello, un filtro que comparara contra el campo equivocado habría seguido "funcionando" por coincidencia.
4. **Aviso 4 (orden de columnas en las tablas de texto)**: `TestSchemaEnTextoMuestraTableNameDeFormaProminente` comprueba el orden COLUMNA/TIPO/DESCRIPCIÓN con `strings.Index` (patrón ya usado en `knowledge_test.go`); `TestQueryEnTextoMuestraTablaConColumnasOrdenadas` usa columnas `id`/`monto` (alfabéticamente `id` < `monto`) con dos filas de valores distintos para comprobar tanto el orden de cabecera como el de cada fila.

Además:

- `TestSchemaYQuerySonAtajosConRunE`: confirma que ambos comandos definen `RunE` y no tienen subcomandos (son atajos, no grupos — lo que dice el brief en su cabecera).
- `TestSchemaTableFiltraPorNombreDeNegocioComoComodidad` y `TestSchemaTableFiltraSinDistinguirMayusculas`: cubren explícitamente la comodidad de filtrar por `Name` y la insensibilidad a mayúsculas, ambas exigidas por la corrección 1.
- `TestQueryResultadoNoJSONDevuelveErrorGenerico`: cubre la rama de error de `resp.Rows()` (un `data` que no es array ni cadena JSON), no ejercitada por el brief.
- `TestQueryCSVYOutputSonIndependientes`: ver Corrección 3.

## Verificación por mutación

Copié el repositorio completo (sin `.git`) a `/private/tmp/.../scratchpad/mutation-copy`, confirmé la base verde, y apliqué/revertí 22 mutaciones sobre `data.go`, cada una ejecutando `go test ./internal/commands/ -run "Schema|Query" -count=1`.

| # | Mutación | Resultado |
|---|---|---|
| M1 | `filterTables`: casa solo contra `Name` (bug del brief original) | FAIL (`TestSchemaTableFiltraEnCliente`, `TestSchemaTableFiltraSinDistinguirMayusculas`, `TestSchemaEnTextoMuestraTableNameDeFormaProminente`) |
| M2 | `filterTables`: casa solo contra `TableName` (sin comodidad por `Name`) | FAIL (`TestSchemaTableFiltraPorNombreDeNegocioComoComodidad`) |
| M3 | `filterTables`: comparación sensible a mayúsculas (sin `EqualFold`) | build failed (import `strings` sin usar) |
| M4 | Mensaje NotFound sin el nombre de la tabla | FAIL (`TestSchemaTableInexistenteEsNotFound`) |
| M5 | `output.CodeNotFound` → `output.CodeGeneric` | FAIL (`TestSchemaTableInexistenteEsNotFound`, código de salida) |
| M6 | Text: imprime `Name` antes que `TableName` | FAIL (`TestSchemaEnTextoMuestraTableNameDeFormaProminente`, orden) |
| M7 | Text: quita el `!= t.TableName` (duplica cuando coinciden) | FAIL (`TestSchemaEnTextoNoDuplicaNombreDeNegocioCuandoCoincideConTableName`) |
| M8 | Text: fila de columna con TIPO/COLUMNA intercambiadas | FAIL (`TestSchemaEnTextoMuestraTableNameDeFormaProminente`, orden de fila) |
| M9 | `"%d tablas"` → `"%d cosas"` | FAIL (`TestSchemaDevuelveTodasLasTablas`) |
| M10 | Breadcrumb `"consultar"` → `"consulta"` | FAIL (`TestSchemaDevuelveTodasLasTablas`) |
| M11 | `query`: `exactArgs(1)` → `cobra.ExactArgs(1)` | FAIL (`TestQuerySinSQLEsErrorDeUso`) |
| M12 | `query`: `formato` ignorado (siempre `""`) | FAIL (`TestQueryOutputSeReenviaAlBackend`) |
| M13 | `"%d filas"` → `"%d resultados"` | FAIL (`TestQueryEnviaElSQLYDevuelveFilas`) |
| M14 | Breadcrumb `"esquema"` → `"esquemas"` | FAIL (`TestQueryEnviaElSQLYDevuelveFilas`) |
| M15 | Error de `Rows()`: `CodeGeneric` → `CodeUsage` | FAIL (`TestQueryResultadoNoJSONDevuelveErrorGenerico`) |
| M16 | Error de `Rows()`: hint sin mención a `--json` | FAIL (`TestQueryResultadoNoJSONDevuelveErrorGenerico`) |
| M17b | `columnsOf`: orden alfabético invertido (sin `sort.Strings` ascendente) | FAIL (`TestQueryCSVEsRenderLocal`, `TestQueryEnTextoMuestraTablaConColumnasOrdenadas`) |
| M18 | `writeCSV`: no escribe la cabecera | FAIL (`TestQueryCSVEsRenderLocal`) |
| M19c | `writeCSV`: cada celda repite el valor de la primera columna | FAIL (`TestQueryCSVEsRenderLocal`) |
| M20b | Text `query`: cada celda repite el valor de la primera columna | FAIL (`TestQueryEnTextoMuestraTablaConColumnasOrdenadas`) |
| M21 | `--csv` fuerza `formato=""` antes de llamar a `Query` (confunde `--csv` con `--output`) | **SOBREVIVIÓ en la primera pasada** — ver abajo |
| M22 | El `if comoCSV` nunca se activa (siempre cae al `Render` con envelope JSON) | FAIL (`TestQueryCSVEsRenderLocal`) |

**M21 sobrevivió a la primera pasada**: ningún test combinaba `--csv` y `--output` en la misma invocación, así que una implementación que (incorrectamente) apagara `--output` en presencia de `--csv` seguía pasando toda la suite. Añadí `TestQueryCSVYOutputSonIndependientes` (ver arriba), confirmé que detecta M21 al reaplicarla, y repetí entonces la mutación sobre la copia con el test ya reforzado: **FAIL** como se esperaba. Tras cada mutación se restauró `data.go` desde `data.go.orig`; al terminar comprobé que tanto la copia externa como el fichero real quedaron **idénticos** al original (el código de producción no cambió en ningún momento; solo se reforzó el test). La suite completa (`go build ./...`, `go vet ./...`, `gofmt -l .`, `go test ./... -race`, `go mod tidy`) se repitió limpia en el árbol de trabajo real después de todo el proceso, antes del commit.

No hice una mutación separada sobre el escape de caracteres especiales en `OrgPath`/`Client.Schema`/`Client.Query`: es código de Task 10, fuera del diff de esta tarea, y las rutas ya se comprueban por integración (`TestSchemaDevuelveTodasLasTablas`, `TestQueryEnviaElSQLYDevuelveFilas` comprueban método y ruta reales contra el servidor de prueba).

## Dudas / puntos para el controlador

Ninguna. Los tres identificadores de nivel de paquete que pedía traducir el brief (`filtrarTablas`, `columnasDe`, `escribirCSV`) ya estaban fijados en el glosario; no hubo ningún otro identificador en español a nivel de paquete sin traducción ya decidida.

---

## Ronda de correcciones 1/5

El revisor confirmó que la corrección 1 (TableName vs Name) quedó bien —verificó con el binario bajo pty que `fct_ventas [tabla SQL]` / `nombre de negocio: Ventas` es inequívoca, y que filtrar solo por `Name` rompe tres tests— y encontró tres defectos Important y uno Minor en la vía de `query`, los tres capaces de corromper o esconder datos de negocio. Son fallos de diseño del brief original, no introducidos por el implementer.

### Estado

DONE

### Commit

`95f0a46` — fix: notación científica, nulos y --csv silencioso en query (ronda 1/5)
(2 ficheros: `internal/commands/data.go`, `internal/commands/data_test.go`)

### Resumen de tests

`go test ./... -race -v` → PASS en los 9 paquetes (195 tests en verde en todo el módulo); 27/27 tests de `Schema*`/`Query*` (26 propios + 1 coincidencia de nombre ajena); `go vet ./...` limpio; `gofmt -l .` sin salida; `go mod tidy` sin cambios. 14 mutaciones dirigidas nuevas verificadas sobre una copia completa del repo fuera del árbol de trabajo: las 14 hicieron fallar al menos un test o el build.

### Qué se corrigió

**IMPORTANT 1 — notación científica.** `fmt.Sprintf("%v", ...)` sobre un valor `any` que tras `json.Unmarshal` es siempre `float64` usa el verbo `%g`, que cambia a exponente con cifras grandes o con muchos decimales. Añadida `formatValue(v any) string`: si `v` es `float64`, usa `strconv.FormatFloat(v, 'f', -1, 64)` (notación decimal fija, representación más corta que conserva el valor exacto — un entero como `1200` sale `"1200"`, no `"1200.0000"` ni `"1.2e+03"`). Usada tanto en `writeCSV` como en el renderer de `Text` de `query`.

**IMPORTANT 2 — `<nil>` en nulos.** Un valor nulo (NULL explícito del backend, o una columna ausente en una fila heterogénea — en Go, `f[c]` da `nil` en ambos casos por igual) se formateaba con `fmt.Sprintf("%v", nil)` → la cadena literal `<nil>`, indistinguible de un valor de negocio real. En CSV, `writeCSV` ahora escribe un campo vacío para un nulo (RFC 4180). En texto, se añadió `cellValue(v any) string`, que devuelve `"NULL"` para un nulo (documentado en el comentario de la función: nunca `<nil>`, que sería indistinguible de una cadena de negocio real con ese valor) y delega en `formatValue` para el resto.

**IMPORTANT 3 — `--csv` anulaba en silencio a `--json`/`--quiet`/`--md`/`--jq`.** El `if comoCSV { return writeCSV(...) }` cortaba antes de llegar a `ctx.Render`, que es lo único que mira esos flags. Añadida `csvConflictsWithOutputMode(cmd *cobra.Command) string`, que devuelve el nombre del primer flag de modo de salida activo (o `""` si no hay conflicto), y una comprobación en `RunE` — **antes** de llamar a `ctx.Client.Query`, para no gastar una llamada de red en un error de uso — que devuelve `output.NewError(output.CodeUsage, ...)` con el flag concreto en el mensaje y un hint que dice cuál usar.

**MINOR 4 — 0 filas en texto no imprimía nada.** El renderer de `Text` de `query` ahora comprueba `len(filas) == 0` primero y escribe `"Sin filas.\n"` en ese caso, en vez de dejar que `presenter.Table` reciba cabeceras y filas vacías (que antes producía, según el caso, una salida sin contenido útil).

No se tocó la pluralización de "1 filas"/"1 tablas" (deuda anotada por el propio controlador, ajena a esta ronda).

### Tests añadidos (11, todos en `internal/commands/data_test.go`)

1. `TestQueryCSVFormateaNumerosSinNotacionCientifica` — CSV con un entero, un decimal con muchos dígitos y un entero grande (15 cifras); exige ausencia de `"e+"`/`"E+"` y la cadena exacta esperada.
2. `TestQueryEnTextoFormateaNumerosSinNotacionCientifica` — variante en `Text` (`IsTTY:true`) del anterior.
3. `TestQueryCSVNuloEsCampoVacio` — NULL explícito del backend; el CSV se relee con `encoding/csv` (no `strings.Contains`) para comprobar el campo vacío sin ambigüedad de comillas.
4. `TestQueryCSVColumnaAusenteEnFilaHeterogeneaEsCampoVacio` — dos filas con columnas distintas entre sí; la fila que no trae la columna también sale con campo vacío.
5. `TestQueryEnTextoNuloNoEsNilLiteral` — variante en `Text` de 3: ni `<nil>`, y sí `NULL`.
6. `TestQueryEnTextoColumnaAusenteEnFilaHeterogeneaNoEsNilLiteral` — variante en `Text` de 4.
7. `TestQueryCSVConJSONEsErrorDeUso` — `--csv --json`: `CodeUsage`, mensaje que nombra `--json`, hint no vacío, código de salida 2, **y** `llamadas == 0` (no debe tocar la red — cierra la vía de un error de uso "correcto" pero perezoso, ver mutación N11 abajo).
8. `TestQueryCSVConQuietEsErrorDeUso` — igual con `--quiet`.
9. `TestQueryCSVConMdEsErrorDeUso` — igual con `--md`.
10. `TestQueryCSVConJQEsErrorDeUso` — igual con `--jq ".data"`.
11. `TestQueryEnTextoConCeroFilasNoQuedaVacio` — 0 filas en `Text`: la salida no debe quedar en blanco y debe contener "Sin filas".

Los seis tests existentes de `--csv`/`--output`/columnas ordenadas se dejaron intactos y siguen en verde sin cambios: los valores usados en ellos (`1200`, `300`, `450`) son enteros pequeños, así que `formatValue`/`cellValue` producen la misma cadena que el `fmt.Sprintf("%v", ...)` anterior — no hubo regresión que enmascarar.

### Verificación por mutación

Copié el repositorio completo (sin `.git`) a una segunda copia de trabajo fuera del árbol (`mutation-copy2`), confirmé la base verde con los 11 tests nuevos ya en rojo→verde, y apliqué/revertí 14 mutaciones sobre el código nuevo de `data.go`, cada una ejecutando `go test ./internal/commands/ -run "Schema|Query" -count=1`.

| # | Mutación | Resultado |
|---|---|---|
| N1 | `formatValue`: quita el caso `float64` (vuelve a `%v`/`%g`) | build failed (`strconv` sin usar) |
| N2 | `formatValue`: `strconv.FormatFloat(f, 'e', -1, 64)` en vez de `'f'` | FAIL (`TestQueryCSVEsRenderLocal`, `TestQueryEnTextoMuestraTablaConColumnasOrdenadas`, `TestQueryCSVFormateaNumerosSinNotacionCientifica`, `TestQueryEnTextoFormateaNumerosSinNotacionCientifica`) |
| N3 | `cellValue`: quita la comprobación de `nil` (siempre `formatValue`) | FAIL (`TestQueryEnTextoNuloNoEsNilLiteral`, `TestQueryEnTextoColumnaAusenteEnFilaHeterogeneaNoEsNilLiteral`) |
| N4 | `cellValue`: nulo → `""` en vez de `"NULL"` | FAIL (mismos dos tests) |
| N5 | `writeCSV`: quita la comprobación de `nil` (usa `formatValue(f[c])` siempre) | FAIL (`TestQueryCSVNuloEsCampoVacio`, `TestQueryCSVColumnaAusenteEnFilaHeterogeneaEsCampoVacio`) |
| N6 | `csvConflictsWithOutputMode`: quita la rama `--quiet` | FAIL (`TestQueryCSVConQuietEsErrorDeUso`) |
| N7 | ídem, rama `--md` | FAIL (`TestQueryCSVConMdEsErrorDeUso`) |
| N8 | ídem, rama `--jq` | FAIL (`TestQueryCSVConJQEsErrorDeUso`) |
| N9 | ídem, rama `--json` | FAIL (`TestQueryCSVConJSONEsErrorDeUso`) |
| N10 | Bloque completo de validación `--csv` eliminado | FAIL (los 4 tests de combinación) |
| N11 | Validación `--csv` movida **después** de `ctx.Client.Query` (sigue devolviendo el error, pero tras llamar a la red) | FAIL (`TestQueryCSVConJSONEsErrorDeUso`, por `llamadas == 0`) — confirma que la comprobación `llamadas` del test discrimina el orden, no solo el resultado final |
| N12 | Hint del error de conflicto `--csv` vacío (`""`) | FAIL (los 4 tests de combinación) |
| N13 | Rama de 0 filas en `Text` eliminada | FAIL (`TestQueryEnTextoConCeroFilasNoQuedaVacio`) |
| N14 | Mensaje de 0 filas cambiado (`"Sin filas."` → `"0 resultados"`) | FAIL (mismo test) |

Las 14 mutaciones quedaron capturadas en la primera pasada — no hizo falta reforzar ningún test. Tras cada mutación se restauró `data.go` desde `data.go.orig`; al terminar comprobé que tanto la copia externa como el fichero real quedaron **idénticos** al original (el código de producción no cambió durante la verificación; solo se ejecutaron mutaciones temporales sobre la copia). La suite completa (`go build ./...`, `go vet ./...`, `gofmt -l .`, `go test ./... -race -v`, `go mod tidy`) se repitió limpia en el árbol de trabajo real después del proceso, antes del commit.

### Verificación en campo (binario real, no solo `testRoot`)

Levanté un servidor HTTP local de prueba y comparé el binario antes/después de las correcciones:

- **Cifra grande + nulo, CSV**: antes → `grande,importe,nulo\n1.23456789012345e+14,1.234567891234e+06,<nil>\n`; después → `grande,importe,nulo\n123456789012345,1234567.891234,\n` (sin exponentes, nulo como campo vacío).
- **`query "SELECT 1" --csv --json`**: antes → CSV de todos modos, sin aviso; después → `{"ok":false,"error":{"code":"USAGE","message":"--csv no se puede combinar con --json: son dos modos de salida distintos.","hint":"Usa --csv solo, o usa --json sin --csv."}}`, código de salida 2.
- **0 filas en texto (bajo pty real, vía `script`)**: antes → salida vacía; después → `Sin filas.`.

### Dudas / puntos para el controlador

Ninguna.
