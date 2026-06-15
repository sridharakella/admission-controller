# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Kubernetes mutating admission webhook controller written in Go. It intercepts pod creation requests and injects a sidecar container into all pods with the `app=frontend` label.

**Key behavior:** The webhook injects a sidecar into every frontend pod at admission time (before scheduling). There is no node-based filtering because mutating admission webhooks run before the Kubernetes scheduler assigns pods to nodes.

## Development Commands

**Use Makefile targets (preferred):**
```bash
make help              # Show all available targets
make build             # Build Go binary
make fmt               # Format code
make vet               # Run go vet
make tidy              # Tidy go modules
make test              # Run tests
make all               # Format, vet, tidy, and build
```

**Docker image:**
```bash
make docker-build IMAGE_TAG=your-registry/admission-controller:v1.0.0
make docker-push IMAGE_TAG=your-registry/admission-controller:v1.0.0
make docker-build-push # Build and push in one command
```

**Run locally (requires TLS certs):**
```bash
go run . -tls-cert=./certs/tls.crt -tls-key=./certs/tls.key -port=8443
```

## Deployment to Kubernetes

**Quick deployment with cert-manager:**
```bash
make install-cert-manager  # If cert-manager not installed
make deploy-with-wait      # Deploy with proper ordering and wait for readiness
```

**Manual deployment:**
```bash
make deploy                # Apply all manifests at once
make logs                  # View admission controller logs
make status                # Check deployment status
```

**Testing:**
```bash
make test-pod              # Create a test frontend pod
kubectl apply -f deploy/test-frontend-deployment.yaml  # Deploy test workload
```

**Uninstall:**
```bash
make undeploy              # Remove all resources (webhook config first)
```

**Important:** Before deploying, update the image reference in `deploy/03-deployment.yaml` to match your container registry.

## Architecture

**Core Components:**

1. **main.go** - Server initialization and routing
   - Configures HTTPS server with TLS (minimum TLS 1.2)
   - Exposes two endpoints: `/mutate` and `/health`
   - Accepts command-line flags: `-tls-cert`, `-tls-key`, `-port` (default 8443)

2. **webhook.go** - Admission webhook logic
   - `WebhookHandler.ServeMutate()` - HTTP handler for admission requests
   - `WebhookHandler.mutate()` - Core mutation logic with filtering
   - `createSidecarPatch()` - Generates JSON Patch operations for sidecar injection

**Request Flow:**

1. Kubernetes API server sends AdmissionReview to `/mutate` endpoint
2. Parse incoming pod object from request
3. Filter: Only process pods with `app=frontend` label
4. If frontend pod → inject sidecar container
5. Return AdmissionResponse with JSONPatch or allow without changes

**Sidecar Injection Details:**

The injected sidecar (webhook.go:117-194):
- Container name: `frontend-sidecar`
- Image: `alpine:latest`
- Command: `tail -f /dev/null` (keeps container running)
- Sets environment variables: `SIDECAR_MODE`, `NODE_NAME`, `POD_NAME`
- Resource requests: 100m CPU, 128Mi memory
- Resource limits: 200m CPU, 256Mi memory
- Adds annotation `sidecar-injector: injected` to pod

The JSONPatch operations handle both empty containers array and existing containers, ensuring the patch works correctly regardless of pod spec structure.

## Deployment Architecture

**Required Kubernetes resources:**
- Namespace: `admission-controller`
- ServiceAccount with RBAC (ClusterRole to list pods - currently unused but may be needed for future enhancements)
- Service exposing the controller on port 443 (routes to container port 8443)
- Deployment with TLS-secured HTTPS server
- TLS certificates (via cert-manager Certificate resource or manual Secret)
- MutatingWebhookConfiguration pointing to the service `/mutate` endpoint

**Certificate Management:**
The deployment uses cert-manager with a self-signed issuer. The CA bundle is automatically injected into the webhook configuration via the `cert-manager.io/inject-ca-from` annotation.

**Security Context:**
The deployment runs as non-root user (UID 1000) with:
- Read-only root filesystem
- No privilege escalation
- All capabilities dropped
- SeccompProfile: RuntimeDefault

## Important Implementation Details

**Dockerfile considerations:**
- Multi-stage build with Go 1.21 builder and Alpine final image
- Binary must have execute permissions (`chmod +x`) for non-root user
- WORKDIR is `/app` (not `/root`) for non-root compatibility
- Binary and `/app` directory owned by UID 1000

**Admission webhook timing:**
Mutating admission webhooks run **before** the Kubernetes scheduler assigns pods to nodes. This means:
- `spec.nodeName` is always empty at admission time
- Any logic requiring node assignment must be handled differently (e.g., via a controller watching pod updates)
- Current implementation injects sidecars into all frontend pods without node-based filtering
