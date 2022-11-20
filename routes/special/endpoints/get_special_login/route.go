package get_special_login

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"net/http"
	"os"
	"popplio/api"
	"popplio/docs"
	"popplio/types"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/login/{act}",
		OpId:        "get_special_login",
		Summary:     "Special Login",
		Description: "This endpoint is used for special login actions. For example, data requests.",
		Tags:        []string{api.CurrentTag},
		Resp:        "[Redirect]",
	})
}

func Route(d api.RouteData, r *http.Request) {
	cliId := os.Getenv("CLIENT_ID")
	redirectUrl := os.Getenv("REDIRECT_URL")

	// Create HMAC of current time in seconds to protect against fucked up redirects
	h := hmac.New(sha512.New, []byte(os.Getenv("CLIENT_SECRET")))

	ctime := strconv.FormatInt(time.Now().Unix(), 10)

	var act = chi.URLParam(r, "act")

	h.Write([]byte(ctime + "@" + act))

	hmacData := hex.EncodeToString(h.Sum(nil))

	d.Resp <- types.HttpResponse{
		Redirect: "https://discord.com/api/oauth2/authorize?client_id=" + cliId + "&scope=identify&response_type=code&redirect_uri=" + redirectUrl + "&state=" + ctime + "." + hmacData + "." + act,
	}
}
