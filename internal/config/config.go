// Package config resuelve la configuración de calliope a partir de capas
// ordenadas, recordando de dónde salió cada valor. Cuando alguien se pregunta
// por qué el CLI apunta a una organización, la respuesta está en un comando.
package config

// Source identifica de qué capa proviene un valor.
type Source string

const (
	SourceFlag    Source = "flag"
	SourceEnv     Source = "env"
	SourceProject Source = "project"
	SourceRepo    Source = "repo"
	SourceGlobal  Source = "global"
	SourceDefault Source = "default"
)

// priority ordena las capas de mayor a menor precedencia.
var priority = map[Source]int{
	SourceFlag:    6,
	SourceEnv:     5,
	SourceProject: 4,
	SourceRepo:    3,
	SourceGlobal:  2,
	SourceDefault: 1,
}

// Claves reconocidas por la configuración.
const (
	KeyOrg     = "org"
	KeyBaseURL = "base_url"
	KeyOutput  = "output"
	KeyTimeout = "timeout"
)

// Value es un valor resuelto junto con su procedencia.
type Value struct {
	Value  string `json:"value"`
	Source Source `json:"source"`
	Path   string `json:"path,omitempty"`
}

// Layer es una fuente de valores sin resolver.
type Layer struct {
	Source Source
	Path   string
	Values map[string]string
}

// Config es el resultado de fusionar las capas.
type Config struct {
	values map[string]Value
}

// Resolve fusiona las capas respetando la precedencia. Los valores vacíos no
// cuentan: un flag no informado no debe pisar a la capa de debajo.
func Resolve(layers []Layer) *Config {
	c := &Config{values: map[string]Value{}}
	for _, l := range layers {
		for k, v := range l.Values {
			if v == "" {
				continue
			}
			actual, existe := c.values[k]
			if existe && priority[actual.Source] >= priority[l.Source] {
				continue
			}
			c.values[k] = Value{Value: v, Source: l.Source, Path: l.Path}
		}
	}
	return c
}

// Get devuelve el valor de una clave; si no existe, uno vacío de origen default.
func (c *Config) Get(key string) Value {
	if v, ok := c.values[key]; ok {
		return v
	}
	return Value{Source: SourceDefault}
}

// All devuelve todos los valores resueltos, para `calliope config list`.
func (c *Config) All() map[string]Value {
	copia := make(map[string]Value, len(c.values))
	for k, v := range c.values {
		copia[k] = v
	}
	return copia
}

func (c *Config) BaseURL() string { return c.Get(KeyBaseURL).Value }
func (c *Config) Org() string     { return c.Get(KeyOrg).Value }
func (c *Config) Output() string  { return c.Get(KeyOutput).Value }
