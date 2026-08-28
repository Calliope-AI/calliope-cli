# Task 7: Credenciales — interfaz, API key y almacenamiento — Informe de Implementación

**Fecha:** 2026-08-28
**Rama:** `feat/cli-v1`
**Hash de commit:** `7cbd222`

## Resumen

Implementación completa del paquete `internal/auth`: la interfaz `Store`, el tipo
`Credential` con `Kind` (`KindAPIKey`, `KindOAuth`), el almacén de fichero
(`fileStore`, permisos `0600`), el almacén de llavero del sistema con respaldo
(`keyringStore`), `DefaultStore` y `Resolve` (precedencia entorno > almacén, con
error `UNAUTHORIZED` + hint cuando no hay credencial).

Se siguió TDD: los tests del brief se escribieron primero, se confirmó que
fallaban por compilación (`undefined: Credential`, etc.), y luego se implementó
hasta que los 9 tests especificados pasaron. Se añadieron 3 tests adicionales
(no incluidos en el brief) para cubrir `keyringStore` sin tocar el llavero real,
usando el proveedor simulado de `go-keyring`.

### Archivos creados

1. **internal/auth/credential.go** — `Kind`, `Credential{Kind, Token, Org}`,
   `Header()`, `Valid()`. Copiado literalmente del brief; todos los
   identificadores ya estaban en inglés.
2. **internal/auth/store.go** — `Store`, `fileStore`, `keyringStore`,
   `NewFileStore`, `NewKeyringStore`, `DefaultStore`.
3. **internal/auth/resolve.go** — `Resolve(env, st) (Credential, string, error)`.
   Copiado literalmente; sin identificadores en español.
4. **internal/auth/credential_test.go** — 3 tests (copiados literalmente).
5. **internal/auth/store_test.go** — 4 tests (copiados literalmente).
6. **internal/auth/resolve_test.go** — 2 tests + ayudante `asCLIError`
   (copiados literalmente).
7. **internal/auth/store_keyring_test.go** — 3 tests adicionales sobre
   `keyringStore` con `keyring.MockInit()` (no forman parte del brief).

### Dependencia añadida

```
go get github.com/zalando/go-keyring@latest
```

`go.mod`/`go.sum` ganaron `github.com/zalando/go-keyring v0.2.8` y sus
indirectas (`danieljoos/wincred`, `godbus/dbus/v5`, `golang.org/x/sys`).

## Traducciones de identificadores (glosario)

El brief de esta tarea ya usaba identificadores de paquete en inglés en casi
todo el código (`Credential`, `Kind`, `Store`, `Header`, `Valid`, `Resolve`,
`fileStore`, `keyringStore`, `NewFileStore`, `NewKeyringStore`,
`DefaultStore`...). Solo hubo que traducir lo siguiente, ninguno tabulado en
`glosario.md`, así que lo anoto aquí como pide el glosario:

| Brief (español) | Usado (inglés) | Motivo |
|---|---|---|
| const `servicioKeyring` | `keyringService` | constante de paquete |
| const `usuarioKeyring` | `keyringUser` | constante de paquete |
| campo `fileStore.ruta` | `fileStore.path` | campo de tipo (nivel de paquete) |
| campo `keyringStore.respaldo` | `keyringStore.fallback` | campo de tipo (nivel de paquete) |

La variable local `crudo` dentro de `keyringStore.Load()` se dejó tal cual
(local dentro del cuerpo de una función, exenta según el glosario).

### Discrepancia de nombres de parámetro en el propio brief

El brief se contradice en dos firmas: la sección "Produces" (arriba del todo)
declara `NewFileStore(path string)`, `NewKeyringStore(fallback Store)` y
`DefaultStore(globalDir string)` — ya en inglés — pero el bloque de código del
Step 4 usa `ruta`, `respaldo` y `dirGlobal` para esos mismos parámetros.

Traté los nombres de parámetro de constructores exportados como parte de la
firma pública (nivel de paquete, no variable local dentro del cuerpo), así que
apliqué la traducción, y para el caso `dirGlobal` vs. `globalDir` — donde ni
el brief ni el glosario coinciden entre sí — usé el que aparece en la sección
"Produces" (`globalDir`) porque es la declaración de interfaz explícita para
esta tarea y ya tiene orden de palabras inglés correcto. El encargo original
que me diste también citaba `DefaultStore(dirGlobal string)`; si el criterio
correcto era ese literal, es un cambio de una línea.

Todos los mensajes de usuario, hints y comentarios se copiaron literales del
brief, en español.

## Pruebas

### Ejecución inicial (Step 2)

```
$ go test ./internal/auth/ -v

internal/auth/credential_test.go:6:7: undefined: Credential
internal/auth/credential_test.go:6:24: undefined: KindAPIKey
...
internal/auth/resolve_test.go:11:8: undefined: NewFileStore
internal/auth/resolve_test.go:23:20: undefined: Resolve
FAIL	github.com/calliope/calliope-cli/internal/auth [build failed]
```

**Resultado:** FAIL — como se esperaba.

### Ejecución tras implementar (Step 6, solo los tests del brief)

```
$ go test ./internal/auth/ -v

--- PASS: TestCabeceraDeAPIKey
--- PASS: TestCabeceraDeOAuth
--- PASS: TestCredencialSinTokenNoEsValida
--- PASS: TestElEntornoGanaAlAlmacen
--- PASS: TestSinCredencialDevuelveErrorConHint
--- PASS: TestFileStoreGuardaYRecupera
--- PASS: TestFileStoreEscribeCon0600
--- PASS: TestFileStoreSinFicheroDevuelveNil
--- PASS: TestFileStoreDelete
PASS
ok  	github.com/calliope/calliope-cli/internal/auth	0.415s
```

**Resultado:** PASS (9 tests, como predice el Step 6).

### Ejecución final con los 3 tests adicionales de `keyringStore`

```
$ go test ./internal/auth/ -v -race

--- PASS: TestCabeceraDeAPIKey
--- PASS: TestCabeceraDeOAuth
--- PASS: TestCredencialSinTokenNoEsValida
--- PASS: TestElEntornoGanaAlAlmacen
--- PASS: TestSinCredencialDevuelveErrorConHint
--- PASS: TestKeyringStoreGuardaYRecuperaConMock
--- PASS: TestKeyringStoreCaeAlRespaldoSiElLlaveroFalla
--- PASS: TestKeyringStoreDeleteBorraDelMock
--- PASS: TestFileStoreGuardaYRecupera
--- PASS: TestFileStoreEscribeCon0600
--- PASS: TestFileStoreSinFicheroDevuelveNil
--- PASS: TestFileStoreDelete
PASS
ok  	github.com/calliope/calliope-cli/internal/auth	1.362s
```

**Resultado:** PASS (12 tests).

### Verificación de todo el repo

```
$ go build ./...      → OK
$ go vet ./...         → OK, sin salida
$ gofmt -l .            → OK, sin salida
$ go test ./... -race   → OK, todos los paquetes pasan
```

## Sobre el llavero en los tests

Ningún test escribe en el Keychain real de macOS ni pide contraseña:

- `store_test.go` y `resolve_test.go` (copiados literalmente del brief) solo
  ejercitan `fileStore` vía `NewFileStore`; nunca instancian `keyringStore`.
- `store_keyring_test.go` (añadido por mí) ejercita `keyringStore` usando
  `keyring.MockInit()` / `keyring.MockInitWithError(err)` de
  `github.com/zalando/go-keyring`, que sustituye el `provider` interno del
  paquete por un almacén en memoria — no toca `security`/Keychain. Verificado
  ejecutando la suite completa sin ningún diálogo de contraseña del sistema.

Sin estos tres tests extra, `keyringStore.Save/Load/Delete` no tenían ninguna
cobertura directa (el brief no incluye tests para esa ruta), así que decidí
añadirlos dado que la instrucción explícitamente permitía cubrir `keyringStore`
con el modo mock si lo consideraba necesario. Lo marco aquí como desviación
del "cópialo literalmente", justificada por el requisito de rigor.

## Pruebas de mutación

Se hizo una copia del repo fuera del árbol de trabajo
(`/private/tmp/.../scratchpad/mutation-copy`, sin `.git`), se aplicó una
mutación a la vez sobre esa copia, se ejecutó solo el test relevante con
`-run`, se confirmó el fallo, y se restauró el fichero desde el repo real
antes de la siguiente mutación. El repo real nunca se tocó durante este
proceso (confirmado con `git status`/`git diff` antes y después).

| # | Test protegido | Mutación | Resultado |
|---|---|---|---|
| 1 | `TestCabeceraDeAPIKey` | `Header()`: `"X-API-Key"` → `"X-Wrong-Key"` | FAIL — `Header() = ("X-Wrong-Key", ...)` |
| 2 | `TestCabeceraDeOAuth` | `Header()`: quitar el prefijo `"Bearer "` | FAIL — `Header() = ("Authorization", "tok")` |
| 3 | `TestCredencialSinTokenNoEsValida` | `Valid()` → siempre `true` | FAIL — "una credencial sin token no debe ser válida" |
| 4 | `TestFileStoreGuardaYRecupera` | `fileStore.Load()` devuelve `&Credential{}` en vez de `&c` | FAIL — `Load() = &{Kind: Token: Org:}` |
| 5 | `TestFileStoreEscribeCon0600` | `os.WriteFile(..., 0o600)` → `0o644` (mutación explícitamente pedida) | FAIL — `permisos = 644, se esperaba 600` |
| 6 | `TestFileStoreSinFicheroDevuelveNil` | Quitar la rama `if os.IsNotExist(err) { return nil, nil }` | FAIL — `Load sin fichero no debe fallar: open ... no such file` |
| 7 | `TestFileStoreDelete` | `fileStore.Delete()` → no-op (`return nil`) | FAIL — "tras Delete, Load debe devolver nil" |
| 8 | `TestElEntornoGanaAlAlmacen` | `env("CALLIOPE_API_KEY")` → `env("CALLIOPE_API_KEY_X")` (rompe el env-check) | FAIL — `token = "del-almacen", la variable de entorno debe ganar` |
| 9 | `TestSinCredencialDevuelveErrorConHint` (hint) | `output.NewError(..., "")` — hint vacío | FAIL — "el error debe decir cómo autenticarse" |
| 9b | `TestSinCredencialDevuelveErrorConHint` (código) | `output.CodeUnauthorized` → `output.CodeNotFound` | FAIL — `código de salida = 4, se esperaba 3` |
| 10 | `TestKeyringStoreGuardaYRecuperaConMock` | `keyringStore.Save` escribe también en `fallback` aunque el llavero funcione | FAIL — "el respaldo debería estar vacío cuando el llavero funciona" |
| 11 | `TestKeyringStoreCaeAlRespaldoSiElLlaveroFalla` | `keyringStore.Save` propaga el error del llavero en vez de caer al respaldo | FAIL — `Save: llavero no disponible` |
| 12 | `TestKeyringStoreDeleteBorraDelMock` | `keyringStore.Delete` no llama a `keyring.Delete` | FAIL — "tras Delete, Load debe devolver nil" |

**Las 12 mutaciones hicieron fallar exactamente el test que se suponía que
protegían.** En particular, la mutación de permisos `0600` → `0644` pedida
explícitamente en el encargo fue verificada (fila 5).

## Checklist de completitud

- [x] `auth.Kind` con `KindAPIKey`, `KindOAuth`
- [x] `auth.Credential{Kind, Token, Org}` con `Header()` y `Valid()`
- [x] `auth.Store` con `Save`, `Load`, `Delete`
- [x] `auth.NewFileStore(path string) Store`
- [x] `auth.NewKeyringStore(fallback Store) Store`
- [x] `auth.DefaultStore(globalDir string) Store`
- [x] `auth.Resolve(env, st) (Credential, string, error)`
- [x] Las credenciales nunca se escriben en configuración de proyecto: solo
      `fileStore` (fichero global, `0600`) o `keyringStore` (llavero del
      sistema, con ese mismo fichero global como respaldo)
- [x] El error sin credencial es `output.CodeUnauthorized` (exit code 3) con
      `Hint` no vacío
- [x] `auth` no importa `config` (recibe `globalDir` ya resuelto)
- [x] No se implementó nada de OAuth más allá de lo que pide el tipo
      (`KindOAuth`, rama en `Header()`) — aplazado a Task 8
- [x] TDD: tests fallan primero (Step 2), luego pasan (Step 6)
- [x] Identificadores de paquete en inglés según glosario;  mensajes/comentarios
      en español literal; nombres de test en español
- [x] Ningún test toca el Keychain real
- [x] `go test ./... -race`, `go vet ./...`, `gofmt -l .` limpios
- [x] Mutación verificada test por test, incluida la de permisos `0644`

## Desviaciones respecto al brief

1. **Identificadores traducidos** no cubiertos por el glosario tabulado:
   `servicioKeyring`→`keyringService`, `usuarioKeyring`→`keyringUser`,
   campo `ruta`→`path`, campo `respaldo`→`fallback`. Anotado arriba para que
   se añadan al glosario si el controlador lo considera oportuno.
2. **Parámetro de `DefaultStore`**: usé `globalDir` (de la sección "Produces"
   del propio brief) en vez de `dirGlobal` (del bloque de código del Step 4 y
   de tu encargo). Cambio de una línea si se prefiere `dirGlobal`.
3. **3 tests adicionales** en `store_keyring_test.go` (no están en el brief)
   para cubrir `keyringStore` con `keyring.MockInit()`/`MockInitWithError`,
   sin tocar el llavero real. Añaden 409→ya incluido en el total de líneas del
   commit.

Ninguna otra desviación: estructura de ficheros, orden de los steps,
mensajes, hints y comentarios son literales del brief.

---

# Ronda de Correcciones 1/5

**Fecha:** 2026-08-28
**Hash de commit:** `5076ebc`
**Coordinador:** Aprobó las 3 decisiones de la ronda anterior tal cual
(traducciones ya en el glosario, `globalDir` correcto, los 3 tests con
`MockInit()` cubren un hueco real del brief). Confirmó `Delete` idempotente,
precedencia de `Resolve` correcta, y que ningún test toca el llavero real.
Identificó 3 hallazgos IMPORTANT y un remate.

## Correcciones aplicadas

### IMPORTANT 1: los permisos no se refuerzan sobre un fichero/directorio ya existente

**Problema:** `os.WriteFile` y `os.MkdirAll` solo aplican el modo (`0600`/
`0700`) **al crear**. Si el fichero o el directorio ya existían con permisos
laxos (restauración de dotfiles, `tar` extraído con umask `022`, un
directorio creado antes por otra parte del CLI con `0755`), `Save()` los
dejaba tal cual — el token quedaba legible por otros usuarios del sistema.
El test original (`TestFileStoreEscribeCon0600`) no lo detectaba porque
siempre partía de un `t.TempDir()` limpio, donde crear-y-escribir coincide
con reforzar.

**Solución:** en `fileStore.Save` (`internal/auth/store.go`), tras
`os.MkdirAll` se añade `os.Chmod(dir, 0o700)`, y tras `os.WriteFile` se añade
`os.Chmod(s.path, 0o600)` — incondicionalmente, sin importar si el
directorio/fichero ya existían.

**Test añadido:** `TestFileStoreRefuerzaPermisosSobreFicheroYDirectorioExistentes`
(`internal/auth/store_test.go`) — crea primero el directorio en `0755` y el
fichero en `0644` con contenido preexistente, llama a `Save`, y comprueba que
quedan en `0700`/`0600`.

**Mutación — quitar el `Chmod` del fichero:**
```
store_test.go:97: permisos del fichero = 644, se esperaba 600 tras Save sobre un fichero preexistente
--- FAIL: TestFileStoreRefuerzaPermisosSobreFicheroYDirectorioExistentes
```

**Mutación — quitar el `Chmod` del directorio:**
```
store_test.go:105: permisos del directorio = 755, se esperaba 700 tras Save sobre un directorio preexistente
--- FAIL: TestFileStoreRefuerzaPermisosSobreFicheroYDirectorioExistentes
```

**Verificado:** ambas mitades del test detectan su mutación de forma
independiente.

*Nota:* como `os.Chmod(s.path, 0o600)` corre siempre después de
`os.WriteFile`, el argumento de modo de `WriteFile` (`0o600`) queda ahora
como defensa redundante — el `Chmod` final es quien realmente garantiza el
resultado. No es un hueco de cobertura nuevo: `TestFileStoreEscribeCon0600`
(el test original) sigue verificando el estado final correcto, y la mutación
de arriba prueba que quitar el refuerzo explícito sí se detecta.

### IMPORTANT 2: el llavero y el fichero de respaldo pueden quedar inconsistentes

**Problema:** `Save` solo escribía en el respaldo si el llavero fallaba, y
nunca sincronizaba el otro sentido. Si una credencial ya vivía en el llavero
y un `Save` posterior sufría un fallo transitorio, la nueva credencial iba
solo al fichero y `Load` seguía devolviendo para siempre la vieja (quizá ya
revocada), porque `Load` prueba el llavero primero y ese seguía respondiendo.
Al revés, una credencial escrita en el fichero mientras el llavero fallaba
quedaba como residuo en claro en disco, listo para resucitar si el llavero
volvía a fallar más tarde.

**Solución — invariante "como mucho un almacén guarda la credencial en cada
momento"** en `keyringStore.Save` (`internal/auth/store.go`):
- Si `keyring.Set` tiene éxito: se borra la copia del respaldo
  (`s.fallback.Delete()`), de mejor esfuerzo (su error se ignora, no rompe el
  `Save`).
- Si `keyring.Set` falla: se borra, también de mejor esfuerzo, la posible
  entrada vieja del llavero (`keyring.Delete`) antes de caer al respaldo, así
  `Load` no sirve una entrada de llavero obsoleta en vez de acudir al
  respaldo recién escrito.

**Test añadido (dirección "éxito limpia el respaldo"):**
`TestKeyringStoreBorraResiduoDelRespaldoAlGuardarConExito` — pre-siembra el
respaldo con una credencial vieja (`vieja-y-revocada`) llamando directamente
a `respaldo.Save`, luego hace un `Save` normal a través de `keyringStore` con
`keyring.MockInit()` (llavero funcionando), y comprueba que el respaldo queda
vacío.

**Mutación — quitar `s.fallback.Delete()` del camino de éxito:**
```
store_keyring_test.go:107: el respaldo debería quedar limpio tras un Save
que sí llegó al llavero, tenía Credential{Kind: api_key, Token: ***, Org: ""}
--- FAIL: TestKeyringStoreBorraResiduoDelRespaldoAlGuardarConExito
```

**Dirección "fallo limpia el llavero" — limitación del mock, documentada
como pide el encargo:** `keyring.MockInitWithError(err)` sustituye el
`provider` interno por uno **nuevo**, vacío, que devuelve `err` en *todas*
las operaciones (`Set`, `Get` y también `Delete`). No hay forma, con la API
pública de mock de `go-keyring` (`MockInit`/`MockInitWithError`), de dejar
una entrada real en el llavero simulado y luego hacer que solo `Set` falle
en la siguiente llamada — cualquier cambio a modo error resetea el store
mock a cero. Así que no pude escribir un test que siembre el llavero, fuerce
un fallo transitorio de `Set`, y compruebe que `keyring.Delete` se invocó
sobre la entrada vieja.

Lo que sí queda cubierto: que la llamada a `keyring.Delete` en el camino de
fallo es de mejor esfuerzo y no rompe `Save` aunque también falle (el mock en
modo error hace fallar `Delete` igual que `Set`). Mutación sobre el test ya
existente `TestKeyringStoreCaeAlRespaldoSiElLlaveroFalla`:

**Mutación — dejar de ignorar el error de `keyring.Delete` en el camino de fallo:**
```
store_keyring_test.go:54: Save: llavero no disponible
--- FAIL: TestKeyringStoreCaeAlRespaldoSiElLlaveroFalla
```

Esto confirma que si alguien "arregla" el best-effort propagando el error de
`keyring.Delete`, el `Save` completo se rompe (regresión real) — que es
justo lo que el "mejor esfuerzo" evita.

### IMPORTANT 3: el error de fichero/llavero corrupto incumplía dos restricciones globales

**Problema:** un `credentials.json` con JSON inválido, o un valor corrupto en
el llavero, propagaban el `*json.SyntaxError` crudo: mensaje en inglés y sin
`hint`. No se enmascaraba como "no hay credencial" (correcto), pero el
mensaje incumplía "mensajes de usuario en español" y "todo error de cara al
usuario lleva hint cuando existe una acción de recuperación".

**Solución:** nueva función `corruptCredentialError(cause error) error` en
`internal/auth/store.go`, usada en `fileStore.Load` y `keyringStore.Load`:

```go
func corruptCredentialError(cause error) error {
	return fmt.Errorf("%w: %w", output.NewError(output.CodeUnauthorized,
		"La credencial guardada está dañada o en un formato inesperado.",
		"Vuelve a autenticarte: calliope auth login --api-key <clave>"), cause)
}
```

`output.CLIError` no tiene `Unwrap`, así que envolver con un solo `%w` habría
obligado a elegir entre "mensaje en español con hint" (un `*output.CLIError`
plano) y "conservar la causa" (un error genérico envuelto, como se hizo en
`layers.go` en la Task 4 — pero ese patrón deja `Hint` vacío al caer al
`CodeGeneric` de `WriteError`). Go 1.20+ permite varios `%w` en un mismo
`fmt.Errorf`, produciendo un árbol de errores: `errors.As(err, &cliErr)` (lo
que usan `output.ExitCodeFor` y `output.WriteError`) sigue encontrando el
`*output.CLIError` y expone `Message`/`Hint`/`Code` limpios en español; la
causa técnica original (p. ej. `*json.SyntaxError`) sigue accesible
recorriendo la cadena, para diagnóstico futuro, sin filtrarse en el mensaje
que ve el usuario.

**Tests añadidos:**
- `TestFileStoreLoadConJSONCorruptoDevuelveErrorAccionable`
  (`internal/auth/store_test.go`): escribe `{no es json` directamente en el
  fichero, comprueba código de salida 3, `Hint` no vacío, mensaje sin llaves
  crudas de JSON, y que la cadena de errores conserva "invalid character".
- `TestKeyringStoreLoadConValorCorruptoDevuelveErrorAccionable`
  (`internal/auth/store_keyring_test.go`): mismo caso vía
  `keyring.Set(keyringService, keyringUser, "esto no es json")` con
  `keyring.MockInit()`.

**Mutación — devolver el error crudo en vez de `corruptCredentialError` (fichero):**
```
store_test.go:122: código de salida = 1, se esperaba 3 (no autorizado)
store_test.go:127: se esperaba poder extraer un *output.CLIError
--- FAIL: TestFileStoreLoadConJSONCorruptoDevuelveErrorAccionable
```

**Mutación — devolver el error crudo en vez de `corruptCredentialError` (llavero):**
```
store_keyring_test.go:126: código de salida = 1, se esperaba 3 (no autorizado)
store_keyring_test.go:131: se esperaba poder extraer un *output.CLIError
--- FAIL: TestKeyringStoreLoadConValorCorruptoDevuelveErrorAccionable
```

### Remate: `Credential.String()` redacta el token

**Solución:** `internal/auth/credential.go` gana un método `String()` que
implementa `fmt.Stringer`, mostrando `Kind` y `Org` pero sustituyendo el
token por `***` (o `<vacío>` si no hay token). Como es un método con receptor
por valor, cubre tanto `Credential` como `*Credential` en cualquier
`%v`/`%s`/`Println`/`Errorf` futuro.

**Test añadido:** `TestCredentialStringRedactaElToken`
(`internal/auth/credential_test.go`) — comprueba que ni `fmt.Sprintf("%v",
c)`, ni `%s`, ni un `fmt.Errorf("...: %v", c)` contienen el token.

**Mutación — hacer que `redactado` sea el token en vez de `"***"`:**
```
credential_test.go:36: String() filtra el token: "Credential{Kind: api_key, Token: cal_live_supersecreto, Org: \"acme\"}"
credential_test.go:42: String() filtra el token con %s: "..."
credential_test.go:46: un error formateado con la credencial filtra el token: "..."
--- FAIL: TestCredentialStringRedactaElToken
```

## Resultados finales

```
$ go test ./internal/auth/ -v -race
17 tests, todos PASS (12 anteriores + 5 nuevos)

$ go vet ./...       → OK, sin salida
$ gofmt -l .          → OK, sin salida
$ go test ./... -race → OK, todos los paquetes pasan
```

### Mutaciones verificadas esta ronda

```
✓ Quitar Chmod del fichero      → TestFileStoreRefuerzaPermisos... falla (permisos = 644)
✓ Quitar Chmod del directorio   → TestFileStoreRefuerzaPermisos... falla (permisos = 755)
✓ corruptCredentialError→err (fichero) → TestFileStoreLoadConJSONCorrupto... falla (exit 1, sin CLIError)
✓ corruptCredentialError→err (llavero) → TestKeyringStoreLoadConValorCorrupto... falla (exit 1, sin CLIError)
✓ Quitar fallback.Delete() en éxito    → TestKeyringStoreBorraResiduo... falla (residuo sigue ahí)
✓ Propagar error de keyring.Delete en el camino de fallo → TestKeyringStoreCaeAlRespaldo... falla (Save rompe)
✓ redactado = c.Token en String()      → TestCredentialStringRedacta... falla (token en las 3 variantes)
```

Las 7 mutaciones (incluida la doble sobre el test de permisos) hicieron
fallar exactamente el test que debían proteger. No se re-verificaron por
mutación las 12 pruebas de la ronda anterior (el encargo solo pedía verificar
los tests nuevos); la suite completa (`go test ./... -race`) confirma que
siguen todas en verde tras los cambios.

## Commit

```
git add internal/auth
git commit -m "fix: refuerza permisos, consistencia llavero/respaldo y errores de credencial corrupta"

[feat/cli-v1 5076ebc] fix: refuerza permisos, consistencia llavero/respaldo y errores de credencial corrupta
 5 files changed, 211 insertions(+), 5 deletions(-)
```

**Hash:** `5076ebc`

## Dudas para el coordinador

1. **Limitación del mock de `go-keyring`** (detallada en IMPORTANT 2): no
   pude simular "el llavero ya tenía una entrada, y la siguiente `Set` falla
   de forma transitoria" con la API pública (`MockInit`/`MockInitWithError`),
   porque cambiar a modo error resetea el store simulado. El código sí hace
   el `keyring.Delete` de mejor esfuerzo en ese camino; lo que no pude cubrir
   con un test directo es que efectivamente borre una entrada *preexistente*
   del llavero real/simulado en ese escenario concreto. Si es importante
   cerrar ese hueco, la vía sería escribir un `Store` doble (una implementación
   propia de la interfaz `keyring.Provider`, si `go-keyring` la expone) en
   vez de usar `MockInit`/`MockInitWithError` directamente — dime si quieres
   que lo persiga.
2. El argumento de modo `0o600` en la llamada a `os.WriteFile` (dentro de
   `fileStore.Save`) queda redundante ahora que hay un `os.Chmod` explícito
   justo después. Lo dejé tal cual (no hace daño, y documenta la intención en
   el propio call site) — dilo si prefieres quitarlo.
