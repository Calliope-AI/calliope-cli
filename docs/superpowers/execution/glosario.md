# Glosario de identificadores — vinculante para todas las tareas

La restricción global del proyecto dice: **mensajes de usuario, ayuda y comentarios en
español; identificadores en inglés.** El plan incumple su propia restricción: sus
bloques de código usan nombres en español para funciones y variables no exportadas.

**Ruling del controlador:** manda la restricción. Traduce los identificadores al
inglés. Los *valores* (cadenas de mensaje, tags json del backend, textos de ayuda,
comentarios) se copian del brief **literalmente**; solo cambian los *nombres*.

`internal/output` e `internal/cli` ya están en inglés; esto los mantiene coherentes.

## Traducciones fijadas

Usa exactamente estos nombres cuando tu brief los mencione. Si tu brief tiene un
identificador en español que no está aquí, tradúcelo al inglés con el criterio obvio
y anótalo en tu informe para que lo añada.

| Brief (español) | Usa (inglés) |
|---|---|
| `escribirJSON` | `writeJSON` |
| `escribirFila` | `writeRow` |
| `resultadoDePrueba` | `testResult` |
| `sanear` | `sanitize` |
| `raizDeRepositorio` / `raizDelRepo` | `repoRoot` |
| `leerFichero` | `readLayerFile` |
| `prioridad` | `priority` |
| `errorDeTransporte` | `transportError` |
| `mapearStatus` | `mapStatus` |
| `esTimeout` | `isTimeout` |
| `filtrarTablas` | `filterTables` |
| `columnasDe` | `columnsOf` |
| `escribirCSV` | `writeCSV` |
| `recortar` | `truncate` |
| `tituloDe` | `titleOf` |
| `activo` | `yesNo` |
| `resumenDe` | `summaryOf` |
| `simbolo` | `symbol` |
| `chequearConectividad` | `checkConnectivity` |
| `escribirAskTexto` | `writeAskText` |
| `escribirAskMarkdown` | `writeAskMarkdown` |
| `canjear` | `exchangeCode` |
| `abrirNavegador` | `openBrowser` |
| `aleatorio` | `randomString` |
| `paginaDeCallback` | `callbackPage` |
| `modoDeSalida` | `outputMode` |
| `timeoutDe` | `timeoutOf` |
| `comandosDocumentados` | `documentedCommands` |
| `soloNombre` | `commandName` |
| `lineaDeComando` | `commandLineRe` |
| `depsDePrueba` | `testDeps` |
| `comandoConFlags` | `commandWithFlags` |
| `escribirConfigProyecto` / `escribirConfigDeProyecto` | `writeProjectConfig` |
| `entornoDePrueba` | `testEnv` |
| `clienteDePrueba` | `testClient` |
| `servidorConFixture` | `serverWithFixture` |
| `errorComoCLI` / `asCLIError` | `asCLIError` |
| `depsConServidor` | `depsWithServer` |
| `raizDePrueba` | `testRoot` |
| `chequeosDe` | `checksOf` |
| `requiereE2E` | `requireE2E` |
| `d0` | `emptyDeps` |
| `servicioKeyring` | `keyringService` |
| `usuarioKeyring` | `keyringUser` |
| campo `ruta` | `path` |
| campo `respaldo` | `fallback` |
| (T19) | `documentedCommand`, `argsUsageOf`, `normalizeDescription`, campo `Args` |
| `BuildSinCredencial` | `BuildWithoutCredential` |
| `renderConfigRota` | `renderBrokenConfig` |
| `outputModeSinConfig` | `outputModeWithoutConfig` |

**Alcance (afinado tras la Task 4):** el glosario aplica a identificadores de nivel de
paquete — tipos, funciones, constantes, campos y tags json. Las variables locales dentro
de un cuerpo de función se quedan como las escribe el brief.
| tipo `Chequeo{Nombre, Estado, Detalle}` | `Check{Name, Status, Detail}`, tags json `name`, `status`, `detail` |

Los nombres de test (`func TestLoQueSea`) **se quedan en español**: describen el
comportamiento y son prosa, no identificadores de API.
