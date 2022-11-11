package docs

import (
	"embed"
	"fmt"
	"popplio/types"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

//go:embed docs-md/*
var docsFiles embed.FS

var docsMd string

func init() {
	ordersFile, err := docsFiles.ReadFile("docs-md/order.yaml")

	if err != nil {
		panic(err)
	}

	var order []string

	err = yaml.Unmarshal(ordersFile, &order)

	if err != nil {
		panic(err)
	}

	entries, err := docsFiles.ReadDir("docs-md")

	if err != nil {
		panic(err)
	}

	for _, entry := range entries {
		if entry.Name() == "order.yaml" {
			continue
		}

		if !slices.Contains(order, entry.Name()) {
			panic(entry.Name() + " not in order.yaml")
		}
	}

	for _, entry := range order {
		docsFile, err := docsFiles.ReadFile("docs-md/" + entry)
		if err != nil {
			panic(err)
		}

		docsMd += string(docsFile) + "\n\n"
	}
}

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
type Parameter struct {
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
	Parameters  []Parameter           `json:"parameters"`
	Responses   map[string]response   `json:"responses"`
	Security    []map[string][]string `json:"security,omitempty"`
	Servers     []server              `json:"servers,omitempty"`
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

func init() {
	api.Info.Description += docsMd
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

	api.Components.Schemas["types.ApiError"] = badRequestSchema
}

func AddTag(name, description string) {
	api.Tags = append(api.Tags, Tag{
		Name:        name,
		Description: description,
	})
}

func AddSecuritySchema(id, header, description string) {
	api.Components.Security[id] = security{
		Type:        "apiKey",
		Name:        header,
		In:          "header",
		Description: description,
	}
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
	AuthType    []string
}

func Route(doc *Doc) {
	// Generate schemaName, taking out bad things

	// Basic checks
	if len(doc.Params) == 0 {
		doc.Params = []Parameter{}
	}

	if len(doc.AuthType) == 0 {
		doc.AuthType = []string{}
	}

	if len(doc.Tags) == 0 {
		panic("no tags set in route: " + doc.Path)
	}

	if len(doc.Params) > 0 {
		for _, param := range doc.Params {
			if param.In == "" {
				panic("no in set in route: " + doc.Path)
			}

			if param.Name == "" {
				panic("no name set in route: " + doc.Path)
			}

			if param.Schema == nil {
				panic("no schema set in route: " + doc.Path)
			}

			if param.Description == "" {
				panic("no description set in route: " + doc.Path)
			}
		}
	}

	if doc.OpId == "" {
		panic("no opId set in route: " + doc.Path)
	}

	if doc.Path == "" {
		panic("no path set in route: " + doc.OpId)
	}

	schemaName := reflect.TypeOf(doc.Resp).String()

	schemaName = strings.ReplaceAll(schemaName, "docs.", "")

	if schemaName != "types.ApiError" {
		fmt.Println(schemaName)

		if _, ok := api.Components.Schemas[schemaName]; !ok {
			schemaRef, err := openapi3gen.NewSchemaRefForValue(doc.Resp, nil)

			if err != nil {
				panic(err)
			}

			api.Components.Schemas[schemaName] = schemaRef
		}
	}

	// Add in requests
	var reqBodyRef *ref
	if doc.Req != nil {
		schemaRef, err := openapi3gen.NewSchemaRefForValue(doc.Req, nil)

		if err != nil {
			panic(err)
		}

		reqSchemaName := reflect.TypeOf(doc.Req).String()

		fmt.Println("REQUEST:", reqSchemaName)

		api.Components.RequestBodies[doc.Method+"_"+reqSchemaName] = reqBody{
			Description: "Request body: " + reflect.TypeOf(doc.Req).String(),
			Required:    true,
			Content: map[string]content{
				"application/json": {
					Schema: schemaRef,
				},
			},
		}

		if _, ok := api.Paths[doc.Path]; !ok {
			api.Paths[doc.Path] = path{}
		}

		reqBodyRef = &ref{Ref: "#/components/requestBodies/" + doc.Method + "_" + reqSchemaName}
	}

	operationData := &operation{
		Tags:        doc.Tags,
		Summary:     doc.Summary,
		Description: doc.Description,
		ID:          doc.OpId,
		Parameters:  doc.Params,
		RequestBody: reqBodyRef,
		Responses: map[string]response{
			"200": {
				Description: "Success",
				Content: map[string]schemaResp{
					"application/json": {
						Schema: ref{
							Ref: "#/components/schemas/" + schemaName,
						},
					},
				},
			},
			"400": {
				Description: "Bad Request",
				Content: map[string]schemaResp{
					"application/json": {
						Schema: ref{
							Ref: "#/components/schemas/types.ApiError",
						},
					},
				},
			},
		},
	}

	if len(doc.AuthType) == 0 {
		doc.AuthType = []string{}
	}

	operationData.Security = []map[string][]string{}

	for _, auth := range doc.AuthType {
		operationData.Security = append(operationData.Security, map[string][]string{
			auth: {},
		})
	}

	op := api.Paths[doc.Path]

	switch strings.ToLower(doc.Method) {
	case "get":
		op.Get = operationData

		api.Paths[doc.Path] = op
	case "post":
		op.Post = operationData

		api.Paths[doc.Path] = op
	case "put":
		op.Put = operationData

		api.Paths[doc.Path] = op
	case "patch":
		op.Patch = operationData

		api.Paths[doc.Path] = op
	case "delete":
		op.Delete = operationData

		api.Paths[doc.Path] = op

	}
}

func GetSchema() any {
	return api
}
