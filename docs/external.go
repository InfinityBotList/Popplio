// Adds external documentation from Arcadia and other microservices to the documentation
package docs

import (
	"io"
	"net/http"
	"strconv"
	"time"

	jsoniter "github.com/json-iterator/go"

	log "github.com/sirupsen/logrus"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

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
		log.Info("Adding documentation from service " + strconv.Itoa(i) + " (" + name + ")" + " [" + url + "]")

		client := http.Client{
			Timeout: time.Second * 10,
		}

		req, err := client.Get(url)

		if err != nil {
			log.Error("Failed to get documentation from "+name+" [ "+url+" ]", err)
			continue
		}

		if req.StatusCode != 200 {
			log.Error("Failed to get documentation from " + name + " [ " + url + " ] with status code " + strconv.Itoa(req.StatusCode))
			continue
		}

		defer req.Body.Close()

		body, err := io.ReadAll(req.Body)

		if err != nil {
			log.Error("Failed to get documentation from "+name+" [ "+url+" ]", err)
			continue
		}

		var doc Openapi

		err = json.Unmarshal(body, &doc)

		if err != nil {
			log.Error("Failed to get documentation from "+name+" [ "+url+" ]", err)
			continue
		}

		for key, path := range doc.Paths {
			for _, v := range []*operation{
				path.Get,
				path.Post,
				path.Put,
				path.Patch,
				path.Delete,
			} {
				if v != nil {
					v.Servers = doc.Servers
					v.Tags = []string{"Arcadia"}
				}
			}

			api.Paths[key] = path
		}

		for key, schema := range doc.Components.Schemas {
			api.Components.Schemas[key] = schema
		}
	}
}
