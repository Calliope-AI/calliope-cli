package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultBaseURL es el backend de Calliope Data.
const DefaultBaseURL = "https://data-0.calliope.so"

// FileName es el fichero de configuración dentro de un directorio .calliope.
const FileName = "config.json"

// Load construye la configuración completa para una invocación. Devuelve
// además los avisos de la frontera de confianza (ver trust.go).
func Load(cwd string, env func(string) string, flags map[string]string) (*Config, []string, error) {
	var avisos []string

	capas := []Layer{
		{Source: SourceDefault, Values: map[string]string{
			KeyBaseURL: DefaultBaseURL,
			KeyTimeout: "60s",
		}},
	}

	if global, err := readLayerFile(SourceGlobal, filepath.Join(globalDir(env), FileName)); err != nil {
		return nil, nil, err
	} else if global != nil {
		capas = append(capas, *global)
	}

	// Raíz del repositorio y directorio actual, en ese orden de precedencia.
	raiz := repoRoot(cwd)
	if raiz != "" && raiz != cwd {
		if l, err := readLayerFile(SourceRepo, filepath.Join(raiz, ".calliope", FileName)); err != nil {
			return nil, nil, err
		} else if l != nil {
			saneada, w := sanitize(*l)
			avisos = append(avisos, w...)
			capas = append(capas, saneada)
		}
	}
	if l, err := readLayerFile(SourceProject, filepath.Join(cwd, ".calliope", FileName)); err != nil {
		return nil, nil, err
	} else if l != nil {
		saneada, w := sanitize(*l)
		avisos = append(avisos, w...)
		capas = append(capas, saneada)
	}

	capas = append(capas, Layer{Source: SourceEnv, Values: map[string]string{
		KeyOrg:     env("CALLIOPE_ORG"),
		KeyBaseURL: env("CALLIOPE_BASE_URL"),
		KeyOutput:  env("CALLIOPE_OUTPUT"),
		KeyTimeout: env("CALLIOPE_TIMEOUT"),
	}})

	capas = append(capas, Layer{Source: SourceFlag, Values: flags})

	// Los avisos se devuelven en vez de imprimirse aquí: quien decide dónde
	// escribirlos es appctx, que conoce el stderr del comando.
	return Resolve(capas), avisos, nil
}

// GlobalPath es la ruta del fichero de configuración global.
func GlobalPath(env func(string) string) string {
	return filepath.Join(globalDir(env), FileName)
}

func globalDir(env func(string) string) string {
	if x := env("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "calliope")
	}
	return filepath.Join(env("HOME"), ".config", "calliope")
}

func readLayerFile(src Source, ruta string) (*Layer, error) {
	b, err := os.ReadFile(ruta)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error al leer configuración de %s (%s): %w", src, ruta, err)
	}
	var vals map[string]string
	if err := json.Unmarshal(b, &vals); err != nil {
		return nil, fmt.Errorf("error al descodificar configuración de %s (%s): %w", src, ruta, err)
	}
	return &Layer{Source: src, Path: ruta, Values: vals}, nil
}

// repoRoot sube desde cwd buscando un .git, directorio o fichero (Diferido
// #5 de la oleada final). En un repositorio normal .git es un directorio,
// pero en un git worktree o un submódulo .git es un FICHERO de una sola
// línea ("gitdir: <ruta>") que apunta al git real en otro sitio. Antes esta
// función exigía fi.IsDir(), así que en cualquiera de esos dos casos
// repoRoot devolvía "" -sin avisar- y la capa de configuración de la raíz
// del repositorio (SourceRepo, ver Load más abajo) desaparecía en
// silencio: en un worktree, un .calliope/config.json legítimo en la raíz
// del repositorio dejaba de aplicarse, y -más grave- también dejaba de
// sanearse un .calliope/config.json hostil que llegara ahí (ver trust.go).
func repoRoot(cwd string) string {
	dir := cwd
	for {
		if fi, err := os.Stat(filepath.Join(dir, ".git")); err == nil && (fi.IsDir() || fi.Mode().IsRegular()) {
			return dir
		}
		padre := filepath.Dir(dir)
		if padre == dir {
			return ""
		}
		dir = padre
	}
}
