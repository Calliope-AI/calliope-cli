package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrecedenciaDeCapas(t *testing.T) {
	cfg := Resolve([]Layer{
		{Source: SourceDefault, Values: map[string]string{"org": "por-defecto", "base_url": "https://data-0.calliope.so"}},
		{Source: SourceGlobal, Path: "/home/u/.config/calliope/config.json", Values: map[string]string{"org": "global"}},
		{Source: SourceRepo, Path: "/repo/.calliope/config.json", Values: map[string]string{"org": "repo"}},
		{Source: SourceProject, Path: "/repo/sub/.calliope/config.json", Values: map[string]string{"org": "proyecto"}},
		{Source: SourceEnv, Values: map[string]string{"org": "entorno"}},
		{Source: SourceFlag, Values: map[string]string{"org": "flag"}},
	})

	if got := cfg.Get("org").Value; got != "flag" {
		t.Errorf("org = %q, el flag debe ganar a todo", got)
	}
	if got := cfg.Get("org").Source; got != SourceFlag {
		t.Errorf("origen = %q, se esperaba flag", got)
	}
	if got := cfg.Get("base_url").Value; got != "https://data-0.calliope.so" {
		t.Errorf("base_url = %q, debe caer al valor por defecto", got)
	}
}

func TestValorRecuerdaSuFicheroDeOrigen(t *testing.T) {
	cfg := Resolve([]Layer{
		{Source: SourceGlobal, Path: "/home/u/.config/calliope/config.json", Values: map[string]string{"org": "acme"}},
	})

	v := cfg.Get("org")
	if v.Path != "/home/u/.config/calliope/config.json" {
		t.Errorf("Path = %q, se esperaba la ruta del fichero global", v.Path)
	}
}

func TestClaveAusenteDevuelveValorVacio(t *testing.T) {
	cfg := Resolve(nil)
	v := cfg.Get("org")
	if v.Value != "" || v.Source != SourceDefault {
		t.Errorf("clave ausente = %+v, se esperaba vacía con origen default", v)
	}
}

func TestLasCapasVaciasNoPisanALasDeMenorPrioridad(t *testing.T) {
	cfg := Resolve([]Layer{
		{Source: SourceGlobal, Values: map[string]string{"org": "acme"}},
		{Source: SourceFlag, Values: map[string]string{"org": ""}}, // flag no informado
	})

	if got := cfg.Get("org").Value; got != "acme" {
		t.Errorf("org = %q, un flag vacío no debe pisar la capa global", got)
	}
}

// Pruebas de precedencia par-a-par para cubrir todas las transiciones
func TestPrecedenciaEnvGanaAProject(t *testing.T) {
	cfg := Resolve([]Layer{
		{Source: SourceProject, Values: map[string]string{"org": "project"}},
		{Source: SourceEnv, Values: map[string]string{"org": "env"}},
	})

	if got := cfg.Get("org").Value; got != "env" {
		t.Errorf("org = %q, env debe ganar a project", got)
	}
	if got := cfg.Get("org").Source; got != SourceEnv {
		t.Errorf("origen = %q, se esperaba env", got)
	}
}

func TestPrecedenciaProjectGanaARepo(t *testing.T) {
	cfg := Resolve([]Layer{
		{Source: SourceRepo, Values: map[string]string{"org": "repo"}},
		{Source: SourceProject, Values: map[string]string{"org": "project"}},
	})

	if got := cfg.Get("org").Value; got != "project" {
		t.Errorf("org = %q, project debe ganar a repo", got)
	}
	if got := cfg.Get("org").Source; got != SourceProject {
		t.Errorf("origen = %q, se esperaba project", got)
	}
}

func TestPrecedenciaRepoGanaAGlobal(t *testing.T) {
	cfg := Resolve([]Layer{
		{Source: SourceGlobal, Values: map[string]string{"org": "global"}},
		{Source: SourceRepo, Values: map[string]string{"org": "repo"}},
	})

	if got := cfg.Get("org").Value; got != "repo" {
		t.Errorf("org = %q, repo debe ganar a global", got)
	}
	if got := cfg.Get("org").Source; got != SourceRepo {
		t.Errorf("origen = %q, se esperaba repo", got)
	}
}

func TestPrecedenciaGlobalGanaADefault(t *testing.T) {
	cfg := Resolve([]Layer{
		{Source: SourceDefault, Values: map[string]string{"org": "default"}},
		{Source: SourceGlobal, Values: map[string]string{"org": "global"}},
	})

	if got := cfg.Get("org").Value; got != "global" {
		t.Errorf("org = %q, global debe ganar a default", got)
	}
	if got := cfg.Get("org").Source; got != SourceGlobal {
		t.Errorf("origen = %q, se esperaba global", got)
	}
}

// Pruebas de los métodos exportados
func TestBaseURL(t *testing.T) {
	cfg := Resolve([]Layer{
		{Source: SourceDefault, Values: map[string]string{"base_url": "https://default.so"}},
		{Source: SourceGlobal, Values: map[string]string{"base_url": "https://global.so"}},
	})

	if got := cfg.BaseURL(); got != "https://global.so" {
		t.Errorf("BaseURL = %q, se esperaba https://global.so", got)
	}
}

func TestOrg(t *testing.T) {
	cfg := Resolve([]Layer{
		{Source: SourceGlobal, Values: map[string]string{"org": "acme"}},
	})

	if got := cfg.Org(); got != "acme" {
		t.Errorf("Org = %q, se esperaba acme", got)
	}
}

func TestOutput(t *testing.T) {
	cfg := Resolve([]Layer{
		{Source: SourceEnv, Values: map[string]string{"output": "json"}},
	})

	if got := cfg.Output(); got != "json" {
		t.Errorf("Output = %q, se esperaba json", got)
	}
}

func TestAllDevuelveCopia(t *testing.T) {
	cfg := Resolve([]Layer{
		{Source: SourceGlobal, Values: map[string]string{"org": "acme", "output": "table"}},
	})

	copia := cfg.All()
	if copia["org"].Value != "acme" {
		t.Errorf("All()[org].Value = %q, se esperaba acme", copia["org"].Value)
	}

	// Mutar la copia devuelta
	copia["org"] = Value{Value: "mutado", Source: SourceEnv}
	copia["nueva_clave"] = Value{Value: "nuevo", Source: SourceEnv}

	// Verificar que Get sigue devolviendo el valor original
	if got := cfg.Get("org").Value; got != "acme" {
		t.Errorf("org después de mutar la copia = %q, se esperaba acme (no mutado)", got)
	}
	if got := cfg.Get("nueva_clave").Value; got != "" {
		t.Errorf("nueva_clave después de añadirla a la copia = %q, se esperaba vacía", got)
	}
}

// Test para IMPORTANT 3: error corrupto debe incluir ruta y capa
func TestErrorConfigJSONCorruptaIncluyeRuta(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".calliope")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(configDir, FileName)
	if err := os.WriteFile(configFile, []byte(`{invalid json}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Simular Load llamando a readLayerFile directamente
	_, err := readLayerFile(SourceProject, configFile)
	if err == nil {
		t.Fatal("se esperaba un error por JSON corrupto")
	}

	// El mensaje debe contener la ruta del fichero y la capa
	errMsg := err.Error()
	if !contains(errMsg, configFile) {
		t.Errorf("error no contiene ruta: %s", errMsg)
	}
	if !contains(errMsg, string(SourceProject)) {
		t.Errorf("error no contiene capa: %s", errMsg)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i < len(haystack)-len(needle)+1; i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
