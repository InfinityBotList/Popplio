package docs

import (
	"embed"
	"fmt"
	"os"
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

var api = Openapi{
	OpenAPI: "3.0.3",
	Info: Info{
		Title: "Infinity Bot List API",
		Description: `
Welcome to the Infinity Bot List API documentation!

`,
		TermsOfService: "https://infinitybotlist.com/terms",
		Version:        "6.0",
		Contact: Contact{
			Name:  "Infinity Bot List",
			URL:   "https://infinitybotlist.com",
			Email: "support@infinitybots.gg",
		},
		License: License{
			Name: "MIT",
			URL:  "https://opensource.org/licenses/MIT",
		},
	},
	Servers: []Server{
		{
			URL:         "https://spider.infinitybotlist.com",
			Description: "Popplio (v6)",
			Variables:   map[string]any{},
		},
	},
	Components: Component{
		Schemas:       make(map[string]any),
		Security:      make(map[string]Security),
		RequestBodies: make(map[string]ReqBody),
	},
}

func init() {
	api.Info.Description += docsMd
	api.Paths = NewMap[string, Path]()
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
	api.Components.Security[id] = Security{
		Type:        "apiKey",
		Name:        header,
		In:          "header",
		Description: description,
	}
}

func Route(doc *Doc) *Doc {
	// Generate schemaName, taking out bad things

	// Basic checks
	if len(doc.Params) == 0 {
		doc.Params = []Parameter{}
	}

	if len(doc.AuthType) == 0 {
		doc.AuthType = []types.TargetType{}
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
		if os.Getenv("DEBUG") == "true" {
			fmt.Println(schemaName)
		}

		if _, ok := api.Components.Schemas[schemaName]; !ok {
			schemaRef, err := openapi3gen.NewSchemaRefForValue(doc.Resp, nil)

			if err != nil {
				panic(err)
			}

			api.Components.Schemas[schemaName] = schemaRef
		}
	}

	// Add in requests
	var reqBodyRef *Schema
	if doc.Req != nil {
		schemaRef, err := openapi3gen.NewSchemaRefForValue(doc.Req, nil)

		if err != nil {
			panic(err)
		}

		reqSchemaName := reflect.TypeOf(doc.Req).String()

		if os.Getenv("DEBUG") == "true" {
			fmt.Println("REQUEST:", reqSchemaName)
		}

		api.Components.RequestBodies[doc.Method+"_"+reqSchemaName] = ReqBody{
			Description: "Request body: " + reflect.TypeOf(doc.Req).String(),
			Required:    true,
			Content: map[string]Content{
				"application/json": {
					Schema: schemaRef,
				},
			},
		}

		if _, ok := api.Paths.Get(doc.Path); !ok {
			api.Paths.Set(doc.Path, Path{})
		}

		reqBodyRef = &Schema{Ref: "#/components/requestBodies/" + doc.Method + "_" + reqSchemaName}
	}

	operationData := &Operation{
		Tags:        doc.Tags,
		Summary:     doc.Summary,
		Description: doc.Description,
		ID:          doc.OpId,
		Parameters:  doc.Params,
		RequestBody: reqBodyRef,
		Responses: map[string]Response{
			"200": {
				Description: "Success",
				Content: map[string]SchemaResp{
					"application/json": {
						Schema: Schema{
							Ref: "#/components/schemas/" + schemaName,
						},
					},
				},
			},
			"400": {
				Description: "Bad Request",
				Content: map[string]SchemaResp{
					"application/json": {
						Schema: Schema{
							Ref: "#/components/schemas/types.ApiError",
						},
					},
				},
			},
		},
	}

	if len(doc.AuthType) == 0 {
		doc.AuthType = []types.TargetType{}
	}

	operationData.Security = []map[string][]string{}

	for _, auth := range doc.AuthType {
		var authSchema string

		switch auth {
		case types.TargetTypeUser:
			authSchema = "User"
		case types.TargetTypeBot:
			authSchema = "Bot"
		case types.TargetTypeServer:
			authSchema = "Server"
		default:
			panic("unknown auth type: " + fmt.Sprint(auth))
		}

		operationData.Security = append(operationData.Security, map[string][]string{
			authSchema: {},
		})
	}

	op, _ := api.Paths.Get(doc.Path)

	switch strings.ToLower(doc.Method) {
	case "get":
		op.Get = operationData

		api.Paths.Set(doc.Path, op)
	case "post":
		op.Post = operationData

		api.Paths.Set(doc.Path, op)
	case "put":
		op.Put = operationData

		api.Paths.Set(doc.Path, op)
	case "patch":
		op.Patch = operationData

		api.Paths.Set(doc.Path, op)
	case "delete":
		op.Delete = operationData

		api.Paths.Set(doc.Path, op)
	}

	return doc
}

func GetSchema() any {
	return api
}
