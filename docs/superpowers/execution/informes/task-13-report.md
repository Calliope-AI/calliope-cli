# Informe — Task 13: Comando `config`

## Estado

DONE

## Commit

`4ace6ae` — feat: comando config con procedencia de cada valor
(3 ficheros: `internal/commands/config.go`, `internal/commands/config_test.go`, `internal/cli/root.go`)

## Resumen de tests

`go test ./... -race` → PASS en los 9 paquetes (7 con tests); `go vet ./...` limpio; `gofmt -l .` sin salida; `go mod tidy` sin cambios. Los 4 tests de `config_test.go` verificados uno por uno por mutación dirigida sobre el propio árbol de trabajo (mutación → test correspondiente falla → revertido con `cp` desde una copia intacta), incluida la mitad de escritura de la frontera de seguridad (`config set` rechazando `base_url` en proyecto).

## Qué se hizo

- `internal/commands/config.go`: grupo `config` con `list`, `get`, `set`, `path`, copiado del brief con las dos correcciones indicadas:
  - `config.IsProjectAllowed(clave)` en vez de `config.ProjectAllowed[clave]` (ya no exportado).
  - `exactArgs(1)` / `exactArgs(2)` en vez de `cobra.ExactArgs(1)` / `cobra.ExactArgs(2)` en `get` y `set`.
  - El grupo (`NewConfigCmd`) no define `RunE`, solo cuelga subcomandos.
- `internal/commands/config_test.go`: los 4 tests del brief, traduciendo `depsConServidor`→`depsWithServer` y `raizDePrueba`→`testRoot` (ya existentes en `auth_test.go`, no redefinidos). Nombres de test en español, sin cambios.
- `internal/cli/root.go`: añadido `commands.NewConfigCmd(d)` a `root.AddCommand(...)`.

## Corrección adicional encontrada durante TDD (no listada en el encargo)

Al ejecutar `TestConfigSetRechazaClavesNoPermitidasEnProyecto` tal como lo trae el brief, falló:

```
el error debe explicar que base_url solo se fija en global: "La clave \"base_url\" no se puede fijar en la configuración de proyecto."
```

Causa: el test comprueba `err.Error()`, pero `output.CLIError.Error()` devuelve **solo** `Message`, nunca `Hint` — contrato fijado explícitamente por `TestCLIErrorErrorDevolverSoloMensaje` en `internal/output/errors_test.go:51-56`, de una tarea anterior ya mergeada. El brief había puesto el texto "--global" únicamente en el `Hint`, así que `err.Error()` nunca lo veía.

No pude tocar `CLIError.Error()` (rompería ese test pinneado). Apliqué el fix mínimo dentro de `newConfigSetCmd`: el `Message` de rechazo ahora incluye explícitamente "usa --global para fijarla en la configuración global", y el `Hint` conserva el comando exacto de recuperación (`calliope config set <clave> <valor> --global`), cumpliendo también la restricción global de que todo error con acción de recuperación lleve `hint`. Documentado con un comentario en el propio código explicando por qué el mensaje repite lo que dice el hint.

Verificado por mutación: revertir el `Message` al texto literal del brief (sin "--global") hace fallar de nuevo el test, confirmando que el fix es necesario y suficiente.

## Verificación manual adicional

Con el binario compilado (`go build ./cmd/calliope`) y HOME/cwd en directorios temporales:
- `config set base_url ... ` (sin `--global`) → rechazado, exit code 2, mensaje y hint correctos, nada escrito en `.calliope/config.json`.
- `config set base_url ... --global` → aceptado, escrito en el fichero de configuración global (`$HOME/.config/calliope/config.json`), nada en el proyecto.
- `config set output json` (clave permitida, sin `--global`) → aceptado en proyecto.
- `config path` → imprime la ruta global correcta.
- `config list --md` → cae a JSON completo (correcto: no se define `Result.Markdown`, igual que `orgs list`), y cada clave muestra su `source`/`path` reales (`env`, `global`, `project`, `default`).

## Dudas / puntos para el controlador

1. La corrección del `Message` en `config set` (ver arriba) no estaba en la lista de "correcciones al brief" del encargo. La apliqué porque era necesaria para que el test literal del brief pasara, sin tocar el contrato ya fijado de `CLIError.Error()`. Si el controlador prefiere una solución distinta (p. ej. que el test comprobara `cliErr.Hint` en vez de `err.Error()`, como hacen los tests de `orgs_test.go`), avisar para ajustar.
2. `config get <clave-inexistente>` no fue un caso cubierto por los tests del brief; con la implementación actual devuelve `Value{Value:"", Source:"default"}` sin error (comportamiento heredado de `config.Config.Get`, ya probado en el paquete `config`). No añadí un test nuevo para esto por no ampliar el alcance más allá de lo pedido; lo señalo por si se quiere cubrir en una tarea posterior.
3. No hubo identificadores en español a nivel de paquete que requirieran traducción nueva para el glosario: `NewConfigCmd`, `newConfigListCmd`, `newConfigGetCmd`, `newConfigSetCmd`, `newConfigPathCmd` ya venían en inglés en el brief.

---

## Ronda de correcciones 1/5

### Estado

DONE

### Qué se corrigió

**IMPORTANT 1 — la ruta `--global` era invisible para los tests.**
Añadidos:
- `TestConfigSetGlobalEscribeEnGlobalNoEnProyecto`: `config set base_url ... --global` se acepta, escribe en la ruta global (`config.GlobalPath`) con el valor correcto, y **no** deja rastro en `.calliope/config.json` del proyecto.
- `TestConfigSetGlobalPreservaLasClavesPrevias`: la variante `--global` de la fusión con valores previos (ver REMATE 4 más abajo).
- `TestConfigSetGlobalArreglaPermisosLaxosDelDirectorioExistente`: cubre a la vez la aceptación de `--global` y los permisos (ver IMPORTANT 2).

La mitad de *rechazo* de `base_url` sin `--global` ya estaba cubierta por `TestConfigSetRechazaClavesNoPermitidasEnProyecto` (usa exactamente esa clave); no se dupicó.

Verificado por mutación (`if global {` → `if false {`): los tres tests nuevos fallan — dos porque `base_url` cae en el rechazo de proyecto (nunca llega a escribirse), uno porque el directorio global pre-existente con permisos laxos nunca se toca.

**IMPORTANT 2 — directorio global creado en 0755 en vez de 0700.**
`newConfigSetCmd` ahora usa un `dirMode` condicional: `0o700` solo cuando `global` es cierto (el directorio de proyecto se queda en `0o755`, igual que `orgs.go`, porque no es una frontera de confianza — ya es público dentro del repo). Tras `MkdirAll`, si `global` es cierto se refuerza con `os.Chmod(dir, 0o700)` explícito, exactamente el patrón y el comentario de `internal/auth/store.go` (`fileStore.Save`), porque es el mismo directorio.

Test nuevo: `TestConfigSetGlobalArreglaPermisosLaxosDelDirectorioExistente` pre-crea el directorio global con `0o755` y comprueba que tras `config set --global` queda en `0o700`.

Verificado por dos mutaciones:
- Quitar el `os.Chmod` de refuerzo (dejando solo `MkdirAll(dir, 0o700)`) → el test falla, porque `MkdirAll` no toca el modo de un directorio que ya existía.
- `--global` inerte (mutación de IMPORTANT 1) → también lo hace fallar, por una vía distinta (el directorio global nunca se toca).

**REMATE 3 — `config get` ignoraba su argumento sin que nadie se enterara.**
Añadido `TestConfigGetRespetaLaClavePedida`: pide `base_url` (no `org`) y compara contra `d.Env("CALLIOPE_BASE_URL")` — el valor exacto que `depsWithServer` fija para esa invocación, no un literal aproximado — y además comprueba que la salida no contiene `"acme"` (el valor de `org`). Verificado por mutación (`ctx.Cfg.Get(args[0])` → `ctx.Cfg.Get(config.KeyOrg)`): falla con las dos aserciones.

**REMATE 4 — la fusión con valores previos no estaba cubierta.**
Añadidos `TestConfigSetPreservaLasClavesPreviasEnProyecto` (siembra `.calliope/config.json` con `output=json`, fija `org=globex`, comprueba que ambas claves sobreviven) y `TestConfigSetGlobalPreservaLasClavesPrevias` (mismo patrón en la ruta global, con `timeout=30s` previo y `base_url` nuevo). Verificado por mutación (sustituir la lectura+`json.Unmarshal` previos por un mapa vacío antes de fijar la clave nueva): ambos tests fallan, cada uno pierde su clave previa.

**REMATE 5 — permisos del fichero no comprobados.**
Añadida la aserción `fi.Mode().Perm() == 0o600` a `TestConfigSetEscribeUnaClavePermitida` (proyecto) y a `TestConfigSetGlobalEscribeEnGlobalNoEnProyecto` (global). Verificado por mutación (`os.WriteFile(ruta, b, 0o600)` → `0o644`): ambos tests fallan.

### Resumen de tests

`go test ./... -race -v` → PASS en todos los paquetes, 9/9 tests de `Config*` en verde; `go vet ./...` limpio; `gofmt -l .` sin salida; `go mod tidy` sin cambios (`go.mod`/`go.sum` intactos). Las 5 mutaciones dirigidas (una por hallazgo, dos para IMPORTANT 2) se aplicaron sobre el árbol de trabajo real, confirmaron el fallo esperado y se revirtieron restaurando desde una copia intacta antes de continuar.

### Alcance no tocado (a propósito)

- `config get <clave-inexistente>`: confirmado como fuera de alcance por el propio controlador.
- No añadí un `os.Chmod` de refuerzo al *fichero* de configuración (solo al directorio global): el controlador acotó IMPORTANT 2 explícitamente al directorio, y `os.WriteFile(ruta, b, 0o600)` ya crea el fichero con el modo correcto en el escenario real (directorio temporal fresco en cada test; no hay caso en la suite donde el fichero preexistiera con permisos más laxos). Lo dejo anotado por si en una ronda futura se quiere el mismo refuerzo que aplica `auth/store.go` a su fichero de credenciales, por simetría — no es una vulnerabilidad confirmada, es una asimetría de diseño.

### Dudas

Ninguna nueva. La corrección aplicada en la ronda anterior (`Message` con "--global") se mantiene sin cambios; el revisor ya la validó.
