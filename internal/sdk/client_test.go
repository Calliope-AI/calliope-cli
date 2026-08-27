package sdk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/output"
)

func testClient(t *testing.T, h http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New(Options{
		BaseURL:    srv.URL,
		Credential: auth.Credential{Kind: auth.KindAPIKey, Token: "cal_live_test"},
		Timeout:    5 * time.Second,
	}), srv
}

func TestLaRutaLlevaElScopeDeOrganizacion(t *testing.T) {
	var visto string
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		visto = r.URL.Path
		w.Write([]byte(`{}`))
	})

	var out map[string]any
	if err := c.Do(context.Background(), http.MethodGet, c.OrgPath("acme", "/rules"), nil, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if visto != "/v1/organizations/acme/rules" {
		t.Errorf("ruta = %q, se esperaba /v1/organizations/acme/rules", visto)
	}
}

func TestElNombreDeOrganizacionSeEscapa(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) })

	// La propiedad que importa no es sintáctica (que no aparezca "..") sino
	// estructural: el nombre de organización no debe introducir ninguna
	// barra sin escapar, porque es la barra sin escapar la que delimita
	// segmentos de URL (RFC 3986). Si org no aporta barras propias, el
	// número total de barras en la ruta resultante es exactamente el de la
	// plantilla fija más el del suffix.
	plantilla := "/v1/organizations/"
	suffix := "/rules"
	esperadas := strings.Count(plantilla, "/") + strings.Count(suffix, "/")

	got := c.OrgPath("acme corp/../otra", suffix)
	if n := strings.Count(got, "/"); n != esperadas {
		t.Errorf("OrgPath = %q — el nombre de organización introduce una barra sin escapar (%d barras, se esperaban %d)", got, n, esperadas)
	}
}

func TestElNombreDeOrganizacionConPuntoNoSeAltera(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) })

	got := c.OrgPath("acme.corp", "/rules")
	if got != "/v1/organizations/acme.corp/rules" {
		t.Errorf("OrgPath = %q, se esperaba /v1/organizations/acme.corp/rules — el punto es válido en un segmento y no debe escaparse", got)
	}
}

func TestSeEnviaLaCabeceraDeAutenticacion(t *testing.T) {
	var visto string
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		visto = r.Header.Get("X-API-Key")
		w.Write([]byte(`{}`))
	})

	var out map[string]any
	if err := c.Do(context.Background(), http.MethodGet, "/v1/auth/me", nil, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if visto != "cal_live_test" {
		t.Errorf("X-API-Key = %q", visto)
	}
}

func TestContentTypeSoloSeFijaConCuerpo(t *testing.T) {
	var vistoSinCuerpo, vistoConCuerpo string
	conCuerpo := false
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if conCuerpo {
			vistoConCuerpo = r.Header.Get("Content-Type")
		} else {
			vistoSinCuerpo = r.Header.Get("Content-Type")
		}
		w.Write([]byte(`{}`))
	})

	var out map[string]any
	if err := c.Do(context.Background(), http.MethodGet, "/x", nil, &out); err != nil {
		t.Fatalf("Do sin cuerpo: %v", err)
	}
	if vistoSinCuerpo != "" {
		t.Errorf("Content-Type sin cuerpo = %q, se esperaba vacío", vistoSinCuerpo)
	}

	conCuerpo = true
	if err := c.Do(context.Background(), http.MethodPost, "/x", map[string]string{"a": "b"}, &out); err != nil {
		t.Fatalf("Do con cuerpo: %v", err)
	}
	if vistoConCuerpo != "application/json" {
		t.Errorf("Content-Type con cuerpo = %q, se esperaba application/json", vistoConCuerpo)
	}
}

func TestMapeoDeStatusACodigosDeSalida(t *testing.T) {
	casos := []struct {
		status int
		codigo output.Code
		salida int
	}{
		{http.StatusUnauthorized, output.CodeUnauthorized, 3},
		{http.StatusForbidden, output.CodeUnauthorized, 3},
		{http.StatusNotFound, output.CodeNotFound, 4},
		{http.StatusTooManyRequests, output.CodeRateLimited, 5},
		{http.StatusInternalServerError, output.CodeGeneric, 1},
	}

	for _, caso := range casos {
		c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(caso.status)
			w.Write([]byte(`{"detail":"traza interna del backend"}`))
		})

		var out map[string]any
		err := c.Do(context.Background(), http.MethodGet, "/x", nil, &out)
		if err == nil {
			t.Fatalf("status %d: se esperaba error", caso.status)
		}
		if got := output.ExitCodeFor(err); got != caso.salida {
			t.Errorf("status %d: código de salida = %d, se esperaba %d", caso.status, got, caso.salida)
		}
	}
}

func TestElErrorNoFiltraElCuerpoDelBackend(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"detail":"panic en /srv/app/handlers/ask.py línea 412"}`))
	})

	var out map[string]any
	err := c.Do(context.Background(), http.MethodGet, "/x", nil, &out)
	if err == nil {
		t.Fatal("se esperaba error")
	}
	if strings.Contains(err.Error(), "ask.py") || strings.Contains(err.Error(), "panic") {
		t.Errorf("el mensaje filtra internals del backend: %q", err.Error())
	}
}

func TestElTimeoutProduceUnErrorAccionable(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte(`{}`))
	})
	c.http.Timeout = 20 * time.Millisecond

	var out map[string]any
	err := c.Do(context.Background(), http.MethodGet, "/x", nil, &out)
	if err == nil {
		t.Fatal("se esperaba un error de timeout")
	}
	var cliErr *output.CLIError
	if !asCLIError(err, &cliErr) {
		t.Fatalf("se esperaba *output.CLIError, se obtuvo %T", err)
	}
	if cliErr.Hint == "" {
		t.Error("el error de timeout debe sugerir qué hacer")
	}
}

func TestRespuestaVaciaNoRompe(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	var out map[string]any
	if err := c.Do(context.Background(), http.MethodGet, "/x", nil, &out); err != nil {
		t.Errorf("una respuesta vacía no debe fallar: %v", err)
	}
}

func asCLIError(err error, target **output.CLIError) bool {
	e, ok := err.(*output.CLIError)
	if ok {
		*target = e
	}
	return ok
}
