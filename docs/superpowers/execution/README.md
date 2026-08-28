# Registro de ejecución del plan de la v1

Este directorio es el **porqué** del CLI. El historial de git guarda qué se
escribió; esto guarda por qué se decidió así.

El CLI se construyó a partir de [el spec](../specs/2026-08-27-calliope-cli-design.md)
y [el plan](../plans/2026-08-27-calliope-cli.md), ejecutando cada tarea con un
agente distinto y revisando cada una antes de pasar a la siguiente.

## Qué hay aquí

| Fichero | Qué contiene |
|---|---|
| [`ledger.md`](ledger.md) | El registro de la ejecución, tarea a tarea: qué encontró cada revisión, qué se corrigió en cada ronda, y **los 25 rulings** que resolvieron los conflictos. Es el fichero más valioso del directorio. |
| [`glosario.md`](glosario.md) | Las traducciones de identificadores fijadas para que veinte agentes distintos no eligieran nombres distintos para lo mismo. |
| [`tipos-corregidos-del-sdk.md`](tipos-corregidos-del-sdk.md) | Los tres tipos del SDK que el plan modelaba mal, corregidos contra los tipos de `calliope-data-ui`. Sigue pendiente confirmarlos contra el backend en vivo. |
| [`informes/`](informes/) | Un informe por tarea, más el de la oleada final. Traen las tablas de pruebas por mutación y las salidas literales de las verificaciones en campo. |

## Por dónde empezar

Si solo vas a leer una cosa, lee la sección de **rulings** del ledger: son las
decisiones que cambiaron el producto, cada una con lo que cuesta si está
equivocada. Varias corrigen defectos del propio plan que solo aparecieron al
implementarlo — el `hint` de los errores que nunca llegaba al usuario, la tabla
del esquema con dos nombres distintos que el plan fundía en uno, o el script del
hook que decía «listo» con el backend caído.

## Qué se dejó fuera a propósito

- **Los diffs de revisión** (41 ficheros, 1,1 MB): son salida de `git diff` sobre
  rangos de commits. Reconstruibles con `git diff <base>..<head>`.
- **Los briefs de tarea** (21 ficheros): son extractos literales del plan, que ya
  está versionado en [`../plans/`](../plans/).

## Lo que sigue pendiente

El [README del proyecto](../../../README.md) lo declara en su sección de estado.
En corto: el smoke de extremo a extremo nunca se ha ejecutado contra un backend
real, la configuración de GoReleaser no se ha verificado con `goreleaser check`,
y el login OAuth no está implementado — solo autenticación por clave de API.
