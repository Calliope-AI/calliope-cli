package version

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDetectaUnaVersionMasReciente(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v1.4.0"}`))
	}))
	defer srv.Close()

	got := LatestVersion(srv.URL, "v1.2.0", time.Second)
	if got != "v1.4.0" {
		t.Errorf("LatestVersion = %q, se esperaba v1.4.0", got)
	}
}

func TestNoAvisaSiYaEstaAlDia(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v1.2.0"}`))
	}))
	defer srv.Close()

	if got := LatestVersion(srv.URL, "v1.2.0", time.Second); got != "" {
		t.Errorf("LatestVersion = %q, no debe avisar estando al día", got)
	}
}

// El ldflag de GoReleaser inyecta {{.Version}} SIN el prefijo "v" (p. ej.
// "1.2.0"), pero el tag_name de la API de GitHub SIEMPRE lo lleva (p. ej.
// "v1.2.0"). Sin normalizar, un binario real compararía "1.2.0" contra
// "v1.2.0" — que nunca son iguales como cadenas — y avisaría siempre de una
// "versión más reciente" que en realidad es la misma que ya tiene instalada.
// Se comprueba en ambas direcciones: no importa cuál de las dos formas
// llegue en `actual` ni cuál llegue en el tag_name.
func TestElPrefijoVNoImportaAlComparar(t *testing.T) {
	casos := []struct {
		nombre string
		actual string
		tag    string
	}{
		{"actual sin v (el caso real), tag con v", "1.2.0", "v1.2.0"},
		{"actual con v, tag sin v", "v1.2.0", "1.2.0"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"tag_name":%q}`, c.tag)
			}))
			defer srv.Close()

			if got := LatestVersion(srv.URL, c.actual, time.Second); got != "" {
				t.Errorf("LatestVersion(actual=%q, tag=%q) = %q, son la misma versión: no debía avisar", c.actual, c.tag, got)
			}
		})
	}
}

// Sin red, el aviso se calla: nunca debe romper `calliope version`.
func TestSinRedNoRompe(t *testing.T) {
	if got := LatestVersion("http://127.0.0.1:1", "v1.2.0", 50*time.Millisecond); got != "" {
		t.Errorf("LatestVersion = %q, sin red debe callarse", got)
	}
}

// El timeout debe respetarse de verdad: un servidor lento no debe colgar el
// aviso, que es de cortesía. TestSinRedNoRompe no lo comprueba: la conexión a
// un puerto cerrado falla al instante por "connection refused", sin llegar
// nunca a esperar el timeout.
func TestRespetaElTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.Write([]byte(`{"tag_name":"v1.4.0"}`))
	}))
	defer srv.Close()

	inicio := time.Now()
	got := LatestVersion(srv.URL, "v1.2.0", 50*time.Millisecond)
	transcurrido := time.Since(inicio)

	if got != "" {
		t.Errorf("LatestVersion = %q, se esperaba vacío por timeout", got)
	}
	if transcurrido > 250*time.Millisecond {
		t.Errorf("LatestVersion tardó %s, el timeout de 50ms no se respetó", transcurrido)
	}
}

func TestEnCompilacionDeDesarrolloNoConsulta(t *testing.T) {
	llamado := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llamado = true
	}))
	defer srv.Close()

	if got := LatestVersion(srv.URL, "dev", time.Second); got != "" {
		t.Errorf("LatestVersion = %q, una compilación dev no debe avisar", got)
	}
	if llamado {
		t.Error("una compilación dev no debe ni consultar la red")
	}
}
