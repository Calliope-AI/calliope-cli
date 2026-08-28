# Corrección de tipos para la Task 10 — vinculante

El Step 7 de tu brief te pide verificar `SchemaResponse`, `Me` y `Organization`
contra el backend real con una API key. **No hay API key disponible**, así que el
controlador los ha verificado contra la otra fuente de verdad: los tipos
TypeScript de `calliope-data-ui`, que es el cliente que hoy consume esos mismos
endpoints (`/Users/j10/repositories/calliope/calliope-data-ui/types/index.ts`).

**Los tres tipos del brief son incorrectos.** Usa estas definiciones en su lugar.
No ejecutes el Step 7 (no hay credencial); en su lugar, deja constancia en el
informe de que los tipos vienen de la UI y siguen sin confirmarse contra el
backend en vivo.

## Me — `GET /v1/auth/me`

El campo del identificador es `id`, **no** `userId` como decía el brief.
La UI lo declara como `BackendUser` (types/index.ts:884).

```go
// Me es el perfil del titular de la credencial.
// Campos verificados contra BackendUser de calliope-data-ui.
type Me struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Username      string    `json:"username,omitempty"`
	FirstName     string    `json:"firstName,omitempty"`
	LastName      string    `json:"lastName,omitempty"`
	Organizations []OrgInfo `json:"organizations"`
}

// OrgInfo es una organización accesible por el usuario, con su rol.
type OrgInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	UserRole string `json:"userRole"`
}
```

## Organization — `GET /v1/organizations`

Ojo: los timestamps van en **snake_case**, a diferencia del resto de la API.
No los necesitamos, pero no los declares en camelCase por inercia.

```go
// Organization es una organización accesible con la credencial actual.
type Organization struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status,omitempty"`
	Description string `json:"description,omitempty"`
}
```

## SchemaResponse — `GET /v1/organizations/{org}/database/schema`

Esta es la corrección más importante. Una tabla tiene **dos** nombres:
`tableName` es el identificador real que va en el SQL, y `name` es el nombre de
negocio. El brief los confundía en un solo campo.

```go
// SchemaColumn es una columna de una tabla.
type SchemaColumn struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Nullable     bool     `json:"nullable"`
	IsPrimaryKey bool     `json:"isPrimaryKey"`
	IsForeignKey bool     `json:"isForeignKey"`
	Description  string   `json:"description,omitempty"`
	SampleValues []string `json:"sampleValues,omitempty"`
}

// SchemaTable es una tabla del esquema. TableName es el identificador que se usa
// en SQL; Name es el nombre de negocio y puede diferir.
type SchemaTable struct {
	ID          string         `json:"id"`
	TableName   string         `json:"tableName"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Source      string         `json:"source,omitempty"`
	Columns     []SchemaColumn `json:"columns"`
}

// SchemaResponse es el esquema de la base de datos de la organización.
type SchemaResponse struct {
	OrganizationID string        `json:"organizationId"`
	Tables         []SchemaTable `json:"tables"`
}
```

## Fixtures

Sustituye el fixture de esquema por uno acorde. Añade también `testdata/me.json`
y `testdata/organizations.json` con estas formas, y un test de decodificación
para cada uno siguiendo el patrón de `TestListDocumentsDecodificaLaPagina`.
En el test del esquema, comprueba explícitamente que `TableName` y `Name` se
decodifican por separado y pueden diferir: es la regresión que quiero fijada.
