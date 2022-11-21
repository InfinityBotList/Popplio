// Adds external documentation from Arcadia and other microservices to the documentation
package docs

import (
	"encoding/json"
	"io"
	"net/http"
	"popplio/state"
	"strconv"
	"time"
)

// Arcadia documentor, returns the openapi schema URL and adds the arcadia tags to the openapi schema
func arcadia() (string, string) {
	AddTag("Arcadia", "The high-performance API server for Infinity Bot List. It powers all endpoints that require compile-time checked queries or high performance to be stable")

	return "Arcadia", "https://sovngarde.infinitybots.gg/eternatus"
}

// Common documentation code
func DocumentMicroservices() {
	services := []func() (string, string){
		arcadia,
	}

	for i, service := range services {
		name, url := service()
		state.Logger.Info("Adding documentation from service " + strconv.Itoa(i) + " (" + name + ")" + " [" + url + "]")

		client := http.Client{
			Timeout: time.Second * 10,
		}

		req, err := client.Get(url)

		if err != nil {
			state.Logger.Error("Failed to get documentation from "+name+" [ "+url+" ]", err)
			continue
		}

		if req.StatusCode != 200 {
			state.Logger.Error("Failed to get documentation from " + name + " [ " + url + " ] with status code " + strconv.Itoa(req.StatusCode))
			continue
		}

		defer req.Body.Close()

		body, err := io.ReadAll(req.Body)

		if err != nil {
			state.Logger.Error("Failed to get documentation from "+name+" [ "+url+" ]", err)
			continue
		}

		var doc Openapi

		err = json.Unmarshal(body, &doc)

		if err != nil {
			state.Logger.Error("Failed to get documentation from "+name+" [ "+url+" ]", err)
			continue
		}

		for pair := doc.Paths.Oldest(); pair != nil; pair = pair.Next() {
			key, path := pair.Key, pair.Value
			for _, v := range []*Operation{
				path.Get,
				path.Post,
				path.Put,
				path.Patch,
				path.Delete,
			} {
				if v != nil {
					v.Servers = doc.Servers
					v.Tags = []string{name}
				}
			}

			api.Paths.Set(key, path)
		}

		for key, schema := range doc.Components.Schemas {
			api.Components.Schemas[key] = schema
		}
	}
}
