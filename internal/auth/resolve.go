package auth

import "github.com/calliope/calliope-cli/internal/output"

// Resolve devuelve la credencial de esta invocación y una descripción legible
// de su origen, que `auth status` y `doctor` muestran tal cual.
//
// Precedencia: variables de entorno antes que el almacén, para que un entorno
// de CI pueda inyectar la credencial sin tocar el llavero del usuario.
func Resolve(env func(string) string, st Store) (Credential, string, error) {
	if k := env("CALLIOPE_API_KEY"); k != "" {
		return Credential{Kind: KindAPIKey, Token: k, Org: env("CALLIOPE_ORG")},
			"variable de entorno CALLIOPE_API_KEY", nil
	}
	if t := env("CALLIOPE_TOKEN"); t != "" {
		return Credential{Kind: KindOAuth, Token: t, Org: env("CALLIOPE_ORG")},
			"variable de entorno CALLIOPE_TOKEN", nil
	}

	c, err := st.Load()
	if err != nil {
		return Credential{}, "", err
	}
	if c != nil && c.Valid() {
		return *c, "almacén local de credenciales", nil
	}

	return Credential{}, "", output.NewError(output.CodeUnauthorized,
		"No hay ninguna credencial de Calliope configurada.",
		"Ejecuta: calliope auth login --api-key <clave>  (créala en el UI, en Observabilidad → Claves API)")
}
