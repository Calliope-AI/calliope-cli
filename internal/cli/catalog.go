package cli

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// CommandInfo es una hoja del árbol de comandos.
type CommandInfo struct {
	Path  string `json:"path"`
	Short string `json:"short"`
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
			out = append(out, CommandInfo{Path: strings.Join(ruta, " "), Short: hijo.Short})
		}
	}
	recorrer(root, nil)

	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
