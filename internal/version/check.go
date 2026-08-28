package version

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// ReleasesURL es el endpoint que se consulta para saber la última versión.
const ReleasesURL = "https://api.github.com/repos/Calliope-AI/calliope-cli/releases/latest"

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
	if cuerpo.TagName == "" || normalizeVersion(cuerpo.TagName) == normalizeVersion(actual) {
		return ""
	}
	return cuerpo.TagName
}

// normalizeVersion quita el prefijo "v" para poder comparar versiones sin
// depender de si lo llevan. El ldflag de GoReleaser inyecta {{.Version}} sin
// "v" (p. ej. "1.2.0"), pero el tag_name de la API de GitHub siempre lo
// lleva (p. ej. "v1.2.0"); sin esto, un binario real avisaría siempre de una
// "versión nueva" que en realidad es la misma que ya tiene instalada. El
// valor que se muestra al usuario sigue siendo el tag_name tal cual llega,
// con su "v" si la trae: esto solo afecta a la comparación.
func normalizeVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}
