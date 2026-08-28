// Package skills embebe la documentación que el CLI publica para agentes.
// Va dentro del binario a propósito: así el skill no puede desincronizarse de
// la versión instalada.
package skills

import _ "embed"

//go:embed calliope/SKILL.md
var SkillMD string
