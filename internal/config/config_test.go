package config

import "testing"

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
