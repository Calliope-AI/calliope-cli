---
name: calliope
description: Consulta los datos, la documentación, la ontología y las reglas de negocio de una organización en Calliope Data mediante el CLI `calliope`. Úsalo para CUALQUIER pregunta o acción sobre Calliope Data.
---

# Calliope Data

`calliope` da acceso gobernado a los datos de una organización: preguntas en
lenguaje natural, documentación, ontología, reglas de negocio y SQL.

Comprueba que está listo con `calliope doctor`. Si falta el binario o la
credencial, ese comando dice exactamente qué ejecutar.

## Invariantes — CUMPLE estas reglas

1. **Elige el modo de salida.** `--jq '<expr>'` para extraer un campo, `--json`
   para el envelope completo, `--md` para presentar a una persona, `--quiet`
   para scripting. **Nunca** hagas pipe a un `jq` externo: el filtro va dentro
   del binario y el externo no existe en muchas máquinas.
2. **`ask` antes que `query`.** La pregunta en lenguaje natural es la vía por
   defecto. Recurre al SQL crudo solo cuando `ask` no baste o necesites una
   columna concreta.
3. **Ejecuta `calliope schema` antes de escribir SQL.** Nunca inventes nombres
   de tabla ni de columna.
4. **El scope de organización es obligatorio.** Usa `--org <nombre>` o fija una
   con `calliope orgs use <nombre>`. Si no sabes cuál, `calliope orgs list`.
5. **Cita las fuentes.** `ask` devuelve `sources`; cuando presentes el
   resultado a una persona, inclúyelas.
6. **Sigue los `breadcrumbs`.** Cada respuesta trae el siguiente comando en el
   campo `breadcrumbs`. Úsalo en vez de adivinar.

## Comandos

<!-- catalogo:inicio -->
- `calliope ask <pregunta>` — pregunta en lenguaje natural sobre tus datos y tu documentación
- `calliope auth login` — guarda y verifica una credencial de Calliope
- `calliope auth logout` — borra la credencial almacenada
- `calliope auth status` — muestra quién eres y de dónde sale la credencial
- `calliope auth token` — imprime la credencial almacenada (para scripts)
- `calliope concepts list` — lista los conceptos de negocio
- `calliope concepts show <id>` — muestra un concepto y sus atributos
- `calliope config get <clave>` — muestra el valor de una clave
- `calliope config list` — muestra cada valor con la capa de la que proviene
- `calliope config path` — imprime la ruta del fichero de configuración global
- `calliope config set <clave> <valor>` — fija una clave en la configuración de proyecto (o global con --global)
- `calliope doctor` — diagnostica la instalación, la credencial y la conectividad
- `calliope docs list` — lista los documentos disponibles
- `calliope docs search <consulta>` — búsqueda semántica en la documentación
- `calliope docs show <id>` — muestra los metadatos de un documento
- `calliope orgs list` — lista las organizaciones accesibles con tu credencial
- `calliope orgs use <organización>` — fija la organización activa en este directorio
- `calliope query <SQL>` — ejecuta SQL contra los datos de la organización
- `calliope rules list` — lista las reglas de negocio de la organización
- `calliope schema` — muestra el esquema de la base de datos de la organización
- `calliope skill` — vuelca la documentación para agentes de esta versión del CLI
- `calliope version` — muestra la versión de calliope
<!-- catalogo:fin -->

## Recetas

**Responder una pregunta de negocio y citar las fuentes:**

    calliope ask "¿cómo evolucionaron las ventas este trimestre?" --md

**Saber qué datos existen antes de preguntar:**

    calliope concepts list --jq '.data[].name'

**SQL, siempre después de mirar el esquema:**

    calliope schema --table ventas
    calliope query "SELECT mes, SUM(importe) FROM ventas GROUP BY mes" --json

**Extraer un solo campo:**

    calliope docs list --status READY --jq '.data[].id'

## Errores

Los fallos salen con esta forma, y `hint` dice qué hacer:

    {"ok": false, "error": {"code": "UNAUTHORIZED", "message": "…", "hint": "Ejecuta: calliope auth login"}}

Códigos de salida: `0` correcto · `1` error · `2` uso incorrecto ·
`3` no autorizado · `4` no encontrado · `5` límite superado.
