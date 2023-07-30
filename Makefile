TEST__USER_ID := 728871946456137770

all:
	CGO_ENABLED=0 go build -v 
	systemctl restart popplio-staging
build-cdocs:
	cd docs/cdocs && FRONTEND_URL=https://botlist.site npm run build && cd ..
tests:
	CGO_ENABLED=0 go test -v -coverprofile=coverage.out ./...
ts:
	rm -rvf /iblcdn/public/dev/bindings/popplio
	~/go/bin/tygo generate

	# Because tygo cant replace all instances of TeamPermission and other useful modifications
	sed -i 's:export type TeamPermission = string; //:// Note that:g' /iblcdn/public/dev/bindings/popplio/types.ts

	# Two steps to replace all instances of TeamPermission with TeamPermissions while keeping TeamPermissions intact	
	sed -i 's:TeamPermissions:TeamPermission:g' /iblcdn/public/dev/bindings/popplio/types.ts

	sed -i 's:TeamPermission:TeamPermissions:g' /iblcdn/public/dev/bindings/popplio/types.ts

	# Copy over go types
	mkdir /iblcdn/public/dev/bindings/popplio/go
	cp -rf types /iblcdn/public/dev/bindings/popplio/go

	# Patch to change package name to 'popltypes'
	sed -i 's:package types:package popltypes:g' /iblcdn/public/dev/bindings/popplio/go/types/*

promoteprod:
	rm -rf ../prod2
	cd .. && cp -rf staging prod2
	echo "prod" > ../prod2/config/current-env
	cd ../prod2 && make && rm -rf ../prod && mv -vf ../prod2 ../prod && systemctl restart popplio-prod
	cd ../prod && make ts

	# Git push to "current-prod" branch
	cd ../prod && git branch current-prod && git add -v . && git commit -m "Promote staging to prod" && git push -u origin HEAD:current-prod --force
