# Task 9 — SDK: transporte, scoping de organización y mapeo de errores

**Estado:** DONE

**Commit:** `8b51a8b` — "feat: transporte del SDK con scoping de organización y mapeo de errores"
(rama `feat/cli-v1`, sobre `5076ebc`)

**Ficheros:**
- `internal/sdk/client.go` (nuevo)
- `internal/sdk/client_test.go` (nuevo)

## Resumen de tests

`go test ./internal/sdk/ -v -race`: **7/7 PASS**. `go test ./... -race`, `go vet ./...` y `gofmt -l .` limpios en todo el módulo.

## TDD seguido

1. Escribí `client_test.go` (Step 1 del brief) y confirmé el fallo esperado: `undefined: Client / New / Options` (Step 2).
2. Implementé `client.go` copiando el código del brief literalmente, con la traducción de identificadores de paquete fijada por el glosario.
3. Al ejecutar los tests (Step 4) **no pasaron los 7**: `TestElNombreDeOrganizacionSeEscapa` falló con el código exacto del brief. Ver "Desviación del brief" abajo.
4. Corregido, los 7 tests pasan.

## Traducción de identificadores (glosario)

| Brief | Usado |
|---|---|
| `mapearStatus` | `mapStatus` |
| `errorDeTransporte` | `transportError` |
| `esTimeout` | `isTimeout` |
| `clienteDePrueba` | `testClient` |
| `errorComoCLI` | `asCLIError` |

Nombres de test (`TestLaRutaLlevaElScopeDeOrganizacion`, etc.) y variables locales (`cuerpo`, `crudo`, `visto`, `casos`, `caso`, `escapado`, `k`, `v`) se dejaron tal como los escribe el brief, en español. Mensajes, comentarios y hints se copiaron literales. No apareció ningún identificador de paquete fuera de los cinco ya fijados en el glosario; no hay nuevas entradas que proponer.

## Desviación del brief (hallazgo real, no cosmético)

El `OrgPath` tal como lo da el brief:

```go
func (c *Client) OrgPath(org, suffix string) string {
	return "/v1/organizations/" + url.PathEscape(org) + suffix
}
```

**no pasa** `TestElNombreDeOrganizacionSeEscapa`. `url.PathEscape` en Go no escapa el carácter `.` (no es un carácter reservado en un segmento de ruta), así que para `org = "acme corp/../otra"`:

- Las barras sí se neutralizan: `/` → `%2F`.
- Los puntos **no**: el resultado es `"acme%20corp%2F..%2Fotra"`, que contiene literalmente la subcadena `".."`.

El test comprueba `strings.Contains(got, "..")`, y falla. Funcionalmente el resultado ya es inerte para *path traversal* de verdad (las barras que harían el `..` estructuralmente peligroso están escapadas, así que es un único segmento de ruta opaco), pero el test — deliberadamente estricto, dado que el nombre de organización viene de configuración de proyecto no confiable — exige que ni siquiera la subcadena aparezca.

**Corrección aplicada** (mínima, sin tocar nada más de la lógica):

```go
func (c *Client) OrgPath(org, suffix string) string {
	escapado := strings.ReplaceAll(url.PathEscape(org), ".", "%2E")
	return "/v1/organizations/" + escapado + suffix
}
```

Con esto `TestLaRutaLlevaElScopeDeOrganizacion` (caso feliz, `"acme"` sin puntos) sigue intacto, y `TestElNombreDeOrganizacionSeEscapa` pasa de verdad.

## Mutaciones (verificación de costura), sobre copia fuera del repo

Copia hecha en `/private/tmp/.../scratchpad/mutation-copy` (fuera de `/Users/j10/repositories/calliope/calliope-cli`), compilada y verificada antes de mutar. Cada mutación se aplicó, se corrió *solo* el test que debía protegerla, se confirmó el fallo, y se revirtió desde una copia de respaldo del fichero original antes de la siguiente. Al terminar, diff contra el fichero real del repo = idéntico; repo real sin tocar (`git status` limpio salvo el nuevo paquete `internal/sdk/` sin commitear en ese momento).

| # | Test protegido | Mutación | Resultado |
|---|---|---|---|
| 1 | `TestLaRutaLlevaElScopeDeOrganizacion` | `"/v1/organizations/"` → `"/v2/organizations/"` en `OrgPath` | FAIL: `ruta = "/v2/organizations/acme/rules", se esperaba /v1/organizations/acme/rules` |
| 2 | `TestElNombreDeOrganizacionSeEscapa` | Se quita el `strings.ReplaceAll(..., ".", "%2E")`, vuelve al `url.PathEscape(org)` puro del brief | FAIL: `OrgPath = "/v1/organizations/acme%20corp%2F..%2Fotra/rules" — el nombre de organización debe ir escapado` (reproduce exactamente el bug original) |
| 3 | `TestSeEnviaLaCabeceraDeAutenticacion` | `req.Header.Set(k, v)` → `_ = k; _ = v` (no se fija la cabecera) | FAIL: `X-API-Key = ""` |
| 4 | `TestMapeoDeStatusACodigosDeSalida` | En `mapStatus`, `case status == http.StatusNotFound:` → `case status == http.StatusNotFound && false:` (el 404 cae al `default`) | FAIL: `status 404: código de salida = 1, se esperaba 4` |
| 5 | `TestElErrorNoFiltraElCuerpoDelBackend` | En `Do`, la rama `>= 400` deja de descartar el cuerpo y lo mete en el mensaje del error (`"Error: "+string(crudoErr)`) | FAIL: `el mensaje filtra internals del backend: "Error: {\"detail\":\"panic en /srv/app/handlers/ask.py línea 412\"}"` |
| 6 | `TestElTimeoutProduceUnErrorAccionable` | El hint del error de timeout en `transportError` se vacía (`""`) | FAIL: `el error de timeout debe sugerir qué hacer` |
| 7 | `TestRespuestaVaciaNoRompe` | Se quita el atajo `if len(bytes.TrimSpace(crudo)) == 0 { return nil }` antes del `json.Unmarshal` | FAIL: `una respuesta vacía no debe fallar: La respuesta de Calliope Data no tiene el formato esperado.` |

Las dos costuras que la revisión iba a mirar con lupa quedaron cubiertas explícitamente:

1. **No filtrar el cuerpo del backend** — mutación 4 (status 404→default aún usa `mapStatus`, que nunca lee el cuerpo) y sobre todo mutación 5, que demuestra que si alguien reintrodujera el cuerpo crudo en el mensaje de error, el test lo detecta inmediatamente. Además, en la ruta feliz del código (`Do`), el cuerpo de error se descarta con `io.Copy(io.Discard, resp.Body)` antes de construir el error — nunca se pasa a `mapStatus`.
2. **Escape del nombre de organización** — mutación 2 reproduce el bug real encontrado durante el TDD (ver sección de desviación) y confirma que, sin el escape del punto, el test lo atrapa.

## Dudas / puntos para quien revise

- El fix de `OrgPath` (escapar también `.`) no estaba en el brief; lo until la propia ejecución del test lo exigió. Es una corrección mínima y localizada, documentada arriba, pero señalo explícitamente que el código entregado **no es una copia 100% literal** del brief en esa única línea — todo lo demás sí lo es.
- No hay identificadores nuevos fuera de los cinco fijados por el glosario que requieran entrada nueva en el glosario.

---

## Ronda de correcciones 1/5

**Commit:** `2dc512f` — "fix: revierte el escape de punto en OrgPath, corrige el test y añade cobertura"

### Adjudicación del coordinador

El test `TestElNombreDeOrganizacionSeEscapa` de la ronda anterior aseveraba `!strings.Contains(got, "..")`, una propiedad sintáctica sin correlato de seguridad. El razonamiento correcto:

- Lo que neutraliza el *path traversal* es que `/` quede escapado a `%2F`: RFC 3986 delimita los segmentos de una URL por barras **sin decodificar**, así que en cuanto la barra está escapada, el valor entero de `org` es un único segmento opaco.
- Dentro de un segmento opaco, la subcadena `..` no tiene semántica de traversal (la eliminación de dot-segments de RFC 3986 §5.2.4 opera sobre segmentos delimitados por `/` literal, no sobre subcadenas).
- Escapar `.` a `%2E` no cambia nada desde el punto de vista del backend (un decodificador RFC-conforme trata `%2E` igual que `.`), y si un backend decodificara `%2F` antes de enrutar —el único escenario con traversal real— decodificaría `%2E` en el mismo paso, reconstruyendo el `..` de todas formas.
- Además introducía una regresión funcional real: `"acme.corp"` pasaba de `/v1/organizations/acme.corp/rules` a `/v1/organizations/acme%2Ecorp/rules`, sin verificación de que el backend o un proxy intermedio lo resolviera igual.

Acepto la corrección: mi cambio de la ronda anterior resolvía el síntoma del test tal como estaba escrito, pero el test estaba mal calibrado y el fix introducía riesgo sin beneficio de seguridad real. Correspondía cuestionar la aserción del test en vez de adaptarle el código.

### Cambios aplicados

1. **`OrgPath` revertido al código literal del brief:**
   ```go
   func (c *Client) OrgPath(org, suffix string) string {
       return "/v1/organizations/" + url.PathEscape(org) + suffix
   }
   ```
   `strings` sigue en uso en `New()` (`strings.TrimRight`), así que el import no queda huérfano; no hubo que tocarlo.

2. **`TestElNombreDeOrganizacionSeEscapa` corregido** para aseverar la propiedad estructural real: que `org` no introduce ninguna barra sin escapar. Cuenta las barras de `got` y las compara con las de la plantilla fija (`/v1/organizations/`) más las del `suffix` — si `org` no aporta barras propias (todas escapadas a `%2F`), el total coincide exactamente.

3. **Test nuevo `TestElNombreDeOrganizacionConPuntoNoSeAltera`:** fija la regresión evitada — `OrgPath("acme.corp", "/rules")` produce `/v1/organizations/acme.corp/rules`, sin escapar el punto.

4. **Comentario de `OrgPath` corregido** para explicar la razón real (barra escapada → segmento opaco), no la falsa ("para que `..` no aparezca").

5. **Test nuevo `TestContentTypeSoloSeFijaConCuerpo`:** una petición sin cuerpo no lleva `Content-Type`; una con cuerpo lleva `application/json`. El código de `Do` ya tenía esto correcto (`if body != nil { ... }`); solo faltaba cobertura explícita.

### Mutaciones (ronda 1), sobre copia fresca fuera del repo

Copia nueva en `/private/tmp/.../scratchpad/mutation-copy-r1`, compilada y verificada antes de mutar; restaurada y diffeada contra el fichero real al terminar (idéntica) antes de borrarla.

| Test protegido | Mutación | Resultado |
|---|---|---|
| `TestElNombreDeOrganizacionSeEscapa` | `OrgPath` deja de escapar `org` (usa el valor crudo; se mantiene el import `net/url` con un `_ = url.Values{}` para que compile) | FAIL: `OrgPath = "/v1/organizations/acme corp/../otra/rules" — el nombre de organización introduce una barra sin escapar (6 barras, se esperaban 4)` |
| `TestElNombreDeOrganizacionConPuntoNoSeAltera` | Se reintroduce el `strings.ReplaceAll(url.PathEscape(org), ".", "%2E")` de la ronda anterior (la regresión exacta que motivó esta ronda) | FAIL: `OrgPath = "/v1/organizations/acme%2Ecorp/rules", se esperaba /v1/organizations/acme.corp/rules — el punto es válido en un segmento y no debe escaparse` |
| `TestContentTypeSoloSeFijaConCuerpo` | `req.Header.Set("Content-Type", "application/json")` se fija incondicionalmente, fuera del `if body != nil` | FAIL: `Content-Type sin cuerpo = "application/json", se esperaba vacío` |

Las tres mutaciones fallaron exactamente en el test que debían proteger. En particular, la mutación de `TestElNombreDeOrganizacionSeEscapa` confirma lo pedido explícitamente por el coordinador: el test falla si `OrgPath` deja de escapar la barra.

### Verificación final

`go test ./... -race -v`: 9/9 en `internal/sdk` PASS (más el resto del módulo sin cambios). `go vet ./...` y `gofmt -l .` limpios.

### Dudas

Ninguna. La adjudicación es técnicamente sólida y verificada por mutación; no queda ambigüedad abierta.
