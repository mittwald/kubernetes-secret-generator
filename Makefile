SHELL=/bin/bash
NAMESPACE=default
KUBECONFIG=/tmp/kubeconfig

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
kind: ## Create a kind cluster to test against
	kind create cluster --name kind-k8s-secret-generator
	kind get kubeconfig --name kind-k8s-secret-generator | tee ${KUBECONFIG}

.PHONY: build
build:
	operator-sdk build --go-build-args "-ldflags -X=version.Version=${SECRET_OPERATOR_VERSION}" ${DOCKER_IMAGE}
	@exit $(.SHELLSTATUS)
