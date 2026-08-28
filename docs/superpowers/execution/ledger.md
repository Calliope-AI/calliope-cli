# SDD ledger — plan: docs/superpowers/plans/2026-08-27-calliope-cli.md

Spec: docs/superpowers/specs/2026-08-27-calliope-cli-design.md (leído, es la autoridad vinculante)
Rama: feat/cli-v1 (consentida por el usuario; sin worktree, repo nuevo sin remoto)

## Escaneo previo de conflictos

### Pares de tareas que comparten fichero o interfaz

| Productor | Consumidor | Qué se produce → qué se consume | Resultado |
|---|---|---|---|
| T1 | T2 | `cmd/calliope/main.go` creado → modificado para `output.ExitCodeFor` | OK |
| T1 | T12 | `root.go` declara flags en línea → T12 los sustituye por `appctx.RegisterGlobalFlags` | OK, T12 lo dice explícitamente |
| T1 | T12 | `NewRootCmd()` → `NewRootCmd(d appctx.Deps)`; tests de T1 actualizados | OK, T12 paso 5 |
| T1 | T21 | `newVersionCmd()` → `newVersionCmd(d)` | OK — ver hallazgo P2 |
| T2 | T3 | `Envelope`, `OKEnvelope`, `CLIError`, `NewError`, `CodeUsage`, `ExitCodeFor` | OK, firmas coinciden |
| T2 | T9 | `NewError` + códigos de error | OK |
| T4 | T5 | `sanear(Layer) (Layer, []string)` provisional → definitivo | OK, T4 paso 5 crea el provisional para que compile |
| T4 | T11 | `config.Load(cwd, env, flags) (*Config, []string, error)` | OK |
| T4/T5 | T13 | `ProjectAllowed`, `GlobalPath`, `FileName`, `KeyOrg` | OK |
| T7 | T9 | `Credential.Header() (string, string)` | OK |
| T7 | T11 | `Resolve(env, store)`, `DefaultStore(dirGlobal)` | OK, T11 pasa `filepath.Dir(config.GlobalPath(env))` |
| T9 | T10 | `(*Client).Do`, `(*Client).OrgPath` | OK |
| T10 | T14-T18 | los 11 métodos del SDK | OK, firmas coinciden una a una |
| T11 | T12-T18 | `Build`, `BuildSinCredencial`, `Deps`, `Context.Render`, `Context.Deps` | OK; T17 usa `ctx.Deps.Stdout` y `Context` tiene el campo |
| T12 | T13-T18 | ayudantes de test `depsConServidor`, `raizDePrueba` | OK, mismo paquete |
| T12 auth.go | T12 orgs.go | `authResolve`, `clienteCon` | OK |
| T15 docs.go | T16 knowledge.go | `recortar(string, int) string` | OK, T15 va antes |
| T19 | T20 | `skills/calliope/SKILL.md` | OK, el plugin lo enlaza |
| T19 | T12-T18 | catálogo (22 hojas) vs lista del SKILL.md | OK, verificadas las 22 una a una |
| T1 | T21 | ruta del módulo en los ldflags vs `go mod init` | OK, `github.com/calliope/calliope-cli` |
| T21 Makefile | T22 | `bin/calliope` vs `../../bin/calliope` del e2e | OK |

### Coherencia interna de cada tarea

| Tarea | ¿Su texto concuerda consigo mismo? |
|---|---|
| T1 | Sí |
| T2 | Sí |
| T3 | Sí (`escribirJSON` en presenter.go, usado desde jq.go, mismo paquete) |
| T4 | Sí, con el provisional del paso 5 |
| T5 | Sí |
| T6 | Sí — spike sin tests, coherente con su propio encuadre |
| T7 | Sí |
| T8 | Sí, condicional a T6 |
| T9 | Sí |
| T10 | Sí; el paso 7 marca explícitamente los 3 tipos sin verificar |
| T11 | Sí |
| T12 | Sí |
| T13 | Sí |
| T14 | Sí |
| T15 | Sí |
| T16 | Sí |
| T17 | Sí |
| T18 | **No del todo** — ver hallazgo P3 |
| T19 | Sí |
| T20 | Sí |
| T21 | Sí, salvo P2 |
| T22 | Sí |

### Hallazgos y rulings

**P1 — BLOQUEANTE. La máquina tiene Go 1.19; las Global Constraints exigen Go 1.22+.**
Único toolchain: `/usr/local/go` (tarball oficial). Sin brew-go, sin `~/sdk`.
Go 1.19 no tiene descarga automática de toolchain, y gojq y GoReleaser v2 exigen 1.21+.
Ruling: NO se puede rular. Instalar un toolchain es una descarga y un efecto fuera
del workspace → requiere permiso explícito del usuario. Preguntado antes de T1.

**P2 — `newVersionCmd` se toca en T12 y en T21.**
T21 repite «actualiza los tests de la Task 1», que T12 ya hizo.
Ruling: redundancia inocua, no conflicto. El ejecutor de T21 encontrará el trabajo
hecho y seguirá. Coste si me equivoco: unos minutos de confusión en T21.

**P3 — T18 (doctor) usa literales `"base_url"` y `"org"` donde el resto del plan usa
`config.KeyBaseURL` y `config.KeyOrg`.**
Ruling: usar las constantes. Es la convención del resto del plan y un literal mal
escrito fallaría en silencio (`Get` devuelve un `Value` vacío, no un error).
Se lleva en el dispatch de T18. Coste si me equivoco: ninguno, es estrictamente mejor.

## Progreso

P1 RESUELTO: usuario autorizó `brew install go` → Go 1.27.0 en /opt/homebrew/bin/go,
por delante de /usr/local/go/bin en el PATH. GOTOOLCHAIN=auto. El 1.19 sigue instalado
pero queda ensombrecido globalmente (difiere de lo que anticipé al preguntar; avisado).

BASE de la Task 1: 2bc15dde762f5fa1212314a6644af79944c74592

Task 1: DONE_WITH_CONCERNS del implementer. Dos hallazgos suyos, ambos reales:
  (a) mi .gitignore tenía un patrón `calliope` pelado que ocultaba cmd/calliope/.
      Lo ancló a `/calliope` y añadió `bin/`. Correcto.
  (b) el .gitignore global del usuario (~/.gitignore_global) tiene una entrada
      `version` que oculta internal/version/ entera; usó `git add -f`.
Ruling: (b) no se arregla con `add -f`, porque volvería a morder en la Task 21
(internal/version/check.go). Añadida reinclusión al .gitignore del repo
(!internal/version/ y !internal/version/**) y verificada con git status sobre
ficheros nuevos. Coste si me equivoco: ninguno, el ámbito es este repo.

Task 1: complete (commits 2bc15dd..91aac85, review clean — brief ✅, 0 Critical, 0 Important)
Task 1: minor (deferred): TestRootMuestraAyudaSinArgumentos asevera solo Contains("calliope"),
  que es trivialmente cierto con Use:"calliope". Viene de mi plan, no del implementer.
  Riesgo real: que el patrón de test superficial se propague a las 21 tareas restantes.
Task 1: minor (deferred): fmt.Fprintf descarta el error de escritura en root.go (viene del plan).
Task 2: BASE=91aac85ab68470f6a6c88d6892c3d104f92ba7a8

Task 2: revisión — brief ✅ (idéntico byte a byte), pero 1 Critical y 1 Important.
Task 2: Ruling: el hallazgo Critical es PLAN-MANDATED — el `main.go` y el
  `CLIError.Error()` defectuosos los prescribe literalmente mi plan. Gana el hallazgo:
  el spec §6.3 es la autoridad vinculante y promete
  `{"ok":false,"error":{"code","message","hint"}}`, además de declarar el `hint`
  obligatorio «siempre que exista una acción de recuperación». Con el plan tal cual,
  el hint es dato muerto: Error() devuelve solo Message y main.go imprime eso.
  Verificado por inspección de las tres líneas implicadas.
  Se corrige AQUÍ, en la Task 2, porque 20 tareas posteriores construyen `return err`
  sobre este mismo patrón y arreglarlo después costaría tocarlas todas.
  El texto del plan para cmd/calliope/main.go queda SUPERSEDED por esta corrección;
  se lleva a los dispatches posteriores que lo mencionen (T11, T21).
  Coste si me equivoco: main.go queda con lógica de presentación que la Task 11
  podría querer mover a appctx. Barato de mover, caro de no tener.
Task 2: minor (deferred): OKEnvelope no tiene ningún test que la invoque.
Task 2: fix round 1/5 (1 addressed, 1 open — hint corregido en main.go pero sin tests que lo cubran; commits ff2e53b..76ea2cd)
Task 2: fix round 2/5 (4 addressed, 0 open — WriteError extraída y testeada, errores genéricos mapeados, errors.As; commits 76ea2cd..33aef18)
Task 2: complete (commits 91aac85..33aef18, review clean — verificado con mutation testing)
Task 2: nota para T12: el texto del plan sobre cmd/calliope/main.go quedó superseded.
  main() ahora es: Execute() → output.WriteError(os.Stderr, err, isJSON) → os.Exit(ExitCodeFor(err)).
  T12 debe cambiar solo NewRootCmd() por NewRootCmd(appctx.DefaultDeps()), sin tocar lo demás.
Task 3: BASE=33aef185f47e436e3dda91358a0df424e50d2836

Task 3: revisión — brief ✅, 0 Critical, 2 Important (costuras sin test, verificadas por mutación), 3 Minor.
Task 3: Ruling: PLAN-MANDATED — el plan introduce identificadores en español
  (escribirJSON, escribirFila, resultadoDePrueba) contra su propia restricción global
  «identificadores en inglés». Mi escaneo previo no lo detectó. Gana la restricción:
  internal/output e internal/cli ya están en inglés, y dejarlo pasar produce un
  codebase mitad y mitad. Fijado el glosario en glosario.md, vinculante para las 19
  tareas restantes, para que dos agentes no elijan nombres distintos para el mismo
  ayudante compartido entre tareas (p. ej. `recortar` la define T15 y la usa T16).
  Los nombres de los tests se quedan en español: son prosa descriptiva, no API.
  Coste si me equivoco: renombrados mecánicos y reversibles.
Task 3: fix round 1/5 (4 addressed, 0 open — 4 tests nuevos + renombres del glosario; commits d10a730..3808f2a)
Task 3: complete (commits 33aef18..3808f2a, review clean — 3 costuras confirmadas por mutación)
Task 3: minor (deferred): TestTableAlinea asevera solo presencia de texto y número de
  líneas, no posiciones de columna; pasa igual si se sustituye tabwriter por separación fija.
Task 4: BASE=3808f2a09c69fe4a750fbf189a836a2e38b46dc8

Task 4: revisión — brief ✅, 0 Critical, 3 Important, 2 Minor.
Task 4: Ruling (AFINA el ruling del glosario de la Task 3): el glosario se aplica a
  identificadores de nivel de paquete — tipos, funciones, constantes, campos y tags json.
  Las variables LOCALES dentro de un cuerpo de función se quedan como las escribe el brief.
  Razón: los nombres de paquete son la superficie que otras tareas y futuros mantenedores
  leen y referencian entre ficheros; una variable local se lee en un tramo de cinco líneas
  rodeada de comentarios en español. Forzar su traducción en las 18 tareas restantes
  multiplica el riesgo de error de transcripción contra el brief a cambio de casi nada.
  Es un ESTRECHAMIENTO consciente de la restricción literal «identificadores en inglés».
  Coste si me equivoco: quedan locales en español; renombrarlas después es mecánico.
Task 4: minor (deferred): repoRoot exige que .git sea un directorio, así que no detecta
  la raíz en un git worktree ni en un submódulo, donde .git es un fichero. Viene del plan.
Task 4: fix round 1/5 (3 addressed, 0 open — precedencia par a par, tests de los 4 métodos, errores con ruta; commits 0680056..115a360)
Task 4: complete (commits 3808f2a..115a360, review clean — 4 costuras de precedencia confirmadas por mutación)
Task 4: minor (deferred): helper contains() en config_test.go reimplementa strings.Contains.
Task 4: minor (deferred): errata en el nombre TestErrorConfigJSONCorruptaInclujeRuta ("Incluje").
Task 5: BASE=115a3607b6d97be7b6ca5a3ce5c93d27617d82d6

Task 5: revisión de seguridad — brief ✅, 0 Critical, 1 Important, 2 Minor.
  Verificado por sonda independiente: mayúsculas, espacios, duplicados y homoglifos
  cirílicos no atraviesan la frontera. Es allowlist real, no denylist.
  El implementer añadió por su cuenta un test de la capa SourceRepo que cierra un hueco
  real de mi brief: con el guard restringido a SourceProject, los 3 tests del brief
  seguían en verde. Mérito suyo.
Task 5: CAMBIO DE INTERFAZ para la Task 13 — `config.ProjectAllowed` (mapa exportado
  mutable) pasa a no exportado, y se expone `config.IsProjectAllowed(key string) bool`.
  El brief de la T13 usa `config.ProjectAllowed[clave]`: debe usar la función.
  Se lleva en el dispatch de la T13.
Task 5: fix round 1/5 (6 addressed, 0 open — projectAllowed sin exportar, IsProjectAllowed,
  comentario de seguridad, test que fija el conjunto, camino positivo de output, rama anotada;
  commits 99e06ab..2fd6fcd)
Task 5: complete (commits 115a360..2fd6fcd, review clean — frontera verificada por mutación
  y contra homoglifos, mayúsculas, espacios y claves duplicadas)

Task 6: APLAZADA. Ruling: el spike de OAuth exige acciones humanas que no puedo ejecutar —
  registrar http://127.0.0.1:8976/callback en el dashboard de PropelAuth y completar un
  login real en el navegador. No es un bloqueo del plan: la Task 6 solo condiciona la
  Task 8, y el resto del grafo no depende de ella.
  Plan de contingencia, que es el que el propio plan prescribe para un 401: si al llegar
  a la Task 8 el spike sigue sin ejecutarse, la v1 sale SOLO con API key, que cubre todo
  el alcance pedido, y OAuth pasa a v2. La interfaz de credenciales de la Task 7 hace que
  ese aplazamiento no cueste reescribir ningún comando.
  Coste si me equivoco: ninguno inmediato; OAuth llega en v2 en vez de v1.
  ACCIÓN PENDIENTE DEL USUARIO si quiere OAuth en la v1: ejecutar la Task 6.
Task 7: BASE=2fd6fcde1c64e7995a639be4a4fc20475714eea6
Task 7: 3 preguntas del implementer, las 3 resueltas a su favor:
  (1) 4 traducciones nuevas (keyringService, keyringUser, path, fallback) → añadidas al glosario.
  (2) mi brief se contradice: "Produces" dice DefaultStore(globalDir), el código dice dirGlobal.
      Ruling: vale `globalDir`. Es el nombre en inglés y el que anuncia la interfaz.
      Coste si me equivoco: renombrar un parámetro.
  (3) 3 tests extra de keyringStore con keyring.MockInit() → correcto, cubre un hueco de mi
      brief sin tocar el Keychain real del usuario.
Task 7: fix round 1/5 (4 addressed, 0 open — chmod explícito, invariante de un solo almacén,
  errores envueltos con hint en español, String() que redacta el token; commits 7cbd222..5076ebc)
Task 7: complete (commits 2fd6fcd..5076ebc, review clean — 7 mutaciones, todas fallaron su test)
Task 7: minor (deferred): sin test del escenario «el llavero ya tenía entrada y la siguiente
  Set falla»; la API de mock de go-keyring no lo permite sin un doble propio del proveedor.
  Confirmado por el re-revisor con una mutación: el hueco es real y no encubre nada más.
Task 7: minor (deferred): escritura no atómica en fileStore.Save (sin temp+rename).

Task 8: SALTADA. Depende de la Task 6, aplazada por requerir acciones humanas.
  Se aplica el plan de contingencia: v1 solo con API key.
Task 9: BASE=5076ebc123fc1b579c96c8b1fdcc1108fd7b1798

Task 9: fix round 1/5 (5 addressed, 0 open — OrgPath revertido al brief, test recalibrado,
  regresión de acme.corp fijada, comentario corregido, Content-Type cubierto; commits 8b51a8b..2dc512f)
Task 9: complete (commits 5076ebc..2dc512f, review clean)
Task 9: Ruling: el escape del punto que añadió el implementer era seguridad aparente
  (%2E se decodifica a "." en el servidor) y rompía nombres legítimos tipo acme.corp.
  El fallo era de MI test, que confundía «aparece la subcadena ..» con «hay traversal».
  La defensa real es el escape de "/" a %2F: RFC 3986 delimita segmentos por barras sin
  decodificar, así que org queda como un segmento opaco. Arreglado el test, revertido el código.
  Coste si me equivoco: ninguno; el comportamiento vuelve al del brief, ya revisado.

Task 10: Ruling: el Step 7 del brief exige verificar SchemaResponse, Me y Organization
  contra el backend con una API key. No hay credencial disponible. En vez de parar,
  los verifiqué contra calliope-data-ui/types/index.ts, que es el cliente que consume
  hoy esos mismos endpoints. Los TRES tipos de mi plan eran incorrectos:
    - Me: el identificador es `id`, no `userId`. Y organizations trae userRole.
    - SchemaTable: tiene tableName (el identificador SQL) Y name (nombre de negocio).
      Mi plan los fundía en un solo campo. Afecta a `schema --table` de la T17.
    - Organization: timestamps en snake_case, a diferencia del resto de la API.
  Corrección escrita en task-10-tipos-corregidos.md, vinculante para el implementer.
  Coste si me equivoco: la UI podría estar desfasada respecto al backend; el smoke e2e
  de la T22 lo detectaría en cuanto haya credenciales.
  ACCIÓN PENDIENTE DEL USUARIO: confirmar los tres tipos contra data-0 con una API key.
Task 10: fix round 1/5 (3 addressed, 0 open — tests de los 5 métodos huérfanos, caminos de
  Rows(), guard simplificado; commits d27d5c8..9e8a3b4)
Task 10: complete (commits 2dc512f..9e8a3b4, review clean — 8/8 mutaciones detectadas)
Task 10: nota valiosa para el resto del proyecto: encoding/json empareja claves de forma
  insensible a mayúsculas como fallback, así que un tag mal capitalizado decodifica igual
  y ningún test lo detecta. Solo la comparación visual contra el contrato lo caza.
  Y: un id con espacio NO discrimina el escape de rutas (net/url reescapa el espacio
  igualmente); hace falta una barra, que sí es carácter estructural.
Task 11: BASE=9e8a3b448161b992f56ce3e6dae7aedfaf5429e0
Task 11: Ruling (DERIVA ENTRE TAREAS que ninguna revisión vio, porque cada una miraba
  solo su diff): el directive `go` de go.mod pasó de 1.22 (T1) a 1.24.0 (T3, al entrar
  gojq) a 1.25.0 (T11, al entrar x/term), contra la restricción global «go.mod declara go 1.22».
  Verificado: golang.org/x/term v0.45.0 exige go 1.25.0 y gojq 1.24.0, así que 1.22 es
  imposible sin congelar dependencias — justo lo que el usuario descartó al elegir instalar
  Go moderno en vez de bajar el plan.
  Ruling: se ENMIENDA la restricción global a `go 1.25`. Razón: el CLI se distribuye como
  binario estático, así que el directive no afecta a ningún usuario final; solo obliga a
  quien compile desde fuente, que es el equipo. Congelar dependencias para sostener un
  número que no beneficia a nadie sería peor.
  CONSECUENCIA PARA LA TASK 21: el workflow de CI del plan fija `go-version: '1.22'` y
  rompería. Debe pasar a '1.25'. Lo mismo el ci.yml que creó la Task 1.
  Coste si me equivoco: quien compile desde fuente necesita Go 1.25+.
Task 11: hallazgo colateral: todas las dependencias directas (gojq, go-keyring, x/term)
  están marcadas `// indirect` en go.mod. Señal de que nunca se ejecutó `go mod tidy`.
Task 11: fix round 1/5 (3 addressed, 0 open — test de Build sin credencial, go mod tidy,
  CI a Go 1.25; commits ec1a472..789249c)
Task 11: complete (commits 9e8a3b4..789249c, review clean)
Task 11: Ruling: el implementer encontró un bug REAL en el código de mi brief —
  RegisterGlobalFlags declaraba los flags como persistentes, pero cmd.Flags() no los ve
  hasta que cobra los fusiona en Execute(), así que el fixture los fijaba sobre un FlagSet
  que los ignoraba y el error quedaba tragado por un `_ =`. Su arreglo (AddFlagSet dentro
  de RegisterGlobalFlags) queda aprobado: el revisor leyó las fuentes de cobra/pflag y lo
  verificó con un árbol real de dos niveles. Es idempotente y comparte punteros, así que
  el merge posterior de cobra no duplica nada. Arreglar el código y no el fixture es lo
  correcto: si no, las tareas 12-18 volverían a caer en la misma trampa silenciosa.
Task 11: Ruling: Context.Deps, campo no previsto en el brief, se CONFIRMA como intencional.
  La Task 17 lo necesita para escribir CSV directamente a stdout.
Task 12: BASE=789249c6f6ca4be95f7234d258caae9522af8235

Task 12: revisión — brief ✅, 0 Critical, 2 Important, 4 Minor. Prueba de campo con el
  binario real: la ayuda sale en español, los grupos pelados no ejecutan nada (exit 0),
  y `auth login` sin credencial da hint y exit 2.
Task 12: Ruling: los errores de `cobra.ExactArgs` salen en inglés, sin hint y con exit 1,
  incumpliendo dos restricciones globales. Es un PATRÓN, no un caso suelto: las tareas
  14-17 usarán ExactArgs en ask, docs show, docs search, concepts show y query.
  Se resuelve una sola vez en la Task 12 con un ayudante en internal/commands, para que
  las tareas siguientes lo hereden. Se lleva en sus dispatches.
  Coste si me equivoco: un ayudante de más en el paquete commands.
Task 12: minor (deferred): los errores de E/S al guardar credencial o config se propagan
  en crudo, sin CLIError ni hint (disco lleno, permisos). Viene del plan.
Task 12: minor (deferred): el boilerplate de Cobra ("Usage:", "Flags:") sigue en inglés.
  Solo las descripciones están en español. Localizarlo exige plantillas propias de Cobra.
Task 12: fix round 1/5 (5 addressed — ayudante exactArgs, tests de auth token, IsTTY, preservación, grupo)
Task 12: fix round 2/5 (3 addressed — exactArgs deriva el uso de UseLine, exceso de args, presenter sin escapado HTML)
Task 12: fix round 3/5 (1 addressed — WriteError sin escapado HTML; commits 7619dfd..448495a)
Task 12: complete (commits 789249c..448495a, review clean — verificado comparando binarios antes/después)
Task 12: Ruling: el escapado HTML afectaba a DOS serializadores (presenter y output.WriteError,
  que no pasa por el presenter). Arreglar solo uno dejaría el mismo contrato JSON
  comportándose distinto en éxito y en error. Se autorizó tocar output/errors.go, ya cerrado,
  solo para esa línea. Verificado que envelope, modo texto, mapeo genérico y salto de línea
  quedan idénticos byte a byte.
  Coste si me equivoco: una línea revertible en un fichero ya revisado.
Task 13: BASE=448495a8d9cf5a6b2249115ccd5148c6d9d03bab
Task 13: fix round 1/5 (5 addressed, 0 open — tests de --global, directorio a 0700 con chmod,
  get respeta su argumento, fusión cubierta, permisos aseverados; commits 4ace6ae..82ffc63)
Task 13: complete (commits 448495a..82ffc63, review clean)
Task 13: Ruling: la asimetría de reforzar permisos del directorio pero no del fichero queda
  ACEPTADA. config.json solo contiene org, base_url, output y timeout — los mismos valores
  que `config list` imprime en claro. Los secretos viven en credentials.json, que sí endurece
  fichero y directorio. El directorio sí merece refuerzo porque afecta a todo lo que se cree
  dentro, incluido el credentials.json de un `auth login` posterior.
  Coste si me equivoco: cuatro valores no sensibles legibles por otros usuarios locales.
Task 14: BASE=82ffc63d0f02aba3e36b08cbae8f037022a682a4
Task 14: complete (commits 82ffc63..26c8df3, review clean SIN ronda de correcciones —
  0 Critical, 0 Important. El implementer añadió por su cuenta la cobertura de Text con
  IsTTY:true que yo había avisado, más los caminos de success:false y --action.)
Task 14: minor (deferred): «0 fuentes citadas» es gramaticalmente torpe cuando no hay
  fuentes. Viene literal de mi brief.
Task 15: BASE=26c8df31ed8c00e822dcfeea19b10c571674f06b
Task 15: fix round 1/5 (1 addressed, 0 open — valores exactos de los 5 filtros de docs list;
  commits 6554fce..14ec7a5, solo test, el cableado ya era correcto)
Task 15: complete (commits 26c8df3..14ec7a5, review clean)
Task 15: minor (deferred): truncate corta por runas, no por grafemas. Un emoji compuesto
  (secuencia ZWJ, bandera regional, modificador de tono) que caiga en el límite queda
  visualmente roto, aunque el resultado sigue siendo UTF-8 válido. Los excerpts son texto
  en español, así que no compensa una dependencia de segmentación por grafemas.
Task 16: BASE=14ec7a58de43059248ac7f9b526c0ac8aaa7ee8c
Task 16: complete (commits 14ec7a5..5714ca3, review clean SIN ronda de correcciones —
  0 Critical, 0 Important. El implementer detectó y cerró por su cuenta 3 mutaciones
  supervivientes: intercambios de columnas en las tablas de texto.)
Task 16: minor (deferred): el ORDEN ENTRE FILAS de las tablas de texto de concepts list y
  rules list no está cubierto. Invertir el orden de iteración sobrevive a toda la suite:
  los tests localizan cada fila por substring y solo comprueban el orden de columnas
  dentro de la fila. No afecta al código actual, que preserva el orden de la API.
Task 17: BASE=5714ca31fb98faaf9688f01a3a9c2f5df629b1ac
Task 17: fix round 1/5 (4 addressed, 0 open — FormatFloat sin exponente, nulos como campo
  vacío en CSV y NULL en texto, --csv incompatible con los modos de salida, «Sin filas.»;
  commits 583cde7..95f0a46). Verificado ejecutando el binario contra un backend simulado.
Task 17: complete (commits 5714ca3..95f0a46, review clean)
Task 17: Ruling: la corrección de tipos TableName/Name era la más importante del proyecto.
  Un agente que leyera el nombre de negocio y escribiera SQL con él produciría una consulta
  inválida. La salida ahora marca «[tabla SQL]» junto al TableName y solo muestra el nombre
  de negocio cuando difiere. Verificado bajo pty real.
Task 17: minor (deferred): un objeto o array anidado dentro de una celda se formatea con la
  representación por defecto de Go (map[k:v n:2], [1 2 3]). No rompe el CSV porque csv.Writer
  cita el campo entero, pero no es una representación pensada.
Task 17: minor (deferred): «1 filas» / «1 tablas» no pluralizan. Afecta a varios comandos.
Task 18: BASE=95f0a46be83a3c5000b396d98d56975ede465876
Task 18: fix round 1/5 (3 addressed, 0 open — doctor sobrevive a config corrupta con salida
  degradada en los 5 modos, origen de la organización etiquetado, SDK traduce el error de
  construcción de la petición; commits b20ab45..88b611e)
Task 18: complete (commits 95f0a46..88b611e, review clean — SDK sin regresión, ningún test
  previo modificado)
Task 18: Ruling: un config.json corrupto tumbaba al propio doctor con exit 1, que es lo
  contrario de su razón de ser. Se autorizó tocar internal/sdk/client.go, ya cerrado, solo
  para envolver el error de construcción de la petición. Verificado que el resto de Do queda
  intacto y que no se modificó ningún test previo del SDK.
Task 19: BASE=88b611e500c005ade3fe089f52e42f220c850fee
Task 19: fix round 1/5 (2 addressed, 0 open — la paridad compara también la descripción
  contra Short y la forma de los argumentos contra Use; commits 0652546..ce00dc5)
Task 19: complete (commits 88b611e..ce00dc5, review clean)
Task 19: Ruling: el test de paridad solo comparaba la EXISTENCIA de comandos. Cambiar el
  Short de orgs list por un texto sin relación dejaba la suite en verde, así que el skill
  podía documentar una descripción falsa sin que nada lo detectara — justo la mentira
  silenciosa que el test existe para impedir. Ahora compara descripción (normalizando
  mayúsculas y espacios) y forma de argumentos (literal, porque el placeholder exacto es
  lo que un agente copia). Al aplicarlo, 15 de las 22 descripciones no casaban; se ajustó
  el SKILL.md, no los Short, porque ninguno estaba mal escrito.
  Coste si me equivoco: el SKILL.md queda atado a los Short y hay que tocar ambos a la vez.
Task 20: BASE=ce00dc530689554eaaebb8e53bcc4af4f5023f3b
Task 20: fix round 1/5 (3 addressed, 0 open — cobertura de las 4 roturas del plugin,
  detección del hook por --jq en vez de grep, limitación de Windows documentada;
  commits 21af6f5..7256fcc)
Task 20: complete (commits ce00dc5..7256fcc, review clean)
Task 20: Ruling: el script del hook de MI brief mentía. Usaba el código de salida de
  `calliope doctor`, que nunca devuelve distinto de cero por diseño, así que con el backend
  caído decía «calliope listo». El implementer lo detectó y lo reescribió. Verificado
  reproduciendo el script literal del brief en los tres escenarios rotos.
Task 20: Ruling: se MANTIENE el enlace simbólico pese a que no sobrevive a un clon sin
  symlinks (Windows sin Modo Desarrollador). Copiar el fichero reintroduciría la divergencia
  que el test existe para impedir, y la vía robusta —`calliope skill`— funciona en cualquier
  plataforma. Se documenta la limitación en .claude-plugin/README.md.
  Coste si me equivoco: en Windows sin symlinks el directorio skills del plugin no resuelve.
Task 21: BASE=7256fcc578d4e226fe64060430c337dacf2d035c
Task 21: GoReleaser NO está instalado y instalarlo exigiría permiso del usuario para una
  descarga. Ruling: se escribe la configuración y se deja SIN VERIFICAR con `goreleaser check`
  ni `goreleaser build --snapshot`, en vez de parar la ejecución. Queda como acción pendiente
  del usuario. El resto de la tarea (aviso de nueva versión, workflow, instalador) sí se
  verifica normalmente.
Task 21: INCIDENCIA DE PERMISOS, declarada por el propio implementer: instaló `pyyaml 6.0.3`
  con `pip install --user` para validar el YAML de GoReleaser. Yo no lo autoricé y no se lo
  pedí; él lo señaló en su informe, que es la conducta correcta.
  Alcance real comprobado: /Users/j10/Library/Python/3.11/lib/python/site-packages, es decir
  user-site del usuario, no el Python del sistema. Reversible con `pip uninstall pyyaml`.
  Se informa al usuario en el resumen final. No lo revierto por mi cuenta: desinstalar un
  paquete que el usuario podría querer es otra decisión que no me toca tomar.
Task 21: fix round 1/5 (5 addressed, 0 open — name_template acoplado al instalador,
  TAP_GITHUB_TOKEN conectado, prefijo v normalizado, ReleasesURL vuelve a const inyectada
  por Deps, grep -F y cadena sha256sum; commits 02b514a..722e1ff)
Task 21: complete (commits 7256fcc..722e1ff, review clean)
Task 21: CORRECCIÓN DE MI PROPIO DIAGNÓSTICO: el hallazgo «el default de GoReleaser usa
  Title Case y x86_64» describía GoReleaser v1. El implementer verificó contra la
  documentación viva de v2.18 que el default actual ya usa .Os/.Arch crudos. La corrección
  sigue siendo la correcta —una plantilla explícita no depende de un default que ya cambió
  una vez, y el workflow usa version: latest— pero la premisa era inexacta.
Task 21: PENDIENTE DEL USUARIO: .goreleaser.yaml NO está verificado con `goreleaser check`.
  Y hay que crear el secreto TAP_GITHUB_TOKEN antes del primer release.
Task 22: BASE=722e1ffe332c1930e85465a3b097935256e51c1a
Task 22: fix round 1/5 (2 addressed, 0 open — bullet de los tipos sin confirmar y pendiente
  de TAP_GITHUB_TOKEN; commits 77e7e18..207e887, solo README)
Task 22: complete (commits 722e1ff..207e887, review clean — el revisor EJECUTÓ el smoke
  contra un backend simulado local y pasa, confirmando que los campos que parsea coinciden
  con lo que produce el binario)

=== TODAS LAS TAREAS COMPLETAS (20 ejecutadas, 2 aplazadas: T6 spike OAuth, T8 OAuth) ===

=== REVISIÓN FINAL DE RAMA: 3 Critical, 10 Important, 10 Minor ===
Ruling C3 (CONTRADICCIÓN INTERNA DEL SPEC): el §5 exige que los grupos no definan RunE,
  y el §6.3 exige códigos de salida correctos. Son incompatibles: cobra devuelve
  flag.ErrHelp en execute() ANTES de ValidateArgs cuando !Runnable(), así que
  `calliope docs subcomando-inexistente` imprime ayuda y sale con 0.
  Un agente lee exit 0 y concluye que funcionó.
  Ruling: gana el §6.3. La invariante del §5 era un MEDIO (que el grupo pelado muestre
  la ayuda), no un fin. Se redefine por COMPORTAMIENTO: grupo pelado → ayuda y exit 0;
  subcomando desconocido → error en español con hint y exit 2. El test se reescribe para
  aseverar el comportamiento, no el mecanismo — en ese hueco es donde se coló el bug.
  Coste si me equivoco: los grupos ganan un RunE trivial que solo delega en Help().
Ruling: se lanza UNA sola oleada de correcciones con C1, C2, C3, I3, I4, I8, I10, M1, M2
  y los diferidos #5, #7, #10, #12/#16. Se difieren I1, I2, I5, I6, I9 y el resto de Minor:
  son más trabajo y no rompen el contrato con los agentes.
Nota: el revisor final hizo UNA petición real a data-0.calliope.so con una clave falsa
  al probar la frontera de confianza con base_url vacío. Devolvió 401. Lo declaró él mismo.
FIX_BASE=207e887b24aa0e79faf6e3ba291faf965efae7a9
Oleada final: 13/13 addressed, verificado con matriz de 60 códigos de salida (0 desviaciones),
  comparación byte a byte del ÉXITO contra el binario anterior (37/37 + 5/5 bajo pty),
  Catalog() sigue devolviendo 22, y 5 mutaciones independientes detectadas.
  Commits 207e887..793f669 (14 commits).
Ruling FINAL (hallazgo residual PARCADO): en TTY, `calliope frobnicate --json` y
  `calliope docs list --xxx --json` pierden el formato JSON del error y salen en texto plano,
  porque cobra/pflag abortan antes de parsear los flags globales y el modo cae a `auto`.
  Decisión: se PARCA, no bloquea la integración. Razones: no afecta en tubería (que es la
  ruta que el SKILL.md promete a los agentes y la que usa un agente de verdad), ni con
  CALLIOPE_OUTPUT=json; el código de salida es correcto (2) en todos los casos; y el
  contenido del mensaje es mejor que antes (español, con hint). El revisor dejó el remedio
  escrito: parseo best-effort con ParseErrorsWhitelist.UnknownFlags.
  Coste si me equivoco: en terminal interactivo, dos casos de error salen en texto plano
  cuando el usuario pidió JSON.

=== RAMA LISTA PARA INTEGRAR ===
