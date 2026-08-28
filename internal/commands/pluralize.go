package commands

import "fmt"

// pluralize antepone n a singular o plural según corresponda -en español,
// solo 1 usa la forma singular; 0, y cualquier valor negativo o mayor que 1,
// usan la plural, igual que "0 manzanas" o "-1 días"- (Diferidos #12 y #16
// de la oleada final). Centraliza el fallo que aparecía en varios summary
// del envelope: "1 filas", "1 tablas", "1 documentos", "1 conceptos", "1
// reglas", "1 organizaciones", "1 valores", "0 fuentes citadas"... un
// número interpolado delante de un sustantivo fijo en plural, cadena de
// cara al cliente en un producto en español.
func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
}
