package appctx

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/output"
)

func commandWithFlags(flags map[string]string) *cobra.Command {
	cmd := &cobra.Command{Use: "x"}
	RegisterGlobalFlags(cmd)
	for k, v := range flags {
		_ = cmd.Flags().Set(k, v)
	}
	return cmd
}

func testDeps(t *testing.T, cwd string) (Deps, *bytes.Buffer) {
	t.Helper()
	st := auth.NewFileStore(filepath.Join(t.TempDir(), "c.json"))
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	home := t.TempDir() // fuera del closure: si no, cada llamada crearía otro
	return Deps{
		Cwd: cwd,
		Env: func(k string) string {
			if k == "HOME" {
				return home
			}
			return ""
		},
		Store:  st,
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
	}, &stderr
}

func writeProjectConfig(t *testing.T, dir string, vals map[string]string) {
	t.Helper()
	d := filepath.Join(dir, ".calliope")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(vals)
	if err := os.WriteFile(filepath.Join(d, "config.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLaOrganizacionSaleDeLaConfiguracionDeProyecto(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, map[string]string{"org": "acme"})
	deps, _ := testDeps(t, dir)

	ctx, err := Build(commandWithFlags(nil), deps)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if ctx.Org != "acme" {
		t.Errorf("Org = %q, se esperaba acme", ctx.Org)
	}
}

func TestElFlagOrgGanaALaConfiguracion(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, map[string]string{"org": "acme"})
	deps, _ := testDeps(t, dir)

	ctx, err := Build(commandWithFlags(map[string]string{"org": "otra"}), deps)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if ctx.Org != "otra" {
		t.Errorf("Org = %q, el flag debe ganar", ctx.Org)
	}
}

func TestSinOrganizacionElErrorDiceComoElegirla(t *testing.T) {
	deps, _ := testDeps(t, t.TempDir())

	_, err := Build(commandWithFlags(nil), deps)
	if err == nil {
		t.Fatal("se esperaba error al no haber organización")
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2 (uso incorrecto)", got)
	}
	if !strings.Contains(err.Error(), "organización") {
		t.Errorf("mensaje poco claro: %q", err.Error())
	}
}

func TestLosAvisosDeConfiguracionVanAStderr(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, map[string]string{"org": "acme", "base_url": "https://atacante.example"})
	deps, stderr := testDeps(t, dir)

	if _, err := Build(commandWithFlags(nil), deps); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(stderr.String(), "base_url") {
		t.Errorf("el aviso debe llegar a stderr, se obtuvo: %q", stderr.String())
	}
}

func TestLosFlagsDeterminanElModoDeSalida(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, map[string]string{"org": "acme"})

	casos := []struct {
		flags  map[string]string
		quiero string
	}{
		{map[string]string{"json": "true"}, "json"},
		{map[string]string{"quiet": "true"}, "quiet"},
		{map[string]string{"md": "true"}, "md"},
		{map[string]string{"jq": ".data"}, "jq"},
		{nil, "auto"},
	}

	for _, caso := range casos {
		deps, _ := testDeps(t, dir)
		ctx, err := Build(commandWithFlags(caso.flags), deps)
		if err != nil {
			t.Fatalf("Build(%v): %v", caso.flags, err)
		}
		if string(ctx.Present.Mode) != caso.quiero {
			t.Errorf("flags %v → modo %q, se esperaba %q", caso.flags, ctx.Present.Mode, caso.quiero)
		}
	}
}

func TestBuildSinCredencialNoFallaSinAutenticacion(t *testing.T) {
	dir := t.TempDir()
	writeProjectConfig(t, dir, map[string]string{"org": "acme"})
	deps, _ := testDeps(t, dir)
	deps.Store = auth.NewFileStore(filepath.Join(t.TempDir(), "vacio.json"))

	ctx, err := BuildSinCredencial(commandWithFlags(nil), deps)
	if err != nil {
		t.Fatalf("BuildSinCredencial no debe exigir credencial: %v", err)
	}
	if ctx.Cred.Valid() {
		t.Error("no debería haber credencial")
	}
}
