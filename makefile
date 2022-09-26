GOOGLEAPIS_DIR = ./googleapis-proto
APP_PREFIX = simple-grpc-transcode
DOCKER_REPO = saifmaruf
RELEASE = 0.1.7

jwks:
	go run src/jwks-tools/jwks.go

proto-user:
	protoc -I${GOOGLEAPIS_DIR} -I. -I/usr/local/include --include_imports --include_source_info --descriptor_set_out=proto/user-grpc-transcode.pd --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative proto/user/user.proto && \
	kubectl create cm user-rpc-proto-descriptor --from-file proto/user-grpc-transcode.pd -n dev --dry-run=client -o yaml > manifests/user/user-rpc-descriptor-cm.yaml

build-user:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o container/user/${APP_PREFIX}-user src/user/main.go && cp -r keys container/user

docker-user:
	docker build -t ${DOCKER_REPO}/${APP_PREFIX}-user:${RELEASE} container/user && \
	docker push ${DOCKER_REPO}/${APP_PREFIX}-user:${RELEASE} && \
	yq -y ".spec.template.spec.containers[0].image = \"${DOCKER_REPO}/${APP_PREFIX}-user:${RELEASE}\"" manifests/user/user-deployment.yaml > temp.yaml && \
	mv temp.yaml manifests/user/user-deployment.yaml

deploy-user:
	kubectl apply -f manifests/user


proto-transaction:
	protoc -I${GOOGLEAPIS_DIR} -I. -I/usr/local/include --include_imports --include_source_info --descriptor_set_out=proto/transaction-grpc-transcode.pd --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative proto/transaction/transaction.proto && \
	kubectl create cm transaction-rpc-proto-descriptor --from-file proto/transaction-grpc-transcode.pd -n dev --dry-run=client -o yaml > manifests/transaction/transaction-rpc-descriptor-cm.yaml

build-transaction:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o container/transaction/${APP_PREFIX}-transaction src/transaction/main.go && cp -r keys container/transaction

docker-transaction:
	docker build -t ${DOCKER_REPO}/${APP_PREFIX}-transaction:${RELEASE} container/transaction && \
	docker push ${DOCKER_REPO}/${APP_PREFIX}-transaction:${RELEASE} && \
	yq -y ".spec.template.spec.containers[0].image = \"${DOCKER_REPO}/${APP_PREFIX}-transaction:${RELEASE}\"" manifests/transaction/transaction-deployment.yaml > temp.yaml && \
	mv temp.yaml manifests/transaction/transaction-deployment.yaml

deploy-transaction:
	kubectl apply -f manifests/transaction

clean:
	find ./proto -regex ".*\.go\|.*\.pd"|xargs rm
	rm -f container/user/${APP_PREFIX}-user
	rm -f container/transaction/${APP_PREFIX}-transaction

deploy-pg: 
	kubectl apply -f manifests/postgres.yaml

deploy-gateway:
	kubectl apply -f manifests/gateway

deploy-authn:
	kubectl apply -f manifests/authetication/jwt

proto-all: proto-user proto-transaction
build-all: build-user build-transaction
docker-all: docker-user docker-transaction
deploy-all: deploy-user deploy-transaction deploy-pg deploy-gateway

all: proto-all build-all docker-all deploy-all