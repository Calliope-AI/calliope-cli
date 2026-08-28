# Task 11 — appctx: el wiring de una invocación

## Estado

DONE

## Commit

`ec1a472` — feat: appctx monta configuración, credencial, cliente y salida

Ficheros: `internal/appctx/appctx.go`, `internal/appctx/appctx_test.go`, `go.mod`, `go.sum`.

## Resumen de tests

`go test ./... -race`: **7/7 paquetes OK** (appctx 6 tests, más auth, cli, config, output,
presenter, sdk sin regresiones); `go vet ./...` y `gofmt -l .` limpios.

## Proceso TDD seguido

1. Escribí `appctx_test.go` copiando literalmente el código del brief, traduciendo solo los
   identificadores de nivel de paquete indicados: `comandoConFlags`→`commandWithFlags`,
   `depsDePrueba`→`testDeps`, `escribirConfigProyecto`→`writeProjectConfig`. Los nombres de
   test se quedaron en español. Presté atención al aviso sobre `t.TempDir()` dentro de
   closures: `home := t.TempDir()` está fuera de la función `Env`, como exige el brief.
2. `go test ./internal/appctx/ -v` → falló en compilación (`undefined: Build`, `undefined:
   Deps`, etc.), como se esperaba.
3. Implementé `appctx.go` copiando el código del brief, traduciendo `modoDeSalida`→`outputMode`
   y `timeoutDe`→`timeoutOf`. Añadí `golang.org/x/term` con `go get golang.org/x/term@latest`
   (subió también `go.mod` de `go 1.24.0` a `go 1.25.0` como efecto colateral de la
   resolución de módulos; sigue cumpliendo el mínimo de Go 1.22+ exigido).
4. Al ejecutar los tests copiados literalmente, **2 de 6 fallaron** por un motivo distinto al
   esperado por el brief (no era el `undefined: Build` del paso 2, sino una discrepancia de
   comportamiento). Investigué con depuración sistemática antes de tocar nada — ver más abajo.
5. Apliqué un único arreglo con causa raíz identificada (no un parche del síntoma) y los 6
   tests pasaron.
6. Verifiqué las 6 costuras por mutación sobre una copia del repo fuera del árbol real
   (`/private/tmp/.../scratchpad/mutation-copy`, generada con `rsync` sin `.git`), más una
   séptima mutación específica para el propio arreglo. Detalle abajo.
7. `go build ./...`, `go vet ./...`, `gofmt -l .` y `go test ./... -race` limpios sobre el
   repo real.
8. Commit.

## Bug encontrado en el código literal del brief (y su arreglo)

**Síntoma:** con el código de `appctx.go` copiado tal cual del brief,
`TestElFlagOrgGanaALaConfiguracion` y `TestLosFlagsDeterminanElModoDeSalida` fallaban: los
flags fijados en `commandWithFlags` mediante `cmd.Flags().Set(k, v)` no llegaban a `Build`.

**Causa raíz** (confirmada leyendo `command.go` de `github.com/spf13/cobra@v1.10.2` y con un
repro mínimo aislado): `RegisterGlobalFlags` declara los flags con `cmd.PersistentFlags()`
(correcto: así los heredan los subcomandos reales). Pero `cmd.Flags()` — el FlagSet *local* —
es un `*pflag.FlagSet` **distinto**, y Cobra solo copia los persistentes dentro de él en
`mergePersistentFlags()`, que se invoca desde `ParseFlags`/`Execute`/`Help`/etc. — nunca desde
`PersistentFlags()` ni desde `Flags()` en frío. El fixture de test
(`commandWithFlags`, literal del brief) construye un `*cobra.Command` suelto y llama
`cmd.Flags().Set("org", "otra")` **sin pasar nunca por `Execute()`**: como `"org"` no existe
todavía en el FlagSet local, `Set` devuelve `no such flag -org` (silenciado por el `_ =` del
propio brief) y el valor se pierde — no llega ni al FlagSet local ni al persistente. Cualquier
lectura posterior con `cmd.Flags().GetString("org")` en `Build` ve, por tanto, el valor por
defecto.

Lo comprobé con un repro de 10 líneas fuera del módulo antes de tocar código (ver política de
depuración sistemática): `cmd.Flags().Set("org", ...)` tras registrar solo en
`PersistentFlags()` devuelve error; añadiendo `cmd.Flags().AddFlagSet(cmd.PersistentFlags())`
justo después de declarar los flags, `Set` y `Get` pasan a compartir el mismo `*pflag.Flag`
(AddFlagSet no copia valores, comparte el puntero), y todo funciona sin necesidad de
`Execute()`.

**Arreglo aplicado** (única línea nueva, con comentario en español explicando el porqué):
en `RegisterGlobalFlags`, tras declarar los 5 flags persistentes, `cmd.Flags().AddFlagSet(f)`.
Verifiqué que esto no cambia el comportamiento real de Cobra para el árbol de comandos
completo (los hijos siguen heredando por el mecanismo normal de `mergePersistentFlags`,
que es independiente de esto; para el propio `root`, esto simplemente adelanta un merge que
Cobra habría hecho de todas formas en su primer `ParseFlags`/`Execute`, y `AddFlagSet` es
idempotente vía `Lookup`, así que no hay doble registro).

No toqué el test: el fixture es literal del brief y, según `progress.md`, lo reutilizarán las
Tasks 12-18 tal cual (`Build`, `BuildSinCredencial`, `Deps`, `Context.Render`, `Context.Deps`),
así que la costura correcta a arreglar era la de producción, no la del test.

## Verificación por mutación (fuera del repo, `/private/tmp/.../scratchpad/mutation-copy`)

Copié el repo completo (sin `.git`) fuera del árbol real con `rsync`, confirmé que compila
igual, y apliqué mutaciones puntuales con `sed`/un script Python corto, una por una,
restaurando desde una copia de respaldo antes de la siguiente. Resultado:

| # | Mutación | Test que la detecta | Resultado |
|---|---|---|---|
| 1 | `Org: cfg.Org()` → `Org: ""` en `BuildSinCredencial` | `TestLaOrganizacionSaleDeLaConfiguracionDeProyecto` | FAIL (capturada) |
| 2 | `config.Load(d.Cwd, d.Env, flags)` → `config.Load(d.Cwd, d.Env, nil)` | `TestElFlagOrgGanaALaConfiguracion` | FAIL (capturada) |
| 3 | `output.CodeUsage` → `output.CodeGeneric` en el error de organización ausente | `TestSinOrganizacionElErrorDiceComoElegirla` | FAIL (capturada; código de salida 1 en vez de 2) |
| 4 | `fmt.Fprintln(d.Stderr, a)` → `fmt.Fprintln(d.Stdout, a)` | `TestLosAvisosDeConfiguracionVanAStderr` | FAIL (capturada) |
| 5 | Invertir la condición del flag `--json` (`v` → `!v`) | `TestLosFlagsDeterminanElModoDeSalida` | FAIL (capturada, en dos de los cinco casos) |
| 6 | `BuildSinCredencial` llama a `auth.Resolve` y falla si no hay credencial | `TestBuildSinCredencialNoFallaSinAutenticacion` | FAIL (capturada) |
| 7 | Quitar la línea `cmd.Flags().AddFlagSet(f)` (el arreglo del bug de Cobra) | `TestElFlagOrgGanaALaConfiguracion` y `TestLosFlagsDeterminanElModoDeSalida` | FAIL en ambos (capturada) |

Las 6 costuras del brief están cubiertas, y la costura del propio arreglo (mutación 7)
también, por dos tests distintos. Tras cada mutación restauré el fichero original y confirmé
con `diff` que no quedaba ningún cambio antes de pasar a la siguiente.

## Identificadores traducidos (glosario)

Todos estaban ya en la lista que me diste; no encontré ninguno adicional en español que
necesitara añadirse al glosario.

- `modoDeSalida` → `outputMode`
- `timeoutDe` → `timeoutOf`
- `depsDePrueba` → `testDeps`
- `comandoConFlags` → `commandWithFlags`
- `escribirConfigProyecto` → `writeProjectConfig`

`BuildSinCredencial` se mantuvo tal cual (en español): no está en el glosario y
`progress.md` confirma que las Tasks 12-18 lo referencian exactamente con ese nombre.

## Dudas / puntos para revisar

1. **El bug de Cobra descrito arriba no está anotado en absoluto en el brief ni en el
   contexto que me diste.** Lo documento aquí porque cualquier task futura que use
   `commandWithFlags`/`RegisterGlobalFlags` para levantar comandos de prueba sin pasar por
   `Execute()` se beneficia del arreglo (`cmd.Flags().AddFlagSet(f)`), pero conviene que quien
   revise Task 12 sepa que esta línea existe y por qué, no vaya a parecer redundante y se
   borre en una limpieza.
2. `go get golang.org/x/term@latest` subió el `go` directive de `go.mod` de `1.24.0` a
   `1.25.0` como efecto colateral (probablemente por la resolución mínima de
   `golang.org/x/sys`). Sigue cumpliendo "Go 1.22+"; no lo revertí porque forzar una versión
   más baja habría requerido fijar una versión antigua de `x/term`/`x/sys` a mano, y el
   binario de Go instalado (1.27.0) lo soporta sin problema. Aviso por si el equipo prefiere
   fijar el `go` directive explícitamente en otro punto del roadmap.
3. No toqué `internal/cli/root.go` ni `cmd/calliope/main.go`, tal como se me indicó — siguen
   con su propia copia en línea de los 5 flags globales, pendiente de que Task 12 la sustituya
   por `appctx.RegisterGlobalFlags`.

---

# Ronda de correcciones 1/5

## Estado

DONE

## Commit

`789249c` — fix: cubre Build sin credencial, ordena go.mod y sube el suelo de Go del CI

## Resumen de lo pedido y lo hecho

**IMPORTANT 1 — faltaba el test de que `Build` exige credencial.**

Añadido `TestBuildExigeCredencialYElErrorTraeCodigoYHint` en
`internal/appctx/appctx_test.go`, junto con el helper `testDepsWithoutCredential`
(nombre nuevo, no venía en el glosario; lo traduzco con el mismo criterio que
`testDeps`/`testDepsSinCredencial` → `testDepsWithoutCredential`, y lo anoto aquí
para que se añada al glosario si hace falta). El helper monta `Deps` con un
`Store` vacío (`auth.NewFileStore` sobre un fichero que nunca se escribe) y un
`Env` que no expone `CALLIOPE_API_KEY` ni `CALLIOPE_TOKEN` — el reverso exacto
de `testDeps`, que siempre deja una credencial válida guardada y por eso nunca
ejercitaba este camino. El test configura `org=acme` en la capa de proyecto
para que el fallo sea inequívocamente el de credencial (aunque, revisando
`Build`, `auth.Resolve` se comprueba antes que el `Org == ""`, así que ni
siquiera hacía falta: cualquier organización que faltase después del fallo de
credencial nunca se llega a evaluar).

El test comprueba tres cosas sobre el mismo `err`, no solo que "no falla":
- `err != nil`.
- `output.ExitCodeFor(err) == 3` (código `CodeUnauthorized`).
- `errors.As(err, &cliErr)` tiene éxito y `cliErr.Hint != ""`.

**Verificación por mutación** (fuera del repo, en una copia nueva vía `rsync`
del mismo estilo que en la ronda anterior): reproduje exactamente la mutación
que describe el revisor —`cred, origen, _ := auth.Resolve(d.Env, d.Store)`, sin
comprobar el error— y confirmé que:
- Los 6 tests originales siguen en verde (igual que reportó el revisor).
- `TestBuildExigeCredencialYElErrorTraeCodigoYHint` pasa a FAIL
  (`se esperaba error al no haber credencial`), capturando la mutación.

Restauré el fichero mutado y confirmé con `diff` que no quedaba ningún cambio
antes de aplicar el arreglo real en el repo.

**IMPORTANT 2 — `go mod tidy` nunca se había ejecutado.**

`go mod tidy` promovió `github.com/itchyny/gojq`, `github.com/zalando/go-keyring`
y `golang.org/x/term` de `// indirect` a directas en `go.mod` (se usan en
`internal/presenter/jq.go`, `internal/auth/store.go` e
`internal/appctx/appctx.go` respectivamente), y actualizó `go.sum` en
consecuencia. Confirmé que una segunda pasada de `go mod tidy` no produce
ningún cambio adicional (idempotente).

**REMATE — versión de Go del CI.**

Cambiado `go-version: '1.22'` → `go-version: '1.25'` en
`.github/workflows/ci.yml`, sin tocar a mano el `go` directive de `go.mod`
(quedó en `go 1.25.0`, tal como lo dejó `go get golang.org/x/term@latest` en
la ronda anterior y confirma `go mod tidy` al no modificarlo).

## Verificación final de esta ronda

- `go build ./...` → OK
- `go vet ./...` → OK
- `gofmt -l .` → sin salida (limpio)
- `go mod tidy` ejecutado; segunda pasada sin diferencias en `go.mod`/`go.sum`
- `go test ./... -race -v` → todos los paquetes OK, incluidos los 7 tests de
  `internal/appctx` (los 6 previos más el nuevo)

## Identificadores traducidos añadidos a esta ronda

| Brief/contexto (español) | Uso (inglés) |
|---|---|
| `depsDePruebaSinCredencial` (nombre que le habría dado siguiendo el patrón español) | `testDepsWithoutCredential` |

## Dudas

Ninguna nueva. Las tres correcciones pedidas quedan cerradas; el punto 1 de mis
dudas de la ronda anterior (documentar el arreglo de `AddFlagSet` para que no
se borre por parecer redundante) sigue siendo válido y ya está confirmado como
aprobado por el revisor.
