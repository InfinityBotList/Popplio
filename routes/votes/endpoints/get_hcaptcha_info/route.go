package get_hcaptcha_info

import (
	"net/http"

	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get HCaptcha Info",
		Description: "Gets hcaptcha info (sitekey)",
		Resp:        types.HCaptchaInfo{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	data := types.HCaptchaInfo{
		SiteKey: state.Config.Hcaptcha.SiteKey,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
