package auth

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Calliope-AI/calliope-cli/internal/output"
)

func TestFileStoreGuardaYRecupera(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "credentials.json")
	st := NewFileStore(ruta)

	quiero := Credential{Kind: KindAPIKey, Token: "cal_live_abc", Org: "acme"}
	if err := st.Save(quiero); err != nil {
		t.Fatalf("Save: %v", err)
	}

	tengo, err := st.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if tengo == nil || *tengo != quiero {
		t.Errorf("Load() = %+v, se esperaba %+v", tengo, quiero)
	}
}

func TestFileStoreEscribeCon0600(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "credentials.json")
	st := NewFileStore(ruta)
	if err := st.Save(Credential{Kind: KindAPIKey, Token: "secreto"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	fi, err := os.Stat(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("permisos = %o, se esperaba 600 — la credencial quedaría legible por otros", perm)
	}
}

func TestFileStoreSinFicheroDevuelveNil(t *testing.T) {
	st := NewFileStore(filepath.Join(t.TempDir(), "no-existe.json"))
	c, err := st.Load()
	if err != nil {
		t.Fatalf("Load sin fichero no debe fallar: %v", err)
	}
	if c != nil {
		t.Errorf("Load() = %+v, se esperaba nil", c)
	}
}

func TestFileStoreDelete(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "credentials.json")
	st := NewFileStore(ruta)
	if err := st.Save(Credential{Kind: KindAPIKey, Token: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := st.Delete(); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if c, _ := st.Load(); c != nil {
		t.Error("tras Delete, Load debe devolver nil")
	}
}

// TestFileStoreSaveSinPermisosDeEscrituraEsCLIError es el test del
// Diferido #10 de la oleada final para `auth login`: en un directorio de
// configuración sin permisos de escritura, fileStore.Save devolvía el error
// crudo de os.MkdirAll -en inglés, sin hint, y con la ruta absoluta del
// sistema de ficheros del cliente incluida en el mensaje-.
func TestFileStoreSaveSinPermisosDeEscrituraEsCLIError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	st := NewFileStore(filepath.Join(dir, "sub", "credentials.json"))
	err := st.Save(Credential{Kind: KindAPIKey, Token: "x"})
	if err == nil {
		t.Fatal("se esperaba un error de E/S")
	}

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T (%v)", err, err)
	}
	if cliErr.Hint == "" {
		t.Error("el error debería traer un hint")
	}
	if strings.Contains(cliErr.Message, dir) {
		t.Errorf("el mensaje filtra la ruta absoluta del sistema: %q", cliErr.Message)
	}
	if strings.Contains(cliErr.Message, "permission denied") {
		t.Errorf("el mensaje no debe ser el de Go en inglés: %q", cliErr.Message)
	}
}

// TestFileStoreDeleteSinPermisosDeEscrituraEsCLIError es la mitad de
// `auth logout` del Diferido #10: si el fichero existe pero su directorio
// no tiene permiso de escritura, os.Remove falla y devolvía el error crudo
// de Go, igual que Save.
func TestFileStoreDeleteSinPermisosDeEscrituraEsCLIError(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "credentials.json")
	if err := os.WriteFile(ruta, []byte(`{"kind":"api_key","token":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	st := NewFileStore(ruta)
	err := st.Delete()
	if err == nil {
		t.Fatal("se esperaba un error de E/S")
	}

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T (%v)", err, err)
	}
	if cliErr.Hint == "" {
		t.Error("el error debería traer un hint")
	}
	if strings.Contains(cliErr.Message, dir) {
		t.Errorf("el mensaje filtra la ruta absoluta del sistema: %q", cliErr.Message)
	}
}

func TestFileStoreRefuerzaPermisosSobreFicheroYDirectorioExistentes(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub")
	ruta := filepath.Join(dir, "credentials.json")

	// El directorio y el fichero ya existen con permisos laxos, como tras
	// restaurar dotfiles o extraer un tar con umask 022. os.MkdirAll y
	// os.WriteFile solo aplican el modo al crear, así que sin refuerzo
	// explícito estos permisos sobrevivirían al Save.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(`{"kind":"api_key","token":"viejo"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	st := NewFileStore(ruta)
	if err := st.Save(Credential{Kind: KindAPIKey, Token: "nuevo"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	fi, err := os.Stat(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("permisos del fichero = %o, se esperaba 600 tras Save sobre un fichero preexistente", perm)
	}

	di, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := di.Mode().Perm(); perm != 0o700 {
		t.Errorf("permisos del directorio = %o, se esperaba 700 tras Save sobre un directorio preexistente", perm)
	}
}

func TestFileStoreLoadConJSONCorruptoDevuelveErrorAccionable(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "credentials.json")
	if err := os.WriteFile(ruta, []byte("{no es json"), 0o600); err != nil {
		t.Fatal(err)
	}

	st := NewFileStore(ruta)
	_, err := st.Load()
	if err == nil {
		t.Fatal("se esperaba un error con un fichero de credenciales corrupto")
	}

	if got := output.ExitCodeFor(err); got != 3 {
		t.Errorf("código de salida = %d, se esperaba 3 (no autorizado)", got)
	}

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatal("se esperaba poder extraer un *output.CLIError")
	}
	if cliErr.Hint == "" {
		t.Error("el error debe decir cómo recuperarse (volver a autenticarse)")
	}
	if strings.ContainsAny(cliErr.Message, "{}") {
		t.Errorf("el mensaje de cara al usuario no debe filtrar detalle crudo de JSON: %q", cliErr.Message)
	}

	// La causa técnica original no se descarta: sigue accesible en la cadena
	// de errores para diagnóstico, aunque el usuario solo vea cliErr.Message.
	if !strings.Contains(err.Error(), "invalid character") {
		t.Errorf("se esperaba conservar la causa original en la cadena de errores: %v", err)
	}
}
