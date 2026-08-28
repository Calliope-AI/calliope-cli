# Informe — Task 16: Comandos `concepts` y `rules`

## Estado

DONE

## Commit

`5714ca3` — feat: comandos concepts y rules
(3 ficheros: `internal/commands/knowledge.go`, `internal/commands/knowledge_test.go`, `internal/cli/root.go`)

## Resumen de tests

`go test ./... -race` → PASS en los 9 paquetes (7 con tests); 21/21 tests de `Concepts*`/`Rules*`/`Activo*` en verde; `go vet ./...` limpio; `gofmt -l .` sin salida; `go mod tidy` sin cambios. 16 mutaciones dirigidas verificadas sobre una copia completa del repo fuera del árbol de trabajo: las 16 hicieron fallar al menos un test tras dos rondas de refuerzo del test (ver abajo — 3 de las 16 sobrevivieron a la primera pasada y se cerraron).

## Qué se hizo

- **Step 1-2 (TDD)**: escribí `internal/commands/knowledge_test.go` con los tests del brief (identificadores traducidos, ver más abajo) y confirmé el fallo esperado: `go test ./internal/commands/ -run "Concepts|Rules|Activo" -v` → `undefined: NewConceptsCmd` / `undefined: NewRulesCmd`.
- **Step 3**: `internal/commands/knowledge.go` — `NewConceptsCmd`, `newConceptsListCmd`, `newConceptsShowCmd`, `NewRulesCmd`, `newRulesListCmd`, `yesNo` — copiado literalmente del brief con dos correcciones del controlador:
  - `Args: exactArgs(1)` en `concepts show`, en vez de `cobra.ExactArgs(1)`.
  - Test `TestConceptsYRulesSonGruposSinRunE`: se omitió el bucle muerto del brief (`for` sobre un slice literal vacío); quedaron solo las comprobaciones de `RunE`/`Run`, más el recuento de subcomandos (2 en `concepts`, 1 en `rules`) siguiendo el patrón ya usado en `TestDocsEsUnGrupoSinRunE`.
  - Traducción de identificadores de nivel de paquete según el glosario: `activo`→`yesNo`. Variables locales (`grupo`, `ctx`, `grafo`, `filas`, `registros`, `det`, `desc`, `cat`, `reglas`, `c`, `a`, `r`) y comentarios se dejaron tal como los escribe el brief, en español. `truncate` se reutilizó de `docs.go` sin redefinir (firma exacta ya existente).
- **Step 4**: `internal/cli/root.go` — añadidos `commands.NewConceptsCmd(d)` y `commands.NewRulesCmd(d)` a `root.AddCommand(...)`.
- Verificado manualmente contra el binario real (no solo `testRoot`): `calliope --help`, `calliope concepts --help` y `calliope rules --help` muestran los grupos, sus subcomandos (`list`/`show` y `list` respectivamente), el `Short` en español y los cinco flags globales heredados.

## Tests añadidos más allá del brief, y por qué

El brief solo comprueba (para `concepts list`) el nombre de un concepto y que hay breadcrumbs, (para `concepts show`) el nombre de un atributo, y (para `rules list`) el nombre de una regla. Siguiendo los cuatro avisos del encargo:

1. **Aviso 1 (Text nunca se ejercita en modo automático)**: `depsWithServer` deja `IsTTY:false`. Añadí `TestConceptsListEnTextoMuestraTabla`, `TestConceptsShowConDescripcionLaImprimeAntesDeLaTabla`, `TestConceptsShowSinDescripcionNoImprimeLineaDeDescripcion`, `TestConceptsShowConDescripcionVaciaNoImprimeLineaDeDescripcion`, `TestConceptsShowEnTextoOrdenaColumnasDeAtributos` y `TestRulesListEnTextoMuestraTabla`/`TestRulesListRecortaDetallesLargos`, todos con `d.IsTTY = true` explícito.

2. **Aviso 2 (una aserción de "la petición se hizo" no detecta un tag mal escrito ni un campo que no se muestra)**: reforcé los tres tests del brief para comparar valores concretos de cada campo devuelto (no solo longitudes/nombres sueltos), y a la vez el `Summary` exacto y los `Breadcrumbs` exactos por acción/cmd (`TestConceptsListDevuelveElGrafo`, `TestConceptsShowDevuelveAtributos`, `TestRulesListDevuelveLasReglas`).

3. **Aviso 3 (si varios valores son iguales entre sí, un cableado cruzado pasa desapercibido)**: en todas las fixtures con dos elementos, los dos traen valores **distintos y observables** entre sí en cada campo (dos conceptos con `id`/`name`/`isActive`/`recordCount` presente-vs-ausente distintos; dos atributos con `name`/`description`/`isActive` distintos; dos reglas con `name`/`details`/`category` presente-vs-ausente/`status` todos distintos). Esto fue justo lo que la primera pasada de mutación detectó que faltaba en los tests de tabla (ver ronda de correcciones abajo).

4. **Aviso 4 (un espacio no discrimina el escape de ruta)**: `TestConceptsShowEscapaLaBarraDelIDEnLaRuta` usa un id con una barra (`carpeta/c-1`), no un espacio, y lee `r.URL.EscapedPath()` (no `r.URL.Path`, que Go ya desescapa) para comprobar que la barra llega como `%2F` en la petición real.

Además:

- `TestConceptsShowSinArgumentoEsErrorDeUso`: cubre la corrección del controlador (`exactArgs` en vez de `cobra.ExactArgs`) — hint exacto `"Uso: calliope concepts show <id>"`, mensaje sin "accepts"/"arg(s)" en inglés, código de salida 2.
- `TestActivoDevuelveSiParaVerdadero` / `TestActivoDevuelveNoParaFalso`: prueba unitaria directa de `yesNo` (mismo paquete, sin pasar por HTTP).
- `TestConceptsShowConDescripcionVaciaNoImprimeLineaDeDescripcion`: caso borde entre "con descripción" y "sin descripción" — un puntero a cadena **vacía** (`""`), no `nil`. Cierra la rama `!= ""` de la condición `det.Concept.Description != nil && *det.Concept.Description != ""`, que ninguno de los dos tests del brief ejercitaba (uno tenía descripción no vacía, el otro directamente `nil`).

## Verificación por mutación

Copié el repositorio completo a `/private/tmp/.../scratchpad/mutation-copy` (baseline verde confirmado antes de empezar) y apliqué/revertí mutaciones sobre `knowledge.go`, cada una ejecutando `go test ./internal/commands/ -run "Concepts|Rules|Activo" -v`.

**Primera pasada — 3 mutaciones sobrevivieron**, exponiendo huecos reales de test (no bugs de producción: el código ya era correcto, los tests no discriminaban):

| # | Mutación | Resultado primera pasada |
|---|---|---|
| M8 | `rules list` Text: fila `{cat, r.Name, r.Status, detalle}` — REGLA y CATEGORÍA intercambiadas | **SOBREVIVIÓ** — el test solo comprobaba orden CATEGORÍA<ESTADO<DETALLE, no la posición de REGLA |
| M11 | `concepts list` Text: fila `{c.Name, c.ID, registros, activo}` — ID y CONCEPTO intercambiadas | **SOBREVIVIÓ** — el test comprobaba REGISTROS/ACTIVO tras el ID, pero como esas dos columnas no se movieron, el orden seguía "pareciendo" correcto |
| M13 | `concepts show` Text: fila de atributo `{desc, a.Name, activo}` — ATRIBUTO y DESCRIPCIÓN intercambiadas | **SOBREVIVIÓ** — ningún test comprobaba el orden relativo de esas dos columnas con una descripción no vacía |

Cerré los tres huecos reforzando los tests correspondientes (índices de posición explícitos, incluyendo el campo que antes no se comprobaba) y añadiendo `TestConceptsShowEnTextoOrdenaColumnasDeAtributos` como test dedicado para M13. Repetí entonces la pasada completa:

| # | Mutación | Resultado |
|---|---|---|
| M1 | `yesNo`: ambas ramas devuelven `"no"` | FAIL (`TestActivoDevuelveSiParaVerdadero`, `TestConceptsListEnTextoMuestraTabla`, `TestConceptsShowSinDescripcionNoImprimeLineaDeDescripcion`) |
| M2 | `concepts list`: `"%d conceptos"` → `"%d cosas"` | FAIL (`TestConceptsListDevuelveElGrafo`) |
| M3 | `concepts list`: breadcrumb `"detalle"` → `"detalles"` | FAIL (`TestConceptsListDevuelveElGrafo`) |
| M4 | `concepts show`: `"%s · %d atributos"` con argumentos invertidos | FAIL (`TestConceptsShowDevuelveAtributos`) |
| M5 | `concepts show`: `Args: cobra.ExactArgs(1)` en vez de `exactArgs(1)` | FAIL (`TestConceptsShowSinArgumentoEsErrorDeUso`) |
| M6 | `rules list`: `truncate(r.Details, 60)` → `truncate(r.Details, 6)` | FAIL (`TestRulesListEnTextoMuestraTabla`, `TestRulesListRecortaDetallesLargos`) |
| M7 | `rules list`: breadcrumb `"conceptos"` → `"concepto"` | FAIL (`TestRulesListDevuelveLasReglas`) |
| M8 | `rules list` Text: REGLA/CATEGORÍA intercambiadas | FAIL (`TestRulesListEnTextoMuestraTabla`, tras el refuerzo) |
| M9 | `concepts show`: breadcrumb pierde las comillas de `"<pregunta sobre …>"` | FAIL (`TestConceptsShowDevuelveAtributos`) |
| M10 | `NewConceptsCmd`: no registra `newConceptsShowCmd` | FAIL (`TestConceptsYRulesSonGruposSinRunE` y los 6 tests de `concepts show`) |
| M11 | `concepts list` Text: ID/CONCEPTO intercambiadas | FAIL (`TestConceptsListEnTextoMuestraTabla`, tras el refuerzo) |
| M12 | `concepts list`: quita el nil-check de `RecordCount` (desreferencia siempre) | FAIL (`TestConceptsListEnTextoMuestraTabla`, panic por puntero nil en c-2) |
| M13 | `concepts show` Text: ATRIBUTO/DESCRIPCIÓN intercambiadas | FAIL (`TestConceptsShowEnTextoOrdenaColumnasDeAtributos`, test nuevo) |
| M14 | `concepts show`: quita la comprobación `!= ""` de la descripción (deja solo `!= nil`) | FAIL (`TestConceptsShowConDescripcionVaciaNoImprimeLineaDeDescripcion`) |
| M15 | `concepts show`: quita el nil-check de `Description` del atributo (desreferencia siempre) | FAIL (`TestConceptsShowSinDescripcionNoImprimeLineaDeDescripcion`, panic) |
| M16 | `rules list`: quita el nil-check de `Category` (desreferencia siempre) | FAIL (`TestRulesListEnTextoMuestraTabla`, panic por puntero nil en r-2) |

Las 16 mutaciones (13 directas + 3 reintentadas tras el refuerzo) quedaron capturadas. Tras cada mutación se restauró el fichero desde una copia intacta (`knowledge.go.orig`) y, al terminar, se comprobó que tanto la copia externa como el fichero del repositorio real quedaron **idénticos** al original (el código de producción no cambió en ningún momento; solo se reforzaron los tests). La suite completa (`go build ./...`, `go vet ./...`, `gofmt -l .`, `go test ./... -race`, `go mod tidy`) se repitió limpia en el árbol de trabajo real después de todo el proceso, antes del commit.

No incluí una mutación sobre el escape de la barra en `concepts show <id>` (aviso 4): ese escape lo aplica `Client.GetConcept`/`OrgPath` en `internal/sdk`, código ya existente de la Task 10 y fuera del diff de esta tarea; `TestConceptsShowEscapaLaBarraDelIDEnLaRuta` verifica la integración end-to-end (que el argumento de `concepts show` llega intacto hasta ahí), no una rama propia de `knowledge.go`.

No añadí un test de registro a nivel de `root.go` que enumere todos los subcomandos: es un hueco preexistente en `internal/cli/root_test.go` (ningún comando anterior — `auth`, `orgs`, `config`, `ask`, `docs` — tiene esa cobertura tampoco), así que seguí la convención del proyecto. En su lugar verifiqué el registro con el binario real compilado (`go build ./cmd/calliope`): `--help`, `concepts --help` y `rules --help` muestran los grupos y subcomandos esperados.

## Dudas / puntos para el controlador

Ninguna. No hubo identificadores en español a nivel de paquete sin traducción ya fijada en el glosario: la única traducción que pedía el encargo (`activo`→`yesNo`) ya estaba en la tabla del glosario, y el resto de nombres del brief (`NewConceptsCmd`, `newConceptsListCmd`, `newConceptsShowCmd`, `NewRulesCmd`, `newRulesListCmd`) ya venían en inglés.
