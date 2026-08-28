package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/calliope/calliope-cli/internal/appctx"
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
