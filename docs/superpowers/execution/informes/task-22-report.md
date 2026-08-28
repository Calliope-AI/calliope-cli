# Informe Task 22: smoke de extremo a extremo y README

## Estado

DONE

## Commit

`77e7e18` — "feat: smoke de extremo a extremo y README"
(rama `feat/cli-v1`, sobre `722e1ff`)

Archivos:
- `test/e2e/smoke_test.go` (nuevo)
- `README.md` (nuevo)
- `.github/workflows/ci.yml` (modificado: solo un comentario explicando por
  qué `go test ./...` no toca `test/e2e`; el workflow ya usaba Go 1.25 y no
  necesitaba cambio funcional)

## Qué se hizo

### `test/e2e/smoke_test.go`

Copiado del brief con la etiqueta `//go:build e2e` y opt-in por
`CALLIOPE_E2E=1` + `CALLIOPE_API_KEY` + `CALLIOPE_ORG`, con los identificadores
renombrados a inglés (`runCalliope`, `requireE2E`) y un helper nuevo
`checksOf` que decodifica `doctor --json`. Dos tests: `TestCadenaCompletaDeAgente`
(orgs list → schema → ask con breadcrumbs y fuentes → concepts list --jq) y
`TestDoctorPasaEnUnEntornoConfigurado`.

Correcciones aplicadas sobre el contenido del brief (verificadas contra el
código fuente real, no solo copiadas de memoria):

- El tipo `Check` de `doctor` (`internal/commands/doctor.go`) usa campos en
  inglés `Name`/`Status`/`Detail` (`json:"name"`, `json:"status"`,
  `json:"detail"`), no `nombre`/`estado`. Verificado además contra el
  `doctor --json` real:
  `{"data":[{"name":"versión","status":"ok","detail":"..."}]}`. El helper
  `checksOf` del smoke usa esos nombres.
- `sdk.AskResponse` (`internal/sdk/models.go`) es lo que viaja entero en
  `data` de `ask --json`; tiene `Text` y `Sources` (no hay envoltorio
  adicional). El struct de decodificación del smoke ahora declara también
  `Sources` junto a `Text`, y comprueba que `breadcrumbs` sale a nivel del
  envelope (no dentro de `data`) — así es como lo serializa
  `output.Envelope`.
- `sdk.Me` no tiene `userId`; tiene `ID` (`json:"id"`). No hace falta en el
  smoke (no decodifica `Me` directamente), pero es relevante para el README:
  la clave `"userId"` que aparece en `auth status --json` es una clave de
  salida elegida a mano en `internal/commands/auth.go`
  (`"userId": me.ID`), no el nombre del campo Go.

Verificación de compilación y exclusión (todas con Go 1.25/1.27 en
`/opt/homebrew/bin/go`, versión efectiva del módulo: `go 1.25.0`):

- `go build ./...`, `go vet ./...`, `go test ./... -race`: **no** compilan ni
  tocan `test/e2e` — `go list ./...` no lo lista y no aparece en la salida de
  `go test ./...`. Confirmado antes y después de crear el fichero.
- `go vet -tags=e2e ./test/e2e/`: compila limpio (exit 0).
- `go test -tags=e2e ./test/e2e/ -v` sin las variables de entorno: los dos
  tests hacen SKIP con el mensaje esperado, exit 0.
- `go build -tags=e2e ./test/e2e/` da "no non-test Go files" — es el
  comportamiento normal de `go build` sobre un paquete que solo tiene
  `_test.go` (no hay nada que archivar), no un fallo de compilación: `go vet`
  y `go test`, que sí compilan el paquete de test, lo confirman limpio.

**El smoke nunca se ha ejecutado contra un backend real.** No había
credenciales de Calliope disponibles en este entorno. Solo se verificó que
compila con la etiqueta `e2e` y que se salta correctamente sin las variables
de entorno — exactamente lo que pedía la corrección 2 del encargo. El día que
haya credenciales, el primer paso es correr:
`CALLIOPE_E2E=1 CALLIOPE_API_KEY=... CALLIOPE_ORG=... go test -tags=e2e ./test/e2e/ -v`

### `README.md`

Escrito a partir del contenido del brief, con estas diferencias respecto al
texto original (todas por las 5 correcciones del encargo, cada una verificada
contra el código o el binario compilado):

1. **Go 1.25**, no 1.22, en la sección Desarrollo (`go.mod` dice
   `go 1.25.0`; `ci.yml` y `release.yml` ya usaban `go-version: '1.25'`).
2. Añadida una frase en "Primeros pasos" dejando explícito que hoy solo hay
   autenticación por API key, con referencia a la sección de estado.
3. Sección "Comandos" nueva: **22 comandos** en total. Contados a mano sobre
   el árbol real construido por `cli.NewRootCmd`
   (`auth login/logout/status/token`, `orgs list/use`,
   `config get/list/path/set`, `docs list/search/show`,
   `concepts list/show`, `rules list`, más `ask`, `query`, `schema`,
   `doctor`, `skill`, `version` = 22) y confirmados ejecutando
   `./bin/calliope <grupo> --help` y `./bin/calliope <comando> --help` para
   cada uno — todos existen con los flags que documenta `SKILL.md`.
4. Sección "Salida": añadido el párrafo sobre `--csv` de `query`
   incompatible con `--json`/`--quiet`/`--md`/`--jq`. Verificado
   ejecutando el binario:
   `CALLIOPE_API_KEY=fake CALLIOPE_ORG=test-org ./bin/calliope query "SELECT 1" --csv --json`
   devuelve `{"ok":false,"error":{"code":"USAGE",...}}` con exit code 2, sin
   tocar la red (el chequeo va antes de `ctx.Client.Query`, en
   `internal/commands/data.go`).
5. Sección "Configuración": añadida la explicación de por qué la capa de
   proyecto solo puede fijar `org` y `output` (`internal/config/trust.go`,
   `projectAllowed`), citando el motivo real del comentario en el código: un
   `.calliope/config.json` viaja dentro de cualquier repositorio clonado, así
   que si pudiera fijar `base_url` un repositorio hostil redirigiría el
   token del usuario.
6. Códigos de salida `0`–`5`: verificados contra `internal/output/errors.go`
   (`Code.ExitCode()`) y calcan los de `skills/calliope/SKILL.md`.
7. Nueva sección **"Estado del proyecto"** con tres puntos, honestos y
   verificados en el propio código, no supuestos:
   - Login OAuth no implementado: solo `auth login --api-key`;
     `internal/auth/resolve.go` acepta un `CALLIOPE_TOKEN` externo con
     `Kind: KindOAuth`, pero no hay ningún comando que lleve a un usuario a
     través de un flujo de login interactivo.
   - `goreleaser check`/`goreleaser build --snapshot` sin verificar:
     `goreleaser` no está instalado en este entorno; confirma lo que ya
     dejó anotado el informe de la Task 21
     (`task-21-report.md`, líneas 63-75, 204, 288, 407).
   - El smoke de extremo a extremo nunca se ha ejecutado contra un backend
     real (ver arriba).

Cada comando y flag que menciona el README fue comprobado uno por uno contra
el binario compilado (`go build -o bin/calliope ./cmd/calliope`), no copiado
sin más del brief. No se encontró ninguna afirmación falsa que corregir más
allá de las 5 ya señaladas en el encargo.

## Verificación final

```
go test ./... -race   → PASS, sin compilar test/e2e
go vet ./...           → limpio
gofmt -l .              → sin salida
go mod tidy             → go.mod y go.sum sin cambios
```

## Resumen de tests

`go test ./... -race` pasa en los 8 paquetes con tests (appctx, auth, cli,
commands, config, output, presenter, sdk, version); `test/e2e` queda fuera
por el build tag y solo se verificó que compila (`go vet -tags=e2e`) y que
sus dos tests se saltan sin las variables de entorno — nunca se ejecutó
contra un backend real por falta de credenciales.

## Dudas / puntos abiertos

- Ninguna duda de diseño. El único pendiente genuino es el que ya declara el
  README: ejecutar el smoke con credenciales reales en cuanto existan, y
  correr `goreleaser check` antes de confiar en el release automático.
- No modifiqué `.github/workflows/ci.yml` más allá de un comentario
  explicativo: ya usaba Go 1.25 y ya excluía `test/e2e` de forma natural (sin
  `-tags=e2e`, el build tag basta). No había nada que arreglar ahí.

## Ronda de correcciones 1/5

El revisor no encontró ninguna afirmación falsa en el README -verificó cada
comando, flag, código de salida y forma de envelope contra el binario- y fue
más allá de la verificación que yo pude hacer: montó un backend simulado
local y ejecutó el smoke de verdad, que pasó. Confirma en la práctica que los
campos que decodifica el smoke coinciden con lo que produce el binario,
incluido `check{Name,Status,Detail}` en inglés y la ubicación de
`breadcrumbs` a nivel de envelope. Las tres correcciones al brief de la
primera pasada eran ciertas.

Quedaban un hallazgo Important y un remate, ambos de documentación; no se
tocó ningún fichero Go.

### IMPORTANT — «Estado del proyecto» omitía el pendiente que más afecta a un usuario

`README.md`, sección "Estado del proyecto": se sustituyó el bullet de smoke
por tres bullets, insertando uno nuevo entre "Login OAuth" y "GoReleaser".

El nuevo bullet nombra los tres grupos de tipos que nunca se han confirmado
contra el backend real -`Me` (usa `auth status`), `Organization` (usa `orgs
list`) y `SchemaResponse`/`SchemaTable`/`SchemaColumn` (usa `schema`), todos
en `internal/sdk/models.go`-, de dónde salen (los tipos TypeScript de
`calliope-data-ui`, no una respuesta real ni el contrato verificado en
`calliope-data-mcp`, según el comentario que ya llevaba `models.go` desde la
Task 10), cuál es el síntoma si algún nombre de campo no coincide
(`encoding/json` deja el campo en blanco sin error; el comando sigue
devolviendo `"ok": true` con valores vacíos, no un fallo visible) y cómo
confirmarlos: ejecutar el smoke con una API key real.

Sobre ese último punto fui más preciso que la instrucción literal, porque no
es enteramente cierto que "el smoke" cubra los tres tipos por igual:
`TestCadenaCompletaDeAgente` sí ejercita `orgs list` y `schema` contra un
backend real; `TestDoctorPasaEnUnEntornoConfigurado` ejercita `Me.Email` de
paso (vía `checkConnectivity`, que llama a `cliente.Me()`), pero ningún test
del smoke actual llama a `auth status`, que es el único comando que expone
`Me.ID` (como `"userId"` en su salida). El bullet lo dice explícitamente y
añade que confirmar `Me.ID` hace falta correr `calliope auth status --json`
a mano, además del smoke. No toqué `smoke_test.go` para añadir esa
llamada -el encargo decía explícitamente "no toques nada más" y limitaba el
alcance a documentación.

### REMATE — pendiente de mantenedor sobre `TAP_GITHUB_TOKEN`

Añadido a la misma frase que ya mencionaba `goreleaser check`/`build
--snapshot` sin verificar: antes del primer release hay que crear el secreto
`TAP_GITHUB_TOKEN` en el repositorio, o la publicación en el tap de Homebrew
y el bucket de Scoop falla aunque el resto del release funcione. Cité el
motivo real, ya documentado en los comentarios de `.goreleaser.yaml`: el
`GITHUB_TOKEN` por defecto del workflow no tiene permiso de escritura sobre
`calliope/homebrew-tap` ni `calliope/scoop-bucket` (confirmado también por el
informe de la Task 21).

### Verificación

```
go test ./... -race   → PASS, sin compilar test/e2e (idéntico a la ronda anterior)
go vet ./...           → limpio
gofmt -l .              → sin salida
go mod tidy             → go.mod y go.sum sin cambios
```

Solo se tocó `README.md` y este informe; no hubo cambios en código Go, así
que la verificación es una repetición exacta de la de la primera pasada, no
una comprobación nueva.

### Commit

`207e887` — "docs: aclara qué tipos del SDK no se han confirmado contra el
backend real y añade el pendiente de TAP_GITHUB_TOKEN" (sobre `77e7e18`,
rama `feat/cli-v1`)

### Dudas

Ninguna. Los dos hallazgos eran correcciones de documentación puntuales, sin
ambigüedad sobre qué cambiar ni dónde.
