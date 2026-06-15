# Variables
DOCKER_REGISTRY ?= your-registry
IMAGE_NAME ?= admission-controller
VERSION ?= v1.0.0
IMAGE_TAG = $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(VERSION)
NAMESPACE = admission-controller

.PHONY: help
help: ## Display this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the Go binary
	go build -o admission-controller .

.PHONY: test
test: ## Run tests
	go test -v ./...

.PHONY: fmt
fmt: ## Format Go code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: tidy
tidy: ## Tidy Go modules
	go mod tidy

.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t $(IMAGE_TAG) .
	docker tag $(IMAGE_TAG) $(DOCKER_REGISTRY)/$(IMAGE_NAME):latest

.PHONY: docker-push
docker-push: ## Push Docker image to registry
	docker push $(IMAGE_TAG)
	docker push $(DOCKER_REGISTRY)/$(IMAGE_NAME):latest

.PHONY: docker-build-push
docker-build-push: docker-build docker-push ## Build and push Docker image

.PHONY: generate-certs
generate-certs: ## Generate self-signed TLS certificates
	./scripts/generate-certs.sh

.PHONY: deploy
deploy: ## Deploy to Kubernetes
	kubectl apply -f deploy/

.PHONY: deploy-with-wait
deploy-with-wait: ## Deploy to Kubernetes and wait for readiness
	kubectl apply -f deploy/00-namespace.yaml
	kubectl apply -f deploy/01-rbac.yaml
	kubectl apply -f deploy/02-service.yaml
	kubectl apply -f deploy/04-certificate.yaml
	@echo "Waiting for certificate to be ready..."
	kubectl wait --for=condition=Ready certificate/admission-controller-tls -n $(NAMESPACE) --timeout=60s || true
	kubectl apply -f deploy/03-deployment.yaml
	@echo "Waiting for deployment to be ready..."
	kubectl wait --for=condition=Available deployment/admission-controller -n $(NAMESPACE) --timeout=120s
	kubectl apply -f deploy/05-webhook-config.yaml
	@echo "Deployment complete!"

.PHONY: undeploy
undeploy: ## Remove from Kubernetes
	kubectl delete -f deploy/05-webhook-config.yaml || true
	kubectl delete -f deploy/ || true

.PHONY: logs
logs: ## View admission controller logs
	kubectl logs -n $(NAMESPACE) -l app=admission-controller -f

.PHONY: status
status: ## Check deployment status
	@echo "Pods:"
	kubectl get pods -n $(NAMESPACE)
	@echo "\nDeployment:"
	kubectl get deployment -n $(NAMESPACE)
	@echo "\nService:"
	kubectl get service -n $(NAMESPACE)
	@echo "\nWebhook Configuration:"
	kubectl get mutatingwebhookconfigurations admission-controller-webhook

.PHONY: test-pod
test-pod: ## Create a test frontend pod
	@NODE_NAME=$$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}'); \
	echo "Creating test pod on node: $$NODE_NAME"; \
	kubectl apply -f - <<EOF
	apiVersion: v1
	kind: Pod
	metadata:
	  name: test-frontend-$$(date +%s)
	  labels:
	    app: frontend
	spec:
	  nodeName: $$NODE_NAME
	  containers:
	  - name: nginx
	    image: nginx:alpine
	    ports:
	    - containerPort: 80
	EOF

.PHONY: clean
clean: ## Clean build artifacts and certificates
	rm -f admission-controller
	rm -rf certs/

.PHONY: install-cert-manager
install-cert-manager: ## Install cert-manager
	kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
	@echo "Waiting for cert-manager to be ready..."
	kubectl wait --for=condition=Available --timeout=300s deployment/cert-manager-webhook -n cert-manager

.PHONY: all
all: fmt vet tidy build ## Run format, vet, tidy, and build
