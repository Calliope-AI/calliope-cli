package auth

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/calliope/calliope-cli/internal/output"
)

// Estos tests cubren keyringStore sin tocar el llavero real del sistema:
// usan el proveedor simulado en memoria que ofrece go-keyring para pruebas.

func TestKeyringStoreGuardaYRecuperaConMock(t *testing.T) {
	keyring.MockInit()

	respaldo := NewFileStore(filepath.Join(t.TempDir(), "credentials.json"))
	st := NewKeyringStore(respaldo)

	quiero := Credential{Kind: KindAPIKey, Token: "cal_live_mock", Org: "acme"}
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

	// Confirma que la credencial fue al llavero simulado, no al respaldo.
	enRespaldo, err := respaldo.Load()
	if err != nil {
		t.Fatalf("Load del respaldo: %v", err)
	}
	if enRespaldo != nil {
		t.Errorf("el respaldo debería estar vacío cuando el llavero funciona, tenía %+v", enRespaldo)
	}
}

func TestKeyringStoreCaeAlRespaldoSiElLlaveroFalla(t *testing.T) {
	keyring.MockInitWithError(errors.New("llavero no disponible"))

	respaldo := NewFileStore(filepath.Join(t.TempDir(), "credentials.json"))
	st := NewKeyringStore(respaldo)

	quiero := Credential{Kind: KindAPIKey, Token: "cal_live_respaldo"}
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

func TestKeyringStoreDeleteBorraDelMock(t *testing.T) {
	keyring.MockInit()

	respaldo := NewFileStore(filepath.Join(t.TempDir(), "credentials.json"))
	st := NewKeyringStore(respaldo)

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

// TestKeyringStoreBorraResiduoDelRespaldoAlGuardarConExito cubre el
// invariante: como mucho un almacén guarda la credencial en cada momento.
// Si el respaldo tenía una credencial vieja (p.ej. residuo de un fallo
// anterior del llavero, ya resuelto) y un Save posterior sí llega al
// llavero, el residuo no debe sobrevivir: podría ser una clave ya revocada
// que resucite más tarde si el llavero volviera a fallar.
func TestKeyringStoreBorraResiduoDelRespaldoAlGuardarConExito(t *testing.T) {
	keyring.MockInit()

	respaldo := NewFileStore(filepath.Join(t.TempDir(), "credentials.json"))
	if err := respaldo.Save(Credential{Kind: KindAPIKey, Token: "vieja-y-revocada"}); err != nil {
		t.Fatal(err)
	}

	st := NewKeyringStore(respaldo)
	if err := st.Save(Credential{Kind: KindAPIKey, Token: "nueva"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	enRespaldo, err := respaldo.Load()
	if err != nil {
		t.Fatalf("Load del respaldo: %v", err)
	}
	if enRespaldo != nil {
		t.Errorf("el respaldo debería quedar limpio tras un Save que sí llegó al llavero, tenía %+v", enRespaldo)
	}
}

func TestKeyringStoreLoadConValorCorruptoDevuelveErrorAccionable(t *testing.T) {
	keyring.MockInit()
	if err := keyring.Set(keyringService, keyringUser, "esto no es json"); err != nil {
		t.Fatal(err)
	}

	respaldo := NewFileStore(filepath.Join(t.TempDir(), "credentials.json"))
	st := NewKeyringStore(respaldo)

	_, err := st.Load()
	if err == nil {
		t.Fatal("se esperaba un error con un valor de llavero corrupto")
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
	if !strings.Contains(err.Error(), "invalid character") {
		t.Errorf("se esperaba conservar la causa original en la cadena de errores: %v", err)
	}
}
