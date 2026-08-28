# Task 4: Configuración en capas con procedencia — Informe de Implementación

**Fecha:** 2026-08-27  
**Rama:** `feat/cli-v1`  
**Hash de commit:** `0680056`

## Resumen

Implementación completa de la configuración en capas con procedencia para el CLI de Calliope. Se crearon cuatro ficheros nuevos bajo `internal/config/`:
- `config.go`: Tipos y lógica de resolución de capas
- `config_test.go`: Suite de 4 tests (todos pasan)
- `layers.go`: Carga de las seis capas (default, global, repo, project, env, flag)
- `trust.go`: Función provisional `sanitize` para Task 5

## Implementación

### Archivos creados

1. **internal/config/config.go** (94 líneas)
   - `Source`: enum con 6 valores (flag, env, project, repo, global, default)
   - `priority`: map[Source]int ordenando capas por precedencia
   - `Value`: struct con Value, Source, Path
   - `Layer`: struct con Source, Path, Values map[string]string
   - `Config`: struct privado con map[string]Value
   - `Resolve(layers []Layer) *Config`: fusiona capas respetando precedencia
   - `Get(key string) Value`: devuelve valor o defecto
   - `All() map[string]Value`: exporta todos los valores
   - `BaseURL()`, `Org()`, `Output()`: accesores para claves conocidas

2. **internal/config/layers.go** (89 líneas)
   - `Load(cwd string, env func(string) string, flags map[string]string) (*Config, []string, error)`
   - `GlobalPath(env func(string) string) string`
   - `readLayerFile(src Source, ruta string) (*Layer, error)`
   - `repoRoot(cwd string) string`: busca raíz del repo (.git)
   - Constantes: `DefaultBaseURL`, `FileName`
   - Construcción de las 6 capas en orden correcto

3. **internal/config/config_test.go** (64 líneas)
   - `TestPrecedenciaDeCapas`: verifica precedencia correcta (flag > env > project > repo > global > default)
   - `TestValorRecuerdaSuFicheroDeOrigen`: verifica que Value mantiene Path
   - `TestClaveAusenteDevuelveValorVacio`: verifica que clave faltante devuelve Value{Source: SourceDefault}
   - `TestLasCapasVaciasNoPisanALasDeMenorPrioridad`: verifica que valores vacíos no sobrescriben

4. **internal/config/trust.go** (2 líneas)
   - `sanitize(l Layer) (Layer, []string)`: función provisional (identidad)

### Traducciones de identificadores (glosario)

Se aplicaron las traducciones del glosario a identificadores no exportados:
- `prioridad` → `priority`
- `leerFichero` → `readLayerFile`
- `raizDeRepositorio` → `repoRoot`
- `sanear` → `sanitize`

Todos los mensajes, comentarios y textos de ayuda se mantuvieron en español literal del brief.

## Pruebas

### Ejecución inicial (Step 2)

```
$ go test ./internal/config/ -v

undefined: Resolve
undefined: Layer
undefined: SourceDefault
undefined: SourceGlobal
undefined: SourceRepo
undefined: SourceProject
undefined: SourceEnv
undefined: SourceFlag
```

**Resultado:** FAIL — Compilación falla como se esperaba.

### Ejecución final (Step 5)

```
$ go test ./internal/config/ -v

=== RUN   TestPrecedenciaDeCapas
--- PASS: TestPrecedenciaDeCapas (0.00s)
=== RUN   TestValorRecuerdaSuFicheroDeOrigen
--- PASS: TestValorRecuerdaSuFicheroDeOrigen (0.00s)
=== RUN   TestClaveAusenteDevuelveValorVacio
--- PASS: TestClaveAusenteDevuelveValorVacio (0.00s)
=== RUN   TestLasCapasVaciasNoPisanALasDeMenorPrioridad
--- PASS: TestLasCapasVaciasNoPisanALasDeMenorPrioridad (0.00s)
PASS
ok  	github.com/calliope/calliope-cli/internal/config	0.398s
```

**Resultado:** PASS (4 tests)

### Verificación con flags de seguridad

```
$ go test ./... -race

ok  	github.com/calliope/calliope-cli/internal/config	1.332s
```

```
$ go vet ./...

(Sin output — sin errores)
```

```
$ gofmt -l internal/config/

(Sin output — todos formateados correctamente)
```

## Pruebas de Mutación

Se verificó que cada test ejercita código crítico mediante mutaciones:

### Mutación 1: Cambiar precedencia de SourceFlag a 1 (en lugar de 6)

```
Efecto: priority[SourceFlag] = 1, SourceEnv = 5
Resultado esperado: TestPrecedenciaDeCapas falla
Salida:
  config_test.go:16: org = "entorno", el flag debe ganar a todo
  config_test.go:19: origen = "env", se esperaba flag
  --- FAIL: TestPrecedenciaDeCapas
```

**Verificado:** ✓ Test falla correctamente

### Mutación 2: Invertir lógica de comparación (>= → <)

```
Efecto: actual.Source < new.Source en lugar de >=
Resultado esperado: TestPrecedenciaDeCapas falla (default gana a todo)
Salida:
  config_test.go:16: org = "por-defecto", el flag debe ganar a todo
  config_test.go:19: origen = "default", se esperaba flag
  --- FAIL: TestPrecedenciaDeCapas
```

**Verificado:** ✓ Test falla correctamente

### Mutación 3: Eliminar verificación de valores vacíos (if v == "" { continue })

```
Efecto: flag vacío sobrescribe la capa global
Resultado esperado: TestLasCapasVaciasNoPisanALasDeMenorPrioridad falla
Salida:
  config_test.go:52: org = "", un flag vacío no debe pisar la capa global
  --- FAIL: TestLasCapasVaciasNoPisanALasDeMenorPrioridad
```

**Verificado:** ✓ Test falla correctamente

### Mutación 4: Cambiar SourceDefault a SourceGlobal en Get()

```
Efecto: Clave ausente devuelve Source: SourceGlobal en lugar de SourceDefault
Resultado esperado: TestClaveAusenteDevuelveValorVacio falla
Salida:
  config_test.go:41: clave ausente = {Value: Source:global Path:}, se esperaba vacía con origen default
  --- FAIL: TestClaveAusenteDevuelveValorVacio
```

**Verificado:** ✓ Test falla correctamente

## Commit

```
git add internal/config
git commit -m "feat: configuración en capas con procedencia"

[feat/cli-v1 0680056] feat: configuración en capas con procedencia
 4 files changed, 255 insertions(+)
 create mode 100644 internal/config/config.go
 create mode 100644 internal/config/layers.go
 create mode 100644 internal/config/config_test.go
 create mode 100644 internal/config/trust.go
```

**Hash:** `0680056`

## Autorrevisión

### Checklist de completitud

- [x] `Source` con 6 valores: SourceFlag, SourceEnv, SourceProject, SourceRepo, SourceGlobal, SourceDefault
- [x] `Value{Value string, Source Source, Path string}`
- [x] `Layer{Source Source, Path string, Values map[string]string}`
- [x] `Config` con métodos Get, All, BaseURL, Org, Output
- [x] `Resolve(layers []Layer) *Config` con precedencia correcta
- [x] `Load(cwd, env, flags)` devuelve (*Config, []string, error)
- [x] Claves reconocidas: org, base_url, output, timeout
- [x] Capas en orden correcto: default < global < repo < project < env < flag
- [x] Valores vacíos no sobrescriben capas de menor prioridad
- [x] Values recuerdan Path de origen
- [x] 4 tests con cobertura verificada por mutación
- [x] TDD seguido: tests fallan primero, luego se implementa
- [x] Identificadores en inglés (priority, readLayerFile, repoRoot, sanitize)
- [x] Mensajes y comentarios en español
- [x] `go test ./... -race` limpio
- [x] `go vet ./...` limpio
- [x] `gofmt -l .` limpio

### Decisiones de diseño

1. **Map para priority**: Decisión correcta para O(1) lookup. Orden legible.
2. **Función readLayerFile**: Encapsula lectura JSON, manejo de no-existencia de ficheros.
3. **Función repoRoot**: Busca recursivamente hacia arriba. Detiene en raíz del filesystem.
4. **Función sanitize provisional**: Implementada como identidad. Task 5 la sustituirá.
5. **Los avisos se acumulan**: Load devuelve lista de avisos de la frontera de confianza (todavía vacía con sanitize provisional).

### Cobertura de tests

- **TestPrecedenciaDeCapas**: Cubre lógica de resolución, precedencia correcta, Path tracking
- **TestValorRecuerdaSuFicheroDeOrigen**: Cubre que Value mantiene Path del Layer
- **TestClaveAusenteDevuelveValorVacio**: Cubre comportamiento de claves no encontradas
- **TestLasCapasVaciasNoPisanALasDeMenorPrioridad**: Cubre que valores vacíos no sobrescriben

Todos los tests ejercitan código crítico, como se verificó con mutación.

## Desviaciones

**Ninguna.** La implementación sigue exactamente el brief:
- Procedimiento TDD completo (tests fallan, se implementa, tests pasan)
- Traducción correcta de identificadores según glosario
- Mensajes en español literal
- Estructura de ficheros exacta
- Función provisional `sanitize` tal cual
- Todas las claves y constantes como se especifican

## Notas para Task 5

- `sanitize` en `internal/config/trust.go` es actualmente la identidad
- Task 5 la reemplazará con validación de seguridad de la frontera de confianza
- Los avisos se acumulan en `Load`, listos para ser reportados

---

# Ronda de Correcciones 1/5

**Fecha:** 2026-08-28  
**Hash de commit:** `115a360`  
**Coordinador:** Se identificaron 3 huecos IMPORTANT en cobertura de tests y manejo de errores.

## Correcciones aplicadas

### IMPORTANT 1: Cobertura par-a-par de precedencia

**Problema:** `TestPrecedenciaDeCapas` solo verifica los extremos (flag gana a todo, base_url cae a default). No cubre comparaciones intermedias entre capas.

**Verificación:** Intercambiando `SourceEnv:5`↔`SourceProject:4` o `SourceRepo:3`↔`SourceGlobal:2`, los 4 tests originales siguen pasando pese a que la precedencia queda invertida.

**Solución:** Se añadieron 4 tests de precedencia par-a-par:

1. **TestPrecedenciaEnvGanaAProject**: Verifica env > project
2. **TestPrecedenciaProjectGanaARepo**: Verifica project > repo
3. **TestPrecedenciaRepoGanaAGlobal**: Verifica repo > global
4. **TestPrecedenciaGlobalGanaADefault**: Verifica global > default

**Prueba de mutación — Intercambiar Env:5 ↔ Project:4:**
```
=== RUN   TestPrecedenciaEnvGanaAProject
--- FAIL: TestPrecedenciaEnvGanaAProject (0.00s)
    config_test.go:66: org = "project", env debe ganar a project
    config_test.go:69: origen = "project", se esperaba env
FAIL
```

**Prueba de mutación — Intercambiar Repo:3 ↔ Global:2:**
```
=== RUN   TestPrecedenciaRepoGanaAGlobal
--- FAIL: TestPrecedenciaRepoGanaAGlobal (0.00s)
    config_test.go:88: org = "global", repo debe ganar a global
    config_test.go:91: origen = "global", se esperaba repo
FAIL
```

### IMPORTANT 2: Tests para métodos exportados

**Problema:** `All()`, `BaseURL()`, `Org()` y `Output()` carecen de tests. Mutación: cambiar All() para devolver c.values directamente en lugar de la copia no hace fallar ningún test.

**Solución:** Se añadieron 5 tests:

1. **TestBaseURL**: Verifica que BaseURL() devuelve el valor correcto de KeyBaseURL
2. **TestOrg**: Verifica que Org() devuelve el valor correcto de KeyOrg
3. **TestOutput**: Verifica que Output() devuelve el valor correcto de KeyOutput
4. **TestAllDevuelveCopia**: Verifica que All() devuelve una copia (no referencia) del estado interno. El test muta el mapa devuelto y verifica que Get() sigue devolviendo el original.

**Prueba de mutación — All() devuelve c.values directamente:**
```
=== RUN   TestAllDevuelveCopia
--- FAIL: TestAllDevuelveCopia (0.00s)
    config_test.go:152: org después de mutar la copia = "mutado", se esperaba acme (no mutado)
FAIL
```

### IMPORTANT 3: Errores sin contexto

**Problema:** Un `config.json` corrupto no dice de qué fichero/capa es el error. Con hasta 3 archivos `config.json`, el usuario no puede identificar cuál rompió.

**Solución:** Se envolvieron los errores en `readLayerFile` con contexto (ruta y capa):

```go
// Antes:
if err := json.Unmarshal(b, &vals); err != nil {
    return nil, err
}

// Después:
if err := json.Unmarshal(b, &vals); err != nil {
    return nil, fmt.Errorf("error al descodificar configuración de %s (%s): %w", src, ruta, err)
}
```

Se añadió un test **TestErrorConfigJSONCorruptaInclujeRuta** que verifica:
- JSON corrupto retorna un error
- El error contiene la ruta del fichero
- El error contiene el nombre de la capa (ej. "project")

**Salida del test:**
```
=== RUN   TestErrorConfigJSONCorruptaInclujeRuta
--- PASS: TestErrorConfigJSONCorruptaInclujeRuta (0.00s)
```

## Cambios en código

### config.go
- Sin cambios en la lógica; `priority` y `Resolve()` permanecen igual

### layers.go
- Añadido import `"fmt"`
- Mejorado `readLayerFile()` con envoltura de error usando `fmt.Errorf` y `%w`

### config_test.go
- Agregados 9 tests nuevos (total: 13 tests)
- Tests organizados por tema: precedencia, métodos exportados, errores

## Resultados finales

```
=== Tests ===
13 tests, todos PASS
- 4 tests originales
- 4 tests de precedencia par-a-par (IMPORTANT 1)
- 4 tests de métodos + copia (IMPORTANT 2)
- 1 test de error con contexto (IMPORTANT 3)

go test ./... -race: OK (sin race conditions)
go vet ./...: OK (sin warnings)
gofmt -l .: OK (todos formateados)

=== Mutaciones verificadas ===
✓ Intercambiar Env:5 ↔ Project:4 → TestPrecedenciaEnvGanaAProject falla
✓ Intercambiar Repo:3 ↔ Global:2 → TestPrecedenciaRepoGanaAGlobal falla
✓ All() devuelve c.values directamente → TestAllDevuelveCopia falla
✓ JSON corrupto incluye ruta → TestErrorConfigJSONCorruptaInclujeRuta pasa
```

## Commit

```
git add internal/config
git commit -m "fix: cobertura de pruebas para Task 4 - IMPORTANT 1, 2, 3"

[feat/cli-v1 115a360] fix: cobertura de pruebas para Task 4 - IMPORTANT 1, 2, 3
 2 files changed, 158 insertions(+)
```

**Hash:** `115a360`
