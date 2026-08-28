package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/version"
)

func TestRootMuestraAyudaSinArgumentos(t *testing.T) {
	cmd := NewRootCmd(appctx.Deps{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if !strings.Contains(out.String(), "calliope") {
		t.Errorf("la ayuda no menciona el binario:\n%s", out.String())
	}
}

func TestVersionImprimeLaVersion(t *testing.T) {
	cmd := NewRootCmd(appctx.Deps{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if !strings.Contains(out.String(), "dev") {
		t.Errorf("se esperaba la versión por defecto 'dev', se obtuvo:\n%s", out.String())
	}
}

// El aviso de nueva versión solo debe salir en terminal interactivo: en una
// tubería la línea extra rompería a quien parsee la salida. TestVersion-
// ImprimeLaVersion no lo comprueba: con Version en "dev" (el valor por
// defecto en los tests) LatestVersion se calla antes de mirar siquiera el
// TTY, así que una mutación que borrara el `if d.IsTTY` pasaría inadvertida.
// Aquí se fuerza una versión "real" y un servidor de prueba (inyectado por
// appctx.Deps.ReleasesURL, no por una variable global) para ejercer el
// camino completo sin tocar la red real.
func TestVersionAvisaSoloEnTTY(t *testing.T) {
	versionOriginal := version.Version
	defer func() { version.Version = versionOriginal }()
	version.Version = "v1.2.0"

	llamado := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llamado = true
		w.Write([]byte(`{"tag_name":"v1.4.0"}`))
	}))
	defer srv.Close()

	t.Run("sin TTY no avisa ni consulta la red", func(t *testing.T) {
		llamado = false
		cmd := NewRootCmd(appctx.Deps{IsTTY: false, ReleasesURL: srv.URL})
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"version"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("no se esperaba error: %v", err)
		}
		if strings.Contains(out.String(), "Hay una versión más reciente") {
			t.Errorf("sin TTY no debía avisar:\n%s", out.String())
		}
		if llamado {
			t.Error("sin TTY no debía consultar la red")
		}
	})

	t.Run("con TTY avisa si hay versión más reciente", func(t *testing.T) {
		llamado = false
		cmd := NewRootCmd(appctx.Deps{IsTTY: true, ReleasesURL: srv.URL})
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"version"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("no se esperaba error: %v", err)
		}
		if !strings.Contains(out.String(), "Hay una versión más reciente: v1.4.0") {
			t.Errorf("con TTY debía avisar de v1.4.0:\n%s", out.String())
		}
		if !llamado {
			t.Error("con TTY debía consultar la red")
		}
	})
}
