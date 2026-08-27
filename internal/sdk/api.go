package sdk

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// Ask hace una pregunta en lenguaje natural sobre los datos y la documentación.
func (c *Client) Ask(ctx context.Context, org, question, action string) (*AskResponse, error) {
	cuerpo := map[string]string{"question": question}
	if action != "" {
		cuerpo["action"] = action
	}
	var out AskResponse
	if err := c.Do(ctx, http.MethodPost, c.OrgPath(org, "/ask"), cuerpo, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SearchDocuments busca semánticamente en la documentación.
func (c *Client) SearchDocuments(ctx context.Context, org, query string, limit int) ([]DocumentSearchResult, error) {
	if limit <= 0 {
		limit = 10
	}
	cuerpo := map[string]any{"query": query, "limit": limit}
	var out []DocumentSearchResult
	if err := c.Do(ctx, http.MethodPost, c.OrgPath(org, "/search/documents"), cuerpo, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListDocumentsParams son los filtros de listado de documentos.
type ListDocumentsParams struct {
	Status string
	Tag    string
	Q      string
	Page   int
	Size   int
}

func (p ListDocumentsParams) query() string {
	q := url.Values{}
	if p.Status != "" {
		q.Set("status", p.Status)
	}
	if p.Tag != "" {
		q.Set("tag", p.Tag)
	}
	if p.Q != "" {
		q.Set("q", p.Q)
	}
	if p.Page > 0 {
		q.Set("page", strconv.Itoa(p.Page))
	}
	if p.Size > 0 {
		q.Set("size", strconv.Itoa(p.Size))
	}
	if len(q) == 0 {
		return ""
	}
	return "?" + q.Encode()
}

// ListDocuments lista los documentos de la organización.
func (c *Client) ListDocuments(ctx context.Context, org string, p ListDocumentsParams) (*DocumentPage, error) {
	var out DocumentPage
	if err := c.Do(ctx, http.MethodGet, c.OrgPath(org, "/documents"+p.query()), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDocument devuelve los metadatos de un documento.
func (c *Client) GetDocument(ctx context.Context, org, id string) (*DocumentResponse, error) {
	var out DocumentResponse
	if err := c.Do(ctx, http.MethodGet, c.OrgPath(org, "/documents/"+url.PathEscape(id)), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListConcepts devuelve el grafo de conceptos de la ontología.
func (c *Client) ListConcepts(ctx context.Context, org string) (*ConceptGraphResponse, error) {
	var out ConceptGraphResponse
	if err := c.Do(ctx, http.MethodGet, c.OrgPath(org, "/knowledge/concepts"), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetConcept devuelve un concepto con sus atributos.
func (c *Client) GetConcept(ctx context.Context, org, id string) (*ConceptDetailResponse, error) {
	var out ConceptDetailResponse
	ruta := c.OrgPath(org, "/knowledge/concepts/"+url.PathEscape(id))
	if err := c.Do(ctx, http.MethodGet, ruta, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListRules devuelve las reglas de negocio compartidas.
func (c *Client) ListRules(ctx context.Context, org string) ([]Rule, error) {
	var out []Rule
	if err := c.Do(ctx, http.MethodGet, c.OrgPath(org, "/rules"), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Schema devuelve el esquema de la base de datos de la organización.
func (c *Client) Schema(ctx context.Context, org string) (*SchemaResponse, error) {
	var out SchemaResponse
	if err := c.Do(ctx, http.MethodGet, c.OrgPath(org, "/database/schema"), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Query ejecuta SQL crudo. El parámetro formato se reenvía tal cual al backend
// en QueryRequest.output; no confundir con el flag --csv, que es render local.
func (c *Client) Query(ctx context.Context, org, sql, formato string) (*QueryResponse, error) {
	cuerpo := map[string]string{"sql": sql}
	if formato != "" {
		cuerpo["output"] = formato
	}
	var out QueryResponse
	if err := c.Do(ctx, http.MethodPost, c.OrgPath(org, "/query"), cuerpo, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Me devuelve el perfil del titular de la credencial.
func (c *Client) Me(ctx context.Context) (*Me, error) {
	var out Me
	if err := c.Do(ctx, http.MethodGet, "/v1/auth/me", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListOrganizations devuelve las organizaciones accesibles con la credencial.
func (c *Client) ListOrganizations(ctx context.Context) ([]Organization, error) {
	var out []Organization
	if err := c.Do(ctx, http.MethodGet, "/v1/organizations", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
