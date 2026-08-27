package auth

import (
	"os"
	"path/filepath"
	"testing"
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
