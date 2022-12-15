package docs

import "popplio/types"

type Server struct {
	URL         string         `json:"url"`
	Description string         `json:"description"`
	Variables   map[string]any `json:"variables"`
}

type Contact struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Email string `json:"email"`
}

type License struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Info struct {
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	TermsOfService string  `json:"termsOfService"`
	Version        string  `json:"version"`
	Contact        Contact `json:"contact"`
	License        License `json:"license"`
}

type Content struct {
	Schema any `json:"schema"`
}

type ReqBody struct {
	Description string             `json:"description,omitempty"`
	Required    bool               `json:"required,omitempty"`
	Content     map[string]Content `json:"content"`
}

type Security struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	In          string `json:"in"` // must be apiKey for Popplio
}

type Component struct {
	Schemas       map[string]any      `json:"schemas"`
	Security      map[string]Security `json:"securitySchemes"`
	RequestBodies map[string]ReqBody  `json:"requestBodies"`
}

type Schema struct {
	Ref        string         `json:"$ref,omitempty"`
	Type       string         `json:"type,omitempty"`
	Required   []string       `json:"required,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
}

type SchemaResp struct {
	Schema Schema `json:"schema"`
}

// Represents a openAPI response
type Response struct {
	Description string                `json:"description"`
	Content     map[string]SchemaResp `json:"content"`
}

// Parameter defines a openAPI parameter
type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Schema      any    `json:"schema"`
}

type Operation struct {
	Summary     string                `json:"summary"`
	Tags        []string              `json:"tags,omitempty"`
	Description string                `json:"description"`
	ID          string                `json:"operationId"`
	RequestBody any                   `json:"requestBody,omitempty"`
	Parameters  []Parameter           `json:"parameters"`
	Responses   map[string]Response   `json:"responses"`
	Security    []map[string][]string `json:"security,omitempty"`
	Servers     []Server              `json:"servers,omitempty"`
}

type Path struct {
	Summary     string     `json:"summary"` // Danger do not use this
	Description string     `json:"description"`
	Get         *Operation `json:"get,omitempty"`
	Post        *Operation `json:"post,omitempty"`
	Put         *Operation `json:"put,omitempty"`
	Patch       *Operation `json:"patch,omitempty"`
	Delete      *Operation `json:"delete,omitempty"`
}

type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Openapi struct {
	OpenAPI    string                    `json:"openapi"`
	Info       Info                      `json:"info"`
	Servers    []Server                  `json:"servers"`
	Components Component                 `json:"components"`
	Paths      *OrderedMap[string, Path] `json:"paths"`
	Tags       []Tag                     `json:"tags,omitempty"`
}

type Doc struct {
	Method      string
	Path        string
	OpId        string
	Summary     string
	Description string
	Params      []Parameter
	Tags        []string
	Req         any
	Resp        any
	AuthType    []types.TargetType

	// Intentionally private to ensure docs.Route has been called
	added bool
}

func (d *Doc) Added() bool {
	return d.added
}
