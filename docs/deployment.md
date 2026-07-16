# Deployment Guide

## 1. Prerequisites

- Kubernetes cluster (v1.26+) with KubeVirt installed
- `kubectl` configured to access the cluster
- Container build tool (`docker` or `podman`)
- `kustomize` (if deploying via kustomize)
- prometheus-operator CRDs installed (for PrometheusRule and ServiceMonitor)

## 2. Build and push the image

```bash
IMG=<your-registry>/virt-observability-controller:latest

make docker-build IMG=$IMG
make docker-push IMG=$IMG
```

To use podman instead of docker:

```bash
make docker-build docker-push IMG=$IMG CONTAINER_TOOL=podman
```

## 3. Deploy

### Option A: Kustomize

```bash
make deploy IMG=$IMG
```

This runs `kustomize build config/default` and applies it. It creates all RBAC resources, the ServiceAccount, and the Deployment with a `virt-observability-controller-` name prefix in the `virt-observability-controller-system` namespace.

### Option B: Static manifest

Generate a single install manifest:

```bash
make build-installer IMG=$IMG
```

Then apply it to any cluster:

```bash
kubectl apply -f dist/install.yaml
```

## 4. Configuration

The controller accepts the following flags (set via the Deployment's `args`):

| Flag | Default | Description |
|------|---------|-------------|
| `--metrics-bind-address` | `:8080` | Address the metrics endpoint binds to (e.g. `:8443` for HTTPS, `:8080` for HTTP) |
| `--metrics-secure` | `true` | Serve metrics over HTTPS with authn/authz |
| `--health-probe-bind-address` | `:8081` | Address the health probe endpoint binds to |
| `--leader-elect` | `false` | Enable leader election for HA deployments |
| `--enable-http2` | `false` | Enable HTTP/2 for metrics and webhook servers |
| `--tls-security-profile` | `""` | TLS security profile for the metrics server: `Old`, `Intermediate`, `Modern`, or `Custom` |
| `--tls-min-version` | `""` | Minimum TLS version (e.g., `VersionTLS12`). Only valid with `--tls-security-profile=Custom` |
| `--tls-ciphers` | `""` | Comma-separated OpenSSL cipher names. Only valid with `--tls-security-profile=Custom` |
| `--metrics-allowlist` | `""` | Comma-separated list of metrics to expose (empty = all) |
| `--alerts-allowlist` | `""` | Comma-separated list of alerts to include in the PrometheusRule (empty = all) |
| `--recording-rules-allowlist` | `""` | Comma-separated list of recording rules to include (empty = all) |
| `--enable-vmstats` | `false` | Enable the vmstats poller for domain-level VM metrics |
| `--vmstats-port` | `8187` | Port for the virt-handler vmstats endpoint |
| `--vmstats-cert-path` | `""` | Directory containing the vmstats client certificate |
| `--metrics-cert-path` | `""` | Directory containing the metrics server TLS certificate |
| `--webhook-cert-path` | `""` | Directory containing the webhook certificate |

The default kustomize overlay sets `--metrics-bind-address=:8443` via `config/default/manager_metrics_patch.yaml`.

### VMStats feature gate

When `--enable-vmstats` is set, KubeVirt must have the `VMStatsCollector` feature gate enabled:

```bash
kubectl patch kubevirt kubevirt -n kubevirt --type merge -p \
  '{"spec":{"configuration":{"developerConfiguration":{"featureGates":["VMStatsCollector"]}}}}'
```

The controller also needs access to the virt-handler monitoring client certificate. Mount the `kubevirt-virt-handler-monitoring-client-certs` secret and pass its path via `--vmstats-cert-path` (the default kustomize deployment handles this automatically).

## 5. Self-managed resources

The controller automatically reconciles the following resources in its own namespace (discovered via the `POD_NAMESPACE` downward API env var):

- **Service** (`virt-observability-controller-metrics`) — exposes the metrics port, selector matches the controller pods
- **Secret** (`virt-observability-controller-metrics-token`) — long-lived ServiceAccount token for Prometheus bearer auth
- **ServiceMonitor** (`virt-observability-controller`) — configures Prometheus to scrape the metrics endpoint with the correct scheme (HTTP/HTTPS) and bearer token

These resources are created on startup and continuously reconciled. If deleted or modified, the controller restores them to the desired state.

## 6. Verify the deployment

Check the controller pods are running:

```bash
kubectl get pods -n virt-observability-controller-system
```

Check the self-managed Service was created:

```bash
kubectl get svc -n virt-observability-controller-system virt-observability-controller-metrics
```

Check the controller logs:

```bash
kubectl logs -n virt-observability-controller-system deploy/virt-observability-controller-controller-manager
```

Port-forward and query metrics:

```bash
kubectl port-forward -n virt-observability-controller-system svc/virt-observability-controller-metrics 8443:8443
```

```bash
TOKEN=$(kubectl get secret virt-observability-controller-metrics-token \
  -n virt-observability-controller-system \
  -o jsonpath='{.data.token}' | base64 -d)

curl -sk -H "Authorization: Bearer $TOKEN" https://localhost:8443/metrics | grep ^kubevirt_ | sort
```

## 7. Uninstall

If deployed with kustomize:

```bash
make undeploy
```

If deployed with the static manifest:

```bash
kubectl delete -f dist/install.yaml
```
