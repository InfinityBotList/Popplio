package ratelimit

import (
	"crypto/sha512"
	"fmt"
	"net/http"
	"popplio/state"
	"popplio/utils"
	"strconv"
	"strings"
	"time"
)

// Represents a moderated bucket typically used in 'combined' endpoints like Get/Create Votes which are just branches off a common function
// This is also the concept used in so-called global ratelimits
type ModeratedBucket struct {
	BucketName string

	// Internally set, dont change
	Global bool

	// Whether or not to keep original rl
	ChangeRL bool

	Requests int
	Time     time.Duration

	// Whether or not to just bypass the ratelimit altogether
	Bypass bool
}

// Default global ratelimit handler
var DefaultGlobalBucket = ModeratedBucket{BucketName: "global", Requests: 500, Time: 2 * time.Minute}

func bucketHandle(bucket ModeratedBucket, id string, w http.ResponseWriter, r *http.Request) bool {
	rlKey := "rl:" + id + "-" + bucket.BucketName

	v := state.Redis.Get(r.Context(), rlKey).Val()

	if v == "" {
		v = "0"

		err := state.Redis.Set(state.Context, rlKey, "0", bucket.Time).Err()

		if err != nil {
			state.Logger.Error(err)
			return false
		}
	}

	err := state.Redis.Incr(state.Context, rlKey).Err()

	if err != nil {
		state.Logger.Error(err)
		return false
	}

	vInt, err := strconv.Atoi(v)

	if err != nil {
		state.Logger.Error(err)
		return false
	}

	if vInt < 0 {
		state.Redis.Expire(state.Context, rlKey, 1*time.Second)
		vInt = 0
	}

	if vInt > bucket.Requests {
		retryAfter := state.Redis.TTL(state.Context, rlKey).Val()

		if bucket.Global {
			w.Header().Set("X-Global-Ratelimit", "true")
		}

		w.Header().Set("Retry-After", strconv.FormatFloat(retryAfter.Seconds(), 'g', -1, 64))

		w.WriteHeader(http.StatusTooManyRequests)

		// Set ratelimit to expire in more time if not global
		if !bucket.Global {
			state.Redis.Expire(state.Context, rlKey, retryAfter+2*time.Second)
		}

		w.Write([]byte("{\"message\":\"You're being rate limited!\",\"error\":true}"))

		return false
	}

	if bucket.Global {
		w.Header().Set("X-Ratelimit-Global-Req-Made", strconv.Itoa(vInt))
	} else {
		w.Header().Set("X-Ratelimit-Req-Made", strconv.Itoa(vInt))
	}
	return true
}

// Public ratelimit handler
func Ratelimit(reqs int, t time.Duration, bucket ModeratedBucket, w http.ResponseWriter, r *http.Request) bool {
	// Get ratelimit from redis
	var id string

	auth := r.Header.Get("Authorization")

	if auth != "" {
		if strings.HasPrefix(auth, "User ") {
			idCheck := utils.AuthCheck(auth, false)

			if idCheck == nil {
				// Bot does not exist, return
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("{\"message\":\"Invalid API token\",\"error\":true}"))
				return false
			}

			id = *idCheck
		} else {
			idCheck := utils.AuthCheck(auth, true)

			if idCheck == nil {
				// Bot does not exist, return
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("{\"message\":\"Invalid API token\",\"error\":true}"))
				return false
			}

			id = *idCheck
		}
	} else {
		remoteIp := strings.Split(strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", ""), ",")

		// For user privacy, hash the remote ip
		hasher := sha512.New()
		hasher.Write([]byte(remoteIp[0]))
		id = fmt.Sprintf("%x", hasher.Sum(nil))
	}

	if ok := bucketHandle(bucket, id, w, r); !ok {
		return false
	}

	return true
}
