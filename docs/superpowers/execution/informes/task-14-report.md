# Informe — Task 14: Comando `ask`

## Estado

DONE

## Commit

`26c8df3` — feat: comando ask con citación de fuentes
(3 ficheros: `internal/commands/ask.go`, `internal/commands/ask_test.go`, `internal/cli/root.go`)

## Resumen de tests

`go test ./... -race` → PASS en los 9 paquetes (7 con tests), 6/6 tests de `Ask*` en verde; `go vet ./...` limpio; `gofmt -l .` sin salida; `go mod tidy` sin cambios. 11 mutaciones dirigidas verificadas sobre una copia completa del repo en `/tmp` (fuera del árbol de trabajo): cada una hizo fallar al menos un test, y el fichero mutado se restauró byte a byte antes de seguir.

## Qué se hizo

- `internal/commands/ask.go`: `NewAskCmd`, `writeAskText`, `writeAskMarkdown` — copiado literalmente del brief con dos cambios:
  - **Corrección indicada por el controlador**: `Args: exactArgs(1)` en vez de `cobra.ExactArgs(1)`.
  - **Fix de `go vet`** (no indicado, necesario para la restricción global de `go vet ./...` limpio): la línea `fmt.Fprintln(w, "\n### Fuentes\n")` del brief dispara `fmt.Fprintln arg list ends with redundant newline` (el string ya termina en `\n` y `Fprintln` añade otro). Cambiada a `fmt.Fprint(w, "\n### Fuentes\n\n")`, que produce exactamente los mismos bytes de salida (verificado: el test de markdown sigue en verde con la cabecera `### Fuentes` intacta).
  - Traducción de identificadores de nivel de paquete según el glosario: `escribirAskTexto`→`writeAskText`, `escribirAskMarkdown`→`writeAskMarkdown`. `NewAskCmd` ya venía en inglés. Variables locales (`accion`, `resumen`, `mensaje`, `titulo`, `resp`, `ctx`) y comentarios se dejaron tal como los escribe el brief, en español.
- `internal/commands/ask_test.go`: los 4 tests del Step 1, usando los ayudantes ya existentes `depsWithServer`/`testRoot` (no redefinidos), más 3 tests añadidos para cerrar huecos de cobertura reales (detalle abajo). Nombres de test en español.
- `internal/cli/root.go`: añadido `commands.NewAskCmd(d)` a `root.AddCommand(...)`.

## Tests añadidos más allá del brief, y por qué

El brief solo prueba `--json` y `--md`. Siguiendo los dos avisos del encargo, añadí:

1. **`TestAskEnTextoCitaLasFuentes`** (aviso 1): `depsWithServer` deja `IsTTY:false`, así que en modo automático la salida siempre cae a JSON y `writeAskText` nunca se ejecuta en ningún test del brief. Con `IsTTY:true` y sin `--json`/`--md` se fuerza el camino de `Text` y se comprueba que cita la fuente con el formato propio de texto plano (`  · Informe anual, p. 12 (doc-1)`), y explícitamente que **no** se cuela el formato Markdown (`### Fuentes`, `**Informe anual**`).

2. **Aserciones reforzadas en los tests del brief** (aviso 2 — "un test que solo verifique que la petición se hizo no detecta un tag mal escrito ni un campo que no se muestra"):
   - `TestAskDevuelveTextoYFuentes`: además de `len(Sources)==1`, ahora compara `Citation` y `DocumentID` exactos, el `Summary` exacto (`"1 fuentes citadas"`) y los dos `Breadcrumbs` exactos (`Action`+`Cmd`), no solo `len(Breadcrumbs) != 0`.
   - `TestAskEnMarkdownCitaLasFuentes`: además de que la cita aparezca en algún sitio, ahora exige la cabecera `### Fuentes` y la línea completa `**Informe anual** — Informe anual, p. 12 (\`doc-1\`)`. Sin esto, el test pasaba igual si por error se llamaba a `writeAskText` en el hueco de Markdown (verificado: ver mutación M3 abajo).

3. **`TestAskConRespuestaSinExitoDevuelveError`**: el camino `!resp.Success` (mensaje del backend + hint de recuperación) no estaba cubierto por ningún test del brief. Comprueba que `success:false` con `error` del backend se traduce en `*output.CLIError` con el mensaje exacto del backend, un hint que menciona `calliope concepts list`, y código de salida 1.

4. **`TestAskEnviaLaAccionSolicitada`**: el flag `--action` se declara pero ningún test del brief comprueba que llegue al backend. Captura el cuerpo de la petición y comprueba `question` y `action` exactos.

## Verificación por mutación

Sobre una copia completa del repo en `/tmp/calliope-mut` (build limpio confirmado antes de empezar), apliqué y revertí 11 mutaciones sobre `ask.go`, cada una con `go test ./internal/commands/ -run Ask`:

| # | Mutación | Resultado |
|---|---|---|
| M1 | `exactArgs(1)` → `cobra.ExactArgs(1)` | FAIL (`TestAskSinPreguntaEsErrorDeUso`: ya no es `*output.CLIError`) |
| M2 | `if !resp.Success` → `if resp.Success` | FAIL (5 tests) |
| M3 | Intercambiar `Text`/`Markdown` (`writeAskText`↔`writeAskMarkdown`) | FAIL (`TestAskEnMarkdownCitaLasFuentes` y `TestAskEnTextoCitaLasFuentes`, cada uno detecta el formato equivocado en su lado) |
| M4 | `writeAskText` pierde `s.DocumentID` en la cita | FAIL (`TestAskEnTextoCitaLasFuentes`) |
| M5 | `writeAskMarkdown` pierde `s.DocumentID` entre backticks | FAIL (`TestAskEnMarkdownCitaLasFuentes`) |
| M6 | Breadcrumb `"calliope docs show <id>"` alterado | FAIL (`TestAskDevuelveTextoYFuentes`) |
| M7 | `resumen` siempre `"0 fuentes citadas"` | FAIL (`TestAskDevuelveTextoYFuentes`) |
| M8 | Hint de error sin mención a `calliope concepts list` | FAIL (`TestAskConRespuestaSinExitoDevuelveError`) |
| M9 | Se ignora `resp.Error`, queda siempre el mensaje genérico | FAIL (`TestAskConRespuestaSinExitoDevuelveError`) |
| M10 | `accion` no se propaga a `Client.Ask` (se manda `""`) | FAIL (`TestAskEnviaLaAccionSolicitada`) |
| M11 | `ctx.Org` sustituido por un literal fijo | FAIL (`TestAskDevuelveTextoYFuentes`: ruta inesperada) |

Tras cada mutación se restauró el fichero desde una copia intacta (`ask.go.orig`); al final se comprobó con `diff` que el fichero mutado quedó **idéntico** al del repositorio real, y la suite completa (`go test ./... -race`, `go vet ./...`, `gofmt -l .`, `go mod tidy`) se repitió limpia en el árbol de trabajo real después de restaurar. El directorio temporal se borró al terminar.

## Verificación manual adicional

Con `go run ./cmd/calliope --help` y `go run ./cmd/calliope ask --help`: `ask` aparece registrado en la raíz real (no solo en `testRoot`), con el `Short` en español y los flags `--action` + los cinco globales heredados correctamente.

## Dudas / puntos para el controlador

1. El fix de `go vet` en `writeAskMarkdown` (`Fprintln`→`Fprint`) no estaba en las correcciones listadas del encargo, pero era necesario para que `go vet ./...` quedara limpio (restricción global). El byte a byte de la salida es idéntico al que produciría el código literal del brief; documentado también como comentario nada — no añadí comentario en el código porque el cambio es mecánico y no cambia intención, pero lo dejo anotado aquí por transparencia.
2. No hubo identificadores en español a nivel de paquete sin traducción ya fijada en el glosario: `NewAskCmd` ya venía en inglés en el brief; `escribirAskTexto`/`escribirAskMarkdown` ya estaban en la tabla de traducciones fijadas (línea 40-41 del glosario).
