# Informe — Task 15: Comando `docs`

## Estado

DONE

## Commit

`6554fce` — feat: comando docs (list, show, search)
(3 ficheros: `internal/commands/docs.go`, `internal/commands/docs_test.go`, `internal/cli/root.go`)

## Resumen de tests

`go test ./... -race` → PASS en los 9 paquetes (7 con tests), 19/19 tests de `Docs*`/`TituloDe*`/`Recorta*` en verde; `go vet ./...` limpio; `gofmt -l .` sin salida; `go mod tidy` sin cambios. 17 mutaciones dirigidas verificadas sobre una copia completa del repo fuera del árbol de trabajo: las 17 hicieron fallar al menos un test, ninguna sobrevivió.

## Qué se hizo

- `internal/commands/docs.go`: `NewDocsCmd`, `newDocsListCmd`, `newDocsShowCmd`, `newDocsSearchCmd`, `titleOf`, `truncate` — copiado literalmente del brief con dos cambios:
  - **Corrección indicada por el controlador**: `Args: exactArgs(1)` en `docs show` y `docs search`, en vez de `cobra.ExactArgs(1)`.
  - Traducción de identificadores de nivel de paquete según el glosario: `tituloDe`→`titleOf`, `recortar`→`truncate` (firma exacta `func truncate(s string, n int) string`, tal como pide el encargo para que la Task 16 la reutilice sin fricción). Variables locales (`grupo`, `p`, `page`, `filas`, `doc`, `limite`, `res`, `r`) y comentarios se dejaron tal como los escribe el brief, en español. Añadí un comentario propio de una línea sobre `truncate` señalando que la Task 16 la reutiliza, para que quede documentado en el propio código.
- `internal/commands/docs_test.go`: los 3 tests del Step 1 del brief (`TestDocsListPasaLosFiltros`, `TestDocsShowLlamaAlEndpointDelDocumento`, `TestDocsSearchUsaPOST`), usando los ayudantes ya existentes `depsWithServer`/`testRoot` (no redefinidos), más 16 tests añadidos para cerrar huecos de cobertura reales (detalle abajo). Nombres de test en español.
- `internal/cli/root.go`: añadido `commands.NewDocsCmd(d)` a `root.AddCommand(...)`.

Verificado manualmente contra el binario real (no solo `testRoot`): `go run ./cmd/calliope docs --help`, `docs show --help` y `docs search --help` muestran los tres subcomandos, el `Short` en español, el flag `--limit` con su default (10) y los cinco flags globales heredados.

## Tests añadidos más allá del brief, y por qué

El brief solo comprueba que la petición viaja y que `data` tiene el tamaño esperado. Siguiendo los tres avisos del encargo:

1. **Aviso 1 (Text nunca se ejercita en modo automático)**: `depsWithServer` deja `IsTTY:false`. Añadí `TestDocsListEnTextoMuestraTabla`, `TestDocsShowEnTextoMuestraLosMetadatos` y `TestDocsSearchEnTextoMuestraTablaConFragmentoRecortado`, todos con `d.IsTTY = true` explícito, para forzar el camino de `Text` y comprobar su contenido real.

2. **Aviso 2 (una aserción de "la petición se hizo" no detecta un tag mal escrito ni un campo que no se muestra)**: reforcé los tres tests del brief para comparar valores concretos, no solo longitudes:
   - `TestDocsListPasaLosFiltros`: `url.ParseQuery` sobre la query real y comparación exacta de `status`/`tag` (detecta un flag cableado al campo equivocado, p. ej. `--status` escribiendo en `p.Tag`); valores exactos de `id`/`filename`/`status`/`sizeBytes` del documento devuelto; `Summary` exacto (`"1 de 1 documentos"`); los dos `Breadcrumbs` exactos.
   - `TestDocsShowLlamaAlEndpointDelDocumento`: valores exactos del documento devuelto, `Summary` exacto y el único `Breadcrumb` exacto.
   - `TestDocsSearchUsaPOST`: cuerpo de la petición decodificado (`query`+`limit`, incluido que el default de `--limit` sea 10 y no 0), valores del fragmento devuelto, `Summary` exacto y `Breadcrumb` exacto.
   - Los tests de tabla en modo `Text` (`TestDocsListEnTextoMuestraTabla`, `TestDocsSearchEnTextoMuestraTablaConFragmentoRecortado`) comprueban además el **orden** de las cabeceras y, dentro de cada fila de datos, el orden relativo de los valores (p. ej. que `READY` preceda a `2048` en la fila de `doc-1`): un `strings.Contains` global no detectaría que dos columnas de datos viajaran intercambiadas si las cabeceras quedan intactas (ver mutaciones M12/M13 abajo).

3. **Aviso 3 (un espacio no discrimina el escape de ruta)**: `TestDocsShowEscapaLaBarraDelIDEnLaRuta` usa un id con una barra (`carpeta/doc-1`) en vez de un espacio, y lee `r.URL.EscapedPath()` (no `r.URL.Path`, que Go ya desescapa `%2F` de vuelta a `/` y no distinguiría los dos casos) para comprobar que la barra llega escapada como `%2F` en la petición real.

Además:

- `TestDocsEsUnGrupoSinRunE`: `docs` no define `RunE`/`Run` y tiene 3 subcomandos — invocarlo pelado muestra la ayuda, siguiendo el patrón de `orgs`/`auth`.
- `TestDocsListSinFiltrosNoEnviaQueryString`: sin flags, la query debe quedar vacía (page/size en 0 se omiten, no se envían como `"0"`).
- `TestDocsShowUsaElTituloDelDocumentoComoResumenSiExiste`: cuando el documento sí trae título, el resumen de `docs show` lo usa en vez del nombre de fichero — sin esto, un `titleOf` que siempre devolviera `Filename` pasaría inadvertido en el test del brief (que no manda título en la fixture).
- `TestDocsShowSinArgumentoEsErrorDeUso` / `TestDocsSearchSinArgumentoEsErrorDeUso`: cubren la corrección del controlador (`exactArgs` en vez de `cobra.ExactArgs`) con el mismo patrón que ya usan `ask`/`orgs use` — hint exacto `"Uso: calliope docs show <id>"` / `"Uso: calliope docs search <consulta>"`, mensaje sin "accepts"/"arg(s)" en inglés, código de salida 2.
- `TestDocsSearchRespetaElLimite`: `--limit 3` llega de verdad al cuerpo de la petición (el default 10 ya se cubre en `TestDocsSearchUsaPOST`, pero nada más comprobaba que el flag explícito se propagara).
- Tests unitarios directos de los dos ayudantes de paquete (mismo paquete, sin pasar por HTTP):
  - `TestTituloDeUsaElTituloSiExiste`, `TestTituloDeCaeAlNombreDeFicheroSinTitulo`, `TestTituloDeCaeAlNombreDeFicheroConTituloVacio` — los tres caminos de `titleOf` (puntero nil, puntero a vacío, puntero con valor).
  - `TestRecortaPorRunesNoPorBytes` — construye una cadena con un carácter multibyte (`ñ`, 2 bytes en UTF-8) repetido 80 veces y comprueba que `truncate` corta por runas (71 runas: 70 + `…`), no por bytes.
  - `TestRecortaNoTocaCadenasCortas` y `TestRecortaCadenaExactamenteEnElLimiteNoAñadePuntosSuspensivos` — los casos borde `len(r) < n` y `len(r) == n` de la condición `<=`.

## Verificación por mutación

Copié el repositorio completo a `/private/tmp/.../scratchpad/mutation-copy` (build limpio confirmado antes de empezar) y apliqué/revertí 17 mutaciones sobre `docs.go`, cada una ejecutando `go test ./internal/commands/ -run 'Docs|TituloDe|Recorta'`:

| # | Mutación | Resultado |
|---|---|---|
| M1 | Intercambiar el cableado de `--status`/`--tag` (cada flag escribe en el campo del otro) | FAIL (`TestDocsListPasaLosFiltros`) |
| M2 | `"%d de %d documentos"` → `"%d/%d documentos"` | FAIL (`TestDocsListPasaLosFiltros`) |
| M3 | Breadcrumb `"detalle"` pierde `<id>` del `Cmd` | FAIL (`TestDocsListPasaLosFiltros`) |
| M4 | `titleOf` ignora `Title`, siempre devuelve `Filename` | FAIL (`TestTituloDeUsaElTituloSiExiste`, `TestDocsShowUsaElTituloDelDocumentoComoResumenSiExiste`, `TestDocsListEnTextoMuestraTabla`) |
| M5 | `titleOf` sin comprobación de título vacío (`doc.Title != nil` a secas) | FAIL (`TestTituloDeCaeAlNombreDeFicheroConTituloVacio`) |
| M6 | `truncate`: `len(r) <= n` → `len(r) < n` (off-by-one) | FAIL (`TestRecortaCadenaExactamenteEnElLimiteNoAñadePuntosSuspensivos`) |
| M7 | `truncate` corta por bytes (`s[:n]`) en vez de por runas | FAIL (`TestRecortaPorRunesNoPorBytes`) |
| M8 | `docs show` sin `Args: exactArgs(1)` (acepta 0 argumentos, panic por `args[0]`) | FAIL (`TestDocsShowSinArgumentoEsErrorDeUso`) |
| M9 | `docs search` sin `Args: exactArgs(1)` | FAIL (`TestDocsSearchSinArgumentoEsErrorDeUso`) |
| M10 | `docs search` ignora `limite`, siempre manda `0` a `SearchDocuments` | FAIL (`TestDocsSearchUsaPOST`: default esperado es 10) |
| M11 | Se borra el registro de `cmd.Flags().IntVar(&limite, "limit", ...)` | FAIL (`TestDocsSearchRespetaElLimite`: flag `--limit` desconocido) |
| M12 | `docs list` Text: intercambia `doc.Status` y `SizeBytes` en la fila de datos (cabeceras intactas) | FAIL (`TestDocsListEnTextoMuestraTabla`: orden READY/2048 en la fila) |
| M13 | `docs search` Text: intercambia `Score` y `Excerpt` en la fila de datos | FAIL (`TestDocsSearchEnTextoMuestraTablaConFragmentoRecortado`: orden doc-1/0.876/excerpt en la fila) |
| M14 | `docs show` usa `doc.Filename` en vez de `titleOf(*doc)` en el resumen | FAIL (`TestDocsShowUsaElTituloDelDocumentoComoResumenSiExiste`) |
| M15 | Breadcrumb `"documento"` de `docs search` pierde `<id>` | FAIL (`TestDocsSearchUsaPOST`) |
| M16 | Breadcrumb `"buscar dentro"` de `docs show` pierde `"<consulta>"` | FAIL (`TestDocsShowLlamaAlEndpointDelDocumento`) |
| M17 | `docs search`: `"%d fragmentos"` → `"%d resultados"` | FAIL (`TestDocsSearchUsaPOST`) |

Las 17 mutaciones hicieron fallar al menos un test; ninguna sobrevivió. Tras cada mutación se restauró el fichero desde una copia intacta (`docs.go.orig`) y, al terminar, se comprobó con `diff` que tanto la copia mutada como el fichero del repositorio real quedaron **idénticos** al original. La suite completa (`go build ./...`, `go vet ./...`, `gofmt -l .`, `go test ./... -race`, `go mod tidy`) se repitió limpia en el árbol de trabajo real después de todo el proceso, antes del commit.

No incluí una mutación sobre el escape de la barra en `docs show <id>` (aviso 3): ese escape lo aplica `Client.GetDocument`/`OrgPath` en `internal/sdk`, código ya existente de la Task 10 y fuera del diff de esta tarea; `TestDocsShowEscapaLaBarraDelIDEnLaRuta` verifica la integración end-to-end (que el argumento de `docs show` llega intacto hasta ahí), no una rama propia de `docs.go`.

## Dudas / puntos para el controlador

Ninguna. No hubo identificadores en español a nivel de paquete sin traducción ya fijada en el glosario: las dos traducciones que pedía el encargo (`tituloDe`→`titleOf`, `recortar`→`truncate`) ya estaban en la tabla del glosario, y el resto de nombres del brief (`NewDocsCmd`, `newDocsListCmd`, `newDocsShowCmd`, `newDocsSearchCmd`) ya venían en inglés.

---

## Ronda de correcciones 1/5

### Hallazgo atendido (Important)

`TestDocsListPasaLosFiltros` comprobaba `--status` y `--tag` con valores exactos, pero `--q`, `--page` y `--size` solo se cubrían por omisión (que no aparecieran en la query cuando no se pasaban). Nada comprobaba que, al pasarlos, llegaran con el valor correcto y al campo correcto: un `--page` cableado a `p.Size` (o viceversa), o un `--q` recableado a `p.Status`/`p.Tag`, habría dejado la suite en verde.

### Cambio

`internal/commands/docs_test.go`, `TestDocsListPasaLosFiltros`: ahora se pasan los cinco flags a la vez (`--status READY --tag finanzas --q factura --page 2 --size 25`), con **valores distintos entre sí** para que cualquier intercambio de cableado entre dos de ellos sea observable, y se comparan uno a uno con `q.Get(...)` contra su valor esperado. La comprobación de que `q`/`page`/`size` se omiten cuando no se pasan sigue cubierta aparte por `TestDocsListSinFiltrosNoEnviaQueryString`, que no se tocó.

No hizo falta cambiar `docs.go`: el hallazgo era un hueco de test, no un bug de producción (verificado: la suite ya estaba en verde antes de esta ronda porque el cableado real es correcto).

### Verificación por mutación

Sobre la misma copia externa al repositorio usada en la ronda anterior (`docs.go` no había cambiado desde entonces; se resincronizó solo `docs_test.go` y se confirmó `diff` vacío contra el original antes de empezar):

| # | Mutación | Resultado |
|---|---|---|
| M18 | Intercambiar `--page`/`--size` (`f.IntVar(&p.Size, "page", ...)` / `f.IntVar(&p.Page, "size", ...)`) | FAIL (`TestDocsListPasaLosFiltros`: `page`/`size` intercambiados en la query) |
| M19 | Recablear `--q` a `p.Status` | FAIL (`TestDocsListPasaLosFiltros`: `status` deja de ser `READY`, `q` deja de aparecer) |
| M20 | Recablear `--q` a `p.Tag` | FAIL (`TestDocsListPasaLosFiltros`: `tag` deja de ser `finanzas`) |
| M21 | `--page` ignora el flag, siempre envía `0` (variable local sin usar en `p.Page`) | FAIL (`TestDocsListPasaLosFiltros`: `page` ausente en la query) |
| M22 | `--size` ignora el flag, siempre envía `0` | FAIL (`TestDocsListPasaLosFiltros`: `size` ausente en la query) |

Las 5 mutaciones nuevas quedaron capturadas. Repetí también las 17 mutaciones de la ronda anterior contra el fichero de test actualizado: las 17 siguen capturadas (22/22 en total entre ambas rondas). Tras cada mutación se restauró `docs.go` desde la copia intacta; al terminar, `diff` confirmó que tanto la copia externa como el fichero del repositorio real quedaron idénticos al original.

### Verificación final

`go vet ./...` limpio; `gofmt -l .` sin salida; `go mod tidy` sin cambios (`go.mod`/`go.sum` sin diff); `go test ./... -race -v` → PASS en los 9 paquetes, sin ningún `FAIL` en la salida completa.

### Commit

`14ec7a5` — test: asevera valores exactos de --q/--page/--size en docs list
