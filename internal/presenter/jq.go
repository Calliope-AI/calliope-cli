package presenter

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/itchyny/gojq"

	"github.com/calliope/calliope-cli/internal/output"
)

// renderJQ aplica una expresión jq al envelope. El filtro va embebido en el
// binario a propósito: el SKILL.md prohíbe el pipe a un jq externo, que falla
// en las máquinas donde no está instalado.
func renderJQ(w io.Writer, env output.Envelope, expr string) error {
	query, err := gojq.Parse(expr)
	if err != nil {
		return output.NewError(output.CodeUsage,
			fmt.Sprintf("Expresión jq inválida: %v", err),
			"Consulta la sintaxis en https://jqlang.github.io/jq/manual/")
	}

	// gojq opera sobre tipos genéricos, así que damos la vuelta por JSON.
	crudo, err := json.Marshal(env)
	if err != nil {
		return err
	}
	var entrada any
	if err := json.Unmarshal(crudo, &entrada); err != nil {
		return err
	}

	iter := query.Run(entrada)
	for {
		v, ok := iter.Next()
		if !ok {
			return nil
		}
		if err, ok := v.(error); ok {
			return output.NewError(output.CodeUsage,
				fmt.Sprintf("Error al evaluar la expresión jq: %v", err), "")
		}
		if err := escribirJSON(w, v); err != nil {
			return err
		}
	}
}
