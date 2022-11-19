// Defines a standard way to define routes
package api

import (
	"context"
	"net/http"
	"popplio/types"
	"popplio/utils"
)

// Stores the current tag
var CurrentTag string

type Method = int

const (
	GET Method = iota
	POST
	PATCH
	PUT
	DELETE
	HEAD
)

type Route struct {
	Method  Method
	Pattern string
	Handler func(d RouteData, r *http.Request)
	Docs    func()
}

type RouteData struct {
	Context context.Context
	Resp    chan types.HttpResponse
}

type Router interface {
	Get(pattern string, h http.HandlerFunc)
	Post(pattern string, h http.HandlerFunc)
	Patch(pattern string, h http.HandlerFunc)
	Put(pattern string, h http.HandlerFunc)
	Delete(pattern string, h http.HandlerFunc)
	Head(pattern string, h http.HandlerFunc)
}

func (r Route) Route(ro Router) {
	if r.Handler == nil {
		panic("Handler is nil")
	}

	if r.Docs == nil {
		panic("Docs is nil")
	}

	if r.Pattern == "" {
		panic("Pattern is empty")
	}

	if CurrentTag == "" {
		panic("CurrentTag is empty")
	}

	r.Docs()

	handle := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		resp := make(chan types.HttpResponse)

		go func() {
			r.Handler(RouteData{
				Context: ctx,
				Resp:    resp,
			}, req)
		}()

		utils.Respond(ctx, w, resp)
	}

	switch r.Method {
	case GET:
		ro.Get(r.Pattern, handle)
	case POST:
		ro.Post(r.Pattern, handle)
	case PATCH:
		ro.Patch(r.Pattern, handle)
	case PUT:
		ro.Put(r.Pattern, handle)
	case DELETE:
		ro.Delete(r.Pattern, handle)
	case HEAD:
		ro.Head(r.Pattern, handle)
	default:
		panic("Unknown method")
	}
}
