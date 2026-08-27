# Calliope CLI — diseño v1

**Fecha:** 2026-08-27
**Estado:** aprobado, pendiente de plan de implementación

## 1. Resumen

`calliope` es la interfaz de línea de comandos de **Calliope Data**. Un binario Go
único, sin runtime, que da acceso a los datos y al conocimiento de una organización
desde el terminal y, sobre todo, desde agentes de IA.

El CLI es el sustrato: expone una superficie estable y auto-documentada
(`calliope skill`) sobre la que se escriben skills de dominio en Markdown, sin
tocar Go.

Referencia estructural: [basecamp-cli](https://github.com/basecamp/basecamp-cli).

## 2. Contexto

### 2.1 El backend objetivo

**Calliope Data** (`https://data-0.calliope.so`) es un backend distinto de
`calliope-py-api` (la plataforma de analítica de marketing de `calliope.so`). Su
consumidor actual es `calliope-data-ui` (Nuxt + PropelAuth), y expone ~130
endpoints bajo `/v1/organizations/{org}/…`.

El CLI apunta **solo** a Calliope Data. `calliope-py-api` queda fuera de alcance.

### 2.2 Lo que ya existe

`calliope-data-mcp` (TypeScript) expone 7 tools read-only sobre este mismo
backend, con tres modos de auth entrante y un `CalliopeClient` que es medio SDK.

**Decisión: CLI y MCP conviven como implementaciones independientes.** El MCP
sigue sirviendo a clientes cloud que no pueden ejecutar binarios (ElevenLabs,
Claude.ai); el CLI sirve al terminal y a los agentes locales. Se acepta
conscientemente la duplicación del cliente HTTP a cambio de no acoplar dos
proyectos con ciclos de vida distintos. El `CalliopeClient` del MCP es la
referencia de comportamiento para el SDK Go (rutas, mapeo de errores, timeouts).

### 2.3 Autenticación disponible hoy

Verificado en el código, no supuesto:

- `Authorization: Bearer <token PropelAuth>` — lo usa la UI (`useApi.ts:292`).
- `X-API-Key: <api key>` — lo usa el MCP (`src/calliope/client.ts`).
- `/v1/auth/api-keys` permite crear y listar claves por API.

Ambas vías existen en el backend. No hace falta ningún cambio de servidor para la
v1 con API key.

## 3. Decisiones y su porqué

| Decisión | Razón |
|---|---|
| **Go + Cobra** | La audiencia incluye clientes finales, así que la distribución es requisito de producto: binario estático, instalación de una línea, GoReleaser da brew/curl/deb/rpm/apk/scoop. Y el arranque de ~5 ms importa porque un agente invoca el CLI decenas de veces por sesión. Coste asumido: lenguaje nuevo en un stack Python + TS. |
| **Alcance v1 = núcleo Q&A + conocimiento** | Cubrir los ~130 endpoints en una v1 no es un spec, son cinco. El núcleo es lo que los agentes usan de verdad. |
| **Incluir `schema` y `query`** | Un CLI que corre en la máquina del cliente, con su identidad, no es el mismo contexto de riesgo que un agente cloud de terceros. La postura no-SQL del MCP se conserva como norma de agente en el SKILL.md, no como límite técnico. |
| **CLI y MCP independientes** | Ver 2.2. |
| **API key primero, OAuth después de un spike** | El único camino garantizado hoy. Ver riesgo R1. |

## 4. Alcance

### 4.1 Dentro de la v1

Q&A en lenguaje natural, documentación, ontología, reglas de negocio, esquema de
base de datos, SQL crudo, autenticación, organizaciones, configuración,
diagnóstico, y la integración con agentes (skill embebido + plugin de Claude Code).

### 4.2 Fuera de la v1

TUI y pickers interactivos · `url parse` · streaming de `ask` (`/ask/stream`) ·
conversations · data-sources y connections · members · llm-usage ·
questions-discovery · modo `mcp serve` · comandos de escritura sobre documentos u
ontología.

Cada uno entra cuando haya una necesidad concreta, no antes.

## 5. Superficie de comandos

```
calliope ask "<pregunta>" [--action <acción>]
calliope docs list [--q] [--tag] [--status] [--page] [--size]
calliope docs show <id>
calliope docs search "<consulta>" [--limit]
calliope concepts list
calliope concepts show <id>
calliope rules list
calliope schema [--table <nombre>]
calliope query "<SQL>" [--output <formato>] [--csv]
calliope orgs list
calliope orgs use <org>
calliope auth login [--api-key] | logout | status | token
calliope config get <clave> | set <clave> <valor> | list | path
calliope doctor
calliope skill
calliope version
```

**`--output` frente a `--csv`.** Son cosas distintas y conviene no confundirlas:
`--output <formato>` se reenvía al backend en `QueryRequest.output`; `--csv` es
render local del conjunto de resultados a CSV, y compone con los modos de la
tabla 6.2 como cualquier otro comando.

**Grupos vs atajos.** Los grupos de recursos (`docs`, `concepts`, `rules`,
`orgs`, `auth`, `config`) no definen `RunE`: invocarlos pelados muestra la ayuda,
que es lo que Cobra hace solo cuando un comando tiene subcomandos y no `RunE`. Los
atajos que ejecutan una acción directa (`ask`, `query`, `schema`, `doctor`) sí lo
definen. Un test de CI comprueba esta invariante.

### 5.1 Mapeo a endpoints

Base de organización: `{base_url}/v1/organizations/{org}`.

| Comando | Método y ruta | Cuerpo / query |
|---|---|---|
| `ask` | `POST /ask` | `{question, action?}` |
| `docs search` | `POST /search/documents` | `{query, limit}` |
| `docs list` | `GET /documents` | `status, tag, q, page, size` |
| `docs show` | `GET /documents/{id}` | — |
| `concepts list` | `GET /knowledge/concepts` | — |
| `concepts show` | `GET /knowledge/concepts/{id}` | — |
| `rules list` | `GET /rules` | — |
| `schema` | `GET /database/schema` | filtrado de `--table` en cliente |
| `query` | `POST /query` | `{sql, output?}` |
| `orgs list` | `GET /v1/organizations` | fuera del scope de org |
| `auth status` | `GET /v1/auth/me` | fuera del scope de org |

`POST /query_with_metadata` existe y devuelve además metadatos de la consulta; se
deja para v2 tras `query --explain`.

## 6. Contrato de salida

### 6.1 Envelope

```json
{
  "ok": true,
  "data": [],
  "summary": "12 documentos",
  "breadcrumbs": [{"action": "show", "cmd": "calliope docs show <id>"}]
}
```

Los **breadcrumbs** son la pieza central del diseño para agentes: la respuesta
enseña el siguiente comando, así que el agente navega sin recargar documentación
en su contexto.

### 6.2 Modos

| Modo | Cuándo | Qué produce |
|---|---|---|
| (por defecto, TTY) | humano en terminal | salida estilizada |
| (por defecto, tubería) | `\| otro-proceso` | JSON con envelope |
| `--json` | JSON explícito | envelope completo |
| `--quiet` | scripting | solo `data`, sin envelope |
| `--md` | presentar a un humano | Markdown |
| `--jq '<expr>'` | extraer un campo | resultado del filtro |

`--jq` va integrado en el binario. El SKILL.md prohíbe el pipe a `jq` externo,
que falla en máquinas donde no está instalado.

### 6.3 Errores y exit codes

```json
{"ok": false, "error": {"code": "UNAUTHORIZED", "message": "…", "hint": "…"}}
```

El campo `hint` es obligatorio siempre que exista una acción de recuperación: es
lo que convierte un fallo en un siguiente paso para el agente.

| Código de salida | Significado |
|---|---|
| 0 | correcto |
| 1 | error genérico |
| 2 | uso incorrecto |
| 3 | no autorizado |
| 4 | no encontrado |
| 5 | límite de peticiones superado |

Los mensajes de error no filtran el cuerpo de la respuesta del backend, siguiendo
el criterio de `mapError` del MCP.

## 7. Autenticación

Dos credenciales tras una única interfaz `CredentialSource`, para que añadir modos
no toque el código de los comandos:

1. **API key** (`X-API-Key`). Vía garantizada. Se toma de `CALLIOPE_API_KEY` o se
   guarda con `calliope auth login --api-key`.
2. **OAuth PropelAuth** (`Authorization: Bearer`). `calliope auth login` abre el
   navegador: authorization code + PKCE, redirect a `http://127.0.0.1:<puerto>/callback`,
   refresh automático. Requiere registrar el redirect de loopback en el dashboard
   de PropelAuth (configuración, no cambio de código).

**Almacenamiento:** keyring del sistema operativo (Keychain, libsecret, Credential
Manager) con *fallback* a fichero `0600` bajo `~/.config/calliope/`. Las
credenciales nunca se escriben en un fichero de configuración de proyecto.

`calliope auth login` valida la credencial con `GET /v1/auth/me` **antes** de
guardarla. Nunca se persiste una credencial no verificada.

## 8. Configuración y contexto

### 8.1 Capas

Precedencia, de mayor a menor:

1. Flags de la invocación
2. Variables de entorno (`CALLIOPE_*`)
3. `.calliope/config.json` del directorio actual
4. `.calliope/config.json` de la raíz del repositorio
5. `~/.config/calliope/config.json`
6. Valores por defecto

Cada valor resuelto recuerda su origen, y `calliope config list` lo muestra.
Cuando un agente se pregunta por qué el CLI apunta a una organización concreta, la
respuesta está en un comando en vez de en una suposición.

### 8.2 Frontera de confianza

Un `.calliope/config.json` llega dentro de un repositorio clonado. Si pudiera
fijar `base_url`, clonar un repositorio hostil y ejecutar `calliope ask` enviaría
el token del usuario a una máquina ajena.

Por tanto los ficheros de configuración **de proyecto** (capas 3 y 4) solo pueden
fijar campos inocuos: `org`, formato de salida por defecto. Los campos
`base_url`, cualquier credencial y los timeouts se **ignoran** en esas capas, con
un aviso visible en stderr. Solo la configuración global y las variables de
entorno pueden fijarlos.

Esta regla se cubre con un test explícito de "repositorio hostil".

### 8.3 Contexto de organización

`calliope orgs use acme` escribe `{"org": "acme"}` en `.calliope/config.json` del
directorio actual. `--org` lo sobrescribe para una invocación.

Si la credencial está limitada a una sola organización, se resuelve sola. Si hay
varias y no hay contexto, el error lista las organizaciones disponibles en `hint`.

## 9. SDK

`internal/sdk` es el cliente HTTP de Calliope Data: el equivalente Go de
`src/calliope/client.ts`. Responsabilidades: construir la URL con el scope de
organización, aplicar la cabecera de autenticación según la credencial, imponer el
timeout, y traducir el status HTTP a un error de dominio tipado.

No conoce Cobra, ni flags, ni formatos de salida. Se prueba de forma aislada
contra un `httptest.Server`.

## 10. Integración con agentes

### 10.1 Skill embebido

`skills/calliope/SKILL.md` se embebe en el binario con `go:embed`, y
`calliope skill` lo vuelca. **El skill no puede desincronizarse de la versión
instalada**: si el agente tiene el binario, tiene su documentación exacta.

El SKILL.md se escribe como invariantes, no como tutorial:

1. Elegir modo de salida: `--jq` para extraer, `--json` para todo, `--md` para
   presentar a un humano. Nunca pipe a `jq` externo.
2. **`ask` antes que `query`.** La pregunta en lenguaje natural es la vía por
   defecto; el SQL crudo solo cuando `ask` no basta.
3. Ejecutar `calliope schema` antes de escribir SQL. Nunca inventar nombres de
   tabla ni de columna.
4. El scope de organización es obligatorio: `--org` o `.calliope/config.json`.
5. `ask` devuelve `sources`; al presentar el resultado a un humano, citarlas.
6. Seguir los `breadcrumbs` de la respuesta anterior en vez de adivinar el
   siguiente comando.

Sobre esta base, las skills de dominio (informes recurrentes, auditoría de reglas)
se escriben en Markdown componiendo comandos, sin tocar Go.

### 10.2 Plugin de Claude Code

`.claude-plugin/` en el mismo repositorio: `plugin.json`, el skill, un comando
`/calliope`, y un hook `session-start` que comprueba binario y autenticación y, si
falta algo, imprime la orden exacta para resolverlo.

### 10.3 `calliope doctor`

Diagnóstico unificado: versión, credencial y su origen, organización activa,
conectividad con el backend y latencia. Lo usan por igual el hook de sesión, el
soporte a clientes y el agente cuando algo falla.

## 11. Estructura del repositorio

```
cmd/calliope/main.go
internal/
  cli/         root, flags globales, wiring
  commands/    un fichero por grupo de comandos
  sdk/         cliente HTTP de Calliope Data
  models/      tipos de respuesta
  output/      envelope, errores, exit codes
  presenter/   render TTY / md / json / jq
  config/      capas + frontera de confianza
  auth/        keyring, loopback OAuth, HTML de callback
  appctx/      contexto de ejecución (org, config, cliente)
skills/calliope/SKILL.md
skills/embed.go
.claude-plugin/
docs/superpowers/specs/
```

Un fichero por grupo de comandos. Ficheros pequeños y enfocados: es donde tanto
las personas como los agentes editan con menos errores.

## 12. Testing

Desarrollo dirigido por tests: el test se escribe antes que la implementación.

- **Unitario.** Precedencia de configuración; frontera de confianza, con un caso
  explícito de repositorio hostil que intenta fijar `base_url` y debe ser
  ignorado; envelope y exit codes; filtrado `--jq`; presenters con *golden files*.
- **SDK.** Contra `httptest.Server` con fixtures grabadas de respuestas reales de
  `data-0`, incluyendo los caminos de error 401, 404, 429 y timeout.
- **Paridad de catálogo.** Todo comando registrado debe aparecer en el catálogo
  de comandos y en el SKILL.md, y viceversa. Este test es la única defensa real
  contra que el skill mienta sobre el CLI.
- **Grupos pelados.** Ningún grupo de recursos define `RunE` (ver 5).
- **E2E.** Smoke del binario real contra una organización de prueba, opt-in por
  variable de entorno, fuera del CI por defecto.

## 13. Distribución

GoReleaser disparado por tag: binarios para macOS, Linux y Windows (amd64 y
arm64), tap de Homebrew, script de instalación `curl -fsSL … | bash`, paquetes
deb/rpm/apk, manifiesto de Scoop y `checksums.txt`. `go install` para el equipo.

`calliope version` avisa cuando hay una versión más reciente.

## 14. Riesgos

**R1 — El token OAuth de PropelAuth puede no ser aceptado por el backend.** El
README de `calliope-data-mcp` documenta que el token que PropelAuth emite por su
flujo OAuth es opaco y que `data-0` tendría que aceptarlo por introspección. La UI
funciona porque usa el SDK de navegador y obtiene un JWT, que no es
necesariamente lo mismo.

*Mitigación:* el plan arranca con un spike de medio día que ejecuta el flujo real
contra PropelAuth y llama a `GET /v1/auth/me`. Si devuelve 401, la v1 sale solo
con API key —que cubre todo el alcance pedido— y OAuth pasa a v2 con el requisito
de backend documentado. La interfaz `CredentialSource` hace que ese aplazamiento
no cueste reescribir ningún comando.

**R2 — Deriva entre el CLI y el MCP.** Al ser implementaciones independientes,
las rutas y los tipos pueden divergir. *Mitigación:* las fixtures del SDK se
graban de respuestas reales; un cambio de contrato rompe los tests del CLI aunque
el MCP no se toque.

**R3 — Go es nuevo en el stack de Calliope.** *Mitigación:* la superficie es
pequeña y muy testeable, y `basecamp-cli` sirve de referencia estructural
concreta. El riesgo real no es escribirlo sino mantenerlo; se acota manteniendo
la v1 estrecha.

**R4 — Exponer `query` amplía la superficie frente a la postura del MCP.** El SQL
crudo se ejecuta con la identidad del usuario y el backend aplica sus propias
ACL, así que el CLI no eleva privilegios. La contención es de comportamiento de
agente: la invariante 2 del SKILL.md establece `ask` como vía por defecto.

## 15. Criterios de aceptación de la v1

1. `calliope ask "…"` devuelve respuesta y fuentes contra una organización real.
2. Los 22 comandos hoja de la sección 5 existen, con ayuda y completions.
3. Los seis modos de salida de la tabla 6.2 producen el formato documentado, y los códigos de
   salida coinciden con la tabla de 6.3.
4. Un repositorio con `.calliope/config.json` hostil no consigue cambiar
   `base_url`, y el test lo demuestra.
5. `calliope skill` vuelca un SKILL.md que el test de paridad valida contra los
   comandos registrados.
6. El plugin se instala en Claude Code y su hook de sesión detecta la ausencia de
   binario o de credencial.
7. GoReleaser produce binarios para las seis combinaciones de plataforma y
   arquitectura, con checksums.
8. Un agente completa, solo con `calliope skill` en contexto, esta cadena:
   descubrir organizaciones, fijar una, consultar el esquema, hacer una pregunta y
   citar las fuentes.
