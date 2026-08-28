// Package sdk es el cliente HTTP de Calliope Data. Construye la URL con el
// scope de organización, aplica la credencial, impone el timeout y traduce el
// status HTTP a un error de dominio. No conoce Cobra ni formatos de salida.
package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/output"
)

// Options configura el cliente.
type Options struct {
	BaseURL    string
	Credential auth.Credential
	Timeout    time.Duration
	UserAgent  string
	HTTPClient *http.Client
}

// Client habla con Calliope Data.
type Client struct {
	baseURL string
	cred    auth.Credential
	http    *http.Client
	ua      string
}

// New construye un cliente. Si no se da HTTPClient, se crea uno con el timeout.
func New(opts Options) *Client {
	hc := opts.HTTPClient
	if hc == nil {
		t := opts.Timeout
		if t == 0 {
			t = 60 * time.Second
		}
		hc = &http.Client{Timeout: t}
	}
	ua := opts.UserAgent
	if ua == "" {
		ua = "calliope-cli"
	}
	return &Client{
		baseURL: strings.TrimRight(opts.BaseURL, "/"),
		cred:    opts.Credential,
		http:    hc,
		ua:      ua,
	}
}

// OrgPath construye una ruta con el scope de organización. El nombre se
// escapa porque llega de configuración de proyecto, que es una entrada no
// confiable: lo que neutraliza el path traversal es que "/" quede escapado
// a "%2F" (RFC 3986: los segmentos de una URL se delimitan por barras sin
// decodificar), de modo que el valor de org entero queda como un único
// segmento opaco.
func (c *Client) OrgPath(org, suffix string) string {
	return "/v1/organizations/" + url.PathEscape(org) + suffix
}

// Do ejecuta una petición y decodifica la respuesta en out. Si out es nil, el
// cuerpo se descarta.
func (c *Client) Do(ctx context.Context, method, path string, body, out any) error {
	var cuerpo io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		cuerpo = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, cuerpo)
	if err != nil {
		// Un error aquí es casi siempre una URL base mal formada (p. ej. con
		// un carácter de control), no un problema de red: se traduce igual
		// que los errores de mapStatus/transportError, en vez de dejar pasar
		// el err.Error() crudo de net/url, que sale en inglés y filtra un
		// detalle interno de la librería estándar en vez de una acción.
		return output.NewError(output.CodeGeneric,
			"No se pudo construir la solicitud a Calliope Data.",
			"Comprueba la URL del backend con: calliope config list")
	}
	k, v := c.cred.Header()
	req.Header.Set(k, v)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.ua)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return transportError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// El cuerpo se descarta a propósito: no filtramos internals del backend.
		io.Copy(io.Discard, resp.Body)
		return mapStatus(resp.StatusCode)
	}

	if out == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}

	crudo, err := io.ReadAll(resp.Body)
	if err != nil {
		return transportError(err)
	}
	if len(bytes.TrimSpace(crudo)) == 0 {
		return nil
	}
	if err := json.Unmarshal(crudo, out); err != nil {
		return output.NewError(output.CodeGeneric,
			"La respuesta de Calliope Data no tiene el formato esperado.",
			"Comprueba la conectividad y la versión del backend con: calliope doctor")
	}
	return nil
}

// mapStatus traduce el status a un error limpio, sin cuerpo del backend.
// Sigue el criterio de mapError en calliope-data-mcp.
func mapStatus(status int) error {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return output.NewError(output.CodeUnauthorized,
			"No autorizado para acceder a estos datos.",
			"Comprueba tu credencial con: calliope auth status")
	case status == http.StatusNotFound:
		return output.NewError(output.CodeNotFound,
			"Recurso no encontrado.",
			"Comprueba el identificador y la organización activa con: calliope config list")
	case status == http.StatusTooManyRequests:
		return output.NewError(output.CodeRateLimited,
			"Se ha superado el límite de solicitudes.",
			"Espera unos segundos y reinténtalo.")
	default:
		return output.NewError(output.CodeGeneric,
			"Error al consultar Calliope Data.",
			"Diagnostica la conexión con: calliope doctor")
	}
}

func transportError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || isTimeout(err) {
		return output.NewError(output.CodeGeneric,
			"La solicitud a Calliope Data superó el tiempo límite.",
			"Sube el límite con CALLIOPE_TIMEOUT=120s, o comprueba la red con: calliope doctor")
	}
	return output.NewError(output.CodeGeneric,
		"No se pudo contactar con Calliope Data.",
		"Comprueba tu conexión y la URL del backend con: calliope doctor")
}

func isTimeout(err error) bool {
	var t interface{ Timeout() bool }
	return errors.As(err, &t) && t.Timeout()
}
