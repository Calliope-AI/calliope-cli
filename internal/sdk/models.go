package sdk

import "encoding/json"

// Los tipos replican el contrato de Calliope Data verificado en
// calliope-data-mcp/src/calliope/types.ts. El JSON del backend es camelCase.
// Solo se declaran los campos que el CLI usa; el resto se ignora.
//
// Me, OrgInfo, Organization y los tipos de SchemaResponse no vienen del MCP:
// se verificaron contra los tipos TypeScript de calliope-data-ui
// (types/index.ts: BackendUser, BackendOrgInfo, Organization, SchemaResponse,
// SchemaTable, SchemaColumn), que es el cliente que hoy consume esos mismos
// endpoints. Siguen sin confirmarse contra el backend en vivo: no había API
// key disponible para el Step 7 del brief original. Ver
// task-10-tipos-corregidos.md y el informe de esta tarea.

// --- /ask ---

// AskDocumentSource es una cita devuelta por /ask.
type AskDocumentSource struct {
	Citation      string  `json:"citation"`
	ChunkID       int     `json:"chunkId"`
	DocumentID    string  `json:"documentId"`
	DocumentTitle *string `json:"documentTitle"`
	Filename      string  `json:"filename"`
	Page          *int    `json:"page"`
	HeadingPath   *string `json:"headingPath"`
	Excerpt       string  `json:"excerpt"`
}

// AskQueryDataset es uno de los conjuntos de datos que respaldan la respuesta.
type AskQueryDataset struct {
	ID       string           `json:"id"`
	Purpose  *string          `json:"purpose"`
	Data     []map[string]any `json:"data"`
	RowCount *int             `json:"rowCount"`
	Error    *string          `json:"error"`
}

// AskResponse es la respuesta de /ask.
type AskResponse struct {
	Success      bool                `json:"success"`
	Text         string              `json:"text"`
	Data         []map[string]any    `json:"data"`
	RowCount     *int                `json:"rowCount"`
	Queries      []AskQueryDataset   `json:"queries"`
	AnalysisType *string             `json:"analysisType"`
	Sources      []AskDocumentSource `json:"sources"`
	Error        *string             `json:"error"`
}

// --- /search/documents ---

// DocumentSearchResult es un fragmento devuelto por la búsqueda semántica.
type DocumentSearchResult struct {
	ChunkID       int     `json:"chunkId"`
	DocumentID    string  `json:"documentId"`
	DocumentTitle *string `json:"documentTitle"`
	Filename      string  `json:"filename"`
	Ordinal       int     `json:"ordinal"`
	PageFrom      *int    `json:"pageFrom"`
	PageTo        *int    `json:"pageTo"`
	HeadingPath   *string `json:"headingPath"`
	Excerpt       string  `json:"excerpt"`
	Score         float64 `json:"score"`
}

// --- /documents ---

// DocumentResponse son los metadatos de un documento.
type DocumentResponse struct {
	ID           string   `json:"id"`
	Filename     string   `json:"filename"`
	Title        *string  `json:"title"`
	MimeType     *string  `json:"mimeType"`
	DeclaredMime string   `json:"declaredMime"`
	SizeBytes    int64    `json:"sizeBytes"`
	Status       string   `json:"status"`
	PageCount    *int     `json:"pageCount"`
	CharCount    *int     `json:"charCount"`
	Language     *string  `json:"language"`
	Tags         []string `json:"tags"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
	ReadyAt      *string  `json:"readyAt"`
}

// DocumentPage es una página de documentos.
type DocumentPage struct {
	Content   []DocumentResponse `json:"content"`
	TotalSize int                `json:"totalSize"`
}

// --- /knowledge/concepts ---

// GraphConceptNode es un concepto en el grafo de la ontología.
type GraphConceptNode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	IsActive    bool   `json:"isActive"`
	RecordCount *int   `json:"recordCount"`
	SourceCount *int   `json:"sourceCount"`
}

// ConceptGraphResponse es el grafo completo de conceptos.
type ConceptGraphResponse struct {
	Concepts []GraphConceptNode `json:"concepts"`
}

// ConceptResponse es la cabecera de un concepto.
type ConceptResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IsActive    bool    `json:"isActive"`
}

// AttributeResponse es un atributo de un concepto.
type AttributeResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IsActive    bool    `json:"isActive"`
}

// ConceptDetailResponse es el detalle de un concepto con sus atributos.
type ConceptDetailResponse struct {
	Concept    ConceptResponse     `json:"concept"`
	Attributes []AttributeResponse `json:"attributes"`
}

// --- /rules ---

// Rule es una regla de negocio compartida de la organización.
type Rule struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Details  string  `json:"details"`
	Category *string `json:"category"`
	Status   string  `json:"status"`
}

// --- /database/schema ---
//
// Corregido contra calliope-data-ui/types/index.ts (SchemaResponse,
// SchemaTable, SchemaColumn): una tabla tiene dos nombres. TableName es el
// identificador real que va en el SQL; Name es el nombre de negocio y puede
// diferir. El brief original los confundía en un solo campo.

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

// SchemaTable es una tabla del esquema. TableName es el identificador que se
// usa en SQL; Name es el nombre de negocio y puede diferir.
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

// --- /query ---

// QueryResponse es el resultado de una consulta SQL. El backend devuelve
// `data` unas veces como array y otras como cadena JSON; Rows normaliza ambas.
type QueryResponse struct {
	Data     json.RawMessage `json:"data"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// Rows devuelve las filas del resultado, venga `data` como array o como cadena.
func (r *QueryResponse) Rows() ([]map[string]any, error) {
	if len(r.Data) == 0 || string(r.Data) == "null" {
		return nil, nil
	}

	var filas []map[string]any
	if err := json.Unmarshal(r.Data, &filas); err == nil {
		return filas, nil
	}

	// El backend la ha enviado como cadena JSON: se deshace una capa.
	var comoCadena string
	if err := json.Unmarshal(r.Data, &comoCadena); err != nil {
		return nil, err
	}
	if comoCadena == "" {
		return nil, nil
	}
	if err := json.Unmarshal([]byte(comoCadena), &filas); err != nil {
		return nil, err
	}
	return filas, nil
}

// --- auth y organizaciones ---
//
// Me, OrgInfo y Organization corregidos contra BackendUser, BackendOrgInfo y
// Organization de calliope-data-ui/types/index.ts.

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

// Organization es una organización accesible con la credencial actual.
// Ojo: en el backend real los timestamps de creación/actualización van en
// snake_case (created_at/updated_at), a diferencia del resto de la API; no se
// declaran aquí porque el CLI no los necesita.
type Organization struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status,omitempty"`
	Description string `json:"description,omitempty"`
}
