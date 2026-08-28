# Informe: oleada de correcciones de la revisión final del CLI

## Estado

DONE. Los 13 puntos (C1, C2, C3, I3, I4, I8, I10, M1, M2, Diferido #10,
Diferido #5, Diferido #7, Diferidos #12/#16) están aplicados, cada uno en su
propio commit, con `go test ./... -race`, `go vet ./...`, `gofmt -l .` y
`go mod tidy` limpios, y `go test -tags=e2e ./test/e2e/` compilando. Se
verificó cada arreglo con tests (mutados donde el encargo lo pedía) y contra
el binario real.

## Commits (rama `feat/cli-v1`, sobre `df919e4`)

| Commit | Punto | Resumen |
|---|---|---|
| `8afeda8` | C1 | `ask` deja de reenviar el cuerpo de error del backend |
| `1edc085` | C2 | Unifica la resolución del modo de salida de un error |
| `2c283bd` | C3 | Subcomando desconocido en un grupo ya no sale con exit 0 |
| `4d68260` | I3 | Comando o flag desconocido son `CLIError`, no el mensaje de Cobra |
| `7d6eb17` | I4 | Los 14 comandos hoja sin posicionales rechazan argumentos de más |
| `bcf7163` | I8 | Una colección vacía siempre serializa `[]` en vez de `null` |
| `88c8775` | I10 | El error de evaluación jq lleva hint |
| `4333156` | M1 | Unifica la construcción del cliente SDK |
| `250a15c` | M2 | Renombra los identificadores de paquete que quedaron en español |
| `e0e33ed` | Diferido #10 | Los errores de E/S no filtran rutas absolutas del sistema |
| `8cf5a4a` | Diferido #5 | `repoRoot` detecta un git worktree o submódulo |
| `526caf2` | Diferido #7 | Corrige la errata `TestErrorConfigJSONCorruptaInclujeRuta` |
| `f99e846` | Diferidos #12/#16 | Pluralización correcta en los `summary` del envelope |
| `793f669` | (doc) | Alinea el spec (§5) con el dictamen de C3, con una nota explicando el porqué |

14 commits nuevos sobre `df919e4`, ninguno usa `--amend` ni toca commits previos.

---

## C1 — `ask` reenvía el cuerpo de error del backend

`internal/commands/ask.go`: la rama `!resp.Success` ahora usa un mensaje fijo
en español ("Calliope no pudo responder a la pregunta."), conservando el
hint. `resp.Error` (el cuerpo de respuesta del backend, solo que dentro de un
200) ya no aparece en ningún sitio de la salida.

Tests: `TestAskConRespuestaSinExitoDevuelveError` (actualizado, ya no espera
el mensaje del backend) y `TestAskConRespuestaSinExitoNuncaFiltraElCuerpoDelBackend`
(nuevo, con un cuerpo con pinta de detalle interno — nombre de tabla, host,
puerto — y comprueba que no aparece ni en `Message` ni en `Hint`).

**Verificado por mutación:** reintroducir `mensaje = *resp.Error` hace fallar
ambos tests.

**Binario real** (backend falso que devuelve `success:false` con un cuerpo
con pinta de detalle interno):
```
$ calliope ask "que tal las ventas" --json
{"ok":false,"error":{"code":"ERROR","message":"Calliope no pudo responder a la pregunta.","hint":"Reformula la pregunta, o mira qué datos existen con: calliope concepts list"}}
exit=1
```

---

## C2 — el modo de salida de los errores estaba desconectado de la resolución real

- `cli.ExecuteRoot(d appctx.Deps) (*cobra.Command, error)` (nuevo, en
  `internal/cli/root.go`) construye la raíz, la ejecuta con `ExecuteC()` (no
  `Execute()`) y devuelve el `*cobra.Command` que de verdad se ejecutó, con
  sus flags ya fusionados y parseados incluso si la ejecución acabó en error.
- `appctx.outputMode` se exporta como `appctx.OutputMode`.
- `appctx.ResolveOutputMode(cmd, d)` (nuevo) resuelve el modo con la misma
  lógica que `Build`/`BuildWithoutCredential`: carga la config con los mismos
  flags, y si esa carga falla usa una config vacía en su lugar (nunca
  propaga el error — solo decide el formato del error que ya se está
  informando), sin re-imprimir los avisos de saneado de config (que ya
  habría impreso, si procede, la construcción original del contexto).
- `presenter.Options.IsMachineReadable()` (nuevo) centraliza en un único
  método la fórmula "todo salvo automático+TTY es para un programa" — antes
  vivía dos veces (el escaneo de `os.Args` en `main`, y la lógica real en
  `appctx`), que es justo el tipo de duplicación que causó el bug.
- `cmd/calliope/main.go` ahora usa `cli.ExecuteRoot` + `appctx.ResolveOutputMode`
  + `opts.IsMachineReadable()` en vez de `slices.Contains(os.Args, "--json")`.

Tests: `TestResolveOutputModeCubreLosSeisModos` (los seis modos de la tabla
6.2 + `CALLIOPE_OUTPUT=json`), `TestResolveOutputModeConConfiguracionRotaNoFalla`,
`TestIsMachineReadable` (tabla de decisión).

**Verificado por mutación:** una versión de `IsMachineReadable` que solo
compara `Mode == ModeJSON` (el bug original, en otra forma) hace fallar 5 de
7 subtests en ambos ficheros de test.

**Binario real:**
```
$ calliope orgs use > out.txt 2>&1; echo exit=$?; cat out.txt
exit=2
{"ok":false,"error":{"code":"USAGE","message":"Número de argumentos incorrecto: se esperaban 1, se recibieron 0.","hint":"Uso: calliope orgs use <organización>"}}

$ calliope ask --jq '.error.message'
{"ok":false,"error":{"code":"USAGE","message":"Número de argumentos incorrecto: se esperaban 1, se recibieron 0.","hint":"Uso: calliope ask <pregunta>"}}
exit=2
```
Ambos casos (tubería sin `--json`, y `--jq`) salen ahora como envelope JSON
completo con exit 2, no como texto plano con exit 1.

---

## C3 — subcomando desconocido en un grupo salía con exit 0

Causa raíz confirmada leyendo `cobra@v1.10.2/command.go`: `execute()` decide
`if !c.Runnable() { return flag.ErrHelp }` **antes** de `ValidateArgs`. Un
grupo sin `RunE` (como pedía el §5 original) nunca podía distinguir "sin
argumentos" de "con un argumento que no casa con ningún subcomando": las dos
rutas llegaban con `Runnable()==false`.

**Dictamen aplicado: gana el §6.3.** `groupRunE`
(`internal/commands/args.go`) se añade como `RunE` a los seis grupos
(`auth`, `orgs`, `config`, `docs`, `concepts`, `rules`): con 0 argumentos
llama a `cmd.Help()` (exit 0); con argumentos, devuelve un `CLIError` en
español con hint y `CodeUsage` (exit 2). El spec (§5) se actualizó para
reflejar el comportamiento en vez del mecanismo, con una nota explicando la
contradicción y el porqué del dictamen (commit `793f669`).

`TestNingunGrupoDeRecursosDefineRunE` (`internal/cli/catalog_test.go`) se
reescribió como `TestGrupoDeRecursosMuestraAyudaPeladoYFallaConSubcomandoDesconocido`:
ejecuta el árbol real con ambos argumentos para los seis grupos, en vez de
comprobar `RunE == nil`. Las cuatro pruebas equivalentes por paquete que
tenían el mismo defecto (`TestAuthEsUnGrupoSinRunE`, `TestOrgsEsUnGrupoSinRunE`,
`TestDocsEsUnGrupoSinRunE`, `TestConceptsYRulesSonGruposSinRunE`) se
reescribieron igual.

**Verificado por mutación:** quitar `RunE: groupRunE` de `docs` hace fallar
tanto el test de `internal/cli` como el de `internal/commands`.

**Binario real, los seis grupos, pelado:**
```
$ calliope auth;     exit=0   → "Gestiona la autenticación con Calliope Data" + Usage:
$ calliope orgs;     exit=0   → "Lista y selecciona la organización activa" + Usage:
$ calliope config;   exit=0   → "Consulta y modifica la configuración de calliope" + Usage:
$ calliope docs;     exit=0   → "Consulta la documentación de la organización" + Usage:
$ calliope concepts; exit=0   → "Explora los conceptos de negocio de la ontología" + Usage:
$ calliope rules;    exit=0   → "Consulta las reglas de negocio compartidas" + Usage:
```

**Binario real, los seis grupos, subcomando inexistente (`typo --json`):**
```
$ calliope auth typo --json
{"ok":false,"error":{"code":"USAGE","message":"\"typo\" no es un subcomando de \"calliope auth\".","hint":"Consulta los subcomandos disponibles con: calliope auth --help"}}
exit=2

$ calliope orgs typo --json
{"ok":false,"error":{"code":"USAGE","message":"\"typo\" no es un subcomando de \"calliope orgs\".","hint":"Consulta los subcomandos disponibles con: calliope orgs --help"}}
exit=2

$ calliope config typo --json
{"ok":false,"error":{"code":"USAGE","message":"\"typo\" no es un subcomando de \"calliope config\".","hint":"Consulta los subcomandos disponibles con: calliope config --help"}}
exit=2

$ calliope docs typo --json
{"ok":false,"error":{"code":"USAGE","message":"\"typo\" no es un subcomando de \"calliope docs\".","hint":"Consulta los subcomandos disponibles con: calliope docs --help"}}
exit=2

$ calliope concepts typo --json
{"ok":false,"error":{"code":"USAGE","message":"\"typo\" no es un subcomando de \"calliope concepts\".","hint":"Consulta los subcomandos disponibles con: calliope concepts --help"}}
exit=2

$ calliope rules typo --json
{"ok":false,"error":{"code":"USAGE","message":"\"typo\" no es un subcomando de \"calliope rules\".","hint":"Consulta los subcomandos disponibles con: calliope rules --help"}}
exit=2
```

---

## I3 — comando o flag desconocido

- `root.SetFlagErrorFunc(flagError)` en `internal/cli/root.go` (se hereda a
  todos los subcomandos): `flagError` (nuevo, `internal/cli/errors.go`)
  detecta `*pflag.NotExistError` (tipado, con `GetSpecifiedName()`/
  `GetSpecifiedShortnames()`, así que no hace falta parsear el texto del
  mensaje) para "flag desconocido" (largo o corto), y da un mensaje genérico
  en español para cualquier otro fallo de flags.
- `wrapUnknownCommand` (nuevo) reconoce por regexp el mensaje fijo que Cobra
  construye en `legacyArgs` para un comando de nivel superior desconocido —
  el único error que Cobra puede producir antes de que se ejecute código
  nuestro, así que no hay tipo ni hook que interceptarlo de otra forma.
  `cli.ExecuteRoot` lo aplica al resultado de `ExecuteC()`.
- `go.mod`: `github.com/spf13/pflag` pasa de indirecta a directa (se importa
  ahora para el type switch de `flagError`); `go mod tidy` lo hizo solo.

Tests nuevos en `internal/cli/errors_test.go`: comando desconocido (directo
y vía `ExecuteRoot`), flag largo desconocido, y que `wrapUnknownCommand` no
toca un error que no coincide con el patrón.

**Binario real:**
```
$ calliope frobnicate --json
{"ok":false,"error":{"code":"USAGE","message":"\"frobnicate\" no es un comando de calliope.","hint":"Consulta los comandos disponibles con: calliope --help"}}
exit=2

$ calliope docs list --xxx --json
{"ok":false,"error":{"code":"USAGE","message":"Flag desconocido: --xxx.","hint":"Consulta los flags disponibles con: calliope docs list --help"}}
exit=2

$ calliope docs list -z
{"ok":false,"error":{"code":"USAGE","message":"Flag desconocido: -z.","hint":"Consulta los flags disponibles con: calliope docs list --help"}}
exit=2
```

---

## I4 — 14 comandos hoja aceptaban argumentos posicionales de más

`NoPositionalArgs()` (nuevo, `internal/commands/args.go`) es `exactArgs(0)`
con un nombre que documenta la intención; exportada porque `version` vive en
el paquete `cli`, no en `commands`. Aplicada a los 14: `version`; `auth
login/logout/status/token`; `orgs list`; `config list/path`; `docs list`;
`concepts list`; `rules list`; `schema`; `doctor`; `skill`.

El test nuevo (`TestComandosHojaSinPosicionalesRechazanArgumentosDeMas`,
`internal/cli/catalog_test.go`) no enumera esos 14 a mano: los deriva del
propio `Catalog()` — cualquier `CommandInfo.Args == ""` debe rechazar un
argumento de más — y ancla el recuento en 14 para que una desviación futura
se note.

**Verificado por mutación:** quitar `Args: NoPositionalArgs()` de `docs list`
hace fallar ese subtest específico (y expone, de paso, que sin la
validación el `RunE` real llega a ejecutarse con `Deps{}` vacías — otra
razón más para que la validación exista).

**Binario real:**
```
$ calliope docs list READY --json
{"ok":false,"error":{"code":"USAGE","message":"Número de argumentos incorrecto: se esperaban 0, se recibieron 1.","hint":"Uso: calliope docs list"}}
exit=2

$ calliope version algo-de-mas --json
{"ok":false,"error":{"code":"USAGE","message":"Número de argumentos incorrecto: se esperaban 0, se recibieron 1.","hint":"Uso: calliope version"}}
exit=2
```

---

## I8 — colección vacía serializaba `null` en vez de `[]`

`output.OKEnvelope` ahora pasa `data` por `normalizeData` (nuevo,
`internal/output/envelope.go`): si es un slice nil, lo sustituye por un
slice vacío del mismo tipo vía reflexión, antes de guardarlo en el
envelope. Un slice nil dentro de un campo `any` no es una interfaz nil —
sigue llevando su tipo, igual que un puntero nil tipado—, así que
`json:"data,omitempty"` no lo omite y `encoding/json` lo serializa como
`null`. Vive en el único punto por el que pasa todo comando de éxito, así
que vale para cualquier lista futura sin tener que acordarse en cada
comando.

Tests: 3 unitarios en `internal/output/envelope_test.go` (varios tipos de
slice nil → `[]`; que no toca slices con datos, punteros ni mapas) + 5 de
extremo a extremo, uno por comando afectado (`docs list`, `rules list`,
`concepts list`, `schema`, `query`), simulando la respuesta del backend que
produce el slice nil en cada caso (`"content":null`, cuerpo `null` a secas
para `rules` porque su endpoint no envuelve el array, `"concepts":null`,
`"tables":null`, `"data":null`).

**Verificado por mutación:** revertir `OKEnvelope` a `Data: data` (sin
normalizar) hace fallar los 3 tests unitarios y los 5 de extremo a extremo.

**Binario real** (la receta exacta del SKILL.md, `rules list --jq '.data[]'`,
con backend que responde `null`):
```
$ calliope rules list --jq '.data[]'
exit=0   (sin salida — antes: "cannot iterate over: null (null)" con exit 2)
```

---

## I10 — el error de evaluación jq no tenía hint

`internal/presenter/jq.go`: el `CLIError` del branch `err, ok := v.(error)`
ahora lleva el mismo hint que el de parseo (tres líneas más arriba: revisar
la expresión contra la forma real del envelope, con `--json`, o consultar el
manual de jq), y el mensaje técnico de gojq queda enmarcado en una frase en
español (`"Error al evaluar la expresión jq: %v."`) en vez de suelto. El
mensaje de gojq describe un fallo de la expresión del propio usuario contra
la forma de nuestros datos — no un dato ajeno como el cuerpo de una
respuesta del backend —, así que conservarlo no es una filtración.

`TestJQErrorEnEvaluacionDevuelveErrorDeUso` se extendió para comprobar hint
no vacío y el prefijo en español del mensaje.

**Binario real:**
```
$ calliope rules list --jq '.data.foo'
{"ok":false,"error":{"code":"USAGE","message":"Error al evaluar la expresión jq: expected an object but got: array (...).","hint":"Comprueba que la expresión encaja con la forma real del envelope -pruébalo primero con --json-, o consulta la sintaxis en https://jqlang.github.io/jq/manual/"}}
exit=2
```

---

## M1 — dos clientes SDK distintos

`clientWith` (`internal/commands/auth.go`, usado por `auth login` y `orgs
list`) construía el cliente sin `Timeout` ni `UserAgent`. Ahora usa
`appctx.TimeoutOf(ctx.Cfg)` (antes `timeoutOf`, sin exportar — exportado
para esto) y `"calliope-cli/"+version.Version`, igual que `appctx.Build`.
`newAuthLoginCmd`, que montaba su propio `sdk.New` en paralelo con la misma
carencia, pasa a llamar a `clientWith` también.

Tests de extremo a extremo para los dos comandos: el `User-Agent` que ve el
backend, y que `CALLIOPE_TIMEOUT` corta una petición lenta (200ms con
timeout de 20ms) en vez de esperar el timeout por defecto de 60s.

**Verificado por mutación:** revertir `clientWith` a la versión sin
`Timeout`/`UserAgent` hace fallar los 4 tests nuevos.

**Binario real** (backend que registra el `User-Agent` que recibe):
```
$ CALLIOPE_BASE_URL=http://127.0.0.1:8995 calliope orgs list --json
→ el backend registra: User-Agent: calliope-cli/dev
```
(antes: `calliope-cli`, sin versión).

---

## M2 — identificadores de paquete en español

Renombrados (glosario actualizado en
`.superpowers/sdd/2026-08-27-calliope-cli/glosario.md`, fuera de control de
versiones por `.gitignore`):

- `appctx.BuildSinCredencial` → `appctx.BuildWithoutCredential` (6 ficheros).
- `renderConfigRota` → `renderBrokenConfig` (`internal/commands/doctor.go`).
- `outputModeSinConfig` → `outputModeWithoutConfig` (íd.; de paso se corrigió
  un comentario que seguía citando `appctx.outputMode`, ya exportado como
  `OutputMode` desde C2).
- `raizDelRepo` → `repoRoot` (`internal/cli/plugin_test.go`; el glosario ya
  traía esta traducción fijada, solo faltaba aplicarla).
- `internal/cli/paridad_test.go` → `parity_test.go` (los identificadores que
  contiene ya estaban en inglés desde una tarea anterior).

Los nombres de test que mencionan `BuildSinCredencial`
(`TestBuildSinCredencialNoFallaSinAutenticacion`) se dejaron tal cual, por la
regla ya fijada en el glosario (son prosa, no identificadores de API); las
llamadas dentro de ese test sí usan el nombre nuevo.

No se encontraron más identificadores de paquete en español fuera de esta
lista explícita (barrido final con grep sobre los cinco términos, limpio).

---

## Diferido #10 — errores de E/S en crudo

`output.WrapIOError(message, hint string, cause error) error` (nuevo,
`internal/output/errors.go`) generaliza el patrón que ya usaba
`corruptCredentialError` (`internal/auth/store.go`): mensaje y hint en
español, causa técnica original conservada solo para
`errors.As`/`errors.Is`, nunca en el mensaje.

Aplicado en:
- `internal/auth/store.go`: `fileStore.Save` y `fileStore.Delete` (cubre
  `auth login` y `auth logout` sin cambios propios en `auth.go`).
- `internal/commands/orgs.go`: `newOrgsUseCmd` (MkdirAll, WriteFile).
- `internal/commands/config.go`: `newConfigSetCmd` (MkdirAll, Chmod,
  WriteFile), con `configIOHint(global)` para distinguir el hint según se
  esté escribiendo la config global o la de proyecto.

Tests: 4 (uno por comando afectado), cada uno con un directorio `chmod 0500`
forzando `permission denied`, comprobando `CLIError` con hint, sin la ruta
absoluta ni "permission denied" en el mensaje.

**Binario real:**
```
$ cd /tmp/dir-sin-permisos && calliope orgs use acme --json
{"ok":false,"error":{"code":"ERROR","message":"No se pudo crear el directorio .calliope del proyecto.","hint":"Comprueba los permisos de escritura en el directorio actual."}}
exit=1
```

---

## Diferido #5 — `repoRoot` no reconocía un git worktree

`internal/config/layers.go`: `repoRoot` exigía `fi.IsDir()`; ahora acepta
también `fi.Mode().IsRegular()` (el caso de un worktree o un submódulo,
donde `.git` es un fichero de una línea `"gitdir: <ruta>"`). Sin esto, la
capa `SourceRepo` entera desaparecía en silencio en un worktree — tanto para
una configuración legítima como, más grave, para una hostil que debía
sanearse.

Test nuevo simulando un worktree (`.git` fichero con contenido `"gitdir:
..."`), y **verificación adicional contra un `git worktree` real** de este
mismo repositorio con un `config.json` hostil en su raíz:
```
$ git worktree add /tmp/x/wt HEAD
$ cat /tmp/x/wt/.git
gitdir: /Users/j10/repositories/calliope/calliope-cli/.git/worktrees/wt
$ echo '{"org":"acme","base_url":"https://atacante.example.com"}' > /tmp/x/wt/.calliope/config.json
$ (cd /tmp/x/wt && calliope config list --json)
aviso: se ignora "base_url" de .../wt/.calliope/config.json; la configuración de proyecto solo puede fijar: org, output
{"data":{"base_url":{"value":"https://data-0.calliope.so","source":"default"}, "org":{"value":"acme","source":"project",...}, ...}}
```
`base_url` se queda en el valor por defecto; el aviso de saneado sale.

**Verificado por mutación:** revertir a `fi.IsDir()` sin la alternativa hace
fallar el test nuevo.

---

## Diferido #7 — errata

`TestErrorConfigJSONCorruptaInclujeRuta` → `TestErrorConfigJSONCorruptaIncluyeRuta`.
Solo el nombre; sin cambio de comportamiento.

---

## Diferidos #12 y #16 — pluralización

`pluralize(n int, singular, plural string) string`
(`internal/commands/pluralize.go`, nuevo, con test unitario): 1 usa
singular; 0 y cualquier otro valor usan plural. Aplicado a todos los
`summary` con conteo: `ask` (fuentes citadas), `config list` (valores),
`schema` (tablas), `query` (filas), `docs list` (documentos — gobernado por
`TotalSize`, el número que precede al sustantivo, no por el tamaño de la
página en curso), `docs search` (fragmentos), `concepts list` (conceptos),
`concepts show` (atributos), `rules list` (reglas), `orgs list`
(organizaciones), y `doctor` (comprobaciones — el verbo se deja en plural
sin acordarlo con el número, porque `doctor` siempre emite al menos dos
comprobaciones y en la práctica nunca es 1).

4 tests existentes aseveraban el texto plural roto con `n=1` y se
actualizaron: `ask` ("1 fuentes citadas"→"1 fuente citada"), `query` ("1
filas"→"1 fila"), `docs list` ("1 de 1 documentos"→"1 de 1 documento"),
`docs search` ("1 fragmentos"→"1 fragmento").

**Binario real:**
```
$ calliope rules list --jq '.summary'   (backend con 1 regla)
"1 regla"
```

---

## Verificación final

```
$ go build ./...                    → limpio
$ go test ./... -race                → 247 subtests, 0 fallos, todos los paquetes ok
$ go vet ./...                       → limpio
$ gofmt -l .                         → sin salida
$ go mod tidy                        → sin cambios en go.mod/go.sum
$ go build -tags=e2e ./...           → limpio
$ go vet -tags=e2e ./...             → limpio
$ go test -tags=e2e ./test/e2e/      → compila, sin credenciales reales disponibles
                                        en este entorno (igual que en la Task 22)
```

## Tests existentes que se actualizaron a propósito

Todos por comportamiento que esta oleada cambia deliberadamente, listados en
sus secciones arriba:

- `TestAskConRespuestaSinExitoDevuelveError` (C1): ya no espera el mensaje
  del backend, espera el mensaje fijo.
- `TestNingunGrupoDeRecursosDefineRunE` y sus cuatro equivalentes por
  paquete (C3): reescritos para asertar comportamiento, no mecanismo.
- 4 asserts de `summary` con `n=1` (Diferidos #12/#16).
- 1 errata de nombre de test (Diferido #7).

Ningún test se debilitó ni se borró para que la suite pasara: donde el
comportamiento cambió, el test nuevo o reescrito cubre el comportamiento
correcto tan estrictamente como el original cubría el incorrecto (verificado
por mutación en los 5 puntos que el encargo señalaba: C1, C2, C3, I4, I8 —
y además en M1 y Diferido #5, por rigor propio).

## Dudas / decisiones que convendría que alguien revise

1. **C3, alcance de `groupRunE`:** el spec (§5) solo enumera los seis grupos
   como "grupos de recursos". La raíz (`calliope` pelado) no está en esa
   lista y sigue sin `RunE` — Cobra ya la trata bien (bare → ayuda, exit 0)
   porque `legacyArgs` solo dispara "unknown command" cuando la raíz no
   tiene padre, no por el chequeo `Runnable()`. No le apliqué `groupRunE` a
   la raíz porque no lo necesita, pero lo dejo anotado por si alguien
   esperaba que la raíz también pasara por ese mismo mecanismo.
2. **I10, alcance de la traducción:** dejé el mensaje técnico de gojq
   embebido (enmarcado en español), igual que ya hacía el error de parseo
   tres líneas arriba. No traduje los mensajes de gojq mensaje a mensaje
   (son dinámicos y no hay una tabla de mapeo razonable) — si el criterio
   real era "cero inglés en el mensaje, sea como sea", esto no lo cumple al
   100%; sí lo dejé estrictamente mejor que estaba (con hint, con marco en
   español) y consistente con el hermano que no se señaló como roto.
3. **I3, alcance de `flagError`:** solo dediqué tratamiento fino (tipo
   `*pflag.NotExistError`, literal exacto reconstruido) al caso "flag
   desconocido", que es el que pedía el encargo. Otros fallos de flags
   (`--foo` sin el valor que necesita, un valor que no parsea) caen en un
   mensaje genérico que envuelve el texto de pflag en una frase en
   español — no está roto, pero tampoco recibió el mismo pulido.
4. **M2, alcance del barrido:** hice un grep final sobre los cinco
   identificadores que el encargo señaló explícitamente (limpio), pero no
   audité el 100% del árbol de identificadores del proyecto en busca de
   otros posibles restos en español fuera de esa lista — el encargo describe
   la lista como cerrada ("los identificadores en español de nivel de
   paquete... son estos"), así que asumí que ya estaba completa.
