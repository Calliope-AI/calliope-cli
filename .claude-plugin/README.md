# Plugin de Claude Code para Calliope Data

Este plugin empaqueta el skill, el comando `/calliope` y un hook
`SessionStart` que avisa si `calliope` no está instalado o no está listo.

## Limitación conocida: Windows sin enlaces simbólicos

`skills` en este directorio es un enlace simbólico a `../skills` (la raíz del
repositorio), no una copia: así el skill que expone el plugin nunca puede
divergir del que el binario embebe con `go:embed`
(`skills/calliope/SKILL.md`, vía `skills/embed.go`) — lo comprueba
`internal/cli/plugin_test.go` (`TestElPluginUsaElMismoSkillQueElBinario`).

Git en Windows, sin el Modo Desarrollador activado (o sin `core.symlinks=true`
en la configuración de git), no materializa enlaces simbólicos al clonar: en
su lugar deja un fichero de texto normal cuyo contenido es literalmente
`../skills`, y el directorio real del skill queda inaccesible desde el
plugin.

Esto no bloquea nada: el comando `/calliope` no lee `skills/` del plugin,
invoca `calliope skill` por CLI, que imprime el mismo contenido leyendo el
binario (embebido con `go:embed`, no el disco) y funciona igual en cualquier
plataforma, con o sin enlaces simbólicos. Si en Windows `.claude-plugin/skills`
aparece como un fichero de texto en vez de un directorio, esa es la vía
fiable: `calliope skill`.
