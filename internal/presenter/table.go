package presenter

import (
	"io"
	"text/tabwriter"
)

// Table escribe una tabla alineada. Los comandos la usan desde su Result.Text.
func Table(w io.Writer, headers []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	escribirFila(tw, headers)
	for _, r := range rows {
		escribirFila(tw, r)
	}
	return tw.Flush()
}

func escribirFila(w io.Writer, celdas []string) {
	for i, c := range celdas {
		if i > 0 {
			io.WriteString(w, "\t")
		}
		io.WriteString(w, c)
	}
	io.WriteString(w, "\n")
}
