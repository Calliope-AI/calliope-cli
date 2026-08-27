// Package auth resuelve y almacena la credencial con la que el CLI llama a
// Calliope Data. Las dos formas viven tras la misma interfaz, para que añadir
// una tercera no toque el código de los comandos.
package auth

// Kind distingue las formas de credencial soportadas.
type Kind string

const (
	KindAPIKey Kind = "api_key"
	KindOAuth  Kind = "oauth"
)

// Credential es la credencial resuelta para una invocación. Org es opcional:
// solo lo rellenan las claves limitadas a una organización.
type Credential struct {
	Kind  Kind   `json:"kind"`
	Token string `json:"token"`
	Org   string `json:"org,omitempty"`
}

// Header devuelve la cabecera HTTP que corresponde a esta credencial.
func (c Credential) Header() (string, string) {
	if c.Kind == KindOAuth {
		return "Authorization", "Bearer " + c.Token
	}
	return "X-API-Key", c.Token
}

// Valid indica si la credencial se puede usar.
func (c Credential) Valid() bool { return c.Token != "" }
