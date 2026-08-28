# calliope

CLI de [Calliope Data](https://data-0.calliope.so): pregunta en lenguaje
natural sobre tus datos, consulta la documentación, la ontología y las reglas
de negocio de tu organización, desde el terminal y desde agentes de IA.

## Instalación

    brew install calliope/tap/calliope

    # o bien
    curl -fsSL https://raw.githubusercontent.com/calliope/calliope-cli/main/install.sh | bash

## Primeros pasos

    calliope auth login --api-key <clave>   # créala en el UI, en Observabilidad → Claves API
    calliope orgs use <organización>
    calliope ask "¿cómo van las ventas este trimestre?"

`calliope doctor` diagnostica instalación, credencial y conectividad.

Hoy la autenticación es solo por clave de API (`auth login --api-key` o la
variable `CALLIOPE_API_KEY`). No hay un flujo de login OAuth interactivo: ver
la sección de estado del proyecto más abajo.

## Comandos

El CLI tiene 22 comandos, agrupados en `auth`, `orgs`, `config`, `docs`,
`concepts` y `rules`, más los sueltos `ask`, `query`, `schema`, `doctor`,
`skill` y `version`. `calliope <grupo> --help` lista los de cada grupo; la
lista completa, con una línea de descripción por comando, vive en el skill
embebido (ver más abajo) y se mantiene sincronizada con el binario porque se
deriva del árbol de comandos real, no se escribe a mano.

## Uso con agentes

El skill va **embebido en el binario**, así que nunca se desincroniza de la
versión instalada:

    calliope skill

Ese comando vuelca el `SKILL.md` completo -invariantes, catálogo de comandos
y recetas- tal como lo empaquetó la versión instalada.

En Claude Code, instala el plugin de este repositorio y el agente queda
equipado: obtiene el skill, un comando `/calliope` y un aviso al empezar la
sesión si falta el binario o la credencial.

Sobre esa base, las skills de dominio se escriben en Markdown componiendo
comandos, sin tocar Go.

## Salida

Toda salida usa un envelope común, y los `breadcrumbs` indican el siguiente
comando:

    {"ok": true, "data": [...], "summary": "12 documentos",
     "breadcrumbs": [{"action": "show", "cmd": "calliope docs show <id>"}]}

Modos: humano en terminal · JSON en tubería · `--json` · `--quiet` · `--md` ·
`--jq '<expr>'` (el filtro va dentro del binario; no hace falta `jq`).

`calliope query` admite además `--csv` para volcar el resultado en CSV. Es un
modo de render distinto a los anteriores, no una variante de ellos: `--csv`
no se puede combinar con `--json`, `--quiet`, `--md` ni `--jq` (da un error de
uso, código de salida 2, antes de tocar la red).

Códigos de salida: `0` correcto · `1` error · `2` uso incorrecto ·
`3` no autorizado · `4` no encontrado · `5` límite superado. Es el mismo
contrato que documenta `skills/calliope/SKILL.md`.

## Configuración

Por capas, de mayor a menor precedencia: flags · variables `CALLIOPE_*` ·
`.calliope/config.json` del directorio · el de la raíz del repositorio ·
`~/.config/calliope/config.json` · valores por defecto.

`calliope config list` muestra cada valor **con la capa de la que proviene**.

Por seguridad, la configuración de proyecto (la del directorio actual y la de
la raíz del repositorio) solo puede fijar `org` y `output`; cualquier otra
clave que traiga se ignora, con un aviso en stderr. La razón: un
`.calliope/config.json` viaja dentro de cualquier repositorio clonado, así que
si pudiera fijar `base_url`, clonar un repositorio hostil y ejecutar
`calliope ask` enviaría la credencial del usuario a un backend controlado por
un tercero. `base_url` y `timeout` solo se pueden fijar en la configuración
global o por variable de entorno, nunca desde un repositorio.

## Desarrollo

Requiere Go 1.25 o superior.

    make test     # go test ./... -race
    make build    # bin/calliope
    make snapshot # binarios de todas las plataformas con GoReleaser

Los tests de extremo a extremo van aparte y necesitan credenciales reales:

    CALLIOPE_E2E=1 CALLIOPE_API_KEY=... CALLIOPE_ORG=... go test -tags=e2e ./test/e2e/

Sin esas variables, esos tests se saltan; y por la etiqueta de compilación
`//go:build e2e`, `go test ./...` ni siquiera los compila, así que el CI
normal no depende de ninguna organización real.

## Estado del proyecto

Con honestidad, esto es lo que falta:

- **Login OAuth no implementado.** Solo hay autenticación por clave de API
  (`auth login --api-key`, o la variable `CALLIOPE_API_KEY`). El tipo de
  credencial `oauth` existe en el código (aceptado vía la variable
  `CALLIOPE_TOKEN`, para inyectar un token ya obtenido por otra vía) pero no
  hay un comando que lleve a un usuario a través de un flujo de login
  interactivo.
- **Tres grupos de tipos nunca se han confirmado contra el backend real.**
  `Me` (lo que usa `auth status`), `Organization` (lo que usa `orgs list`) y
  `SchemaResponse`/`SchemaTable`/`SchemaColumn` (lo que usa `schema`), todos
  en `internal/sdk/models.go`, no salen de una respuesta real del backend ni
  del contrato verificado en `calliope-data-mcp`: se dedujeron leyendo los
  tipos TypeScript de `calliope-data-ui` (`BackendUser`, `Organization`,
  `SchemaResponse`/`SchemaTable`/`SchemaColumn`), que es el cliente que hoy
  consume esos mismos endpoints — una fuente razonable, pero no lo mismo que
  haberlos visto responder. Si el backend real usa otro nombre de campo para
  alguno de ellos, el síntoma **no es un error**: `encoding/json` deja en
  blanco el campo que no encuentra y el comando sigue devolviendo
  `"ok": true`, así que `orgs list`, `auth status` o `schema` saldrían con
  valores vacíos (nombres, IDs, columnas) sin ningún aviso. Quien vaya a
  scriptear contra esas tres salidas debería saberlo antes de confiar en
  ellas a ciegas. La forma de confirmarlos es ejecutar el smoke
  (`test/e2e/smoke_test.go`) con una API key real: `TestCadenaCompletaDeAgente`
  ya ejercita `orgs list` (`Organization`) y `schema`
  (`SchemaResponse`/`SchemaTable`/`SchemaColumn`) contra el backend de
  verdad, y `doctor` ejercita `Me.Email` de paso, en su comprobación de
  conectividad. Ninguno de los dos tests llama a `auth status`, así que para
  confirmar el resto de campos de `Me` —empezando por `id`, el que `auth
  status` expone como `"userId"`— hace falta además correr
  `calliope auth status --json` a mano contra una organización real.
- **La configuración de GoReleaser no se ha verificado.** `.goreleaser.yaml`
  no se ha comprobado con `goreleaser check` ni con `goreleaser build
  --snapshot` porque la herramienta no estaba disponible al escribir esto.
  Antes de depender del release automático conviene ejecutar ambos; además,
  antes del primer release hay que crear el secreto `TAP_GITHUB_TOKEN` en
  este repositorio (`.goreleaser.yaml` lo exige para `brews[].repository` y
  `scoops[].repository`: el `GITHUB_TOKEN` por defecto del workflow no tiene
  permiso de escritura sobre `calliope/homebrew-tap` ni
  `calliope/scoop-bucket`), o la publicación en el tap de Homebrew y en el
  bucket de Scoop fallará aunque el resto del release funcione.
- **El smoke de extremo a extremo nunca se ha ejecutado contra un backend
  real.** `test/e2e/smoke_test.go` compila y sus tests se saltan
  correctamente sin credenciales, pero nadie lo ha corrido todavía con
  `CALLIOPE_E2E=1` y una organización real: no hay credenciales de Calliope
  disponibles en el entorno donde se escribió. Es la primera comprobación
  pendiente en cuanto existan, y de paso confirmaría los tipos `Me`,
  `Organization` y `SchemaResponse`/`SchemaTable`/`SchemaColumn` del bullet
  de más arriba.
