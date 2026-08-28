# Informe — Task 1: Bootstrap del módulo, comando raíz y CI

## Estado
DONE_WITH_CONCERNS (ver "Desviaciones respecto al brief" — no afectan al resultado final, pero implican cambios no literales al `.gitignore`).

## Qué se implementó

Siguiendo el brief al pie de la letra:

- `go.mod` — módulo `github.com/calliope/calliope-cli`, directiva `go 1.22`.
- `go.sum` — generado por `go mod tidy` tras `go get github.com/spf13/cobra@latest`.
- `internal/version/version.go` — variables de paquete `Version`, `Commit`, `Date` con sus valores por defecto (`dev`, `none`, `unknown`).
- `internal/cli/root.go` — `NewRootCmd()` con los 5 flags globales persistentes (`--org`, `--json`, `--quiet`, `--md`, `--jq`) y el subcomando `version`.
- `internal/cli/root_test.go` — los dos tests exactos del brief.
- `cmd/calliope/main.go` — punto de entrada.
- `Makefile` — targets `build`, `test`, `lint`.
- `.github/workflows/ci.yml` — job `test` en `ubuntu-latest`, Go 1.22, `go vet` + `go test -race`.

Todo el código en inglés (identificadores), textos de usuario y comentarios en español, tal como exige el proyecto.

## Orden TDD seguido

1. Escribí `internal/cli/root_test.go` primero.
2. Ejecuté el test sin `go.mod`: falló con `go: cannot find main module` (no es el fallo "correcto" del brief, así que antes de considerar visto el fallo esperado creé el `go.mod` mínimo con `go mod init github.com/calliope/calliope-cli`, sin tocar `internal/cli/root.go`).
3. Reejecuté: **FAIL** con `undefined: NewRootCmd` en ambas líneas del test — el fallo exacto que predice el brief.
4. Implementé `internal/version/version.go`, `internal/cli/root.go`, `cmd/calliope/main.go` (código literal del brief).
5. Reejecuté: **PASS** los 2 tests.
6. Añadí `Makefile` y `.github/workflows/ci.yml`.
7. `make build && ./bin/calliope version` → `calliope dev (none, unknown)`, coincide exactamente con lo esperado.
8. Commit.

## Comandos de test ejecutados y salida literal

### Paso 2 — test debía fallar (tras `go mod init`, sin implementación)

```
$ go test ./internal/cli/ -run TestRoot -v
# github.com/calliope/calliope-cli/internal/cli [github.com/calliope/calliope-cli/internal/cli.test]
internal/cli/root_test.go:10:9: undefined: NewRootCmd
internal/cli/root_test.go:25:9: undefined: NewRootCmd
FAIL	github.com/calliope/calliope-cli/internal/cli [build failed]
FAIL
```

(Antes de crear `go.mod`, la misma orden fallaba con `go: cannot find main module, but found .git/config...` — un fallo de infraestructura, no del código; documentado por transparencia pero no es el fallo que cuenta a efectos de TDD.)

### Paso 4 — tests debían pasar

```
$ go test ./... -v
?   	github.com/calliope/calliope-cli/cmd/calliope	[no test files]
=== RUN   TestRootMuestraAyudaSinArgumentos
--- PASS: TestRootMuestraAyudaSinArgumentos (0.00s)
=== RUN   TestVersionImprimeLaVersion
--- PASS: TestVersionImprimeLaVersion (0.00s)
PASS
ok  	github.com/calliope/calliope-cli/internal/cli	0.202s
?   	github.com/calliope/calliope-cli/internal/version	[no test files]
```

2/2 tests pasan.

### `make lint` y `make test`

```
$ make lint
go vet ./...
$ make test
go test ./... -race
?   	github.com/calliope/calliope-cli/cmd/calliope	[no test files]
ok  	github.com/calliope/calliope-cli/internal/cli	1.349s
?   	github.com/calliope/calliope-cli/internal/version	[no test files]
```

Sin avisos de `go vet`, `-race` pasa.

### Paso 6 — build y ejecución

```
$ make build && ./bin/calliope version
go build -o bin/calliope ./cmd/calliope
calliope dev (none, unknown)
```

Coincide literalmente con lo esperado por el brief.

### Verificación final tras el commit (árbol limpio)

```
$ git status
On branch feat/cli-v1
nothing to commit, working tree clean

$ go test ./... -v   → 2/2 PASS
$ go vet ./...        → sin salida (limpio)
$ ./bin/calliope       → muestra ayuda con "calliope" en el uso
```

## Entorno

- `go version` → `go version go1.27.0 darwin/arm64` (con `/opt/homebrew/bin` primero en `PATH`, según indicó el contexto).
- `go.mod` declara `go 1.22` como suelo mínimo, no `1.27.0` — corregido manualmente tras `go mod init`, que por defecto escribe la versión del toolchain instalado.

## Commits

- `091a0039c54ecd75b06b695509970fed0d7ec25a` (`091a003`) — `feat: bootstrap del módulo, comando raíz y CI`
  - 9 ficheros: `.github/workflows/ci.yml`, `.gitignore` (modificado), `Makefile`, `cmd/calliope/main.go`, `go.mod`, `go.sum`, `internal/cli/root.go`, `internal/cli/root_test.go`, `internal/version/version.go`.

## Desviaciones respecto al brief

Dos desviaciones, ambas necesarias para poder completar el Paso 6 tal como está escrito (`git add go.mod go.sum cmd internal .github Makefile`), no anticipadas por el brief:

1. **`.gitignore` del repo tenía un problema real**: la entrada `calliope` (sin barra, sin ancla) es un patrón de gitignore que coincide con *cualquier* ruta llamada `calliope` en cualquier profundidad — no solo el binario compilado en la raíz, sino también el propio directorio `cmd/calliope/`. Esto hacía que `cmd/calliope/main.go` fuera invisible para `git add` / `git status` (ignorado silenciosamente, sin error). El contexto que me diste explícitamente contemplaba esta posibilidad ("No lo toques salvo que necesites añadir algo (p. ej. bin/)"), así que:
   - Cambié `calliope` → `/calliope` (ancla a la raíz del repo, preserva la intención original de ignorar un binario suelto en la raíz sin capturar `cmd/calliope/`).
   - Añadí `bin/` (el destino real de `make build`), tal como sugería el contexto.
   
   Verificado con `git check-ignore -v` antes y después: `cmd/calliope/main.go` pasó de estar ignorado a no estarlo; `bin/calliope` sigue ignorado por la nueva entrada `bin/`.

2. **Gitignore global del usuario** (`~/.gitignore_global`, fuera de este repo) tiene una entrada `version` en su línea 34, que también ignoraba `internal/version/version.go` por homonimia de directorio. No toqué ese fichero (es de fuera del repo y de tu cuenta, no del proyecto); en su lugar usé `git add -f internal/version/version.go` para forzar el staging de ese fichero concreto, ya que el brief lo exige explícitamente como parte de la interfaz (`version.Version`, `version.Commit`, `version.Date`). Confirmado en el commit final: el fichero está incluido.

Ninguna de las dos desviaciones cambia código de aplicación ni anticipa tareas futuras; son correcciones de infraestructura de control de versiones estrictamente necesarias para cumplir el propio Paso 6 del brief.

## Autorrevisión

- Código: idéntico línea por línea al brief en `root.go`, `root_test.go`, `version.go`, `main.go`, `Makefile`, `ci.yml`. No añadí nada no pedido (ni flags extra, ni subcomandos extra, ni lógica de `appctx`/`presenter`, que el propio brief indica que llegan en tareas posteriores).
- `go.mod`: verificado que la directiva es `go 1.22` (no `1.27.0`, que es lo que `go mod init` habría escrito por defecto con el toolchain instalado).
- Mensajes de usuario y ayuda: en español (`Short`, textos de flags). Identificadores en inglés (`NewRootCmd`, `newVersionCmd`, etc.).
- TDD: test visto fallar por el motivo correcto (`undefined: NewRootCmd`) antes de escribir cualquier implementación.
- No se despacharon subagentes ni revisores, conforme a instrucción explícita.
- No toqué `.superpowers/` (sigue ignorado, no es parte de los ficheros a commitear).

## Dudas

Ninguna duda bloqueante. La única nota es la desviación de `.gitignore` documentada arriba, que considero correcta pero señalo explícitamente para tu revisión, dado que modifica un fichero que el brief no mencionaba tocar.
