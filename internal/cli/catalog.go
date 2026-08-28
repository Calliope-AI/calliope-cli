package cli

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// CommandInfo es una hoja del árbol de comandos: su ruta, su descripción
// corta y la forma de sus argumentos tal como aparece en Use (p. ej. "<id>"
// o "<clave> <valor>"; vacío si el comando no toma ninguno).
type CommandInfo struct {
	Path  string `json:"path"`
	Short string `json:"short"`
	Args  string `json:"args"`
}

// Catalog recorre el árbol y devuelve sus hojas, ordenadas. Se deriva del
// árbol real en vez de mantenerse a mano: así no puede desviarse de lo que el
// binario registra.
func Catalog(root *cobra.Command) []CommandInfo {
	var out []CommandInfo
	var recorrer func(c *cobra.Command, prefijo []string)

	recorrer = func(c *cobra.Command, prefijo []string) {
		for _, hijo := range c.Commands() {
			if hijo.Hidden || hijo.Name() == "help" || hijo.Name() == "completion" {
				continue
			}
			ruta := append(append([]string{}, prefijo...), hijo.Name())
			if len(hijo.Commands()) > 0 {
				recorrer(hijo, ruta)
				continue
			}
			// hijo.Use siempre empieza por hijo.Name() -es justo de ahí de
			// donde Cobra saca el nombre-, así que lo que sobra tras
			// recortarlo es la forma de los argumentos.
			args := strings.TrimSpace(strings.TrimPrefix(hijo.Use, hijo.Name()))
			out = append(out, CommandInfo{Path: strings.Join(ruta, " "), Short: hijo.Short, Args: args})
		}
	}
	recorrer(root, nil)

	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
