package main

import (
	"net/http"
	"time"
)

func createBucketMods() {
	// Please place all custom buckets here

	// This endpoint needs a bucket moderator to handle PUT and GET at the same time
	bucketModerators["gvotes"] = func(r *http.Request) moderatedBucket {
		newBucket := moderatedBucket{}

		if r.Method == "PUT" {
			newBucket.BucketName = "cvotes"
			newBucket.ChangeRL = true
			newBucket.Requests = 3
			newBucket.Time = 2 * time.Minute
		} else {
			newBucket.BucketName = "gvotes"
		}

		return newBucket
	}

	bucketModerators["greminder"] = func(r *http.Request) moderatedBucket {
		newBucket := moderatedBucket{}

		if r.Method == "PUT" {
			newBucket.BucketName = "creminder"
			newBucket.ChangeRL = true
			newBucket.Requests = 5
			newBucket.Time = 1 * time.Minute
		} else {
			newBucket.BucketName = "greminder"
		}

		return newBucket
	}

	bucketModerators["gbot"] = func(r *http.Request) moderatedBucket {
		return moderatedBucket{
			BucketName: "gbot",
			Bypass:     true,
		}
	}

	bucketModerators["guser"] = func(r *http.Request) moderatedBucket {
		return moderatedBucket{
			BucketName: "guser",
			Bypass:     true,
		}
	}

	bucketModerators["glstats"] = func(r *http.Request) moderatedBucket {
		return moderatedBucket{
			BucketName: "glstats",
			Bypass:     true,
		}
	}
}
