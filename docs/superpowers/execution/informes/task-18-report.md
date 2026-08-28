# Informe — Task 18: Comando `doctor`

## Estado

DONE

## Commit

`b20ab45` — feat: comando doctor
(3 ficheros: `internal/commands/doctor.go`, `internal/commands/doctor_test.go`, `internal/cli/root.go`)

## Resumen de tests

`go test ./... -race` → PASS en los 9 paquetes; 4/4 tests propios de `doctor_test.go` en verde (incluida la variante con subtests del token); `go vet ./...` limpio; `gofmt -l .` sin salida; `go mod tidy` sin cambios. 6 mutaciones dirigidas verificadas sobre una copia completa del repo fuera del árbol de trabajo: las 6 hicieron fallar al menos un test.

## Qué se hizo

- **Step 1-2 (TDD)**: escribí `internal/commands/doctor_test.go` (identificadores traducidos, ver más abajo) y confirmé el fallo esperado: `go test ./internal/commands/ -run Doctor -v` → `undefined: NewDoctorCmd` (4 apariciones).
- **Step 3**: `internal/commands/doctor.go` — copiado literalmente del brief salvo las tres correcciones del controlador (abajo). Los mensajes, el `Short`, los nombres de comprobación (`"versión"`, `"backend"`, `"credencial"`, `"organización"`, `"conectividad"`) y los valores de estado (`ok`/`aviso`/`error`) son los del brief, sin cambios.
- **Step 4**: `internal/cli/root.go` — añadido `commands.NewDoctorCmd(d)` a `root.AddCommand(...)`.
- Verificado contra el binario real (`go run ./cmd/calliope doctor`) con `CALLIOPE_BASE_URL` apuntando a un puerto cerrado, `HOME` a un directorio vacío y sin credencial: exit code 0, envelope `"ok": true`, `"summary": "2 de 4 comprobaciones fallan"`, cada comprobación rota reporta su propio `status: "error"` con su hint — el comando informa, nunca falla, exactamente cuando la autenticación está rota.

## Las tres correcciones aplicadas

1. **Identificadores en inglés.** `Chequeo{Nombre, Estado, Detalle}` (tags `nombre`/`estado`/`detalle`) → `Check{Name, Status, Detail}` (tags `name`/`status`/`detail`), tal como fija el glosario en su tabla de "Alcance (afinado tras la Task 4)". Ayudantes: `chequeosDe`→`checksOf`, `resumenDe`→`summaryOf`, `simbolo`→`symbol`, `chequearConectividad`→`checkConnectivity`. Los tests del brief, que buscaban las claves JSON en español, se reescribieron para leer `name`/`status`. Las cadenas de valor (`"credencial"`, `"organización"`, `"conectividad"`, `"ok"`, `"error"`, todos los mensajes) se dejaron literales, en español, tal como pide el ruling del glosario: solo cambian los nombres, no los valores.
2. **Constantes de configuración.** `ctx.Cfg.Get("base_url")`/`ctx.Cfg.Get("org")` → `ctx.Cfg.Get(config.KeyBaseURL)`/`ctx.Cfg.Get(config.KeyOrg)`. Añadido el import de `internal/config`.
3. **Solo `appctx.BuildSinCredencial`.** El brief ya lo usaba así; no se tocó. Verificado explícitamente por mutación (ver M6 abajo): si `doctor` propagara el error de `auth.Resolve` en vez de informarlo, `TestDoctorSinCredencialInformaEnVezDeFallar` lo detecta.

No hubo ningún otro identificador de nivel de paquete en español sin traducción ya fijada en el glosario o en las correcciones del encargo.

## Corrección al brief del test de `TestDoctorTodoCorrecto`

El brief hace que el servidor de prueba devuelva `{"userId":"u-1","email":"a@b.c"}`, pero `sdk.Me` (Task 10, corregido contra `calliope-data-ui`) usa el campo `id`, no `userId` — así lo señala explícitamente el encargo ("Me tiene el campo ID, no UserID"). Usé `{"id":"u-1","email":"a@b.c"}` en las tres fixtures que simulan `/v1/auth/me`. No afecta al resultado del chequeo `conectividad` en sí (solo se usa `me.Email` en el detalle), pero con `userId` el campo `ID` habría quedado silenciosamente vacío sin que ningún test lo notara — no añadí una aserción sobre `me.ID` porque `doctor` no lo expone en ningún detalle; lo dejo anotado por si una tarea futura lo necesita.

## Test añadido más allá del brief: fuga del token

El aviso 4 del encargo pedía cuidado especial en que el token no aparezca nunca en la salida de `doctor`, en ningún modo, cubierto con un test. Añadí `TestDoctorNuncaImprimeElToken`, con tres subtests:

- `json/todo-correcto`: modo `--json`, conectividad ok.
- `json/backend-caido`: modo `--json`, conectividad en error (por si el error de `sdk` alguna vez arrastrara detalles de la petición).
- `texto/tty`: sin `--json`, con `d.IsTTY = true` (aviso 1 del encargo: el renderer de `Text` no se ejercita si no se fija `IsTTY` explícitamente).

Guardé una credencial con un token distintivo (`cal_live_no_debe_salir_nunca`) y comprobé `!strings.Contains(stdout, token)` en los tres casos. Confirmé con una mutación (N4 abajo) que el test detecta una fuga real: interpolar `cred.Token` en el detalle de `conectividad` hace fallar `json/todo-correcto` y `texto/tty` (pero no `json/backend-caido`, que nunca llega a construir ese detalle — comportamiento esperado, no un hueco).

## Verificación por mutación

Copié el repositorio completo (sin `.git`) a `/private/tmp/.../scratchpad/mutcopy`, confirmé la base verde, y apliqué/revertí 6 mutaciones sobre `doctor.go`, cada una ejecutando `go test ./internal/commands/ -run Doctor -v`.

| # | Mutación | Resultado |
|---|---|---|
| 1 | `credencial`: estado `"error"` → `"ok"` en la rama sin credencial (esconde el fallo) | FAIL (`TestDoctorSinCredencialInformaEnVezDeFallar`) |
| 2 | `checkConnectivity`: la rama de error queda inalcanzable (`if false`), sigue leyendo `me.Email` con `me == nil` | FAIL — panic (nil pointer) en `TestDoctorConBackendCaidoLoDetecta`, capturado por el test runner como fallo |
| 3 | Chequeo de `credencial` eliminado por completo (ni rama ok ni error) | FAIL (`TestDoctorTodoCorrecto`, `TestDoctorSinCredencialInformaEnVezDeFallar`) |
| 4 | `checkConnectivity`: el detalle incluye `cred.Token` (fuga del token) | FAIL (`TestDoctorNuncaImprimeElToken/json/todo-correcto`, `.../texto/tty`) |
| 5 | Chequeo de `organización`: condición invertida (`org != ""` en vez de `org == ""`, marca error cuando sí hay org) | FAIL (`TestDoctorTodoCorrecto`) |
| 6 | `doctor` propaga `errCred` con `return errCred` en vez de informar y seguir | FAIL (`TestDoctorSinCredencialInformaEnVezDeFallar`, `doctor nunca debe fallar, debe informar: ...`) |

Las 6 mutaciones quedaron capturadas en la primera pasada — ningún test necesitó refuerzo tras la ronda. Tras cada mutación se restauró `doctor.go` desde una copia (`doctor.go.orig`) guardada antes de empezar; al terminar comprobé con `diff` que la copia externa quedó **idéntica** al original antes de borrar el directorio de mutación. El árbol de trabajo real (`/Users/j10/repositories/calliope/calliope-cli`) no se tocó en ningún momento durante el proceso de mutación — solo se editó la copia en el scratchpad. La suite completa (`go build ./...`, `go vet ./...`, `gofmt -l .`, `go test ./... -race`, `go mod tidy`) se repitió limpia en el árbol de trabajo real después del proceso, antes del commit.

No mutué el `Text`/`Table` de la ruta de texto más allá de lo que cubre `TestDoctorNuncaImprimeElToken` (orden de columnas, símbolo por estado): el brief no exige una comprobación de formato del render humano de `doctor` y no hay una fixture con múltiples estados de `aviso` que lo justifique; lo anoto por si una ronda de revisión lo considera necesario.

## Dudas / puntos para el controlador

1. Fixture del brief (`{"userId":...}`) corregida a `{"id":...}` en los tests de `doctor` — documentado arriba, coherente con lo que ya señalaba el propio encargo sobre el campo `ID` de `sdk.Me`.
2. `checkConnectivity` no usa el `Timeout` de configuración (`config.KeyTimeout` / `timeoutOf` en `appctx.go`): construye su propio cliente SDK con un timeout fijo de 10s, tal como especifica el brief literalmente. Es intencional — un `doctor` que respetara un timeout de usuario configurado a, por ejemplo, 5 minutos tardaría igual de lento en informar de que el backend no responde — pero lo señalo por si el controlador prefiere documentarlo explícitamente en un comentario.

---

## Ronda de correcciones 1/5

El revisor confirmó que las tres correcciones ordenadas en la tarea original estaban bien aplicadas (identificadores en inglés, constantes `config.KeyBaseURL`/`config.KeyOrg`, solo `appctx.BuildSinCredencial`), que el token no se filtra en ninguno de los cinco modos de salida, y que `doctor` aguanta sin fallar seis escenarios malos de red/credencial. Encontró un Critical y dos Important, los tres reproducidos en vivo contra el binario real.

### Estado

DONE

### Commit

`88b611e` — fix: doctor sobrevive a un config.json corrupto, etiqueta bien el origen de la organización, y el SDK traduce el error de construcción de la petición
(4 ficheros: `internal/commands/doctor.go`, `internal/commands/doctor_test.go`, `internal/sdk/client.go`, `internal/sdk/client_test.go`)

### Resumen de tests

`go test ./... -race -v` → PASS en los 9 paquetes (204 tests en verde en todo el módulo, 0 fallos); `go vet ./...` limpio; `gofmt -l .` sin salida; `go mod tidy` sin cambios. 11 mutaciones dirigidas verificadas sobre una segunda copia completa del repo fuera del árbol de trabajo (una de ellas, la de la rama `--md`, sobrevivió en la primera pasada y quedó cerrada reforzando el test — ver abajo). Verificación en campo de los cinco modos de salida con el binario real contra un `config.json` global corrupto, incluida la variante de texto bajo un pty real (`script`).

### CRITICAL — `doctor` ya no falla con un `config.json` global corrupto

**Causa.** `appctx.BuildSinCredencial` llama a `config.Load`, que devuelve error si un fichero de configuración no parsea (`readLayerFile` en `internal/config/layers.go`). `doctor` propagaba ese error con `return err`, y como no es un `*output.CLIError`, `output.ExitCodeFor` lo mapeaba a exit 1 — justo el tipo de instalación rota que `doctor` existe para diagnosticar, tumbándolo en vez de informar.

**Corrección.** En `internal/commands/doctor.go`, `RunE` ahora captura ese error y lo delega a una función nueva, `renderConfigRota(cmd, d, errCfg) error`:

- Emite dos comprobaciones: `"versión"` (`ok`, no depende de la configuración) y `"configuración"` (`error`, con `Detail` que incluye el `err.Error()` completo de `config.Load` — que ya trae la ruta del fichero corrupto, ver `readLayerFile` — más el sufijo `"(corrígelo o bórralo y vuelve a intentarlo)"`).
- Sin `*appctx.Context` no hay `ctx.Render`, así que el render se arma a mano: `presenter.Render(result, outputModeSinConfig(cmd, d))`. `outputModeSinConfig` replica `appctx.outputMode` (no exportado) leyendo los flags del propio comando (`--jq`, `--json`, `--quiet`, `--md`) y `d.IsTTY`/`d.Stdout`, sin la capa `cfg.Output()` que aquí no existe porque la configuración es precisamente lo que falló.
- Devuelve el resultado de `presenter.Render` (puede fallar solo por un error de escritura real, no por el diagnóstico) — mismo patrón que el camino normal de `doctor`.

No añadí el chequeo de `credencial` a este camino degradado (aunque `auth.Resolve` no depende de `*config.Config` y técnicamente podría ejecutarse): el encargo pedía "al menos la versión", y mantener el conjunto mínimo (`versión` + `configuración`) acota el riesgo de esta ronda a lo pedido. Lo anoto por si una ronda futura quiere ampliarlo.

**Tests.** `TestDoctorConConfigCorruptaInformaEnVezDeFallar` (código 0, `versión`=`ok`, `configuración`=`error`, el `Detail` nombra el fichero — comprobado con `strings.Contains(detalle, "config.json")` — y dice cómo arreglarlo — `Contains(detalle, "corríge")`/`"bórra"`) y `TestDoctorConConfigCorruptaFuncionaEnLosCincoModos`, con una tabla de 6 casos (`auto-tty`, `auto-pipe`, `json`, `quiet`, `md`, `jq`) que no solo comprueban código 0 y stdout no vacío, sino la **forma específica** que solo ese modo produce (ver la corrección de test descrita abajo, "M2/M4 sobrevivieron").

Ambos tests usan un helper nuevo, `writeGlobalConfig(t, d, contenido)`, que escribe directamente en `HOME/.config/calliope/config.json` (la misma resolución de ruta que `config.globalDir`, confirmada leyendo `internal/config/layers.go` antes de escribir el test) — sin tocar `depsWithServer`, que no se modificó.

### IMPORTANT 1 — la organización que sale de la credencial ya no se etiqueta `(default)`

**Causa.** `doctor` etiquetaba siempre con `ctx.Cfg.Get(config.KeyOrg).Source`, pero el propio código tiene el fallback `if org == "" { org = cred.Org }`. Cuando la organización viene de ahí, `ctx.Cfg.Get(config.KeyOrg)` nunca tuvo ningún valor, así que `Source` cae al default de `config.Config.Get` (`SourceDefault` = `"default"`) — una etiqueta falsa.

**Corrección.** Se introduce `origenOrg`, inicializado igual que antes (`ctx.Cfg.Get(config.KeyOrg).Source`); solo cuando el fallback a `cred.Org` se dispara de verdad (`org == "" && cred.Org != ""`) se reasigna a la cadena literal `"credencial"`. El resto del comportamiento (incluida la rama de error cuando ninguna de las dos fuentes tiene valor) no cambia.

**Test.** `TestDoctorOrganizacionDesdeCredencialSeEtiquetaCorrectamente`: anula `CALLIOPE_ORG` en el entorno simulado (que `depsWithServer` sí fija, a `"acme"`) para que la única fuente posible sea la credencial, guarda una con `Org: "org-desde-credencial"` (valor distinto de `"acme"` y de cualquier otra fixture del fichero, para que un cableado cruzado sea observable) y comprueba el `Detail` exacto: `"org-desde-credencial (credencial)"`.

### IMPORTANT 2 — el error de construir la petición ya sale en español

**Causa.** En `internal/sdk/client.go`, `Do` propagaba tal cual el error de `http.NewRequestWithContext` (típicamente una URL con un carácter de control, p. ej. un `base_url` mal formado), sin pasar por `mapStatus`/`transportError`. `net/url` produce mensajes en inglés (`"parse \"...\": net/url: invalid control character in URL"`), filtrando detalle interno de la librería estándar e incumpliendo la restricción global de mensajes en español.

**Corrección.** Autorizado explícitamente a tocar `internal/sdk/client.go`, y solo para esto: el `if err != nil` tras `http.NewRequestWithContext` ahora devuelve `output.NewError(output.CodeGeneric, "No se pudo construir la solicitud a Calliope Data.", "Comprueba la URL del backend con: calliope config list")`, mismo patrón que `mapStatus`/`transportError`. El resto de `Do` no se tocó.

**Tests.**

1. `internal/sdk/client_test.go` — `TestErrorAlConstruirLaPeticionSaleEnEspanol`: `c.Do(ctx, GET, "/x\n", nil, &out)` (el `\n` es un carácter de control que `net/url` rechaza al parsear, sin tocar la red). Comprueba `*output.CLIError`, `Hint != ""`, y que el mensaje no contiene `"net/url"` ni `"control character"`.
2. `internal/commands/doctor_test.go` — `TestDoctorConectividadConBaseURLInvalidaSaleEnEspanol`, variante end-to-end: fija `CALLIOPE_BASE_URL` a una URL con un byte `\x00`, ejecuta `doctor --json` y comprueba que el `Detail` del chequeo `conectividad` (que es `err.Error()`, ver `checkConnectivity`) tampoco contiene esas cadenas en inglés.

Confirmé con TDD real (no solo razonamiento): con `git stash` sobre `client.go` (dejando el test nuevo en su sitio), `TestErrorAlConstruirLaPeticionSaleEnEspanol` falla con `se esperaba *output.CLIError, se obtuvo *url.Error: parse "http://127.0.0.1:NNNNN/x\n": net/url: invalid control character in URL` — el mensaje exacto que reportó el revisor. `git stash pop` restauró el fix antes de continuar.

### Un gap propio detectado y cerrado durante la verificación por mutación

Al mutar `outputModeSinConfig` quitando la rama `--jq` y, por separado, la rama `--md`, ambas mutaciones **sobrevivieron** en la primera pasada:

- **`--jq` (M2):** `TestDoctorConConfigCorruptaFuncionaEnLosCincoModos/jq` solo comprobaba `stdout.Len() != 0`. Sin la rama `--jq`, el modo cae a `ModeAuto` y, con `IsTTY:false`, produce el envelope JSON completo — no vacío, así que la aserción débil no lo detectaba.
- **`--md` (M4):** `doctor` no define `Result.Markdown`, así que `ModeMarkdown` cae a `writeJSON(w, r.Envelope)` — el mismo JSON que produciría `ModeAuto` con `IsTTY:false` por pura coincidencia de este caso concreto. Con el `isTTY` del caso de test en `false` (su cero por defecto), quitar la rama `--md` no cambiaba la salida observable.

Corregí el test, no la producción (el bug era del test, no del código): cada caso de la tabla ahora tiene un `comprueba func(t, salida)` que verifica la **forma específica** de su modo —

- `auto-tty`: contiene `"COMPROBACIÓN"` y no empieza por `{`.
- `auto-pipe`/`json`: decodifica como envelope con `"ok": true`.
- `quiet`: empieza por `[` y **no** contiene la clave de nivel superior `"ok":` (nota: no basta con buscar la subcadena `"ok"`, porque un chequeo con `Status: "ok"` también la contiene como *valor* — el primer intento de este test tuvo ese falso positivo, cerrado antes de dar la ronda por buena).
- `md`: ahora con `isTTY: true` a propósito — así se distingue de verdad de `ModeAuto` (que sí miraría `IsTTY` y renderizaría texto) en vez de coincidir con él por accidente; comprueba que sigue siendo el envelope JSON completo pese al TTY.
- `jq`: cambié la expresión de `.data` a `.data | length` y comprobé que la salida es exactamente `"2"` — así, si la rama `--jq` se perdiera, el resultado (el envelope o el array completo) nunca sería la cadena `"2"`, cerrando también el hueco de que un array por accidente pasara la comprobación de forma.

Con los tests reforzados, repetí las 2 mutaciones que habían sobrevivido: ambas quedaron capturadas.

### Verificación por mutación

Copié el repositorio completo (sin `.git`) a una segunda copia de trabajo fuera del árbol (`scratchpad/mutcopy2`), confirmé la base verde, y apliqué/revertí mutaciones sobre `doctor.go` y `client.go`, cada una ejecutando `go test ./internal/commands/ -run Doctor -count=1` o `go test ./internal/sdk/... -run Espanol -count=1` según el fichero mutado.

| # | Fichero | Mutación | Resultado |
|---|---|---|---|
| M1 | `doctor.go` | `return renderConfigRota(...)` → `return err` (reintroduce el Critical) | FAIL (`TestDoctorConConfigCorruptaInformaEnVezDeFallar`, `TestDoctorConConfigCorruptaFuncionaEnLosCincoModos` completo) |
| M2 | `doctor.go` | `outputModeSinConfig`: quita la rama `--jq` | **Sobrevivió en la 1ª pasada** (ver arriba) → FAIL tras reforzar el test |
| M3 | `doctor.go` | `outputModeSinConfig`: quita la rama `--quiet` | FAIL (`.../quiet`: dos aserciones, no empieza por `[` y filtra el envelope completo) |
| M4 | `doctor.go` | `outputModeSinConfig`: quita la rama `--md` | **Sobrevivió en la 1ª pasada** (ver arriba) → FAIL tras reforzar el test (`isTTY:true`) |
| M5 | `doctor.go` | `renderConfigRota`: `Out: d.Stdout` → `Out: d.Stderr` | FAIL (4 de 6 subtests: "no escribió nada en stdout") |
| M6 | `doctor.go` | `renderConfigRota`: quita el chequeo `"configuración"` (deja solo `versión`) | FAIL (`.../jq`: `.data | length` da `"1"`, no `"2"`) |
| M7 | `doctor.go` | `renderConfigRota`: `Detail` fijo (`"hay un problema con la configuración"`), pierde la ruta del fichero | FAIL (`TestDoctorConConfigCorruptaInformaEnVezDeFallar`, las dos aserciones sobre el `Detail`) |
| M8 | `doctor.go` | Quita `origenOrg = "credencial"` (reintroduce el Important 1) | FAIL (`TestDoctorOrganizacionDesdeCredencialSeEtiquetaCorrectamente`: `"(default)"` en vez de `"(credencial)"`) |
| N1 | `client.go` | Revierte el `output.NewError(...)` a `return err` crudo (reintroduce el Important 2) | FAIL (`TestErrorAlConstruirLaPeticionSaleEnEspanol`: `*url.Error` en vez de `*output.CLIError`) — confirmado también end-to-end vía `TestDoctorConectividadConBaseURLInvalidaSaleEnEspanol` |
| N2 | `client.go` | `Hint` del nuevo error vacío (`""`) | FAIL (`TestErrorAlConstruirLaPeticionSaleEnEspanol`: "el error debe traer un hint accionable") |

Dos intentos de mutación (uno sobre el bloque completo de `RunE` con comentarios de varias líneas, otro sobre el bloque completo de `organización`) no llegaron a aplicarse por un `perl -0pi` cuyo patrón multilínea no casaba exactamente con el fichero real — se detectaron porque el test seguía en verde sin que `grep` confirmara el cambio, y se repitieron con una sustitución de una sola línea, más simple y verificable con `grep` antes de correr el test; están reflejados como M1 y M8 en la tabla, ya con el patrón correcto.

Tras cada mutación se restauró el fichero mutado desde su copia `.orig2` guardada antes de empezar (`/tmp/doctor.go.orig2`, `/tmp/client.go.orig2`); al terminar comprobé con `diff` que ambos ficheros de la copia externa quedaron **idénticos** al original antes de borrar el directorio de mutación y los ficheros `.orig2`. El árbol de trabajo real no se tocó durante el proceso de mutación (con una excepción deliberada y ya revertida: el `git stash`/`git stash pop` sobre `client.go` y `doctor.go` para confirmar el TDD rojo→verde de los cuatro tests nuevos, documentado arriba). La suite completa (`go build ./...`, `go vet ./...`, `gofmt -l .`, `go test ./... -race -v`, `go mod tidy`) se repitió limpia en el árbol de trabajo real después de todo el proceso, antes del commit.

### Verificación en campo (binario real)

Con un `config.json` global corrupto (`{not valid json`) en un `HOME` temporal:

- `doctor --json`, `--quiet`, `--md`, `--jq '.data | length'` y `doctor | cat` (auto sin TTY): los cinco, exit 0, con el chequeo `configuración` en `error` nombrando la ruta real del fichero corrupto y el chequeo `versión` en `ok`. `--jq '.data | length'` imprime `2`.
- `doctor` bajo un pty real (vía `script -q /dev/null`, para forzar `auto` con TTY): exit 0, tabla de texto con columnas `COMPROBACIÓN`/`DETALLE`, símbolos `✓`/`✗` correctos.

### Dudas / puntos para el controlador

Ninguna nueva. Las dos dudas de la ronda anterior (fixture `id` vs `userId`, timeout fijo de `checkConnectivity`) siguen sin resolver por el controlador; no las he tocado en esta ronda porque no formaban parte de los tres defectos señalados.
