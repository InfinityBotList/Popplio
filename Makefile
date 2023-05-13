TEST__USER_ID := 728871946456137770

all:
	CGO_ENABLED=0 go build -v 
tests:
	CGO_ENABLED=0 go test -v -coverprofile=coverage.out ./...
ts:
	~/go/bin/tygo generate
