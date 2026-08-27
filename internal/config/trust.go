package config

import "fmt"

// ProjectAllowed son las únicas claves que una capa de proyecto puede fijar.
//
// La regla existe porque un .calliope/config.json viaja dentro de cualquier
// repositorio clonado. Si pudiera fijar base_url, clonar un repositorio hostil
// y ejecutar `calliope ask` enviaría el token del usuario a una máquina ajena.
// Los timeouts quedan fuera porque un timeout absurdo convierte cualquier
// comando en un fallo silencioso.
var ProjectAllowed = map[string]bool{
	KeyOrg:    true,
	KeyOutput: true,
}

// sanitize elimina de una capa de proyecto todo campo no permitido y devuelve
// un aviso por cada uno, para que el usuario vea qué se ha ignorado.
func sanitize(l Layer) (Layer, []string) {
	if l.Source != SourceProject && l.Source != SourceRepo {
		return l, nil
	}

	limpia := Layer{Source: l.Source, Path: l.Path, Values: map[string]string{}}
	var avisos []string
	for k, v := range l.Values {
		if ProjectAllowed[k] {
			limpia.Values[k] = v
			continue
		}
		avisos = append(avisos, fmt.Sprintf(
			"aviso: se ignora %q de %s; la configuración de proyecto solo puede fijar: org, output",
			k, l.Path))
	}
	return limpia, avisos
}
