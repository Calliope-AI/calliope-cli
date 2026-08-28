# Informe — Task 21: Distribución con GoReleaser

## Estado

DONE_WITH_CONCERNS

Todo el código Go (aviso de nueva versión, wiring de `version` en el comando
`version`) está implementado con TDD y verificado por mutación, incluida una
comprobación de extremo a extremo del comando real. El único motivo de
"WITH_CONCERNS" es explícito y estaba previsto por el encargo: **`.goreleaser.yaml`
no se ha verificado con la propia herramienta GoReleaser**, porque no está
instalada y no debía instalarse sin permiso del usuario (sería una descarga).
Ver "GoReleaser sin verificar" más abajo.

## Commit

`02b514a` — feat: distribución con GoReleaser, instalador y aviso de nueva versión
(8 ficheros: `.goreleaser.yaml`, `.github/workflows/release.yml`, `install.sh`,
`Makefile`, `internal/version/check.go`, `internal/version/check_test.go`,
`internal/cli/root.go`, `internal/cli/root_test.go`)

BASE = `7256fcc578d4e226fe64060430c337dacf2d035c`

## Resumen de tests

`go build ./...`, `go vet ./...`, `gofmt -l .` (sin salida) y `go test ./... -race`
→ PASS en los 9 paquetes; `go mod tidy` sin cambios en `go.mod`/`go.sum`. 6/6
mutaciones dirigidas sobre `internal/version/check.go` cazadas (una necesitó un
test nuevo, ver abajo) y 1/1 mutación dirigida sobre el gate TTY de
`internal/cli/root.go` cazada, todas verificadas restaurando el fichero
original tras cada mutación. `install.sh`: `bash -n` limpio, detección de
arquitectura probada por separado para `x86_64`/`aarch64`/`arm64`/no soportada,
y confirmado por lectura que la verificación de checksum precede a
`tar`/`install`.

## Qué se hizo

- **Step 6-7 (TDD)**: escribí `internal/version/check_test.go` (los 4 tests del
  brief, copiados literalmente salvo el nombre de la función bajo prueba) y
  confirmé el fallo esperado: `go test ./internal/version/ -v` →
  `undefined: LatestVersion` (4 apariciones). Implementé
  `internal/version/check.go` — cuerpo copiado literalmente del brief, con la
  única diferencia de que `ReleasesURL` es `var`, no `const` (ver "Decisión:
  `var` en vez de `const`" abajo). 4/4 tests en verde.
- **`internal/cli/root.go`**: `newVersionCmd` pasa a tomar `d appctx.Deps` y,
  solo si `d.IsTTY`, llama a `version.LatestVersion(version.ReleasesURL,
  version.Version, 2*time.Second)` e imprime el aviso si hay versión más
  reciente. `NewRootCmd` actualizado a `newVersionCmd(d)`.
  `internal/cli/root_test.go` (Task 1) **ya** usaba `NewRootCmd(appctx.Deps{})`
  — con `IsTTY` en cero (falso) no consulta la red; confirmado cronometrando
  `TestVersionImprimeLaVersion` en 0.00s. No hizo falta tocarlo, pero añadí un
  test nuevo (`TestVersionAvisaSoloEnTTY`, ver abajo).
- **`.goreleaser.yaml`, `.github/workflows/release.yml`, `install.sh`,
  `Makefile`**: copiados literalmente del brief, con la corrección de
  `go-version: '1.25'` en `release.yml` (el brief traía `'1.22'`; el suelo real
  del módulo es 1.25, igual que ya tiene `ci.yml`, que no se tocó).
- Commit único con los ocho ficheros, mensaje explicando ambas correcciones
  (Go 1.25, GoReleaser sin verificar) para que quede en el historial, no solo
  en este informe.

## Las cuatro correcciones del encargo, aplicadas

1. **GoReleaser no instalado.** Salté `goreleaser check` y
   `goreleaser build --snapshot --clean --single-target` (Step 2) y la mitad de
   `goreleaser check && go test ./... -race` del Step 8 (solo corrí
   `go test ./... -race`). Validé el YAML a mano: `python3 -c "import yaml;
   yaml.safe_load(open('.goreleaser.yaml'))"` parsea sin error y el árbol
   resultante (impreso como JSON para inspección) tiene la forma esperada —
   `builds`/`archives`/`checksum`/`nfpms`/`brews`/`scoops`/`release` con las
   claves del brief. **Esto NO es una verificación semántica contra el schema
   real de GoReleaser v2** (p. ej. si `formats` es el nombre de campo correcto
   en la versión de GoReleaser que se use en CI, o si `nfpms`/`brews`/`scoops`
   aceptan exactamente esas claves sin typos). Copié el bloque literal del
   brief sin inventar ningún campo adicional, pero la única verificación real
   —`goreleaser check`— queda pendiente como acción del usuario o de CI.
2. **Go 1.25, no 1.22.** `release.yml` usa `go-version: '1.25'`. Confirmé por
   YAML que coincide exactamente con `ci.yml` (que no toqué).
3. **`newVersionCmd(d)`.** Hecho; ver arriba. `root_test.go` de la Task 1 ya
   estaba en la forma pedida.
4. **`cmd/calliope/main.go` no tocado.** Confirmado: no aparece en el diff del
   commit (`git show --stat 02b514a`).

## Decisión: `var` en vez de `const` para `ReleasesURL`

El brief declara `const URLReleases = "https://api.github.com/repos/..."`. Lo
implementé como `var`, mismo nombre traducido (`ReleasesURL`), mismo valor por
defecto, mismo uso. Motivo: al verificar por mutación el gate `if d.IsTTY` de
`root.go` (ver tabla abajo), descubrí que **ningún test existente lo
ejercitaba de verdad** — `version.Version` vale `"dev"` en todos los tests de
`internal/cli`, así que `LatestVersion` se calla por la guarda de compilación
de desarrollo *antes* de que importe si `d.IsTTY` es verdadero o falso. Un
`if d.IsTTY {` cambiado a `if true {` pasaba la suite entera en verde. Para
cerrar ese hueco con un test real de extremo a extremo (no solo de
`LatestVersion` en aislado) necesitaba poder apuntar `newVersionCmd` a un
servidor de prueba sin tocar la red real, y con `const` eso es imposible.
Cambiarlo a `var` es el patrón estándar de Go para esto, no cambia el
comportamiento en producción (sigue siendo un string de solo lectura fijado al
arrancar) y el comentario en el propio código explica por qué es variable.

**Coste si me equivoco:** ninguno en producción — nadie reasigna
`version.ReleasesURL` fuera de tests; en tests, si alguien no restaura el valor
original con `defer`, un test de `internal/version` que llame a
`LatestVersion(ReleasesURL, ...)` sin pasar su propia URL heredaría la URL
mutada de otro test que corriera antes. Los tests de `internal/version`
siempre pasan su propia `url` como argumento (nunca usan la constante global),
así que este riesgo no se materializa hoy.

## Traducción de identificadores (glosario)

No estaban en la tabla fijada del glosario; los añado aquí para que se
incorpore:

| Brief (español) | Usado |
|---|---|
| `UltimaVersion` | `LatestVersion` |
| `URLReleases` | `ReleasesURL` |

Nota sobre `ReleasesURL` vs `URLReleases`: no es estrictamente una traducción
(ambas palabras ya son inglesas), es un reordenamiento al patrón que ya usa el
propio proyecto (`ctx.Cfg.BaseURL()` en `appctx`), para que `XxxURL` sea
consistente en toda la base. Parámetros de función (`url`, `actual`,
`timeout`) y variables locales del cuerpo (`cliente`, `resp`, `cuerpo`) se
dejaron tal como las escribe el brief, en español, conforme al alcance del
glosario ("las variables locales dentro de un cuerpo de función se quedan como
las escribe el brief"). Los nombres de test se dejaron en español.

## Verificación por mutación — `internal/version/check.go`

Copié `version.go`, `check.go` y `check_test.go` a un módulo Go aislado en el
scratchpad (fuera del repo), confirmé la base en verde, y apliqué/revertí 6
mutaciones, restaurando desde una copia `check.go.orig` tras cada una:

| # | Mutación | Resultado |
|---|---|---|
| 1 | Quitar la guarda `actual == "" \|\| actual == "dev"` (`if false {`) | FAIL — `TestEnCompilacionDeDesarrolloNoConsulta` (consulta la red en dev) |
| 2 | Invertir `resp.StatusCode != http.StatusOK` a `==` | FAIL — `TestDetectaUnaVersionMasReciente` |
| 3 | Invertir `cuerpo.TagName == actual` a `!=` | FAIL — `TestDetectaUnaVersionMasReciente` y `TestNoAvisaSiYaEstaAlDia` |
| 4 | Quitar `Timeout: timeout` del `http.Client{}` | **SOBREVIVIÓ** a los 4 tests del brief (ver abajo) |
| 5 | Quitar la comprobación `cuerpo.TagName == "" \|\| cuerpo.TagName == actual` | FAIL — `TestNoAvisaSiYaEstaAlDia` |
| 6 | Quitar `if err != nil { return "" }` tras `cliente.Get` | FAIL — `TestSinRedNoRompe`, panic por nil pointer (`resp` nil) recuperado por el test runner como fallo |

**Mutación 4 sobrevivió** porque `TestSinRedNoRompe` apunta a
`http://127.0.0.1:1`, un puerto que rechaza la conexión al instante
("connection refused") — nunca llega a esperar el timeout, así que quitar el
`Timeout` del cliente no cambia su resultado. Añadí
`TestRespetaElTimeout` (servidor de prueba que duerme 300 ms, timeout de 50 ms,
comprobando tanto el resultado vacío como que el tiempo transcurrido queda
acotado por debajo de 250 ms) y confirmé que la misma mutación 4, repetida
sobre la copia con el test nuevo, ahora falla:
`LatestVersion tardó 303.420417ms, el timeout de 50ms no se respetó`.

Tras cada mutación comparé con `diff` que la copia quedó idéntica al original
antes de borrar el directorio de mutación. El árbol de trabajo real no se tocó
en ningún momento durante este proceso.

## Verificación por mutación — gate TTY en `internal/cli/root.go`

Sobre el árbol de trabajo real, con backup y restauración inmediata
(`cp root.go /tmp/root.go.backup` → mutar → test → `cp /tmp/root.go.backup
root.go` → `diff` confirmando restauración exacta):

| # | Mutación | Resultado antes de añadir el test | Resultado después |
|---|---|---|---|
| 1 | `if d.IsTTY {` → `if true {` (el aviso se intentaría siempre) | **SOBREVIVIÓ** — ningún test de `internal/cli` fija `version.Version` a algo distinto de `"dev"`, así que `LatestVersion` se calla igual por la guarda de dev, TTY o no | FAIL — `TestVersionAvisaSoloEnTTY/sin_TTY_no_avisa_ni_consulta_la_red` (con el test nuevo, que sí fija `version.Version = "v1.2.0"`) |

Esto es lo que motivó el test `TestVersionAvisaSoloEnTTY` (y, indirectamente,
la decisión de `var` sobre `ReleasesURL` de más arriba): sin él, el requisito
explícito "el aviso solo sale en TTY" no tenía ningún test que de verdad lo
protegiera — dependía por accidente de que `version.Version` fuera siempre
`"dev"` en la suite.

## Verificación manual del binario real

- **Con `Version` en `dev` (build normal), en pipe:** `calliope version | cat`
  imprime solo `calliope dev (none, unknown)`, en 0.008s — confirma que sin
  TTY no hay intento de red, ni con `Version` real ni con `dev`.
- **Con `Version` inyectada por `-ldflags` (`v0.1.0`), bajo un pty real
  (`script -q /dev/null`):** el sandbox sí tiene red de salida; el endpoint
  real (`api.github.com/repos/calliope/calliope-cli/...`, que no existe como
  repo público) respondió 404. El binario imprimió solo la línea de versión,
  sin aviso (correcto: `LatestVersion` se calla ante un `StatusCode != 200`),
  sin colgarse, en 0.7s — bien por debajo del timeout de 2s.
- Confirmado con `curl` directo que hay conectividad de red real en este
  entorno (HTTP 404 en 0.3s), así que la ausencia de aviso en la prueba
  anterior es por el 404 real, no por falta de red.

## `install.sh`: verificación sin ejecutar descargas

- `bash -n install.sh` → sin errores de sintaxis.
- Detección de arquitectura probada en aislado (extrayendo el mismo `case`
  a un subshell, sin tocar el resto del script): `x86_64`→`amd64`,
  `aarch64`→`arm64`, `arm64`→`arm64`, `riscv64`→sale con código 1 y el mensaje
  `Arquitectura no soportada: riscv64` en stderr.
- Confirmado por lectura (`grep -n`) que la línea del checksum
  (`grep " ${archivo}\$" checksums.txt | shasum -a 256 -c -`, línea 35) está
  **antes** de `tar -xzf` (línea 37) e `install -m 0755` (línea 38); con
  `set -euo pipefail` activo, un checksum que no verifique aborta el script
  antes de extraer o instalar nada.
- No ejecuté el script de verdad: ni `curl` contra GitHub Releases (no existe
  ningún release real todavía) ni `install` sobre `/usr/local/bin`.

## Pendiente / fuera de mi alcance

- **`goreleaser check` y `goreleaser build --snapshot`**: acción pendiente del
  usuario, o de la primera ejecución de CI. No instalé GoReleaser.
- **`TAP_GITHUB_TOKEN`**: como señala el propio brief, hay que crear ese
  secreto (con permiso de escritura sobre `calliope/homebrew-tap` y
  `calliope/scoop-bucket`) antes del primer release de verdad; no es algo que
  yo pueda hacer desde aquí.
- No se ha creado ningún tag `v*` ni disparado el workflow de release —
  correcto, no era parte del encargo.

---

# Ronda de correcciones 1/5

## Commit de esta ronda

`722e1ffe332c1930e85465a3b097935256e51c1a`

8 ficheros: `.goreleaser.yaml`, `install.sh`, `internal/appctx/appctx.go`,
`internal/appctx/appctx_test.go`, `internal/cli/root.go`,
`internal/cli/root_test.go`, `internal/version/check.go`,
`internal/version/check_test.go`.

## Resumen de tests (una línea)

`go build ./...`, `go vet ./...`, `gofmt -l .` (sin salida), `go test ./... -race`
(9 paquetes, PASS) y `go mod tidy` (sin cambios) limpios; `bash -n install.sh`
limpio; 4 mutaciones dirigidas nuevas cazadas (normalización del prefijo `v`
×2, `DefaultDeps` sin cablear `ReleasesURL`, checksum manipulado tras el
cambio a `sha256sum`/`shasum` con fallback) más las 7 mutaciones de la ronda
anterior repetidas y siguiendo cazadas tras los cambios de esta ronda.

## CRITICAL 1 — `name_template` acoplado a `install.sh`

**Antes de tocar nada, verifiqué el comportamiento real contra la
documentación de GoReleaser en vivo** (no de memoria): hice `curl` a
`https://goreleaser.com/customization/archive/` y a
`https://goreleaser.com/customization/templates/`. La página confirma que la
build actual (**GoReleaser v2.18**, "Last updated on August 11, 2026") es:

- El **valor por defecto documentado** de `name_template` en `archives:` es:
  `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}`
  — es decir, usa `.Os`/`.Arch` **crudos**, sin `title` ni sustitución a
  `x86_64`.
- `.Os` = GOOS literal (`linux`, `darwin`, `windows`); `.Arch` = GOARCH literal
  (`amd64`, `arm64`) — confirmado en la sección "Single-artifact extra
  fields" de la página de templates.
- `.Version` = "el tag sin el prefijo `v`" (footnote explícita: *"The v
  prefix is stripped"*).

**Esto no coincide exactamente con lo que decía la corrección** ("por
defecto: sistema operativo en Title Case y `amd64` traducido a `x86_64`").
Ese comportamiento (`Darwin`, `x86_64`) era el de GoReleaser v1.x vía el
campo `replacements:`, deprecado y **eliminado** en v2. Con la doc actual, el
default de v2.18 para nuestra matriz (sin ARM32/MIPS, `GOAMD64` por defecto
`v1`) se simplifica a `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}`
— que coincidiría con `install.sh` **hoy**, sin necesidad de `name_template`
explícito.

**Implementé el `name_template` explícito de todas formas**, porque el
argumento de fondo de la corrección sigue siendo válido y de hecho mi propia
verificación lo refuerza: el default **ya cambió una vez** entre v1 y v2, el
workflow fija `version: latest` para `goreleaser-action`, y depender de qué
produzca el default de turno es exactamente la clase de acoplamiento
implícito y frágil que un `name_template` explícito elimina. Lo dejé
comentado en ambos ficheros, tal como se pidió.

**Tabla comparativa**, `.Version` = `1.4.0` (tag `v1.4.0`, el mismo de los
fixtures del brief), `.ProjectName` = `calliope`:

| goos | goarch | Mi `name_template` explícito | Default de GoReleaser **v2.18** (verificado en vivo) | Default "clásico" v1.x (`Title(Os)` + `amd64`→`x86_64`, el que describía la corrección) | Lo que `install.sh` construye |
|---|---|---|---|---|---|
| darwin | amd64 | `calliope_1.4.0_darwin_amd64.tar.gz` | `calliope_1.4.0_darwin_amd64.tar.gz` (**coincide**) | `calliope_Darwin_x86_64.tar.gz` (sin versión, Title Case, x86_64 — **no coincide**) | `calliope_1.4.0_darwin_amd64.tar.gz` |
| darwin | arm64 | `calliope_1.4.0_darwin_arm64.tar.gz` | `calliope_1.4.0_darwin_arm64.tar.gz` (**coincide**) | `calliope_Darwin_arm64.tar.gz` (**no coincide**) | `calliope_1.4.0_darwin_arm64.tar.gz` |
| linux | amd64 | `calliope_1.4.0_linux_amd64.tar.gz` | `calliope_1.4.0_linux_amd64.tar.gz` (**coincide**) | `calliope_Linux_x86_64.tar.gz` (**no coincide**) | `calliope_1.4.0_linux_amd64.tar.gz` |
| linux | arm64 | `calliope_1.4.0_linux_arm64.tar.gz` | `calliope_1.4.0_linux_arm64.tar.gz` (**coincide**) | `calliope_Linux_arm64.tar.gz` (**no coincide**) | `calliope_1.4.0_linux_arm64.tar.gz` |
| windows | amd64 | `calliope_1.4.0_windows_amd64.zip` | `calliope_1.4.0_windows_amd64.zip` (**coincide**) | `calliope_Windows_x86_64.zip` (**no coincide**) | (`install.sh` no soporta Windows; fuera de alcance) |
| windows | arm64 | `calliope_1.4.0_windows_arm64.zip` | `calliope_1.4.0_windows_arm64.zip` (**coincide**) | `calliope_Windows_arm64.zip` (**no coincide**) | (idem) |

Con mi `name_template` explícito, las cuatro combinaciones que `install.sh`
sí instala (darwin/linux × amd64/arm64) coinciden byte a byte con lo que el
script construye, **con independencia de qué versión de GoReleaser se acabe
usando** en CI. Esto sigue sin estar verificado contra la herramienta real
(ver "GoReleaser sin verificar" al principio de este informe): la
verificación de arriba es contra la documentación en vivo, no una ejecución
de `goreleaser check`/`build`.

## CRITICAL 2 — `TAP_GITHUB_TOKEN` sin conectar

Confirmé el nombre exacto del campo contra la documentación en vivo de
`goreleaser.com/customization/homebrew/` y `.../scoop/`: ambos publicadores
comparten el mismo tipo `repository:` (`owner`, `name`, `token`, ...). Añadí
`token: "{{ .Env.TAP_GITHUB_TOKEN }}"` dentro de `brews[].repository` y
`scoops[].repository`, con un comentario explicando por qué hace falta (el
`GITHUB_TOKEN` del workflow no tiene permiso de escritura sobre los repos del
tap). Verificado por lectura del YAML parseado (ver JSON en la sección
siguiente): ambos bloques `repository` incluyen ahora la clave `token`.

## IMPORTANT 3 — prefijo `v` al comparar versiones

TDD: añadí `TestElPrefijoVNoImportaAlComparar` (dos subtests: `actual` sin
`v`/tag con `v` — el caso real de un binario compilado por GoReleaser — y al
revés) a `internal/version/check_test.go`, confirmé que fallaba contra la
implementación anterior (`LatestVersion(actual="1.2.0", tag="v1.2.0")`
devolvía `"v1.2.0"` en vez de `""`), y luego añadí `normalizeVersion` (recorta
un prefijo `"v"` con `strings.TrimPrefix`) y comparé versiones normalizadas
en `LatestVersion`. El valor **devuelto** al usuario sigue siendo el
`tag_name` tal cual (con su `v` si lo trae): solo cambia la comparación.

## Punto 4 — `ReleasesURL` vuelve a `const`, enrutado por `appctx.Deps`

- `internal/version/check.go`: `ReleasesURL` vuelve a ser `const` (como en el
  brief original).
- `internal/appctx/appctx.go`: campo nuevo `Deps.ReleasesURL string`, con
  comentario explicando por qué es inyectable. `DefaultDeps()` lo rellena con
  `version.ReleasesURL`.
- `internal/cli/root.go`: `newVersionCmd` pasa a llamar
  `version.LatestVersion(d.ReleasesURL, version.Version, 2*time.Second)` en
  vez de `version.ReleasesURL`.
- `internal/cli/root_test.go`: `TestVersionAvisaSoloEnTTY` ya no muta
  `version.ReleasesURL` (ahora imposible, es `const`); construye
  `appctx.Deps{IsTTY: ..., ReleasesURL: srv.URL}` en los dos subtests. El
  test del gate TTY se conserva íntegro en su intención.
- Añadí `TestDefaultDepsUsaElEndpointRealDeVersion` en
  `internal/appctx/appctx_test.go`: cierra un hueco que mi propio cambio
  introducía (si `DefaultDeps` olvidara cablear el campo nuevo, el binario
  real dejaría de avisar de nuevas versiones sin que ningún test existente lo
  notara, porque todos los tests de `cli` fijan `ReleasesURL` a mano).
  Verificado por mutación (ver tabla abajo).

No hubo ciclo de imports: `appctx` ya importaba `internal/version` (lo usa
para `sdk.Options.UserAgent`).

## Punto 5 — remates de robustez en `install.sh`

- `grep " ${archivo}\$" checksums.txt` → `grep -F -- " ${archivo}" checksums.txt`.
  Probé en aislado (ver "Verificación de `install.sh`" abajo) que con `-F` un
  fichero de checksums con un nombre parecido pero con caracteres literales
  en vez de los puntos de la versión/extensión **no** produce un falso
  positivo, y que el fichero correcto sí se encuentra y verifica.
- Añadido el fallback `sha256sum` (preferido, típico en Linux) → `shasum -a
  256` (macOS) → mensaje de error claro y `exit 1` si no hay ninguno de los
  dos, usando un array (`verificador=(...)` / `"${verificador[@]}"`) para
  evitar el *word-splitting* frágil de interpolar un string con espacios.
- El orden se conserva: la verificación del checksum (línea 53) sigue yendo
  antes de `tar -xzf` (línea 55) e `install -m 0755` (línea 56).

## Verificación de `install.sh` (sin descargar nada)

- `bash -n install.sh` → limpio.
- **`grep -F` vs. colisión por puntos no escapados**: en un directorio de
  scratch, creé `calliope_1.4.0_darwin_arm64.tar.gz` y
  `calliopeX1X4X0_darwin_arm64.tar.gz` (mismo nombre salvo que los puntos de
  la versión se sustituyeron por letras), generé un `checksums.txt` real con
  `sha256sum`/`shasum` para ambos, y comprobé que
  `grep -F -- " ${archivo}" checksums.txt` selecciona **solo** la línea
  correcta y que la verificación pasa (`OK`). Después corrompí el archivo y
  confirmé que la verificación **falla** (`FAILED`, exit 1) — el camino de
  fallo también funciona.
- **Fallback `sha256sum`/`shasum`**: probé la rama `sha256sum` (entorno
  normal, macOS con coreutils de Homebrew) y forcé la rama `shasum`
  vaciando `PATH` a un directorio con solo `bash`/`grep`/`shasum`/`cat`
  simulando un `sha256sum` ausente — ambas ramas verifican correctamente.
  También probé la rama "ninguno disponible": imprime el mensaje y sale con
  código 1.
- No ejecuté el script contra GitHub de verdad (no hay ningún release
  publicado todavía).

## Verificación por mutación de esta ronda

Sobre una copia externa de `internal/version` (mismo procedimiento que en la
ronda anterior, restaurando `check.go.orig` tras cada mutación):

| # | Mutación | Resultado |
|---|---|---|
| M7 | Comparar `cuerpo.TagName == actual` en vez de las versiones normalizadas (revive el bug real) | FAIL — `TestElPrefijoVNoImportaAlComparar` (los dos subtests) |
| M8 | `normalizeVersion` recorta un prefijo que nunca aparece (equivale a no normalizar) | FAIL — `TestElPrefijoVNoImportaAlComparar` (los dos subtests) |

Repetí también M1 (quitar la guarda `dev`) y M4 (quitar el `Timeout` del
`http.Client`) de la ronda anterior sobre el `check.go` ya actualizado: siguen
cazadas por `TestEnCompilacionDeDesarrolloNoConsulta` y `TestRespetaElTimeout`
respectivamente.

Sobre el árbol de trabajo real, con backup/restauración inmediata:

| # | Mutación | Fichero | Resultado |
|---|---|---|---|
| M9 | `if d.IsTTY {` → `if true {` (repetida tras enrutar la URL por `Deps` en vez de la variable global) | `internal/cli/root.go` | FAIL — `TestVersionAvisaSoloEnTTY/sin_TTY_no_avisa_ni_consulta_la_red` |
| M10 | Se borra la línea `ReleasesURL: version.ReleasesURL,` de `DefaultDeps` | `internal/appctx/appctx.go` | FAIL — `TestDefaultDepsUsaElEndpointRealDeVersion` |

Las 4 mutaciones nuevas de esta ronda (M7-M10) quedaron cazadas en la primera
pasada. Cada fichero se restauró desde su copia de backup y se confirmó con
`diff` antes de continuar; el árbol de trabajo quedó exactamente como antes
de cada experimento.

## Seguí sin instalar nada

Reconfirmé el YAML completo con
`python3 -c "import yaml; yaml.safe_load(open('.goreleaser.yaml'))"` (parsea,
estructura esperada — ver el JSON completo en la sección CRITICAL 2 de este
apéndice, o reprodúzcalo con el mismo comando). Las únicas peticiones de red
de esta ronda fueron lecturas `GET` a `goreleaser.com` para verificar el
comportamiento documentado — no se instaló GoReleaser ni ningún otro paquete.
**`.goreleaser.yaml` sigue sin verificarse contra la herramienta real**
(`goreleaser check` / `goreleaser build --snapshot`); lo de arriba es
verificación contra la documentación pública, no una ejecución real.

## No tocado (deuda menor, tal como se indicó)

- El mensaje inalcanzable «No se pudo determinar la última versión.» en
  `install.sh`.
- La versión de `goreleaser-action` (`version: latest`).
