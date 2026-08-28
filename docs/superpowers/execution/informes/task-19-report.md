# Informe — Task 19: Skill embebido, catálogo y test de paridad

## Estado

DONE

## Commit

`0652546` — feat: skill embebido en el binario con test de paridad del catálogo
(7 ficheros: `skills/calliope/SKILL.md`, `skills/embed.go`, `internal/cli/catalog.go`,
`internal/cli/catalog_test.go`, `internal/cli/paridad_test.go`, `internal/commands/skill.go`,
`internal/cli/root.go`)

## Resumen de tests

`go test ./... -race` → PASS en los 10 paquetes (8 con tests); `go vet ./...` limpio;
`gofmt -l .` sin salida; `go mod tidy` sin cambios. Los tres tests nuevos (paridad ×2
sentidos + "ningún grupo define RunE") verificados por mutación dirigida sobre una copia
completa del árbol de trabajo fuera del repositorio (`rsync` a `scratchpad/mutate`,
excluyendo `.git` y `bin`), no sobre el repo real.

## Qué se hizo

Todo copiado literalmente del brief (`task-19-brief.md`), en el orden TDD indicado:

1. `skills/calliope/SKILL.md` — contenido literal del brief, en español, con el bloque
   `<!-- catalogo:inicio -->` / `<!-- catalogo:fin -->` con los 22 comandos.
2. Tests que fallan primero: `internal/cli/catalog_test.go` y `internal/cli/paridad_test.go`.
   Verificado con `go test ./internal/cli/ -v` → `undefined: Catalog` (el paquete `skills`
   ya existía porque lo había creado en el paso 1 del brief antes de correr esta
   verificación, así que el segundo error esperado del brief, "no required module
   provides package .../skills", no llegó a manifestarse — no afecta al resultado, la
   pieza real bajo prueba, `Catalog`, sí falló como se esperaba).
3. `skills/embed.go`, `internal/cli/catalog.go`, `internal/commands/skill.go` — literales
   del brief.
4. `internal/cli/root.go` — añadido `commands.NewSkillCmd()` al `root.AddCommand(...)`.
5. `go test ./internal/cli/ -v` → PASS a la primera (incluido el test de paridad, sin
   necesidad de ajustar el SKILL.md).
6. `make build && ./bin/calliope skill | head -20` → imprime la cabecera del SKILL.md
   embebido.
7. Commit.

## Traducción de identificadores (glosario)

Aplicadas exactamente las entradas ya fijadas en el glosario, sin necesidad de acuñar
ninguna nueva:

- `comandosDocumentados` → `documentedCommands` (función de nivel de paquete en
  `paridad_test.go`).
- `soloNombre` → `commandName` (función de nivel de paquete).
- `lineaDeComando` → `commandLineRe` (variable de nivel de paquete, la regexp).

El resto de identificadores del brief (`CommandInfo{Path, Short}`, `Catalog`,
`NewSkillCmd`, `SkillMD`) ya venían en inglés. Las variables locales dentro de cuerpos de
función (`recorrer`, `prefijo`, `hijo`, `ruta`, `enElCLI`, `enElSkill`, `faltan`,
`sobran`, `inicio`, `fin`, `linea`, `partes`) se dejaron tal cual las escribe el brief,
en español, conforme al alcance del glosario. Los nombres de test
(`TestCatalogDevuelveSoloLasHojas`, `TestNingunGrupoDeRecursosDefineRunE`,
`TestParidadEntreElCatalogoYElSkill`) se quedaron en español, sin cambios.

## Verificación por mutación (fuera del repo)

Copié el árbol de trabajo completo (incluidos los ficheros nuevos sin commitear) a
`scratchpad/mutate` con `rsync`, confirmé que la suite pasaba allí igual que en el repo
real, y apliqué tres mutaciones, una por una, revirtiendo cada una antes de la siguiente:

1. **Comando añadido al árbol, no documentado en el SKILL.md.** Registré un
   `&cobra.Command{Use: "ping", ...}` extra en `root.AddCommand(...)` de la copia.
   Resultado: `TestParidadEntreElCatalogoYElSkill` falla con
   `comandos del CLI sin documentar en SKILL.md: [ping]`. Confirma la rama "faltan".

2. **Comando fantasma en el SKILL.md que ya no existe en el árbol.** Añadí la línea
   `` - `calliope fantasma` — comando que ya no existe `` dentro del bloque de catálogo
   de la copia del SKILL.md. Resultado: el mismo test falla con
   `comandos documentados en SKILL.md que ya no existen: [fantasma]`. Confirma la rama
   "sobran".

3. **Grupo de recursos con `RunE`.** Añadí `RunE: func(cmd *cobra.Command, args []string)
   error { return nil }` al `&cobra.Command{Use: "orgs", ...}` de `orgs.go` en la copia
   (un grupo real, con subcomandos). Resultado: `TestNingunGrupoDeRecursosDefineRunE`
   falla con `"orgs" tiene subcomandos y además define RunE; invocarlo pelado debe
   mostrar la ayuda`.

Las tres mutaciones se revirtieron y `go test ./... -race` volvió a pasar en la copia
antes de borrarla. El repo real (`/Users/j10/repositories/calliope/calliope-cli`) no se
tocó durante este proceso — se verificó con `git status --porcelain` antes y después.

## Dudas / puntos para el controlador

Ninguna. El test de paridad pasó a la primera con el SKILL.md copiado literalmente del
brief, sin necesidad de ajustarlo; la lista de 21 comandos hoja existentes + `skill`
coincidió exactamente con la que dio el controlador en el encargo.

---

## Ronda de correcciones 1/5

### Estado

DONE

### Commit

`ce00dc5` — fix: paridad compara también descripción y forma de los argumentos, no solo
la ruta del comando
(3 ficheros: `internal/cli/catalog.go`, `internal/cli/paridad_test.go`, `skills/calliope/SKILL.md`)

### Qué se corrigió

**IMPORTANT 1 — la paridad ignoraba las descripciones.**

`CommandInfo` (en `internal/cli/catalog.go`) gana un campo `Args string` (tag json
`args`), poblado desde `hijo.Use` recortando el prefijo `hijo.Name()` — Cobra siempre
construye `Name()` como el primer token de `Use`, así que el resto es exactamente la
forma de los argumentos ("" si el comando no toma ninguno).

`internal/cli/paridad_test.go` se reescribió para:
- Indexar `enElCLI` por `CommandInfo` completo (antes solo guardaba un `bool`), para
  tener `Short` y `Args` disponibles en la comparación.
- `documentedCommands` ahora devuelve `map[string]documentedCommand`
  (`documentedCommand{Args, Description string}`), extraído con un regex ampliado
  (`commandLineRe = ^- \`calliope ([^\`]+)\` — (.+)$`) que captura también el texto tras
  el em dash.
- Nueva función `argsUsageOf(s string) string`, complementaria de `commandName`: donde
  `commandName` se queda con los tokens antes del primero que empieza por `<`, `[` o
  `--`, `argsUsageOf` se queda con ese token en adelante. Entre las dos reparten `s` sin
  solapamiento ni hueco.
- Nueva función `normalizeDescription(s string) string` (`strings.ToLower` +
  `strings.TrimSpace`), usada solo para comparar descripciones — la comparación de
  `Args` es con `strings.TrimSpace` a secas, sin normalizar mayúsculas (los placeholders
  no tienen ambigüedad de caso).
- El test añade dos bloques de comparación nuevos, solo sobre los comandos presentes en
  ambos lados (los que faltan/sobran ya se reportan en el bloque existente, y no tienen
  con qué compararse): uno para `Short` vs `Description` normalizados, otro para `Args`
  (Use real) vs `Args` (SKILL.md) tal cual.

Al aplicar esto contra el SKILL.md existente, **15 de los 22 comandos** tenían una
descripción que no coincidía con su `Short` real más allá de mayúsculas/espacios — no
eran abreviaciones triviales, eran texto genuinamente distinto (p. ej. `config path`
decía "imprime la ruta de la configuración global" y el `Short` real dice "imprime la
ruta **del fichero de** configuración global"; `query` decía "ejecuta SQL contra los
datos" y el `Short` real añade "**de la organización**"). Ninguno de los `Short` en sí
parecía mal escrito o incorrecto — eran redacciones válidas, solo que el SKILL.md tenía
su propia versión más corta, escrita a mano en la Task 19 original. Apliqué la regla por
defecto del controlador: **ajusté el SKILL.md**, copiando literalmente cada `Short` real
(con la inicial en minúscula para mantener el estilo del bloque de catálogo; el resto de
la redacción, sin tocar) para las 15 líneas discrepantes. Las 7 que ya coincidían
(`auth logout`, `auth status`, `concepts list`, `concepts show`, `config get`,
`docs list`, `docs search`) se dejaron intactas.

Los argumentos (`Args`) de las 22 líneas ya coincidían con el `Use` real antes de tocar
nada — la comparación nueva no encontró ninguna discrepancia ahí, así que el SKILL.md no
necesitó cambios en esa dimensión.

**IMPORTANT 2 — la paridad era ciega a la forma de los argumentos.**

Implementado dentro del mismo cambio que IMPORTANT 1 (`Args` en `CommandInfo` +
`argsUsageOf` + la segunda comparación en el test), descrito arriba.

### Verificación por mutación (fuera del repo, copia nueva)

Repetí el proceso de la ronda anterior: `rsync` del árbol de trabajo completo (con los
cambios de esta ronda ya aplicados) a `scratchpad/mutate2`, confirmé la línea base en
verde, y apliqué cuatro mutaciones una por una, revirtiendo cada una antes de la
siguiente:

1. **`Short` real cambiado por un texto sin ninguna relación** (`orgs list` →
   "Convierte megabytes en pulgadas de arcoíris"). `TestParidadEntreElCatalogoYElSkill`
   falla con `orgs list: Short="Convierte megabytes en pulgadas de arcoíris"
   SKILL.md="lista las organizaciones accesibles con tu credencial"`. Confirma que
   IMPORTANT 1 está atrapado.

2. **Contraprueba de que la normalización no da falsos positivos**: mismo `Short`,
   pero en mayúsculas y con espacios extra al principio/final
   (`"  LISTA LAS ORGANIZACIONES ACCESIBLES CON TU CREDENCIAL  "`). El test sigue en
   verde, como debe: la normalización tiene que ignorar exactamente eso y nada más.

3. **Argumento añadido a un `Use` real** (`config set <clave> <valor>` →
   `config set <clave> <valor> <capa>`). El test falla con
   `config set: Use="<clave> <valor> <capa>" SKILL.md="<clave> <valor>"`.

4. **Argumento quitado de un `Use` real** (`docs show <id>` → `docs show`). El test
   falla con `docs show: Use="" SKILL.md="<id>"`.

Las cuatro mutaciones se revirtieron y `go test ./... -race` volvió a pasar en la copia
antes de borrarla (`rm -rf scratchpad/mutate2`). El repo real no se tocó durante el
proceso — verificado con `git status --porcelain` antes y después, mostrando solo los
tres ficheros que sí edité a propósito (`catalog.go`, `paridad_test.go`, `SKILL.md`).

### Comprobaciones finales

`go test ./... -race -v` → PASS en todos los paquetes, incluidos los tres tests de
`internal/cli` (`TestCatalogDevuelveSoloLasHojas`,
`TestNingunGrupoDeRecursosDefineRunE`, `TestParidadEntreElCatalogoYElSkill`); `go vet
./...` limpio; `gofmt -l .` sin salida; `go mod tidy` sin cambios en `go.mod`/`go.sum`.
`./bin/calliope skill` comparado con `cmp` contra `skills/calliope/SKILL.md`: idéntico
byte a byte.

### Identificadores nuevos (no estaban en el glosario)

Añadidos con criterio obvio, para que el controlador los incorpore si procede:

| Español (si aplica) | Usado |
|---|---|
| — | `documentedCommand` (tipo, antes solo existía como `bool` en el mapa) |
| `formaDeArgumentos` | `argsUsageOf` |
| — | `normalizeDescription` |
| campo `Args` de `CommandInfo` | `Args`, tag json `args` |

`commandName` se conservó sin cambios (glosario: `soloNombre` → `commandName`); no fue
necesario tocarlo porque `argsUsageOf` es una función independiente con la misma
estructura de recorrido, no un envoltorio sobre `commandName`.

### Alcance no tocado (a propósito)

Los tres puntos de deuda menor que el controlador marcó explícitamente como "no toques"
siguen igual: `TestNingunGrupoDeRecursosDefineRunE` solo recorre el primer nivel de
`root.Commands()`, `Catalog` sigue usando `sort.Slice` (no estable), y no hay validación
del frontmatter YAML del SKILL.md.

### Dudas

Ninguna. Las 15 correcciones de texto en el SKILL.md son mecánicas (copiar el `Short`
real palabra por palabra, solo con la inicial en minúscula); no encontré ningún `Short`
que pareciera mal escrito o incorrecto en sí mismo, así que no toqué ningún fichero de
`internal/commands`.
