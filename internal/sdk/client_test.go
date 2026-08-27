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

	got := c.OrgPath("acme corp/../otra", "/rules")
	if strings.Contains(got, "..") || strings.Contains(got, " ") {
		t.Errorf("OrgPath = %q — el nombre de organización debe ir escapado", got)
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
