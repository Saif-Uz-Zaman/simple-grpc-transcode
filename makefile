GOOGLEAPIS_DIR = ./googleapis-proto
APP_PREFIX = simple-grpc-transcode


proto-user:
	protoc -I${GOOGLEAPIS_DIR} -I. -I/usr/local/include --include_imports --include_source_info --descriptor_set_out=proto/user-grpc-transcode.pd --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative proto/user/user.proto

build-user:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o container/user/${APP_PREFIX}-user src/user/main.go