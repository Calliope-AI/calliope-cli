package config

import "fmt"

// projectAllowed son las únicas claves que una capa de proyecto puede fijar.
//
// La regla existe porque un .calliope/config.json viaja dentro de cualquier
// repositorio clonado. Si pudiera fijar base_url, clonar un repositorio hostil
// y ejecutar `calliope ask` enviaría el token del usuario a una máquina ajena.
// Los timeouts quedan fuera porque un timeout absurdo convierte cualquier
// comando en un fallo silencioso.
//
// AVISO DE SEGURIDAD: este mapa es la frontera de confianza completa entre un
// repositorio clonado y las credenciales/red del usuario. No se exporta a
// propósito: ampliarlo (por ejemplo, para que el proyecto pueda apuntar a un
// proxy local) exige tocar este fichero y, con él, el test que fija su
// contenido exacto en trust_test.go. Si ese test no cambia, la revisión de
// código lo verá como lo que es: una ampliación deliberada de lo que un
// repositorio ajeno puede hacerle al CLI, no un efecto colateral silencioso
// de otro cambio.
var projectAllowed = map[string]bool{
	KeyOrg:    true,
	KeyOutput: true,
}

// IsProjectAllowed indica si una capa de proyecto puede fijar la clave dada.
// Es la única forma en que el resto del paquete (y otras tareas) debe
// consultar la frontera de confianza; el mapa que la respalda no se exporta.
func IsProjectAllowed(key string) bool {
	return projectAllowed[key]
}

// sanitize elimina de una capa de proyecto todo campo no permitido y devuelve
// un aviso por cada uno, para que el usuario vea qué se ha ignorado.
func sanitize(l Layer) (Layer, []string) {
	// Esta comprobación es defensa en profundidad: hoy Load (ver layers.go)
	// solo llama a sanitize para capas SourceProject y SourceRepo, así que la
	// rama "return l, nil" nunca se ejecuta en producción. Se deja aquí a
	// propósito — no es código muerto — para que sanitize siga siendo segura
	// por sí misma si algún día se la llama desde otro punto (p. ej. sobre la
	// capa global) sin repetir el filtro en el llamador.
	if l.Source != SourceProject && l.Source != SourceRepo {
		return l, nil
	}

	limpia := Layer{Source: l.Source, Path: l.Path, Values: map[string]string{}}
	var avisos []string
	for k, v := range l.Values {
		if IsProjectAllowed(k) {
			limpia.Values[k] = v
			continue
		}
		avisos = append(avisos, fmt.Sprintf(
			"aviso: se ignora %q de %s; la configuración de proyecto solo puede fijar: org, output",
			k, l.Path))
	}
	return limpia, avisos
}
