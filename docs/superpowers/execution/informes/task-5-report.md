# Task 5 — Frontera de confianza de la configuración de proyecto

**Estado:** DONE

**Commit:** `99e06ab` — "feat: frontera de confianza de la configuración de proyecto"
(rama `feat/cli-v1`, sobre `115a360`)

## Resumen de tests

`go test ./... -race` → 16/16 tests en `internal/config` en verde (13 previos de la
Task 4 intactos + 3 nuevos del brief + 1 test adicional que añadí yo); `go vet ./...`
limpio; `gofmt -l .` sin salida.

## Qué se hizo

1. **`internal/config/trust_test.go`** (nuevo). Copié literalmente los tres tests del
   brief y sus ayudantes, traduciendo solo los identificadores de nivel de paquete
   según el glosario:
   - `escribirConfigDeProyecto` → `writeProjectConfig`
   - `escribirJSON` → `writeJSON`
   - `entornoDePrueba` → `testEnv`
   - Los nombres de test (`TestRepositorioHostilNoPuedeCambiarBaseURL`,
     `TestLaConfiguracionGlobalSiPuedeFijarBaseURL`,
     `TestLaConfiguracionDeProyectoNuncaAportaCredenciales`) se dejaron en español,
     tal cual el brief.
   - Comprobé colisiones con `config_test.go` (que ya existía de la Task 4, con
     precedencia par a par y tests de `All`/`BaseURL`/`Org`/`Output`/error con
     ruta): ninguno de los tres ayudantes (`writeProjectConfig`, `writeJSON`,
     `testEnv`) estaba ya definido en el paquete, así que los creé sin necesidad de
     reutilizar nada. No toqué `config_test.go`.

2. **`internal/config/trust.go`** — sustituido por completo. El `sanitize` provisional
   de la Task 4 (identidad) fue reemplazado por la implementación definitiva del
   brief: `ProjectAllowed` (solo `org` y `output`) y `sanitize` filtrando capas
   `SourceProject`/`SourceRepo`, devolviendo un aviso por cada campo descartado.
   Copiado literalmente (mensajes, comentarios) salvo el nombre `sanear` → `sanitize`
   (ya fijado por el glosario y ya en vigor desde la Task 4).

3. **`internal/config/layers.go`** — añadido el comentario pedido en el Step 5, justo
   antes del `return` final de `Load`, indicando que los avisos se devuelven en vez
   de imprimirse porque quien decide dónde escribirlos es `appctx`.

4. Seguí el orden TDD del brief: escribí `trust_test.go` primero, confirmé que 2 de
   los 3 tests fallaban con el `sanitize` identidad provisional (`go test
   ./internal/config/ -run Test -v`, salida exacta: `base_url =
   "https://atacante.example.com" — un repositorio hostil ha conseguido redirigir el
   CLI`), implementé `trust.go`, y confirmé que los 16 tests pasan.

## Test adicional (no está en el brief) y por qué lo añadí

Al hacer la verificación por mutación pedida (ver abajo), descubrí que **ningún test
de la suite — ni los 13 de la Task 4 ni los 3 del brief — detecta si se elimina el
saneado de la capa `SourceRepo`** (dejando `sanitize` activo solo para
`SourceProject`). La razón: `Load` solo puebla `SourceRepo` cuando `cwd` es un
*subdirectorio* de la raíz del repo (`raiz != cwd`); los tres tests del brief ejecutan
`Load` con `cwd` igual a la raíz del repositorio clonado, así que nunca ejercitan esa
rama. Es exactamente el escenario de un monorepo o de `calliope ask` lanzado desde un
subdirectorio de un repositorio hostil clonado — el propio caso de amenaza que motiva
la tarea.

Añadí `TestRepositorioHostilEnLaRaizNoPuedeCambiarBaseURLDesdeUnSubdirectorio`: crea
`raiz/.git/` y `raiz/.calliope/config.json` con `base_url` hostil, y llama a
`Load(raiz/paquetes/app, ...)`. Confirmé que:
- Con la implementación real, pasa.
- Con la mutación que quita `SourceRepo` del guard de `sanitize`, falla (ver
  Mutación 6 abajo), mientras que sin este test esa misma mutación pasaba
  desapercibida por toda la suite.

Uso los mismos ayudantes (`writeProjectConfig`, `testEnv`) para no duplicar nada.

## Verificación por mutación (fuera del repo, en un directorio temporal separado)

Copié el árbol completo a un directorio fuera del repositorio
(`.../scratchpad/mutation-test`, borrado al terminar) y apliqué mutaciones puntuales
sobre `trust.go`, ejecutando `go test ./internal/config/ -v` tras cada una. Resultado:

| # | Mutación | Test(s) que debía(n) fallar | Resultado |
|---|---|---|---|
| 1 | `sanitize` vuelve a ser la identidad (`return l, nil`) — el provisional original de la Task 4 | `TestRepositorioHostilNoPuedeCambiarBaseURL`, `TestRepositorioHostilEnLaRaizNoPuedeCambiarBaseURLDesdeUnSubdirectorio`, `TestLaConfiguracionDeProyectoNuncaAportaCredenciales` | **FALLAN los tres**, como exige el encargo. `TestLaConfiguracionGlobalSiPuedeFijarBaseURL` sigue en verde (correcto: la capa global nunca se sanea). |
| 2 | `ProjectAllowed` añade `KeyBaseURL: true` | `TestRepositorioHostilNoPuedeCambiarBaseURL` | FALLA (`base_url = "https://atacante.example.com"...`) |
| 3 | `ProjectAllowed` añade `KeyTimeout: true` | `TestRepositorioHostilNoPuedeCambiarBaseURL` | FALLA (`timeout = "1ms"...`) |
| 4 | `sanitize` filtra bien los valores pero siempre devuelve `avisos == nil` | `TestRepositorioHostilNoPuedeCambiarBaseURL`, `TestLaConfiguracionDeProyectoNuncaAportaCredenciales` | FALLAN ambos (aserciones sobre `len(avisos)` y el contenido del aviso) |
| 5 | El guard de `sanitize` también incluye `SourceGlobal` (sobre-restrictivo) | *(exploratoria)* | **No falla ningún test** — hallazgo documentado: `Load` nunca invoca `sanitize` sobre la capa global (el guard está en el punto de llamada, no solo dentro de `sanitize`), así que esta rama del `if` es defensiva pero no alcanzable desde `Load` hoy. No es un bug: la propiedad de seguridad relevante (la capa global sí puede fijar `base_url`) ya está cubierta extremo a extremo por `TestLaConfiguracionGlobalSiPuedeFijarBaseURL`, que ejercita el camino real. Lo dejo anotado por transparencia, no requiere acción. |
| 6 | El guard de `sanitize` deja de cubrir `SourceRepo` (solo `SourceProject`) | *Antes de añadir el test nuevo:* toda la suite (16 tests) pasaba — hueco real. *Después de añadir `TestRepositorioHostilEnLaRaizNoPuedeCambiarBaseURLDesdeUnSubdirectorio`:* ese test FALLA como se esperaba. | Motivó el test adicional descrito arriba. |

Tras cada mutación restauré `trust.go` a la implementación real antes de la siguiente,
y al terminar confirmé de nuevo la suite completa en verde sobre la copia y borré el
directorio de mutación. El repo real (`/Users/j10/repositories/calliope/calliope-cli`)
no fue tocado por estas mutaciones en ningún momento; solo se modificó la copia en
`scratchpad/mutation-test`.

## Ayudantes: colisiones comprobadas

Ninguno de `writeProjectConfig`, `writeJSON`, `testEnv` existía ya en el paquete
`config` (revisado `config.go`, `layers.go`, `config_test.go`). No hubo que reutilizar
nada; tampoco hubo que renombrar nada del brief más allá de lo que ya fija el
glosario.

## Dudas / nada pendiente

- El glosario no menciona `escribirConfigDeProyecto`/`escribirConfigProyecto` con ese
  nombre exacto de este brief (usa `writeProjectConfig`), coincide con la entrada ya
  existente `escribirConfigProyecto / escribirConfigDeProyecto → writeProjectConfig`,
  así que no hay ambigüedad ni necesidad de anotar una traducción nueva.
- El único hallazgo no trivial es la Mutación 6 / el test añadido; lo marco explícito
  por si el equipo prefiere que ese test viva en su propia tarea o revisión en vez de
  aquí. Lo dejé en este commit porque protege exactamente la propiedad de seguridad
  que da nombre a la tarea (frontera de confianza de la configuración de proyecto)
  y se descubrió durante el propio trabajo de esta tarea.

---

## Ronda de correcciones 1/5

**Motivada por la revisión de seguridad.** Confirmó que la frontera bloquea mayúsculas,
espacios, claves duplicadas, homoglifos cirílicos y claves futuras desconocidas, y que
el test añadido en la ronda anterior (subdirectorio de un repo hostil) cierra un hueco
real. Quedaba un hallazgo Important y dos remates.

### IMPORTANT — `ProjectAllowed` exportado y mutable → endurecido

1. El mapa pasa a llamarse `projectAllowed` (sin exportar).
2. Se añade `func IsProjectAllowed(key string) bool`, único punto de consulta público
   de la frontera para el resto del paquete y para tareas futuras.
3. Se añadió un comentario de "AVISO DE SEGURIDAD" sobre el mapa explicando por qué no
   se exporta y por qué ampliarlo debe tocar el test que fija su contenido exacto.
4. Se añadió `TestProjectAllowedEsExactamenteOrgYOutput`, que recorre las cuatro claves
   conocidas del paquete (`KeyOrg`, `KeyBaseURL`, `KeyOutput`, `KeyTimeout`) y varias
   claves fuera del catálogo (`api_key`, `token`, `proxy`, `""`), comprobando que
   `IsProjectAllowed` es `true` exactamente para `org` y `output` y `false` para todo
   lo demás.

**Prueba de que el endurecimiento es real, no solo nominal:** además del test, se
comprobó por compilación que el símbolo ya no es accesible desde fuera del paquete.
Se creó temporalmente `internal/config/_probe_tmp/main.go` con
`config.ProjectAllowed[config.KeyBaseURL] = true` y se intentó compilar:

```
internal/config/_probe_tmp/main.go:6:9: undefined: config.ProjectAllowed
```

El compilador rechaza el acceso — ya no es un bypass posible en absoluto, ni siquiera
por descuido de un futuro PR que use el nombre viejo. El directorio de sondeo se borró
inmediatamente después (no forma parte del commit; `git status` quedó limpio antes de
comitear).

### Remate 1 — camino positivo de `output`

Se añadió `TestLaConfiguracionDeProyectoSiPuedeFijarOutput`: escribe
`{"output": "json"}` en `.calliope/config.json` de proyecto, carga con `Load` y
comprueba `cfg.Output() == "json"`, que la procedencia (`Source`) sea `SourceProject`,
y que no se genere ningún aviso (output es una clave permitida, no debe reportarse
como ignorada).

### Remate 2 — comentario sobre la rama redundante en `sanitize`

Se documentó con un comentario el `if l.Source != SourceProject && l.Source !=
SourceRepo` dentro de `sanitize`: explica que hoy es inalcanzable desde `Load` (el
filtro real está en el punto de llamada, ver `layers.go`), que es defensa en
profundidad deliberada y que no debe eliminarse por parecer código muerto.

### Verificación por mutación de esta ronda

Igual que en la ronda anterior: copia completa fuera del repo
(`scratchpad/mutation-test-r1`, borrada al terminar), mutaciones puntuales sobre
`trust.go`, `go test ./internal/config/ -v` tras cada una, restauración entre
mutaciones. El repo real no se tocó en ningún momento durante estas pruebas.

| # | Mutación | Test(s) que debía(n) fallar | Resultado |
|---|---|---|---|
| 7 | `projectAllowed` gana una tercera clave (`KeyTimeout: true`) sin tocar ningún test | `TestProjectAllowedEsExactamenteOrgYOutput`, `TestRepositorioHostilNoPuedeCambiarBaseURL` | **FALLAN ambos.** `IsProjectAllowed("timeout") = true, se esperaba false`; y el timeout hostil (`1ms`) pasa a fijarse. |
| 8 | `IsProjectAllowed` siempre devuelve `true` (bypass total del filtro) | `TestRepositorioHostilNoPuedeCambiarBaseURL`, `TestRepositorioHostilEnLaRaizNoPuedeCambiarBaseURLDesdeUnSubdirectorio`, `TestLaConfiguracionDeProyectoNuncaAportaCredenciales`, `TestProjectAllowedEsExactamenteOrgYOutput` | **FALLAN los cuatro.** |
| 9 | `sanitize` dejar de consultar `IsProjectAllowed` y solo permitir `KeyOrg` a mano (ignora `output`) | `TestLaConfiguracionDeProyectoSiPuedeFijarOutput` | **FALLA solo este test.** Los tres tests del repositorio hostil (`org` sigue funcionando, `base_url`/`api_key` siguen bloqueados) **pasan igual** — confirma exactamente lo que pedía el Remate 1: sin este test, una regresión que rompe el camino de `output` habría pasado desapercibida por toda la suite anterior. |

Tras la última mutación se restauró `trust.go` a la implementación real, se confirmó
la suite completa (18 tests) en verde sobre la copia, y se borró el directorio de
mutación.

### Verificación final

`go build ./...` OK · `go test ./... -race -v` → 18/18 en `internal/config` en verde
(el resto de paquetes sin cambios) · `go vet ./...` limpio · `gofmt -l .` sin salida.

### Commit

`2fd6fcd` — "fix: endurece la frontera de confianza tras revisión de seguridad"

### Dudas

Ninguna. El único punto a vigilar hacia delante: en cuanto otra tarea necesite
consultar la frontera de confianza desde fuera de `internal/config` (p. ej. `calliope
config list` marcando qué claves son fijables por proyecto), debe usar
`config.IsProjectAllowed(key)` y no reintroducir un mapa exportado.
