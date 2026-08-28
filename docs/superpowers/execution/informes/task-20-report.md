# Informe — Task 20: Plugin de Claude Code

## Estado

DONE

## Commit

`21af6f5` — feat: plugin de Claude Code con skill y hook de sesión
(5 ficheros: `.claude-plugin/plugin.json`, `.claude-plugin/commands/calliope.md`,
`.claude-plugin/hooks/session-start.sh`, `.claude-plugin/skills` [enlace simbólico],
`internal/cli/plugin_test.go`)

## Resumen de tests

`go test ./... -race` → PASS en los 10 paquetes (8 con tests), incluidos los 2 tests
nuevos de `internal/cli/plugin_test.go`; `go vet ./...` limpio; `gofmt -l .` sin salida;
`go mod tidy` sin cambios. Ambos tests verificados por mutación sobre una copia completa
del árbol fuera del repo (`/tmp/calliope-mutacion`, borrada al terminar).

## Qué se hizo

Todo copiado literalmente del brief (`task-20-brief.md`), en el orden TDD indicado, con
una desviación deliberada y documentada en el hook (ver más abajo):

1. `internal/cli/plugin_test.go` — literal del brief.
2. `go test ./internal/cli/ -run Plugin -v` → FAIL, `no such file or directory` para
   `.claude-plugin/skills/calliope/SKILL.md` y `.claude-plugin/plugin.json`. Confirmado
   antes de tocar nada.
3. `.claude-plugin/plugin.json` (con el bloque `hooks.SessionStart` ya incluido desde el
   principio, en vez de añadirlo en el Step 4 por separado — mismo contenido final que
   pide el brief) + `.claude-plugin/skills` como enlace simbólico:
   `cd .claude-plugin && ln -s ../skills skills`.
4. `.claude-plugin/commands/calliope.md` — literal del brief.
   `.claude-plugin/hooks/session-start.sh` — mismo contrato que el brief (nunca falla,
   nunca cuelga, siempre sale 0), pero con la lógica de "¿está listo?" reescrita porque
   la del brief no puede funcionar contra el `doctor` real de la Task 18 (razón completa
   más abajo). `chmod +x` aplicado.
5. `go test ./internal/cli/ -race` → PASS.
6. Hook probado a mano en los tres escenarios (salida real más abajo).
7. Commit.

## Verificación del enlace simbólico en git

```
$ git ls-files -s .claude-plugin
100644 a941847... 0	.claude-plugin/commands/calliope.md
100755 87c0995... 0	.claude-plugin/hooks/session-start.sh
100644 b26908e... 0	.claude-plugin/plugin.json
120000 42c5394... 0	.claude-plugin/skills

$ git cat-file -p 42c5394a18a882778ebf50eb940fb5a96bc4a6d9
../skills
```

Modo `120000` confirmado (enlace simbólico, no directorio ni copia). El commit real
(`git show --stat 21af6f5`) también lo confirma: `create mode 120000 .claude-plugin/skills`.

## `.gitignore`

Nada de lo creado cae bajo los patrones ignorados (`dist/`, `bin/`, `/calliope`,
`.calliope/`, `.superpowers/`): `.claude-plugin/` no coincide con ninguno.
`git status --short` tras el commit no muestra nada inesperado. Sí until: durante las
pruebas manuales del hook (ver abajo) se creó un `.calliope/config.json` de prueba **en
el propio repo** (efecto colateral de un `calliope config set org` lanzado desde ese cwd
sin `--global`); estaba correctamente ignorado por git (no apareció nunca en
`git status`), pero lo borré igualmente (`rm -rf .calliope`) para no dejar basura de
prueba en el árbol de trabajo.

## La decisión sobre el timeout de red del hook

El brief pide pensar si el timeout interno de 10s de `calliope doctor` (cuando hay
credencial configurada) es aceptable en un hook de arranque, y resolverlo si no lo es.
Investigando esto encontré dos problemas, no uno:

**1. `calliope doctor` nunca devuelve código de salida distinto de cero.** Está así por
diseño (`TestDoctorConConfigCorruptaInformaEnVezDeFallar` en `doctor_test.go`: "doctor
nunca debe fallar"). Confirmado en vivo: `calliope doctor --quiet` sin ninguna credencial
configurada sale con código `0` e imprime igualmente el JSON de chequeos (con
`"status": "error"` dentro). El script tal cual lo da el brief
(`if ! calliope doctor --quiet ...`) **nunca entraría en la rama de "no listo"**: con
`calliope` instalado pero sin credencial, habría impreso `calliope listo: ""` (probado en
vivo, ver Escenario 2 más abajo con el script literal antes de corregirlo) — exactamente
lo contrario de lo que el propio comentario del hook dice que tiene que hacer ("avisar si
no está listo"). Por eso `session-start.sh` no usa el código de salida como señal de
"listo": parsea el diagnóstico en modo `--quiet` buscando `"status": "error"`.

**2. El timeout de 10s en sí.** `checkConnectivity` en `doctor.go` solo se ejecuta cuando
hay credencial (`if errCred == nil`), así que los dos escenarios más frecuentes de un
hook de arranque — `calliope` no instalado, o instalado sin credencial — vuelven casi
instantáneos (medido: 0.03s y 0.5s respectivamente, ver abajo). El caso lento es
específicamente "credencial configurada + red lenta o caída", y ahí sí se nota un hook de
10s en cada arranque de sesión. Decisión: acotar la llamada a `calliope doctor` con
`timeout 5` / `gtimeout 5` (GNU coreutils) cuando estén disponibles en el `PATH`, y dejar
el límite interno de 10s del propio binario como red de seguridad cuando no lo estén (p.
ej. macOS de fábrica, que no trae `timeout`). Descarté implementar un timeout manual en
bash puro (con un proceso "vigía" en background que mata al principal) tras reproducir
una condición de carrera real de `bash` 3.2 (el que trae macOS de fábrica, sin Homebrew):
dentro de una subshell de sustitución de comandos `$(...)`, `wait $pid` no devuelve el
control en cuanto el proceso objetivo termina si hay un segundo job en background
corriendo a la vez (el vigía) — se queda bloqueado hasta que el vigía también termina,
inutilizando el propio timeout. Lo reproduje de forma aislada y consistente (ver
`/tmp/test5.sh` / `/tmp/test8.sh` durante la sesión, ya borrados); dado que ese bug es
exactamente el entorno más probable en la máquina de un usuario real, monté algo frágil
para resolver un problema de latencia habría sido peor que el problema. Resultado neto:
nunca cuelga indefinidamente (el límite interno de 10s del SDK lo garantiza en cualquier
caso), y cuando `timeout`/`gtimeout` están disponibles el peor caso baja a 5s.

## Prueba en los tres escenarios (salida real, con el hook ya definitivo)

**Escenario 1 — sin `calliope` en el PATH:**
```
$ PATH="/usr/bin:/bin" HOME=/tmp/no_home ./session-start.sh
calliope no está instalado. Instálalo con: brew install calliope/tap/calliope
exit code: 0
```

**Escenario 2 — `calliope` instalado, sin credencial:**
```
$ PATH="<bin con calliope>:/usr/bin:/bin" HOME=<vacío> ./session-start.sh
calliope está instalado pero no listo. Diagnostica con: calliope doctor
exit code: 0   (0.5s — checkConnectivity se salta sin credencial, ni siquiera toca la red)
```

**Escenario 3 — todo listo** (credencial válida en el almacén de fichero, org
configurada, backend apuntando a un servidor HTTP local de prueba que responde
`GET /v1/auth/me`, vía `CALLIOPE_BASE_URL`):
```
$ ./session-start.sh
calliope listo: miorg
exit code: 0   (0.038s)
```

Nota sobre cómo monté el Escenario 3: `calliope auth login` usa `go-keyring`
(Llavero de macOS) para guardar la credencial, y en este sandbox de ejecución
`keyring.Set` se queda colgado indefinidamente (probablemente esperando una
autorización de UI que nunca llega — sin relación con este hook, que nunca escribe en el
llavero). Lo esquivé escribiendo directamente el fichero de credenciales
(`~/.config/calliope/credentials.json`, el mismo formato que usa el almacén de
respaldo); `keyring.Get` sobre un ítem inexistente sí falla rápido y cae al fichero, así
que la lectura no tiene el mismo problema.

**Extra — verificación de que el timeout realmente corta:** monté un servidor "agujero
negro" (acepta la conexión TCP y nunca responde) y comprobé los dos caminos del hook:

```
# con `timeout` real en el PATH → corta a los 5s
calliope está instalado pero no listo. Diagnostica con: calliope doctor
   real 0m5.014s

# sin `timeout`/`gtimeout` en el PATH → cae al límite interno de 10s del SDK
calliope está instalado pero no listo. Diagnostica con: calliope doctor
   real 0m10.470s
```

Ambos casos: nunca cuelga, siempre sale `0`, siempre imprime una línea legible.

## Verificación por mutación (fuera del repo)

Copié el árbol completo (con el commit ya hecho) a `/tmp/calliope-mutacion` con `cp -R`
(preserva enlaces simbólicos). Confirmé la línea base en verde, y apliqué dos mutaciones,
una por una, revirtiendo cada una antes de la siguiente:

1. **Enlace simbólico sustituido por una copia divergente.** Borré
   `.claude-plugin/skills` y creé en su lugar un directorio real con una copia de
   `SKILL.md` más una línea extra añadida al final. Resultado:
   `TestElPluginUsaElMismoSkillQueElBinario` falla con "el SKILL.md del plugin y el del
   binario han divergido; el plugin debe ser un enlace simbólico". Repuesto el enlace
   simbólico, el test vuelve a pasar.

2. **`plugin.json` corrompido.** Primero JSON inválido
   (`{ "name": "calliope", "version": `) → `TestElManifiestoDelPluginEsValido` falla con
   "plugin.json no es JSON válido: unexpected end of JSON input". Después, JSON válido
   pero con `description` vacía → el mismo test falla con "plugin.json incompleto:
   {Name:calliope Version:0.1.0 Description:}".

Las dos mutaciones se revirtieron y `go test ./internal/cli/ -run Plugin -v` volvió a
pasar en la copia antes de borrarla (`rm -rf /tmp/calliope-mutacion`). El repo real no se
tocó durante el proceso.

## Traducción de identificadores (glosario)

Nada nuevo que decidir: el brief no introduce identificadores Go nuevos en esta tarea más
allá de los ya fijados en la Task 19 (`raizDelRepo`, ya usado en `paridad_test.go` con el
mismo nombre — se reutilizó tal cual, sin duplicar la función; de hecho `plugin_test.go`
declara su propia copia porque el brief la trae inline, ver duda más abajo). El resto del
contenido nuevo es Markdown/JSON/shell (comando, manifiesto, hook), fuera del alcance del
glosario de identificadores Go.

## Comprobaciones finales

`go test ./... -race` → PASS en los 10 paquetes; `go vet ./...` limpio; `gofmt -l .` sin
salida; `go mod tidy` sin cambios en `go.mod`/`go.sum`. `shellcheck` no está instalado en
esta máquina (no es requisito del brief); comprobé al menos la sintaxis con
`bash -n .claude-plugin/hooks/session-start.sh` → OK.

## Dudas / puntos para el controlador

1. **`raizDelRepo` duplicada.** El brief trae `raizDelRepo` inline dentro de
   `plugin_test.go`, idéntica a la que ya existe en `paridad_test.go` (mismo paquete
   `cli`). Como ambas están en el mismo paquete, esto es en realidad una declaración
   duplicada — compilé y pasó porque Go permite que una función se declare una sola vez
   por paquete... revisé esto con cuidado: **no compilaría si estuviera duplicada de
   verdad.** Lo que hice fue copiar el brief literalmente (incluida su copia de
   `raizDelRepo`) y `go build`/`go test` no protestó, así que confirmé que
   `paridad_test.go` no tiene ya una función con ese nombre exacto — no la tiene (tiene
   su propia lógica inline, no una función `raizDelRepo` extraída). No hay colisión real;
   lo apunto solo por si el controlador esperaba que reutilizara alguna función
   existente en vez de la copia literal del brief.
2. La reescritura de la lógica "¿está listo?" en el hook (parsear el diagnóstico en vez
   de mirar el código de salida) es una desviación real del texto literal del brief,
   justificada arriba con evidencia en vivo. Si el controlador prefiere que el hook siga
   el código de salida tal cual aunque hoy sea una señal inútil (por si `doctor` cambia
   de comportamiento en el futuro), es un cambio de una línea revertirlo — pero tal como
   está hoy el `doctor` de la Task 18, el script literal del brief no cumple su propio
   propósito declarado.

---

## Ronda de correcciones 1/5

### Estado

DONE

### Commit

`7256fcc` — fix: cierra huecos de cobertura del plugin y sustituye grep por jq en el hook
(3 ficheros: `internal/cli/plugin_test.go`, `.claude-plugin/hooks/session-start.sh`,
`.claude-plugin/README.md` [nuevo])

### Qué se corrigió

**CRITICAL — cobertura.** El revisor verificó por mutación que la suite original seguía
en verde tras (a) quitar el bit ejecutable al hook, (b) borrar `hooks.SessionStart` de
`plugin.json`, (c) borrar `commands/calliope.md` y (d) borrar `session-start.sh` entero.
Añadidos tres tests nuevos a `internal/cli/plugin_test.go`, cada uno responsable de una
de esas cuatro roturas:

- `TestElHookDeSesionEsEjecutable` — `os.Stat` sobre
  `.claude-plugin/hooks/session-start.sh`, comprueba que no es directorio y que
  `info.Mode()&0o111 != 0`. Cubre (a) y (d) (sin fichero, `os.Stat` falla igual).
- `TestElComandoCalliopeNoEstaVacio` — `os.ReadFile` sobre
  `.claude-plugin/commands/calliope.md`, comprueba que existe y que
  `strings.TrimSpace(...)` no es cadena vacía. Cubre (c).
- `TestElManifiestoDeclaraElHookDeSesion` — decodifica `plugin.json` en un struct que
  refleja `hooks.SessionStart[].hooks[].{type,command}`, comprueba que el array no está
  vacío, que `type == "command"` y que `command` contiene literalmente
  `${CLAUDE_PLUGIN_ROOT}/hooks/session-start.sh`. Cubre (b).

**IMPORTANT — grep frágil.** `session-start.sh` migrado de
`calliope doctor --quiet` + `grep -q '"status": "error"'` a
`calliope doctor --json --jq '[.data[]|select(.status=="error")]|length'`. La condición
de "no listo" ahora es `[ "$estado" -ne 0 ] || [ "$fallos" != "0" ]`: cualquier valor que
no sea exactamente `"0"` (incluida cadena vacía por timeout o fallo del proceso) se trata
como "no listo", así que un formato de salida distinto nunca puede hacer que el hook
mienta silenciosamente. De paso, el hook deja de depender de texto y pasa a depender de
`--jq`, que es jq embebido en el binario (`gojq`, sin proceso externo) — coherente con la
primera invariante del SKILL.md, que prohíbe el jq del sistema.

**IMPORTANT — symlink en Windows sin soporte.** Documentado en
`.claude-plugin/README.md` (nuevo fichero, junto al propio enlace simbólico, el sitio más
probable donde alguien lo encuentre al mirar por qué `.claude-plugin/skills` es un
fichero de texto en vez de un directorio en un clon de Windows sin Modo Desarrollador):
explica que la vía fiable en cualquier plataforma es `calliope skill`, porque `/calliope`
nunca lee `skills/` del plugin — invoca el CLI. No se tocó `commands/calliope.md`: ya
decía "ejecuta `calliope skill`" como primer paso, sin mencionar el directorio del
plugin en absoluto, así que no había nada que corregir ahí; el README es la única pieza
nueva de documentación.

### Verificación por mutación (fuera del repo, copia nueva)

`cp -R` del árbol completo (con los cambios de esta ronda ya aplicados, antes de
comitear) a `/tmp/calliope-mutacion2`. Confirmada la línea base en verde (los 5 tests de
`plugin_test.go`), y aplicadas las cuatro mutaciones del revisor, una por una,
revirtiendo cada una antes de la siguiente:

1. **`chmod 644 .claude-plugin/hooks/session-start.sh`** → `TestElHookDeSesionEsEjecutable`
   falla con "no tiene el bit ejecutable (modo -rw-r--r--)". El resto sigue en verde.
2. **`plugin.json` reescrito sin el bloque `hooks`** → `TestElManifiestoDeclaraElHookDeSesion`
   falla con "plugin.json no declara el hook SessionStart". El resto sigue en verde.
3. **`commands/calliope.md` movido fuera** → `TestElComandoCalliopeNoEstaVacio` falla con
   "commands/calliope.md debe existir: ... no such file or directory". El resto sigue en
   verde.
4. **`hooks/session-start.sh` movido fuera** → `TestElHookDeSesionEsEjecutable` falla con
   "el hook de sesión debe existir: stat ... no such file or directory". El resto sigue
   en verde.

Las cuatro mutaciones se revirtieron y la suite completa de `plugin_test.go` volvió a
pasar en la copia antes de borrarla (`rm -rf /tmp/calliope-mutacion2`). El repo real no
se tocó durante el proceso.

### Los cinco escenarios, reprobados con el hook basado en jq

Reconstruido el entorno de prueba (servidor HTTP local en `127.0.0.1:8934` para el
camino feliz, servidor "agujero negro" en `127.0.0.1:8935` que acepta la conexión y
nunca responde, credencial escrita directamente en el almacén de fichero para esquivar
el llavero de macOS — mismo motivo que en la ronda anterior, `keyring.Set` se cuelga en
este sandbox esperando una autorización de UI que nunca llega, sin relación con el
hook).

**1 — sin `calliope` en el PATH:**
```
calliope no está instalado. Instálalo con: brew install calliope/tap/calliope
exit: 0   (0.33s)
```

**2 — `calliope` instalado, sin credencial:**
```
calliope está instalado pero no listo. Diagnostica con: calliope doctor
exit: 0   (0.35s)
```

**3 — todo listo** (credencial + org + backend local que responde):
```
calliope listo: miorg
exit: 0   (0.033s)
```

**4 — credencial configurada + backend caído (agujero negro), CON `timeout`/`gtimeout`
en el PATH:**
```
calliope está instalado pero no listo. Diagnostica con: calliope doctor
exit: 0   (5.02s — cortado por el limitador)
```

**5 — credencial configurada + backend caído (agujero negro), SIN `timeout`/`gtimeout`
en el PATH:**
```
calliope está instalado pero no listo. Diagnostica con: calliope doctor
exit: 0   (10.38s — cae al límite interno de 10s del SDK)
```

Los cinco escenarios: nunca cuelga, nunca predica éxito sin confirmarlo, siempre sale
con `0`. Los tiempos y el comportamiento coinciden exactamente con la versión basada en
grep de la ronda anterior — el cambio a `--jq` es un cambio de robustez, no de
comportamiento observable.

### Comprobaciones finales

`go test ./... -race -v` → PASS en todos los paquetes; `go vet ./...` limpio; `gofmt -l .`
sin salida; `go mod tidy` sin cambios en `go.mod`/`go.sum`.

### Dudas

Ninguna.
