package patch_pack

import (
	"popplio/api"
	"popplio/docs"
	"popplio/types"
)

var compiledMessages = api.CompileValidationErrors(PatchPack{})

type PatchPack struct {
	Name  string   `json:"name" validate:"required,min=3,max=20" msg:"Name must be between 3 and 20 characters"`
	URL   string   `json:"url" validate:"required,min=3,max=20,nospaces,notblank,alpha" msg:"URL must be between 3 and 20 characters without spaces and must be alphabetic"`
	Short string   `json:"short" validate:"required,min=10,max=100" msg:"Description must be between 10 and 100 characters"`
	Tags  []string `json:"tags" validate:"required,unique,min=1,max=5,dive,min=3,max=20,alpha,notblank,nonvulgar,nospaces" msg:"There must be between 1 and 5 tags without duplicates" amsg:"Each tag must be between 3 and 20 characters and alphabetic"`
	Bots  []string `json:"bots" validate:"required,unique,min=1,max=10,dive,numeric" msg:"There must be between 1 and 10 bots without duplicates"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Patch Pack",
		Description: "Edits a pack you are owner of. Returns 204 on success",
		Req:         PatchPack{},
		Resp:        types.AllPacks{},
	}
}
