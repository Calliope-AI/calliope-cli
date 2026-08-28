package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"

	"github.com/Calliope-AI/calliope-cli/internal/output"
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

// ioErrorHint es el hint común de los errores de E/S de este almacén: quien
// lo lee puede cambiar dónde vive el fichero de credenciales sin tocar
// permisos del sistema.
const ioErrorHint = "Comprueba los permisos de escritura en tu directorio de configuración, o cambia su ubicación con la variable de entorno XDG_CONFIG_HOME."

func (s *fileStore) Save(c Credential) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return output.WrapIOError("No se pudo crear el directorio de configuración.", ioErrorHint, err)
	}
	// os.MkdirAll solo aplica el modo al crear: si el directorio ya existía
	// (p.ej. tras restaurar dotfiles o extraer un tar con umask laxo) se
	// queda con los permisos que ya tuviera. Se refuerza explícitamente.
	if err := os.Chmod(dir, 0o700); err != nil {
		return output.WrapIOError("No se pudo ajustar los permisos del directorio de configuración.", ioErrorHint, err)
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.path, b, 0o600); err != nil {
		return output.WrapIOError("No se pudo guardar la credencial.", ioErrorHint, err)
	}
	// Igual que con el directorio: si el fichero ya existía, os.WriteFile no
	// toca sus permisos. Se refuerza para que la credencial no quede legible
	// por otros usuarios del sistema.
	if err := os.Chmod(s.path, 0o600); err != nil {
		return output.WrapIOError("No se pudo ajustar los permisos de la credencial guardada.", ioErrorHint, err)
	}
	return nil
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
		return nil, corruptCredentialError(err)
	}
	return &c, nil
}

func (s *fileStore) Delete() error {
	err := os.Remove(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return output.WrapIOError("No se pudo borrar la credencial guardada.", ioErrorHint, err)
	}
	return nil
}

// keyringStore usa el llavero del sistema y cae al respaldo cuando no hay
// ninguno disponible.
type keyringStore struct{ fallback Store }

// NewKeyringStore crea un almacén sobre el llavero del sistema.
func NewKeyringStore(fallback Store) Store { return &keyringStore{fallback: fallback} }

// Save mantiene el invariante de que, como mucho, un almacén guarda la
// credencial en cada momento: si el llavero la acepta, se borra el residuo
// del respaldo (podría revivir una credencial vieja, quizá ya revocada); si
// el llavero falla, se borra la posible entrada vieja del llavero antes de
// caer al respaldo, para que Load no la sirva en vez de la recién guardada.
// Los borrados son de mejor esfuerzo: que fallen no debe romper el Save.
func (s *keyringStore) Save(c Credential) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := keyring.Set(keyringService, keyringUser, string(b)); err != nil {
		_ = keyring.Delete(keyringService, keyringUser)
		return s.fallback.Save(c)
	}
	_ = s.fallback.Delete()
	return nil
}

func (s *keyringStore) Load() (*Credential, error) {
	crudo, err := keyring.Get(keyringService, keyringUser)
	if err != nil {
		return s.fallback.Load()
	}
	var c Credential
	if err := json.Unmarshal([]byte(crudo), &c); err != nil {
		return nil, corruptCredentialError(err)
	}
	return &c, nil
}

func (s *keyringStore) Delete() error {
	// Se borra en ambos sitios: da igual dónde acabara al guardarse.
	_ = keyring.Delete(keyringService, keyringUser)
	return s.fallback.Delete()
}

// corruptCredentialError envuelve un fallo al descodificar una credencial
// guardada (fichero o llavero) en un CLIError con mensaje y pista en
// español. La causa técnica original no se descarta: sigue accesible vía
// errors.As/errors.Is, para que un log o diagnóstico futuro pueda inspeccionarla.
func corruptCredentialError(cause error) error {
	return fmt.Errorf("%w: %w", output.NewError(output.CodeUnauthorized,
		"La credencial guardada está dañada o en un formato inesperado.",
		"Vuelve a autenticarte: calliope auth login --api-key <clave>"), cause)
}

// DefaultStore es el almacén que usa el CLI: llavero del sistema con respaldo
// en el fichero global. Nunca escribe en configuración de proyecto.
func DefaultStore(globalDir string) Store {
	return NewKeyringStore(NewFileStore(filepath.Join(globalDir, "credentials.json")))
}
