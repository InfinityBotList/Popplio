package docs

import (
	"popplio/types"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

type server struct {
	URL         string         `json:"url"`
	Description string         `json:"description"`
	Variables   map[string]any `json:"variables"`
}

type contact struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Email string `json:"email"`
}

type license struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type info struct {
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	TermsOfService string  `json:"termsOfService"`
	Version        string  `json:"version"`
	Contact        contact `json:"contact"`
	License        license `json:"license"`
}

type content struct {
	Schema any `json:"schema"`
}

type reqBody struct {
	Description string             `json:"description"`
	Required    bool               `json:"required"`
	Content     map[string]content `json:"content"`
}

type security struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	In          string `json:"in"` // must be apiKey for Popplio
}

type component struct {
	Schemas       map[string]any      `json:"schemas"`
	Security      map[string]security `json:"securitySchemes"`
	RequestBodies map[string]reqBody  `json:"requestBodies"`
}

type ref struct {
	Ref string `json:"$ref"`
}

type schemaResp struct {
	Schema ref `json:"schema"`
}

// Represents a openAPI response
type response struct {
	Description string                `json:"description"`
	Content     map[string]schemaResp `json:"content"`
}

// Parameter defines a openAPI parameter
type Paramater struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Schema      any    `json:"schema"`
}

type operation struct {
	Summary     string                `json:"summary"`
	Tags        []string              `json:"tags,omitempty"`
	Description string                `json:"description"`
	ID          string                `json:"operationId"`
	RequestBody *ref                  `json:"requestBody,omitempty"`
	Parameters  []Paramater           `json:"parameters"`
	Responses   map[string]response   `json:"responses"`
	Security    []map[string][]string `json:"security,omitempty"`
}

type path struct {
	Summary     string     `json:"summary"` // Danger do not use this
	Description string     `json:"description"`
	Get         *operation `json:"get,omitempty"`
	Post        *operation `json:"post,omitempty"`
	Put         *operation `json:"put,omitempty"`
	Patch       *operation `json:"patch,omitempty"`
	Delete      *operation `json:"delete,omitempty"`
}

type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Openapi struct {
	OpenAPI    string          `json:"openapi"`
	Info       info            `json:"info"`
	Servers    []server        `json:"servers"`
	Components component       `json:"components"`
	Paths      map[string]path `json:"paths"`
	Tags       []Tag           `json:"tags,omitempty"`
}

var api = Openapi{
	OpenAPI: "3.0.3",
	Info: info{
		Title: "Infinity Bot List API",
		Description: `
Welcome to the Infinity Bot List API documentation!

## Libraries

We offer several libraries for interacting with the API:

- [Java](https://guide.infinitybots.gg/docs/libraries/java)
- [JavaScript](https://guide.infinitybots.gg/docs/libraries/javascript)
- [Python](https://guide.infinitybots.gg/docs/libraries/python)

## Ratelimits

You can find documentation on ratelimits and other resources [here](https://guide.infinitybots.gg/docs/resources/ratelimits)
`,
		TermsOfService: "https://infinitybotlist.com/terms",
		Version:        "6.0",
		Contact: contact{
			Name:  "Infinity Bot List",
			URL:   "https://infinitybotlist.com",
			Email: "support@infinitybots.gg",
		},
		License: license{
			Name: "MIT",
			URL:  "https://opensource.org/licenses/MIT",
		},
	},
	Servers: []server{
		{
			URL:         "https://spider.infinitybotlist.com",
			Description: "Popplio (v6)",
			Variables:   map[string]any{},
		},
	},
	Components: component{
		Schemas:       make(map[string]any),
		Security:      make(map[string]security),
		RequestBodies: make(map[string]reqBody),
	},
	Paths: make(map[string]path),
}

var badRequestSchema *openapi3.SchemaRef

var IdSchema *openapi3.SchemaRef
var BoolSchema *openapi3.SchemaRef

func init() {
	var err error

	badRequestSchema, err = openapi3gen.NewSchemaRefForValue(types.ApiError{}, nil)

	if err != nil {
		panic(err)
	}

	IdSchema, err = openapi3gen.NewSchemaRefForValue("1234567890", nil)

	if err != nil {
		panic(err)
	}

	BoolSchema, err = openapi3gen.NewSchemaRefForValue(true, nil)

	if err != nil {
		panic(err)
	}

	api.Components.Schemas["ApiError"] = badRequestSchema
}

func AddTag(name, description string) {
	api.Tags = append(api.Tags, Tag{
		Name:        name,
		Description: description,
	})
}

func AddSecuritySchema(id string, description string) {
	api.Components.Security[id] = security{
		Type:        "apiKey",
		Name:        "Authorization",
		In:          "header",
		Description: description,
	}
}

func AddDocs(method string, pathStr string, opId string, summary string, description string, params []Paramater, tags []string, req any, resp any, authType []string) {
	// Generate schemaName, taking out bad things
	schemaName := strings.ReplaceAll(reflect.TypeOf(resp).String(), "[", "-")

	schemaName = strings.ReplaceAll(schemaName, "]", "-")
	schemaName = strings.ReplaceAll(schemaName, " ", "")
	schemaName = strings.ReplaceAll(schemaName, "{", "")
	schemaName = strings.ReplaceAll(schemaName, "}", "")

	// Remove last - if it exists
	schemaName = strings.TrimSuffix(schemaName, "-")

	schemaName = strings.ReplaceAll(schemaName, "docs.", "")

	if _, ok := api.Components.Schemas[schemaName]; !ok {

		schemaRef, err := openapi3gen.NewSchemaRefForValue(resp, nil)

		if err != nil {
			panic(err)
		}

		api.Components.Schemas[schemaName] = schemaRef
	}

	// Add in requests
	if req != nil {
		schemaRef, err := openapi3gen.NewSchemaRefForValue(req, nil)

		if err != nil {
			panic(err)
		}

		api.Components.RequestBodies["method-"+schemaName] = reqBody{
			Description: "Request body",
			Required:    true,
			Content: map[string]content{
				"application/json": {
					Schema: schemaRef,
				},
			},
		}
	}

	if _, ok := api.Paths[pathStr]; !ok {
		api.Paths[pathStr] = path{}
	}

	refName := "#/components/schemas/" + schemaName
	reqName := "#/components/requestBodies/" + "method-" + schemaName

	var reqBody *ref

	if req != nil {
		reqBody = &ref{Ref: reqName}
	}

	operationData := &operation{
		Tags:        tags,
		Summary:     summary,
		Description: description,
		ID:          opId,
		Parameters:  params,
		RequestBody: reqBody,
		Responses: map[string]response{
			"200": {
				Description: "Success",
				Content: map[string]schemaResp{
					"application/json": {
						Schema: ref{
							Ref: refName,
						},
					},
				},
			},
			"400": {
				Description: "Bad Request",
				Content: map[string]schemaResp{
					"application/json": {
						Schema: ref{
							Ref: "#/components/schemas/ApiError",
						},
					},
				},
			},
		},
	}

	if len(authType) == 0 {
		authType = []string{"None"}
	}

	operationData.Security = []map[string][]string{}

	for _, auth := range authType {

		operationData.Security = append(operationData.Security, map[string][]string{
			auth: {},
		})
	}

	op := api.Paths[pathStr]

	switch strings.ToLower(method) {
	case "get":
		op.Get = operationData

		api.Paths[pathStr] = op
	case "post":
		op.Post = operationData

		api.Paths[pathStr] = op
	case "put":
		op.Put = operationData

		api.Paths[pathStr] = op
	case "patch":
		op.Patch = operationData

		api.Paths[pathStr] = op
	case "delete":
		op.Delete = operationData

		api.Paths[pathStr] = op

	}
}

func GetSchema() any {
	return api
}
