SHELL=/usr/bin/env bash -o pipefail
NAMESPACE=default
KUBECONFIG=/tmp/kubeconfig
VERSION ?= latest
IMAGE_TAG_BASE ?= quay.io/mittwald/kubernetes-secret-generator
IMG ?= secret-generator:${VERSION}

.PHONY: install
install: ## Install all resources (RBAC and Operator)
	@echo ....... Applying Rules and Service Account .......
	kubectl apply -f deploy/role.yaml -n ${NAMESPACE}
	kubectl apply -f deploy/role_binding.yaml  -n ${NAMESPACE}
	kubectl apply -f deploy/service_account.yaml  -n ${NAMESPACE}
	@echo ....... Applying Operator .......
	kubectl apply -f deploy/operator.yaml -n ${NAMESPACE}

.PHONY: installwithmonitoring
installwithmonitoring: ## Install all resources (RBAC and Operator) with monitoring role
	@echo ....... Applying Rules and Service Account .......
	kubectl apply -f deploy/role_with_service_permissions.yaml -n ${NAMESPACE}
	kubectl apply -f deploy/role_binding.yaml  -n ${NAMESPACE}
	kubectl apply -f deploy/service_account.yaml  -n ${NAMESPACE}
	@echo ....... Applying Operator .......
	kubectl apply -f deploy/operator.yaml -n ${NAMESPACE}


.PHONY: uninstall
uninstall: ## Uninstall all that all performed in the $ make install
	@echo ....... Uninstalling .......
	@echo ....... Deleting Rules and Service Account .......
	kubectl delete -f deploy/role.yaml -n ${NAMESPACE}
	kubectl delete -f deploy/role_binding.yaml -n ${NAMESPACE}
	kubectl delete -f deploy/service_account.yaml -n ${NAMESPACE}
	@echo ....... Deleting Operator .......
	kubectl delete -f deploy/operator.yaml -n ${NAMESPACE}

.PHONY: uninstallwithmonitoring
uninstallwithmonitoring: ## Uninstall all that all performed in the $ make installwithmonitoring
	@echo ....... Uninstalling .......
	@echo ....... Deleting Rules and Service Account .......
	kubectl delete -f deploy/role_with_service_permissions.yaml -n ${NAMESPACE}
	kubectl delete -f deploy/role_binding.yaml -n ${NAMESPACE}
	kubectl delete -f deploy/service_account.yaml -n ${NAMESPACE}
	@echo ....... Deleting Operator .......
	kubectl delete -f deploy/operator.yaml -n ${NAMESPACE}

.PHONY: test
test: kind
	@echo go test
	go test ./... -v

.PHONY: fmt
fmt:
	@echo go fmt
	go fmt $$(go list ./...)

.PHONY: kind
kind: ## Create a kind cluster to tefmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

.PHONY: build
build: fmt vet ## Build manager binary.
	go build -o bin/manager ./cmd/manager/main.go
	@exit $(.SHELLSTATUS)

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	docker build -t ${IMG} .
	@exit $(.SHELLSTATUS)

docker-push: ## Push docker image with the manager.
	docker tag ${IMG} ${IMAGE_TAG_BASE}:${VERSION}
	docker push ${IMAGE_TAG_BASE}
