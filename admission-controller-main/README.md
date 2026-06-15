# Kubernetes Admission Controller

A Kubernetes mutating admission webhook controller that conditionally injects sidecar containers into frontend pods based on node affinity.

## Overview

This admission controller intercepts pod creation requests and intelligently injects a sidecar container when:

1. Pod has the `app=frontend` label
2. Pod is scheduled to a specific node (`spec.nodeName` is set)
3. No other frontend pod with a sidecar is currently running on that node

This implements a **node-affinity-based sidecar injection pattern**, ensuring only one sidecar per node for frontend workloads.

## Features

- Smart sidecar injection based on node scheduling
- Kubernetes API integration to check existing pods
- TLS-secured webhook endpoint
- Health check endpoint
- Production-ready security contexts
- Support for both cert-manager and manual certificate management

## Quick Start

### Prerequisites

- Kubernetes cluster (v1.19+)
- `kubectl` configured
- Docker
- Container registry access

### Deploy to Kubernetes

1. **Build and push the Docker image:**

```bash
make docker-build IMAGE_TAG=your-registry/admission-controller:v1.0.0
make docker-push IMAGE_TAG=your-registry/admission-controller:v1.0.0
```

2. **Update the image reference in `deploy/03-deployment.yaml`**

3. **Deploy using cert-manager (recommended):**

```bash
# Install cert-manager
make install-cert-manager

# Deploy the admission controller
make deploy-with-wait
```

4. **Or deploy with manual certificates:**

```bash
# Generate certificates
make generate-certs

# Deploy
make deploy-with-wait
```

For detailed deployment instructions, see [DEPLOYMENT.md](DEPLOYMENT.md).

## Local Development

### Build

```bash
make build
```

### Run locally

Requires TLS certificates:

```bash
go run . -tls-cert=./certs/tls.crt -tls-key=./certs/tls.key -port=8443
```

### Format and test

```bash
make fmt
make vet
make test
```

## Architecture

### Components

- **main.go** - HTTPS server setup and routing
- **webhook.go** - Admission webhook logic and sidecar injection

### Request Flow

```
API Server → /mutate endpoint → Parse AdmissionReview
    → Check app=frontend label
    → Check spec.nodeName is set
    → Query API for existing frontend pods on node
    → Inject sidecar if conditions met
    → Return AdmissionResponse
```

### Injected Sidecar

The sidecar container includes:
- **Image:** `nginx:alpine` (customizable in webhook.go:175-217)
- **Port:** 8080
- **Environment variables:** `SIDECAR_MODE`, `NODE_NAME`, `POD_NAME`
- **Resources:** 100m CPU / 128Mi memory (requests), 200m CPU / 256Mi memory (limits)
- **Annotation:** `sidecar-injector: injected`

## Configuration

### Webhook Settings

Edit `deploy/05-webhook-config.yaml` to configure:

- **Failure Policy:** `Ignore` (default) or `Fail`
- **Namespace Selector:** Exclude specific namespaces
- **Timeout:** Default 10 seconds

### Sidecar Customization

Modify the sidecar configuration in `webhook.go:175-217`:

```go
sidecar := corev1.Container{
    Name:  "frontend-sidecar",
    Image: "nginx:alpine",  // Change this
    // ... customize other settings
}
```

## Testing

Create a test frontend pod:

```bash
make test-pod
```

Verify sidecar injection:

```bash
kubectl get pods -l app=frontend -o jsonpath='{.items[*].spec.containers[*].name}'
```

## Monitoring

View logs:

```bash
make logs
```

Check status:

```bash
make status
```

## Troubleshooting

See the [Troubleshooting section in DEPLOYMENT.md](DEPLOYMENT.md#troubleshooting) for common issues and solutions.

## Makefile Targets

```bash
make help  # Display all available targets
```

Common targets:
- `make build` - Build Go binary
- `make docker-build` - Build Docker image
- `make deploy` - Deploy to Kubernetes
- `make logs` - View logs
- `make status` - Check deployment status
- `make undeploy` - Remove from Kubernetes

## Project Structure

```
.
├── main.go                      # Server initialization
├── webhook.go                   # Webhook logic
├── go.mod                       # Go dependencies
├── Dockerfile                   # Container image
├── Makefile                     # Build automation
├── DEPLOYMENT.md                # Deployment guide
├── deploy/                      # Kubernetes manifests
│   ├── 00-namespace.yaml
│   ├── 01-rbac.yaml
│   ├── 02-service.yaml
│   ├── 03-deployment.yaml
│   ├── 04-certificate.yaml
│   └── 05-webhook-config.yaml
└── scripts/
    └── generate-certs.sh        # Certificate generation
```

## Production Considerations

- Use cert-manager with a proper CA issuer
- Configure high availability with multiple replicas
- Set `failurePolicy: Fail` for strict enforcement
- Implement monitoring and alerting
- Review and adjust resource limits
- Apply network policies

## Contributing

This is a reference implementation. Feel free to customize the sidecar injection logic in `webhook.go` for your specific use case.

## License

This project is provided as-is for educational and reference purposes.
