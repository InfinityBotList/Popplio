package special

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strconv"
	"strings"
	"time"

	b64 "encoding/base64"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgtype"
	jsoniter "github.com/json-iterator/go"
)

const (
	tagName = "Special Routes"
	ddrStr  = `
SELECT 
  tbl.relname AS table_name,
  col.attname AS column_name,
  referenced_tbl.relname AS foreign_table_name,
  referenced_field.attname AS foreign_column_name
FROM pg_constraint c
    INNER JOIN pg_namespace AS sh ON sh.oid = c.connamespace
    INNER JOIN (SELECT oid, unnest(conkey) as conkey FROM pg_constraint) con ON c.oid = con.oid
    INNER JOIN pg_class tbl ON tbl.oid = c.conrelid
    INNER JOIN pg_attribute col ON (col.attrelid = tbl.oid AND col.attnum = con.conkey)
    INNER JOIN pg_class referenced_tbl ON c.confrelid = referenced_tbl.oid
    INNER JOIN pg_namespace AS referenced_sh ON referenced_sh.oid = referenced_tbl.relnamespace
    INNER JOIN (SELECT oid, unnest(confkey) as confkey FROM pg_constraint) conf ON c.oid = conf.oid
    INNER JOIN pg_attribute referenced_field ON (referenced_field.attrelid = c.confrelid AND referenced_field.attnum = conf.confkey)
WHERE c.contype = 'f'`
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type InternalOauthUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Disc     string `json:"discriminator"`
	TID      string `json:"-"` // Only set in taskFn
}

type KVPair struct {
	Key   string
	Value any
}

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "Special endpoints, these don't return JSONs and are purely for browser use."
}

func (b Router) Routes(r *chi.Mux) {
	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/login/{act}",
		OpId:        "get_special_login",
		Summary:     "Special Login",
		Description: "This endpoint is used for special login actions. For example, data requests.",
		Tags:        []string{tagName},
		Resp:        "[Redirect]",
	})
	r.Get("/login/{act}", func(w http.ResponseWriter, r *http.Request) {
		cliId := os.Getenv("CLIENT_ID")
		redirectUrl := os.Getenv("REDIRECT_URL")

		// Create HMAC of current time in seconds to protect against fucked up redirects
		h := hmac.New(sha512.New, []byte(os.Getenv("CLIENT_SECRET")))

		ctime := strconv.FormatInt(time.Now().Unix(), 10)

		var act = chi.URLParam(r, "act")

		h.Write([]byte(ctime + "@" + act))

		hmacData := hex.EncodeToString(h.Sum(nil))

		http.Redirect(w, r, "https://discord.com/api/oauth2/authorize?client_id="+cliId+"&scope=identify&response_type=code&redirect_uri="+redirectUrl+"&state="+ctime+"."+hmacData+"."+act, http.StatusFound)
	})

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/cosmog",
		OpId:        "get_special_login_resp",
		Summary:     "Special Login Handler",
		Description: "This endpoint is used to respond to a special login. It then spawns the task such as data requests etc.",
		Tags:        []string{tagName},
		Resp:        "[Redirect+Task Creation]",
	})
	r.Get("/cosmog", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		resp := make(chan types.HttpResponse)

		go func() {
			act := r.URL.Query().Get("state")

			// Split act and hmac
			actSplit := strings.Split(act, ".")

			if len(actSplit) != 3 {
				resp <- types.HttpResponse{
					Status: http.StatusBadRequest,
					Data:   "Invalid state",
				}
				return
			}

			// Check hmac
			h := hmac.New(sha512.New, []byte(os.Getenv("CLIENT_SECRET")))

			h.Write([]byte(actSplit[0] + "@" + actSplit[2]))

			hmacData := hex.EncodeToString(h.Sum(nil))

			if hmacData != actSplit[1] {
				resp <- types.HttpResponse{
					Status: http.StatusBadRequest,
					Data:   "Invalid state",
				}
				return
			}

			// Check time
			ctime, err := strconv.ParseInt(actSplit[0], 10, 64)

			if err != nil {
				resp <- types.HttpResponse{
					Status: http.StatusBadRequest,
					Data:   "Invalid state",
				}
				return
			}

			if time.Now().Unix()-ctime > 300 {
				resp <- types.HttpResponse{
					Status: http.StatusBadRequest,
					Data:   "Invalid state, HMAC is too old",
				}
				return
			}

			// Remove out the actual action
			act = actSplit[2]

			// Check code with discords api
			data := url.Values{}

			data.Set("client_id", os.Getenv("CLIENT_ID"))
			data.Set("client_secret", os.Getenv("CLIENT_SECRET"))
			data.Set("grant_type", "authorization_code")
			data.Set("code", r.URL.Query().Get("code"))
			data.Set("redirect_uri", os.Getenv("REDIRECT_URL"))

			response, err := http.PostForm("https://discord.com/api/oauth2/token", data)

			if err != nil {
				resp <- types.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
				return
			}

			defer response.Body.Close()

			body, err := io.ReadAll(response.Body)

			if err != nil {
				resp <- types.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
				return
			}

			var token struct {
				AccessToken string `json:"access_token"`
				Scope       string `json:"scope"`
			}

			err = json.Unmarshal(body, &token)

			if err != nil {
				resp <- types.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
				return
			}

			state.Logger.Info(token)

			if !strings.Contains(token.Scope, "identify") {
				resp <- types.HttpResponse{
					Status: http.StatusBadRequest,
					Data:   "Invalid scope: scope contain identify, is currently " + token.Scope,
				}
				return
			}

			// Get user info
			req, err := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)

			if err != nil {
				resp <- types.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
				return
			}

			req.Header.Set("Authorization", "Bearer "+token.AccessToken)

			client := &http.Client{Timeout: time.Second * 10}

			response, err = client.Do(req)

			if err != nil {
				resp <- types.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
				return
			}

			defer response.Body.Close()

			body, err = ioutil.ReadAll(response.Body)

			if err != nil {
				resp <- types.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
				return
			}

			var user InternalOauthUser

			err = json.Unmarshal(body, &user)

			if err != nil {
				resp <- types.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
				return
			}

			taskId := utils.RandString(196)

			err = state.Redis.Set(ctx, taskId, "WAITING", time.Hour*8).Err()

			if err != nil {
				resp <- types.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
				return
			}

			remoteIp := strings.Split(strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", ""), ",")

			if act == "dr" {
				go dataTask(taskId, user.ID, remoteIp[0], false)
			} else if act == "ddr" {
				go dataTask(taskId, user.ID, remoteIp[0], true)
			} else if act == "gettoken" {
				token := utils.RandString(128)

				_, err := state.Pool.Exec(ctx, "UPDATE users SET api_token = $1 WHERE user_id = $2", token, user.ID)

				if err != nil {
					resp <- types.HttpResponse{
						Status: http.StatusInternalServerError,
						Data:   err.Error(),
					}
					return
				}

				resp <- types.HttpResponse{
					Data: token,
				}
				return

			} else {
				resp <- utils.ApiDefaultReturn(http.StatusNotFound)
				return
			}

			resp <- types.HttpResponse{
				Redirect: "/cosmog/tasks/" + taskId + "?n=" + b64.URLEncoding.EncodeToString(body),
			}
		}()

		utils.Respond(ctx, w, resp)
	})

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/cosmog/tasks/{tid}",
		OpId:        "get_cosmog_task_status",
		Summary:     "Special Login Task View",
		Description: "Shows the status of a task that has been started by a special login.",
		Tags:        []string{tagName},
		Resp:        "[HTML]",
	})
	r.Get("/cosmog/tasks/{tid}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		resp := make(chan types.HttpResponse)

		go func() {
			var user InternalOauthUser

			tid := chi.URLParam(r, "tid")

			if tid == "" {
				resp <- types.HttpResponse{
					Status: http.StatusBadRequest,
					Data:   "Invalid task id",
				}
				return
			}

			userStr := r.URL.Query().Get("n")

			if userStr == "" {
				user = InternalOauthUser{
					ID:       "Unknown",
					Username: "Unknown",
					Disc:     "0000",
				}
			} else {
				body, err := b64.URLEncoding.DecodeString(userStr)

				if err != nil {
					resp <- types.HttpResponse{
						Status: http.StatusInternalServerError,
						Data:   err.Error(),
					}
					return
				}

				err = json.Unmarshal(body, &user)

				if err != nil {
					resp <- types.HttpResponse{
						Status: http.StatusInternalServerError,
						Data:   err.Error(),
					}
					return
				}
			}

			user.TID = tid

			t, err := template.ParseFiles("html/taskpage.html")

			if err != nil {
				resp <- types.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			t.Execute(w, user)

			resp <- types.HttpResponse{
				Stub: true,
			}
		}()

		utils.Respond(ctx, w, resp)
	})

	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/cosmog/tasks/{tid}.arceus",
		OpId:        "get_cosmog_task_tid",
		Summary:     "Special Login Task View JSON",
		Description: "Returns the status of a task as a arbitary json.",
		Tags:        []string{tagName},
		Resp:        "[JSON]",
	})
	r.Get("/cosmog/tasks/{tid}.arceus", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		resp := make(chan types.HttpResponse)

		go func() {
			tid := chi.URLParam(r, "tid")

			if tid == "" {
				resp <- types.HttpResponse{
					Status: http.StatusBadRequest,
					Data:   "Invalid task id",
				}
				return
			}

			task, err := state.Redis.Get(ctx, tid).Result()

			if err == redis.Nil {
				resp <- types.HttpResponse{
					Status: http.StatusNotFound,
					Data:   "Task not found",
				}
				return
			}

			if err != nil {
				resp <- types.HttpResponse{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
				}
				return
			}

			resp <- types.HttpResponse{
				Data: task,
			}
		}()

		utils.Respond(ctx, w, resp)
	})
}

func dataTask(taskId string, id string, ip string, del bool) {
	ctx := state.Context

	state.Redis.SetArgs(ctx, taskId, "Fetching basic user data", redis.SetArgs{
		KeepTTL: true,
	}).Err()

	var keys []*struct {
		ForeignTable      string `db:"foreign_table_name"`
		TableName         string `db:"table_name"`
		ColumnName        string `db:"column_name"`
		ForeignColumnName string `db:"foreign_column_name"`
	}

	data, err := state.Pool.Query(ctx, ddrStr)

	if err != nil {
		state.Logger.Error(err)

		state.Redis.SetArgs(ctx, taskId, "Critical:"+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})

		return
	}

	if err := pgxscan.ScanAll(&keys, data); err != nil {
		state.Logger.Error(err)

		state.Redis.SetArgs(ctx, taskId, "Critical:"+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})

		return
	}

	finalDump := make(map[string]any)

	for _, key := range keys {
		if key.ForeignTable == "users" {
			sqlStmt := "SELECT * FROM " + key.TableName + " WHERE " + key.ColumnName + "= $1"

			data, err := state.Pool.Query(ctx, sqlStmt, id)

			if err != nil {
				state.Logger.Error(err)
			}

			var rows []map[string]any

			if err := pgxscan.ScanAll(&rows, data); err != nil {
				state.Logger.Error(err)

				state.Redis.SetArgs(ctx, taskId, "Critical:"+err.Error(), redis.SetArgs{
					KeepTTL: true,
				})

				return
			}

			if del {
				sqlStmt = "DELETE FROM " + key.TableName + " WHERE " + key.ColumnName + "= $1"

				_, err := state.Pool.Exec(ctx, sqlStmt, id)

				if err != nil {
					state.Logger.Error(err)

					state.Redis.SetArgs(ctx, taskId, "Critical:"+err.Error(), redis.SetArgs{
						KeepTTL: true,
					})

					return
				}
			}

			finalDump[key.TableName] = rows
		}
	}

	state.Redis.SetArgs(ctx, taskId, "Fetching postgres backups on this user", redis.SetArgs{
		KeepTTL: true,
	})

	rows, err := state.BackupsPool.Query(ctx, "SELECT col, data, ts, id FROM backups")

	if err != nil {
		state.Logger.Error("Failed to get backups")
		state.Redis.SetArgs(ctx, taskId, "Failed to fetch backup data: "+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})
		return
	}

	defer rows.Close()

	var backups []any

	var foundBackup bool

	for rows.Next() {
		var col pgtype.Text
		var data pgtype.JSONB
		var ts pgtype.Timestamptz
		var uid pgtype.UUID

		err = rows.Scan(&col, &data, &ts, &uid)

		if err != nil {
			state.Logger.Error("Failed to scan backup")
			state.Redis.SetArgs(ctx, taskId, "Failed to fetch backup data: "+err.Error()+". Ignoring", redis.SetArgs{
				KeepTTL: true,
			})
			continue
		}

		var dataPacket []KVPair

		err = json.Unmarshal([]byte(data.Bytes), &dataPacket)

		if err != nil {
			state.Logger.Error("Failed to decode backup")
			state.Redis.SetArgs(ctx, taskId, "Failed to fetch backup data: "+err.Error()+". Ignoring", redis.SetArgs{
				KeepTTL: true,
			})
			continue
		}

		var backupDat = make(map[string]any)

		for _, kvpair := range dataPacket {
			if kvpair.Key == "userID" || kvpair.Key == "author" || kvpair.Key == "main_owner" {
				val, ok := kvpair.Value.(string)
				if !ok {
					continue
				}

				if val == id {
					foundBackup = true
					break
				}
			}
		}

		if foundBackup {
			backupDat["col"] = col.String
			backupDat["data"] = dataPacket
			backupDat["ts"] = ts.Time
			backupDat["id"] = toString(uid)
			backups = append(backups, backupDat)

			if del {
				_, err := state.BackupsPool.Exec(ctx, "DELETE FROM backups WHERE id=$1", toString(uid))
				if err != nil {
					state.Logger.Error("Failed to delete backup")
					state.Redis.SetArgs(ctx, taskId, "Failed to delete backup: "+err.Error(), redis.SetArgs{
						KeepTTL: true,
					})
					return
				}
			}
		}

		foundBackup = false
	}

	finalDump["backups"] = backups

	bytes, err := json.Marshal(finalDump)

	if err != nil {
		state.Logger.Error("Failed to encode data")
		state.Redis.SetArgs(ctx, taskId, "Failed to encode data: "+err.Error(), redis.SetArgs{
			KeepTTL: true,
		})
		return
	}

	state.Redis.SetArgs(ctx, taskId, string(bytes), redis.SetArgs{
		KeepTTL: false,
	})
}

// Given a UUID, returns a string representation of it
func toString(myUUID pgtype.UUID) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", myUUID.Bytes[0:4], myUUID.Bytes[4:6], myUUID.Bytes[6:8], myUUID.Bytes[8:10], myUUID.Bytes[10:16])
}