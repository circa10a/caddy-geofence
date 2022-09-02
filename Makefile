PROJECT="circa10a/caddy-geofence"

run:
	@if ! command -v xcaddy 1>/dev/null; then\
		echo "Need to install golangci-lint";\
		exit 1;\
	fi;
	xcaddy run

build:
	xcaddy build --with github.com/$(PROJECT)=./

lint:
	@if ! command -v golangci-lint 1>/dev/null; then\
		echo "Need to install golangci-lint";\
		exit 1;\
	fi;\
	golangci-lint run -v

build-linux: export GOOS = linux
build-linux:
	xcaddy build --with github.com/$(PROJECT)=./

redis: build-linux
	docker-compose up
