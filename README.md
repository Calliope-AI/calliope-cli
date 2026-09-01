# calliope

CLI de [Calliope Data](https://data-0.calliope.so): pregunta en lenguaje
natural sobre tus datos, consulta la documentación, la ontología y las reglas
de negocio de tu organización, desde el terminal y desde agentes de IA.

## Instalación

    brew install Calliope-AI/tap/calliope

    # o bien
    curl -fsSL https://raw.githubusercontent.com/Calliope-AI/calliope-cli/main/install.sh | bash

## Primeros pasos

    calliope auth login --api-key <clave>
    calliope orgs use <organización>
    calliope ask "¿cómo van las ventas este trimestre?"

La clave se crea en el UI, en **Configuración → Claves de API**, con rol Owner
o Admin. Pertenece a **una sola organización**: la que estuviera seleccionada
al crearla. Si `orgs use` apunta a otra, el CLI corta con «No autorizado» y
código de salida 3. Esa pantalla lista las claves de la organización entera, no
solo las tuyas, así que también se ve ahí la que creó un compañero, con su
último uso.

Al lado, en **Configuración → API**, está la referencia HTTP: qué llamada hay
debajo de cada comando, con un `curl` por endpoint y la forma de la respuesta.
Útil para saber qué se puede pedir, y para integrar desde donde no llegue este
CLI.

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

## Apuntar a otro backend

`calliope` habla por defecto con `https://data-0.calliope.so`. Para probar
contra un despliegue local o de staging, basta una variable de entorno:

    CALLIOPE_BASE_URL=http://localhost:8080 calliope doctor

`doctor` dice siempre contra qué backend está hablando y de dónde salió ese
valor, así que nunca hay duda:

    calliope doctor --json --jq '.data[]|select(.name=="backend")'

También se puede fijar de forma permanente en la configuración global:

    calliope config set base_url http://localhost:8080

Lo que **no** puede fijarlo es un `.calliope/config.json` de proyecto. Es
deliberado: ese fichero viaja dentro de cualquier repositorio que clones, y si
pudiera cambiar el backend, clonar un repositorio hostil y ejecutar
`calliope ask` mandaría tu token a una máquina ajena. El CLI ignora ese campo
en las capas de proyecto y avisa por stderr.

### Correr el smoke sin backend

`test/e2e/backend-falso.py` levanta un backend de mentira que implementa los
endpoints del smoke, sin dependencias:

    python3 test/e2e/backend-falso.py &
    CALLIOPE_E2E=1 CALLIOPE_API_KEY=cualquiera CALLIOPE_ORG=miorg \
      CALLIOPE_BASE_URL=http://127.0.0.1:8899 \
      go test -tags=e2e ./test/e2e/ -v

Sirve para desarrollar y para CI, pero **no sustituye** al smoke contra el
backend real: un stub que devuelve lo que el CLI espera no puede descubrir
que el CLI espera algo equivocado.

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
- **El release automático ya está probado de extremo a extremo.** El tag
  `v0.0.1-test` produjo los 13 artefactos (6 archivos, 6 paquetes deb/rpm/apk
  y `checksums.txt`), escribió la fórmula en `Calliope-AI/homebrew-tap` y el
  manifiesto en `Calliope-AI/scoop-bucket`, y
  `brew install Calliope-AI/tap/calliope` instaló un binario que arranca con
  la versión inyectada. `goreleaser check` pasa limpio y el workflow fija
  `version: "~> v2.18"` para que una deprecación futura no rompa un release
  en marcha.

- **El smoke de extremo a extremo nunca se ha ejecutado contra un backend
  real.** `test/e2e/smoke_test.go` compila y sus tests se saltan
  correctamente sin credenciales, pero nadie lo ha corrido todavía con
  `CALLIOPE_E2E=1` y una organización real: no hay credenciales de Calliope
  disponibles en el entorno donde se escribió. Es la primera comprobación
  pendiente en cuanto existan, y de paso confirmaría los tipos `Me`,
  `Organization` y `SchemaResponse`/`SchemaTable`/`SchemaColumn` del bullet
  de más arriba.
