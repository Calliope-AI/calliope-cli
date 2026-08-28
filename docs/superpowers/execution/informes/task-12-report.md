# Task 12 — Comandos `auth` y `orgs`

## Estado

DONE

## Commit

`c10a709` — feat: comandos auth y orgs

Ficheros: `internal/commands/auth.go`, `internal/commands/auth_test.go`,
`internal/commands/orgs.go`, `internal/commands/orgs_test.go`,
`internal/cli/root.go`, `internal/cli/root_test.go`, `cmd/calliope/main.go`.

## Resumen de tests

`go test ./... -race`: **8/8 paquetes OK** (`internal/commands` con 7 tests nuevos,
sin regresiones en appctx, auth, cli, config, output, presenter, sdk); `go vet ./...`
y `gofmt -l .` limpios; `go mod tidy` sin cambios en `go.mod`/`go.sum`.

## Proceso TDD seguido

1. Leí el brief y el glosario, y contrasté con el estado real del repo indicado en el
   encargo: `main.go` ya reducido a su forma final, `root.go` con los flags en línea (Task 1)
   pendientes de sustituir, y las interfaces reales de `appctx`, `sdk`, `auth`, `config`,
   `output` y `presenter` (las leí todas antes de escribir una línea).
2. Escribí `internal/commands/auth_test.go` y `orgs_test.go` copiando literalmente el código
   del brief, traduciendo solo los dos identificadores de nivel de paquete indicados:
   `raizDePrueba`→`testRoot`, `depsConServidor`→`depsWithServer`. Los nombres de test se
   quedaron en español; las variables locales, tal cual el brief.
3. `go vet ./internal/commands/` y `go test ./internal/commands/ -v` → fallaron en compilación
   (`undefined: NewAuthCmd`, `undefined: NewOrgsCmd`), como se esperaba.
4. Implementé `auth.go` copiando el código del brief, con dos correcciones respecto al texto
   literal:
   - `clienteCon`→`clientWith` (glosario, indicado explícitamente en el encargo).
   - `me.UserID`→`me.ID`: `sdk.Me` (Task 10) no tiene campo `UserID`, tiene `ID`. El brief se
     equivoca en esto, tal como advertía el encargo; usé el campo real.
5. Implementé `orgs.go` copiando el código del brief, actualizando la llamada
   `clienteCon(ctx, cred)`→`clientWith(ctx, cred)` para que coincida con el ayudante renombrado.
6. `go test ./internal/commands/... -v` → los 7 tests pasaron a la primera, sin sorpresas de
   comportamiento (a diferencia de la Task 11, aquí el código del brief, una vez corregido el
   campo `ID`, coincidía con las interfaces reales).
7. Actualicé `internal/cli/root.go`: `NewRootCmd` pasa a aceptar `d appctx.Deps`, sustituí el
   bloque de 5 líneas de `PersistentFlags()` por `appctx.RegisterGlobalFlags(root)`, y colgué
   `commands.NewAuthCmd(d)` y `commands.NewOrgsCmd(d)` junto a `newVersionCmd()` (sin parámetro,
   como se indicó explícitamente que no tocara).
8. Actualicé `internal/cli/root_test.go`: las dos llamadas a `NewRootCmd()` pasan a
   `NewRootCmd(appctx.Deps{})`, con el import de `appctx` añadido.
9. Actualicé `cmd/calliope/main.go` con el único cambio indicado:
   `cli.NewRootCmd()`→`cli.NewRootCmd(appctx.DefaultDeps())`, añadiendo el import de `appctx`.
   No toqué nada más del fichero (el renderizado de errores queda intacto).
10. `go build ./...` y `go test ./... -race` → todo OK. `go vet ./...` y `gofmt -l .` limpios.
    `go mod tidy` no produjo diferencias en `go.mod`/`go.sum`.
11. Verifiqué las 7 costuras por mutación sobre una copia del repo fuera del árbol real
    (detalle abajo), con especial atención a la garantía de seguridad (credencial rechazada no
    se persiste).
12. Commit.

## Corrección aplicada al código literal del brief

**`sdk.Me` no tiene campo `UserID`.** El tipo real (`internal/sdk/models.go`, Task 10) es:

```go
type Me struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Username      string    `json:"username,omitempty"`
	FirstName     string    `json:"firstName,omitempty"`
	LastName      string    `json:"lastName,omitempty"`
	Organizations []OrgInfo `json:"organizations"`
}
```

El brief usaba `me.UserID` en `newAuthStatusCmd` para la clave `"userId"` del mapa de salida de
`auth status`. Usé `me.ID`, tal como advertía el encargo explícitamente. Sin esta corrección el
código ni compila (`me.UserID undefined (type *sdk.Me has no field or method UserID)`).

## Verificación por mutación (fuera del repo, `/private/tmp/.../scratchpad/mutation-copy`)

Copié el repo completo (con `.git`, vía `cp -R`) fuera del árbol real, confirmé que compila y
pasan los tests igual que en el repo real, y apliqué mutaciones puntuales con `perl -pi`, una
por una, restaurando desde una copia de respaldo (`auth.go.orig`/`orgs.go.orig`) antes de la
siguiente. Al terminar comparé con `diff` que ambos ficheros quedaban idénticos al original y
borré la copia completa.

| # | Mutación | Test que la detecta | Resultado |
|---|---|---|---|
| 1 | `NewAuthCmd` gana un `RunE` (deja de ser un grupo puro) | `TestAuthEsUnGrupoSinRunE` | FAIL (capturada) |
| 2 | `NewAuthCmd` no añade subcomandos | `TestAuthEsUnGrupoSinRunE` | FAIL (capturada, segunda aserción) |
| 3 | `login` no llama a `cliente.Me` (guarda sin validar) | `TestAuthLoginValidaLaCredencialAntesDeGuardarla` | FAIL (capturada) |
| 4 | **(crítica)** `login` guarda la credencial en `d.Store.Save` *antes* de comprobar el error de `cliente.Me`, pero sigue devolviendo el error si la validación falla | `TestAuthLoginNoGuardaUnaCredencialRechazada` | FAIL (capturada): `"nunca se debe persistir una credencial no verificada, se guardó: Credential{Kind: api_key, Token: ***, Org: \"\"}"` |
| 5 | `logout` no llama a `d.Store.Delete()` | `TestAuthLogoutBorraLaCredencial` | FAIL (capturada) |
| 6 | `auth status` añade el token a la línea de `Text` | `TestAuthStatusNoImprimeElToken` | **PASS inesperado** — ver nota abajo |
| 7 | `orgs list` construye el envelope con `Data: nil` en vez de `orgs` | `TestOrgsListDevuelveLasOrganizaciones` | FAIL (capturada) |
| 8 | `orgs use` no escribe `vals[config.KeyOrg] = args[0]` | `TestOrgsUseEscribeLaConfiguracionDeProyecto` | FAIL (capturada) |

La mutación 4 es la que more importa: reproduce exactamente el escenario que la garantía de
seguridad de esta tarea prohíbe (persistir sin verificar), moviendo el `Save` antes de
comprobar el error de `Me` en lugar de después. El test la detecta sin ambigüedad.

**Nota sobre la mutación 6 (hallazgo, no bug de producción):** el mutante que hace que el
renderer `Text` de `auth status` imprima el token **no lo detecta el test tal como está
escrito**. Investigué la causa: `depsWithServer` no fija `d.IsTTY` (queda en `false`, el cero de
`bool`), así que `appctx.outputMode` calcula `Present.Mode = ModeAuto`, y en
`presenter.Render`, el caso `ModeAuto` solo invoca `r.Text` cuando `opts.IsTTY` es cierto — en
este test, al ser `false`, el render cae directamente a JSON (el mapa `datos`, que sí excluye
el token). El test comprueba de verdad "el envelope JSON no lleva el token", pero su nombre y
su intención ("`auth status` no debe imprimir el token") sugieren que también cubre el
renderer humano (`Text`), y no lo hace: ese camino nunca se ejecuta con las `Deps` que usa el
test. En un terminal real (`IsTTY=true`) sin flags `--json`/`--quiet`/`--md`, si `Text`
tuviera esa fuga, se imprimiría igualmente. El código que entregué (copiado literalmente del
brief) **no tiene esa fuga** — ambos caminos (`datos` y `Text`) excluyen el token — así que no
hay ningún bug de producción aquí; es un hueco en la cobertura del test tal como lo especifica
el brief. No lo toqué, siguiendo la instrucción de copiar los tests literalmente; lo dejo
anotado para quien revise, por si quiere añadir `IsTTY: true` a una variante del test o un
test específico de `Text` en una ronda de correcciones.

## Identificadores traducidos (glosario)

Los cuatro que indicaba el encargo explícitamente, todos aplicados:

| Brief (español) | Uso (inglés) |
|---|---|
| `raizDePrueba` | `testRoot` |
| `depsConServidor` | `depsWithServer` |
| `authResolve` | `authResolve` (ya en inglés, sin cambio) |
| `clienteCon` | `clientWith` |

No encontré ningún otro identificador de nivel de paquete en español en el código de este
brief que no estuviera ya cubierto por el glosario o por el encargo.

## Dudas / puntos para revisar

1. Ver la nota de la mutación 6 arriba: el test `TestAuthStatusNoImprimeElToken`, tal como lo
   especifica el brief, no ejercita el renderer `Text` (solo el JSON), porque `depsWithServer`
   deja `IsTTY` en `false`. No es un bug de producción — ambos caminos del código que entregué
   excluyen el token — pero la cobertura real es menor de lo que el nombre del test sugiere.
2. `auth token` (imprime `cred.Token` en crudo, para scripts) no tiene test en el brief; lo
   implementé tal cual porque así lo pide el Step 3, pero no hay TDD detrás de esa costura
   concreta. Es un hueco de cobertura esperable, ya que el brief no incluye ningún test para
   ella.
3. Ni el brief ni el encargo piden un test de que `orgs`/`auth` pelados (sin subcomando)
   muestren ayuda vía la raíz real (`NewRootCmd`); solo hay test directo sobre
   `NewAuthCmd(appctx.Deps{})`. Lo doy por cubierto porque Cobra garantiza ese comportamiento
   para cualquier `*cobra.Command` sin `RunE`/`Run`, independientemente de si cuelga de la raíz
   real o de un `testRoot` de prueba.

---

# Ronda de correcciones 1/5

## Estado

DONE

## Commit

`7619dfd` — fix: hint en español para argumentos, cubre auth token y huecos de test

Ficheros: `internal/commands/args.go` (nuevo), `internal/commands/auth_test.go`,
`internal/commands/orgs.go`, `internal/commands/orgs_test.go`.

## Resumen de lo pedido y lo hecho

**IMPORTANT 1 — validación de argumentos de Cobra en inglés, sin hint, código de salida 1.**

Nuevo `internal/commands/args.go` con:

```go
func exactArgs(n int, uso string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == n {
			return nil
		}
		return output.NewError(output.CodeUsage,
			fmt.Sprintf("Número de argumentos incorrecto: se esperaban %d, se recibieron %d.", n, len(args)),
			"Uso: "+uso)
	}
}
```

Documentado con un comentario explícito de que las Tasks 14-17 (`ask`, `docs show`,
`concepts show`, `docs search`, `query`) deben usarlo en vez de `cobra.ExactArgs`
directamente. Aplicado a `orgs use`: `Args: exactArgs(1, "calliope orgs use <organización>")`.
El hint queda como `"Uso: calliope orgs use <organización>"` — nombra el comando concreto, no
una fórmula genérica.

Test nuevo `TestOrgsUseSinArgumentoDaErrorEnEspanolConHintYCodigoDeUso`: ejecuta `orgs use` sin
argumento y comprueba que el error es un `*output.CLIError`, que el hint no está vacío y
menciona `"calliope orgs use"`, que el mensaje no contiene `"accepts"`/`"arg(s)"` (el inglés de
Cobra), y que `output.ExitCodeFor(err) == 2`.

**IMPORTANT 2 — `auth token` sin ningún test.**

Dos tests nuevos en `auth_test.go`:
- `TestAuthTokenImprimeElTokenResuelto`: guarda una credencial en el almacén, ejecuta
  `auth token` y comprueba que stdout es exactamente el token (`strings.TrimSpace`).
- `TestAuthTokenSinCredencialDevuelveErrorConHint`: sin nada guardado ni variables de entorno
  de credencial, comprueba que el error es un `*output.CLIError` con `Hint != ""` y código de
  salida 3 (`CodeUnauthorized`).

**MINOR 3 — hueco de cobertura en `TestAuthStatusNoImprimeElToken`.**

Añadido `TestAuthStatusEnTTYNoImprimeElToken`: idéntico al test original salvo `d.IsTTY = true`
antes de construir la raíz. Con `IsTTY:true` y sin flags de salida, `appctx.outputMode` deja
`Present.Mode = ModeAuto`, y en `presenter.Render` eso sí invoca `r.Text` (la condición es
`opts.IsTTY && r.Text != nil`), a diferencia del test original donde `IsTTY` es el cero de
`bool` (`false`) y la salida cae siempre a JSON sin tocar `Text`.

**MINOR 4 — `orgs use` sin test de preservación.**

Añadido `TestOrgsUsePreservaLasClavesPrevias`: siembra `.calliope/config.json` con
`{"output":"json","timeout":"30s"}`, ejecuta `orgs use globex`, y comprueba que las tres claves
(`org`, `output`, `timeout`) siguen presentes con los valores esperados.

**MINOR 5 — test simétrico de `orgs` como grupo sin `RunE`.**

Añadido `TestOrgsEsUnGrupoSinRunE`, calcado de `TestAuthEsUnGrupoSinRunE`: comprueba
`cmd.RunE == nil && cmd.Run == nil` y `len(cmd.Commands()) > 0`.

## Verificación por mutación (fuera del repo, `/private/tmp/.../scratchpad/mutation-copy-2`)

Copié el repo completo (con `.git`) fuera del árbol real, confirmé que compila y pasan los 13
tests igual que en el repo real, y apliqué mutaciones puntuales con `perl -pi` sobre
`auth.go`/`orgs.go`/`args.go`, una por una, restaurando desde copias de respaldo
(`*.go.orig`) antes de la siguiente. Al terminar comparé con `diff` que los tres ficheros
quedaban idénticos al original y borré la copia completa.

| # | Mutación | Test que la detecta | Resultado |
|---|---|---|---|
| 1 | `orgs use` vuelve a `Args: cobra.ExactArgs(1)` directo | `TestOrgsUseSinArgumentoDaErrorEnEspanolConHintYCodigoDeUso` | FAIL (capturada): el error deja de ser `*output.CLIError` |
| 2 | `exactArgs` deja el hint en `""` | `TestOrgsUseSinArgumentoDaErrorEnEspanolConHintYCodigoDeUso` | FAIL (capturada, dos aserciones) |
| 3 | `auth token` imprime `string(cred.Kind)` en vez de `cred.Token` | `TestAuthTokenImprimeElTokenResuelto` | FAIL (capturada): `stdout = "api_key", se esperaba el token resuelto` |
| 4 | `auth token` ignora el error de `auth.Resolve` (`cred, _, _ := ...`) | `TestAuthTokenSinCredencialDevuelveErrorConHint` | FAIL (capturada): `se esperaba error sin credencial` |
| 5 | `Text` de `auth status` añade `Token: %s` con `ctx.Cred.Token` | `TestAuthStatusEnTTYNoImprimeElToken` **sí** la detecta; `TestAuthStatusNoImprimeElToken` (el original, `IsTTY:false`) **no** la detecta, confirmando exactamente el hueco reportado | Resultado esperado en ambos casos |
| 6 | `orgs use` deja de leer/mergear el `config.json` existente (siempre `vals := map[string]string{}` sin merge) | `TestOrgsUsePreservaLasClavesPrevias` **sí** la detecta; `TestOrgsUseEscribeLaConfiguracionDeProyecto` (el original, sin siembra previa) **no** la detecta — confirma el hueco reportado | Resultado esperado en ambos casos |
| 7 | `NewOrgsCmd` gana un `RunE` | `TestOrgsEsUnGrupoSinRunE` | FAIL (capturada) |

Las mutaciones 5 y 6 se ejecutaron deliberadamente contra los dos tests (el viejo y el nuevo)
para confirmar por evidencia, no solo por lectura del código, que el hueco de cobertura
reportado por el revisor era real y que el test nuevo lo cierra sin romper el viejo (que sigue
siendo válido, solo insuficiente por sí solo).

## Verificación final de esta ronda

- `go build ./...` → OK
- `go vet ./...` → OK
- `gofmt -l .` → sin salida (limpio)
- `go mod tidy` → sin diferencias en `go.mod`/`go.sum`
- `go test ./... -race -v` → todos los paquetes OK, 13 tests en `internal/commands`
  (los 7 originales de la Task 12 más los 6 nuevos de esta ronda)

## Dudas

Ninguna. Las cinco correcciones pedidas quedan cerradas y verificadas por mutación,
incluyendo la confirmación explícita de que los dos huecos autoreportados en la Task 12
(`auth token` sin test, `TestAuthStatusNoImprimeElToken` sin ejercitar `Text`) eran reales:
reproduje el mutante correspondiente contra el test viejo antes de escribir el nuevo, y en
ambos casos el viejo lo dejó pasar.

---

# Ronda de correcciones 2/5

## Estado

DONE, con un hallazgo importante para decidir en la siguiente ronda (ver abajo).

## Commit

`cf3b018` — fix: deriva el hint de uso del propio comando, cubre exceso de argumentos y no escapa HTML en JSON

Ficheros: `internal/commands/args.go`, `internal/commands/orgs.go`, `internal/commands/orgs_test.go`,
`internal/presenter/presenter.go`, `internal/presenter/presenter_test.go`.

## Resumen de lo pedido y lo hecho

**1 — `exactArgs` duplicaba la forma de uso.**

Cambié la firma de `exactArgs(n int, uso string)` a `exactArgs(n int)`. El hint se deriva ahora
de `cmd.UseLine()` (compone padres + `Use`) dentro de un nuevo ayudante `usageLine(cmd)`.

Antes de fijarlo así, **probé con el binario real** tal como se pidió explícitamente, y
confirmé el aviso del propio encargo: `cmd.UseLine()` añade `" [flags]"` en cuanto el comando
tiene algún flag disponible, y aquí siempre lo tiene (los 5 flags globales son persistentes de
la raíz y los hereda cualquier subcomando). Con `cmd.UseLine()` sin tocar, el hint salía como
`"Uso: calliope orgs use <organización> [flags]"` — ruido que no aporta nada a un error sobre
número de argumentos.

No lo dejé así (siguiendo la instrucción explícita de no forzarlo si empeora el mensaje), pero
tampoco descarté la idea de derivar del `Use` del comando: añadí `usageLine(cmd)`, que recorta
el sufijo `" [flags]"` con `strings.TrimSuffix`. Verificado de nuevo contra el binario real: el
hint vuelve a ser exactamente `"Uso: calliope orgs use <organización>"`, igual que con el
literal hardcodeado de antes, pero ahora derivado del `Use` real del comando — si alguien
cambia el placeholder en `Use` sin tocar nada más, el hint lo sigue automáticamente.

Fortalecí las aserciones de hint en los dos tests de `orgs use` (el de cero argumentos y el
nuevo de exceso) de `strings.Contains` a comparación exacta contra
`"Uso: calliope orgs use <organización>"`, para que el propio recorte de `"[flags]"` quede
protegido por test y no solo por lectura del código.

**2 — sin test de argumentos de más.**

Añadido `TestOrgsUseConArgumentosDeMasDaErrorEnEspanolConHintYCodigoDeUso`: ejecuta
`orgs use acme globex` (dos argumentos donde se espera uno) y comprueba las mismas cuatro
propiedades que el test de cero argumentos (CLIError, hint exacto, mensaje sin inglés de
Cobra, exit 2).

**3 — escapado HTML en `presenter.writeJSON`.**

Añadida la línea `enc.SetEscapeHTML(false)` en `internal/presenter/presenter.go` (única línea
tocada de ese fichero, más su test, tal como se autorizó). Test nuevo
`TestRenderJSONNoEscapaHTML`: construye un `Result` con un valor `"calliope orgs use
<organización>"` en `Data`, renderiza en `ModeJSON`, y comprueba que `"<organización>"` sale
literal y que no aparece `<`/`>` en la salida.

## Verificación por mutación (fuera del repo, `/private/tmp/.../scratchpad/mutation-copy-3`)

Copié el repo completo (con `.git`) fuera del árbol real, confirmé que compila y pasan los
tests igual que en el repo real, apliqué mutaciones puntuales con `perl -pi`, restaurando desde
copias de respaldo antes de la siguiente, y confirmé con `diff` que los ficheros quedaban
idénticos al original al terminar.

| # | Mutación | Test que la detecta | Resultado |
|---|---|---|---|
| 1 | `usageLine` recorta un sufijo que nunca aparece (deja de quitar `" [flags]"`) | `TestOrgsUseSinArgumentoDaErrorEnEspanolConHintYCodigoDeUso` y `TestOrgsUseConArgumentosDeMasDaErrorEnEspanolConHintYCodigoDeUso` | FAIL en ambos: `hint = "Uso: calliope orgs use <organización> [flags]", se esperaba "Uso: calliope orgs use <organización>"` |
| 2 | `exactArgs` pasa de "exactamente n" a "como mínimo n" (`len(args) == n` → `len(args) >= n`), dejando pasar el exceso | `TestOrgsUseConArgumentosDeMasDaErrorEnEspanolConHintYCodigoDeUso` | FAIL (capturada): `se esperaba error con argumentos de más` |
| 3 | Se quita `enc.SetEscapeHTML(false)` de `writeJSON` | `TestRenderJSONNoEscapaHTML` | FAIL (capturada): el JSON vuelve a llevar `<organización>` |

Nota sobre la mutación 1: mi primer intento (recortar directamente `" [flags]"` de vuelta a
`cmd.UseLine()` crudo) hacía que `strings` quedara sin usar y no compilara — un mutante que
"se detecta" solo porque rompe la build, no porque el test ejerza la aserción. Lo repetí con un
sufijo inocuo (`" [banderas-que-no-existen]"`) para que el mutante compilara y la detección
viniera de verdad de la comparación de hint, no de un error de compilación accidental. Mismo
cuidado con la mutación 2: mi primer intento (`len(args) <= n`) no alteraba el caso de exceso
(2 args, n=1: `2<=1` sigue siendo falso), así que no mutaba nada observable; lo corregí a
`len(args) >= n`, que sí deja pasar el exceso y es la mutación semánticamente correcta para "ya
no rechaza argumentos de más".

## Comprobación en campo (binario real, no solo tests)

```
$ calliope orgs use
Número de argumentos incorrecto: se esperaban 1, se recibieron 0.
Uso: calliope orgs use <organización>
exit=2

$ calliope orgs use a b
Número de argumentos incorrecto: se esperaban 1, se recibieron 2.
Uso: calliope orgs use <organización>
exit=2

$ calliope orgs use --json
{"ok":false,"error":{"code":"USAGE","message":"Número de argumentos incorrecto: se esperaban 1, se recibieron 0.","hint":"Uso: calliope orgs use <organización>"}}
exit=2
```

Mensaje en español, hint concreto e idéntico en ambos casos (cero y exceso de argumentos), exit
2 — confirmado en los tres, tal como se pidió.

## Hallazgo importante: el fix de `presenter.go` no cubre el escenario `--json` que motivó el punto 3

**El tercer bloque de arriba es la comprobación de campo exacta del escenario que motivó el
punto 3 del encargo, y el hint sigue saliendo escapado (`<organización>`) pese al fix
en `presenter.go`.** Lo investigué antes de dar la ronda por cerrada:

Un error de validación de `Args` (como el de `orgs use` sin argumento) nunca pasa por
`presenter.Render`. `root.Execute()` lo devuelve directamente como `error` de Go, y quien lo
serializa es `cmd/calliope/main.go`:

```go
func main() {
	if err := cli.NewRootCmd(appctx.DefaultDeps()).Execute(); err != nil {
		isJSON := slices.Contains(os.Args, "--json")
		output.WriteError(os.Stderr, err, isJSON)
		...
	}
}
```

Y `output.WriteError` (`internal/output/errors.go`) serializa con `json.Marshal(envelope)` a
secas — un `json.Marshal` normal, no un `json.Encoder` con `SetEscapeHTML(false)`, así que
sigue escapando `<`/`>` a `<`/`>` sin que mi cambio en `presenter.go` lo toque para
nada: son dos serializadores JSON completamente distintos en el mismo binario.

**Verifiqué que el fix de `presenter.go` sí es correcto para lo que sí cubre**: construí un
test ad-hoc (no commiteado) que ejecuta `orgs list --json` con éxito -ese sí pasa por
`ctx.Render` → `presenter.Render` → `writeJSON`- y el breadcrumb
`"cmd": "calliope orgs use <nombre>"` sale literal, sin escapar, confirmando que el arreglo
funciona en la ruta de éxito (datos, breadcrumbs de un `Envelope` correcto).

**No toqué `internal/output/errors.go`.** Por dos motivos: (1) el encargo de esta ronda
autorizaba explícitamente una sola línea de `presenter.go` y nada más; (2) el encargo original
de la Task 12 ya advertía que "el renderizado de errores es de otra tarea y está revisado" y
pedía no tocar nada de eso. Ampliar el alcance por mi cuenta habría incumplido ambas
instrucciones a la vez.

Lo dejo señalado explícitamente para que se decida en la siguiente ronda: si se quiere que
`--json` con un error también deje `<`/`>` literales, el cambio equivalente
(`json.Marshal(envelope)` → un `json.Encoder` con `SetEscapeHTML(false)`) tendría que ir en
`internal/output/errors.go`, fuera de mi tarea tal como está delimitada hoy.

## Verificación final de esta ronda

- `go build ./...` → OK
- `go vet ./...` → OK
- `gofmt -l .` → sin salida (limpio)
- `go mod tidy` → sin diferencias en `go.mod`/`go.sum`
- `go test ./... -race -v` → todos los paquetes OK, 15 tests en `internal/commands`
  (13 de rondas anteriores más el de exceso de argumentos; el de cero argumentos se mantiene)
  y 10 en `internal/presenter` (9 más `TestRenderJSONNoEscapaHTML`)
- Comprobación en campo con el binario real (ver arriba)

## Dudas

Una: el hallazgo de arriba sobre `output.WriteError`/`internal/output/errors.go`. No es una
duda sobre qué hacer en mi propio código -ahí no hay ambigüedad, y no lo toqué- sino sobre si
el punto 3 se da por cerrado tal como quedó (fix correcto pero de alcance más estrecho de lo
que el ejemplo del encargo sugería) o si se abre explícitamente el alcance a
`internal/output/errors.go` en una ronda futura.

---

# Ronda de correcciones 3/5 (última)

## Estado

DONE

## Commit

`448495a` — fix: WriteError no escapa HTML en modo --json, cierra el contrato con presenter

Ficheros: `internal/output/errors.go`, `internal/output/errors_test.go` (los dos únicos
ficheros autorizados para esta ronda, y solo la línea de escapado en el primero).

## Resumen de lo pedido y lo hecho

Cierra la asimetría que quedé señalando al final de la ronda 2: `presenter.writeJSON` ya no
escapaba `<`/`>`, pero `output.WriteError` seguía haciéndolo, así que el mismo contrato de
envelope se comportaba distinto en JSON según el resultado fuera éxito o error.

**1 — `json.Marshal` → `json.Encoder` con `SetEscapeHTML(false)`.**

En `WriteError`, sustituí:

```go
envelope := cliErr.Envelope()
b, err := json.Marshal(envelope)
if err != nil {
	return err
}
_, err = fmt.Fprintln(w, string(b))
return err
```

por:

```go
enc := json.NewEncoder(w)
enc.SetEscapeHTML(false)
return enc.Encode(cliErr.Envelope())
```

Sobre el detalle del salto de línea que se pedía decidir explícitamente: `Encoder.Encode` ya
añade su propio `\n` final (lo mismo que antes hacía el `fmt.Fprintln` envolviendo el resultado
de `Marshal`, que no lo lleva), así que retiré el `Fprintln` en vez de apilarlo encima -apilarlo
habría dejado dos saltos de línea, que sí habría sido una diferencia observable-. El resultado
es exactamente un salto de línea final, igual que antes, y consistente con lo que ya hace
`presenter.writeJSON` (que también usa `Encoder.Encode`, nunca `Marshal` + `Fprintln`). Los 10
tests preexistentes de `errors_test.go` (formas de envelope, mapeo a `CodeGeneric`, `omitempty`
del hint, modo texto) siguen en verde sin tocarlos ni un carácter.

**2 — test simétrico del presenter.**

`TestWriteErrorModoJSONNoEscapaHTML`: construye un `*CLIError` con
`Hint: "Uso: calliope orgs use <organización>"`, llama `WriteError(buf, err, true)`, y
comprueba que `"<organización>"` sale literal y que no aparece la forma escapada
(`<`/`>`) en la salida. Cometí el mismo desliz que en la ronda anterior al escribirlo
la primera vez -comprobar `!strings.Contains(output, "<")` en vez de
`!strings.Contains(output, `<`)`, que falla trivialmente porque el placeholder correcto
también lleva `<`- y lo corregí antes de darlo por bueno, igual que hice con el test del
presenter en la ronda 2.

**3 — comprobación en campo.**

```
$ calliope orgs use --json
{"ok":false,"error":{"code":"USAGE","message":"Número de argumentos incorrecto: se esperaban 1, se recibieron 0.","hint":"Uso: calliope orgs use <organización>"}}
exit=2

$ calliope orgs use a b --json
{"ok":false,"error":{"code":"USAGE","message":"Número de argumentos incorrecto: se esperaban 1, se recibieron 2.","hint":"Uso: calliope orgs use <organización>"}}
exit=2
```

`<organización>` sale sin escapar en ambos casos (antes salía como `<organización>`), exit 2
en los dos. Es exactamente el escenario que quedó documentado como hallazgo al final de la
ronda 2. Los tests de renderizado de errores de tareas anteriores (los 10 originales de
`errors_test.go`) siguen en verde, confirmado con `go test ./internal/output/... -v -race`.

## Verificación por mutación (fuera del repo, `/private/tmp/.../scratchpad/mutation-copy-4`)

Copié el repo completo (con `.git`) fuera del árbol real, confirmé que compila y pasan los
tests igual que en el real, apliqué la mutación con un script Python, y restauré desde una
copia de respaldo (`errors.go.orig`), confirmando con `diff` que quedaba idéntico al terminar.

| # | Mutación | Test que la detecta | Resultado |
|---|---|---|---|
| 1 | `WriteError` vuelve a `json.Marshal` + `fmt.Fprintln` (revierte el cambio completo de esta ronda) | `TestWriteErrorModoJSONNoEscapaHTML` | FAIL (capturada): el hint vuelve a salir como `<organización>` |

## Verificación final de esta ronda

- `go build ./...` → OK
- `go vet ./...` → OK
- `gofmt -l .` → sin salida (limpio)
- `go mod tidy` → sin diferencias en `go.mod`/`go.sum`
- `go test ./... -race -v` → todos los paquetes OK; `internal/output` con 18 tests (los 17
  previos más `TestWriteErrorModoJSONNoEscapaHTML`)
- Comprobación en campo con el binario real (ver arriba)

## Dudas

Ninguna. Con esto se cierran las cinco rondas: los tres puntos de la ronda 1 (hint en español
con `exactArgs`, cobertura de `auth token`, y los dos huecos de test autoreportados en la
Task 12 original), los tres de la ronda 2 (`exactArgs` sin duplicar el `Use`, caso de exceso de
argumentos, no-escapado en `presenter`), y el cierre de la ronda 3 (no-escapado también en
`output.WriteError`). El contrato JSON de éxito y de error se comporta ahora igual respecto al
escapado de HTML.
