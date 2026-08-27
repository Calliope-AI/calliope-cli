package auth

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
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
