package presenter

import (
	"io"
	"text/tabwriter"
)

// Table escribe una tabla alineada. Los comandos la usan desde su Result.Text.
func Table(w io.Writer, headers []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	writeRow(tw, headers)
	for _, r := range rows {
		writeRow(tw, r)
	}
	return tw.Flush()
}

func writeRow(w io.Writer, celdas []string) {
	for i, c := range celdas {
		if i > 0 {
			io.WriteString(w, "\t")
		}
		io.WriteString(w, c)
	}
	io.WriteString(w, "\n")
}
