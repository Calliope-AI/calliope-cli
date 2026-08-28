package version

import (
	"encoding/json"
	"net/http"
	"time"
)

// ReleasesURL es el endpoint que se consulta para saber la última versión.
// Es variable (no constante) para que los tests de internal/cli puedan
// apuntarlo a un servidor de prueba y comprobar el aviso de principio a fin
// sin tocar la red real.
var ReleasesURL = "https://api.github.com/repos/calliope/calliope-cli/releases/latest"

// LatestVersion devuelve la última versión publicada si es distinta de la
// actual, o cadena vacía. Nunca devuelve error: un aviso de cortesía jamás
// debe hacer fallar `calliope version`.
func LatestVersion(url, actual string, timeout time.Duration) string {
	// Una compilación de desarrollo no tiene con qué comparar.
	if actual == "" || actual == "dev" {
		return ""
	}

	cliente := &http.Client{Timeout: timeout}
	resp, err := cliente.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var cuerpo struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cuerpo); err != nil {
		return ""
	}
	if cuerpo.TagName == "" || cuerpo.TagName == actual {
		return ""
	}
	return cuerpo.TagName
}
