# Deployment Guide

This guide walks through deploying the admission controller to a Kubernetes cluster.

## Prerequisites

- Kubernetes cluster (v1.19+)
- `kubectl` configured to access your cluster
- Docker for building the container image
- Access to a container registry (Docker Hub, GCR, ECR, etc.)

## Deployment Steps

### 1. Build and Push Container Image

Build the Docker image:

```bash
docker build -t your-registry/admission-controller:v1.0.0 .
```

Push to your container registry:

```bash
docker push your-registry/admission-controller:v1.0.0
```

**Update the image reference in `deploy/03-deployment.yaml`:**

```yaml
spec:
  containers:
    - name: admission-controller
      image: your-registry/admission-controller:v1.0.0  # Update this line
```

### 2. Choose Certificate Management Method

You have two options for TLS certificates:

#### Option A: Using cert-manager (Recommended)

Install cert-manager if not already installed:

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

Wait for cert-manager to be ready:

```bash
kubectl wait --for=condition=Available --timeout=300s \
  deployment/cert-manager-webhook -n cert-manager
```

The certificates will be automatically created when you deploy the manifests.

#### Option B: Generate Self-Signed Certificates Manually

Run the certificate generation script:

```bash
./scripts/generate-certs.sh
```

This will:
- Generate CA and server certificates in `./certs/`
- Optionally create the Kubernetes secret
- Display the CA bundle for webhook configuration

If you don't use cert-manager, you'll need to manually add the CA bundle to `deploy/05-webhook-config.yaml`:

```yaml
clientConfig:
  caBundle: <base64-encoded-ca-cert>  # Add the output from generate-certs.sh
```

### 3. Deploy to Kubernetes

Deploy all resources in order:

```bash
# Create namespace
kubectl apply -f deploy/00-namespace.yaml

# Create RBAC resources
kubectl apply -f deploy/01-rbac.yaml

# Create service
kubectl apply -f deploy/02-service.yaml

# If using cert-manager, create certificate resources
kubectl apply -f deploy/04-certificate.yaml

# Wait for certificate to be ready (if using cert-manager)
kubectl wait --for=condition=Ready certificate/admission-controller-tls \
  -n admission-controller --timeout=60s

# Deploy the admission controller
kubectl apply -f deploy/03-deployment.yaml

# Wait for deployment to be ready
kubectl wait --for=condition=Available deployment/admission-controller \
  -n admission-controller --timeout=120s

# Deploy webhook configuration
kubectl apply -f deploy/05-webhook-config.yaml
```

Or deploy all at once:

```bash
kubectl apply -f deploy/
```

### 4. Verify Deployment

Check that the admission controller is running:

```bash
kubectl get pods -n admission-controller
```

Expected output:
```
NAME                                   READY   STATUS    RESTARTS   AGE
admission-controller-xxxxxxxxx-xxxxx   1/1     Running   0          30s
```

Check logs:

```bash
kubectl logs -n admission-controller deployment/admission-controller
```

Test the health endpoint:

```bash
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl -k https://admission-controller.admission-controller.svc:443/health
```

### 5. Test the Webhook

Create a test pod with the `app=frontend` label scheduled to a specific node:

```bash
# First, get a node name
NODE_NAME=$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}')

# Create a test pod
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: test-frontend
  labels:
    app: frontend
spec:
  nodeName: ${NODE_NAME}
  containers:
  - name: nginx
    image: nginx:alpine
    ports:
    - containerPort: 80
EOF
```

Check if the sidecar was injected:

```bash
kubectl get pod test-frontend -o jsonpath='{.spec.containers[*].name}'
```

Expected output (if sidecar was injected):
```
nginx frontend-sidecar
```

Check the annotation:

```bash
kubectl get pod test-frontend -o jsonpath='{.metadata.annotations.sidecar-injector}'
```

Clean up:

```bash
kubectl delete pod test-frontend
```

## Configuration

### Webhook Behavior

The webhook will inject a sidecar when:
1. Pod has `app=frontend` label
2. Pod is scheduled to a specific node (`spec.nodeName` is set)
3. No other frontend pod with a sidecar is currently running on that node

### Failure Policy

The webhook is configured with `failurePolicy: Ignore` by default. This means if the webhook is unavailable, pod creation will proceed normally.

For production, consider changing to `failurePolicy: Fail` in `deploy/05-webhook-config.yaml`:

```yaml
failurePolicy: Fail  # Strict enforcement
```

### Namespace Selector

The webhook excludes `kube-system` and `admission-controller` namespaces by default. Modify the namespace selector in `deploy/05-webhook-config.yaml` to customize this behavior.

## Troubleshooting

### Pod creation is slow or timing out

Check webhook logs:
```bash
kubectl logs -n admission-controller deployment/admission-controller -f
```

### Certificate issues

If using cert-manager, check certificate status:
```bash
kubectl describe certificate admission-controller-tls -n admission-controller
```

View the certificate secret:
```bash
kubectl get secret admission-controller-tls -n admission-controller -o yaml
```

### Webhook not being called

Verify webhook configuration:
```bash
kubectl get mutatingwebhookconfigurations admission-controller-webhook -o yaml
```

Check that the CA bundle is present:
```bash
kubectl get mutatingwebhookconfigurations admission-controller-webhook \
  -o jsonpath='{.webhooks[0].clientConfig.caBundle}' | base64 -d
```

### RBAC permission issues

Check service account permissions:
```bash
kubectl auth can-i list pods --as=system:serviceaccount:admission-controller:admission-controller
```

## Updating the Deployment

To update the admission controller:

1. Build and push new image with a new tag
2. Update image in `deploy/03-deployment.yaml`
3. Apply the changes:

```bash
kubectl apply -f deploy/03-deployment.yaml
```

Or perform a rolling update:

```bash
kubectl set image deployment/admission-controller \
  admission-controller=your-registry/admission-controller:v1.1.0 \
  -n admission-controller
```

## Uninstalling

To remove the admission controller:

```bash
# Remove webhook configuration first (important!)
kubectl delete -f deploy/05-webhook-config.yaml

# Remove other resources
kubectl delete -f deploy/

# If you created certificates manually
kubectl delete secret admission-controller-tls -n admission-controller
```

## Production Considerations

1. **TLS Certificates**: Use cert-manager with a proper CA issuer (not self-signed)
2. **High Availability**: Increase replicas in `deploy/03-deployment.yaml`
3. **Resource Limits**: Adjust CPU/memory based on your cluster size
4. **Failure Policy**: Consider using `failurePolicy: Fail` for strict enforcement
5. **Monitoring**: Add Prometheus metrics and alerting
6. **Network Policies**: Restrict network access to the webhook service
7. **Pod Security**: The deployment includes security contexts and follows best practices

## Next Steps

- Customize the sidecar container image and configuration in `webhook.go:175-217`
- Add health checks and metrics endpoints
- Implement admission logic for additional use cases
- Configure cert-manager with a production CA issuer
