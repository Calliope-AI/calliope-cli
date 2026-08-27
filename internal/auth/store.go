package auth

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "calliope-cli"
	keyringUser    = "default"
)

// Store guarda la credencial entre invocaciones.
type Store interface {
	Save(c Credential) error
	Load() (*Credential, error)
	Delete() error
}

// fileStore guarda la credencial en un fichero con permisos 0600. Es el
// respaldo para máquinas sin keyring (contenedores, servidores sin sesión).
type fileStore struct{ path string }

// NewFileStore crea un almacén respaldado por fichero.
func NewFileStore(path string) Store { return &fileStore{path: path} }

func (s *fileStore) Save(c Credential) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o600)
}

func (s *fileStore) Load() (*Credential, error) {
	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var c Credential
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *fileStore) Delete() error {
	err := os.Remove(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// keyringStore usa el llavero del sistema y cae al respaldo cuando no hay
// ninguno disponible.
type keyringStore struct{ fallback Store }

// NewKeyringStore crea un almacén sobre el llavero del sistema.
func NewKeyringStore(fallback Store) Store { return &keyringStore{fallback: fallback} }

func (s *keyringStore) Save(c Credential) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := keyring.Set(keyringService, keyringUser, string(b)); err != nil {
		return s.fallback.Save(c)
	}
	return nil
}

func (s *keyringStore) Load() (*Credential, error) {
	crudo, err := keyring.Get(keyringService, keyringUser)
	if err != nil {
		return s.fallback.Load()
	}
	var c Credential
	if err := json.Unmarshal([]byte(crudo), &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *keyringStore) Delete() error {
	// Se borra en ambos sitios: da igual dónde acabara al guardarse.
	_ = keyring.Delete(keyringService, keyringUser)
	return s.fallback.Delete()
}

// DefaultStore es el almacén que usa el CLI: llavero del sistema con respaldo
// en el fichero global. Nunca escribe en configuración de proyecto.
func DefaultStore(globalDir string) Store {
	return NewKeyringStore(NewFileStore(filepath.Join(globalDir, "credentials.json")))
}
