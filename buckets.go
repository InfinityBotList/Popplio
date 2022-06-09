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
}
