# DB Backup Helm Chart

Enterprise-grade database backup and restoration utility Helm chart for Kubernetes.

## Prerequisites

- Kubernetes 1.24+
- Helm 3.8+
- PV provisioner support in the underlying infrastructure (for persistence)
- (Optional) Istio 1.16+ for service mesh integration

## Installing the Chart

### Add the Helm repository

```bash
helm repo add db-backup https://charts.backup.example.com
helm repo update
```

### Install with default values

```bash
helm install my-backup db-backup/db-backup \
  --namespace db-backup \
  --create-namespace
```

### Install with custom values

```bash
helm install my-backup db-backup/db-backup \
  --namespace db-backup \
  --create-namespace \
  --values custom-values.yaml
```

### Install with inline values

```bash
helm install my-backup db-backup/db-backup \
  --namespace db-backup \
  --create-namespace \
  --set image.tag=1.0.0 \
  --set ingress.hosts[0].host=api.backup.example.com \
  --set secrets.jwtSecret="my-secure-jwt-secret"
```

## Uninstalling the Chart

```bash
helm uninstall my-backup --namespace db-backup
```

## Configuration

The following table lists the configurable parameters and their default values.

### Global Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.imageRegistry` | Global Docker image registry | `""` |
| `global.imagePullSecrets` | Global Docker registry secret names | `[]` |

### Common Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `2` |
| `image.repository` | Image repository | `db-backup` |
| `image.tag` | Image tag | `1.0.0` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `nameOverride` | Override chart name | `""` |
| `fullnameOverride` | Override fullname | `""` |

### Service Account

| Parameter | Description | Default |
|-----------|-------------|---------|
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.annotations` | Service account annotations | `{}` |
| `serviceAccount.name` | Service account name | `""` |

### Service Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `service.type` | Service type | `ClusterIP` |
| `service.port` | Service port | `8080` |
| `service.metricsPort` | Metrics port | `9090` |

### Ingress Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ingress.enabled` | Enable ingress | `true` |
| `ingress.className` | Ingress class name | `nginx` |
| `ingress.hosts` | Ingress hosts | `[api.backup.example.com]` |
| `ingress.tls` | Ingress TLS configuration | `[]` |

### Resources

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resources.limits.cpu` | CPU limit | `2000m` |
| `resources.limits.memory` | Memory limit | `2Gi` |
| `resources.requests.cpu` | CPU request | `500m` |
| `resources.requests.memory` | Memory request | `512Mi` |

### Autoscaling

| Parameter | Description | Default |
|-----------|-------------|---------|
| `autoscaling.enabled` | Enable HPA | `true` |
| `autoscaling.minReplicas` | Minimum replicas | `2` |
| `autoscaling.maxReplicas` | Maximum replicas | `10` |
| `autoscaling.targetCPUUtilizationPercentage` | Target CPU % | `70` |

### Persistence

| Parameter | Description | Default |
|-----------|-------------|---------|
| `persistence.enabled` | Enable persistence | `true` |
| `persistence.storageClass` | Storage class | `efs-sc` |
| `persistence.size` | Storage size | `100Gi` |

### Istio Integration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `istio.enabled` | Enable Istio integration | `false` |
| `istio.virtualService.enabled` | Enable VirtualService | `true` |
| `istio.gateway.enabled` | Enable Gateway | `true` |
| `istio.peerAuthentication.enabled` | Enable mTLS | `true` |

## Examples

### Example 1: Basic Installation

```yaml
# values-basic.yaml
replicaCount: 2

secrets:
  jwtSecret: "my-secure-jwt-secret-min-32-characters"
  awsAccessKeyId: "AKIAIOSFODNN7EXAMPLE"
  awsSecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

ingress:
  hosts:
    - host: api.backup.example.com
      paths:
        - path: /
          pathType: Prefix
```

```bash
helm install my-backup db-backup/db-backup \
  --namespace db-backup \
  --create-namespace \
  --values values-basic.yaml
```

### Example 2: High Availability Setup

```yaml
# values-ha.yaml
replicaCount: 3

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 20

podDisruptionBudget:
  enabled: true
  minAvailable: 2

resources:
  requests:
    cpu: 1000m
    memory: 1Gi
  limits:
    cpu: 4000m
    memory: 4Gi

persistence:
  enabled: true
  size: 500Gi

affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values:
                - db-backup
        topologyKey: topology.kubernetes.io/zone
```

### Example 3: Istio Service Mesh

```yaml
# values-istio.yaml
istio:
  enabled: true

  virtualService:
    enabled: true
    hosts:
      - api.backup.example.com

  gateway:
    enabled: true

  peerAuthentication:
    enabled: true
    mtlsMode: STRICT

  destinationRule:
    enabled: true
    trafficPolicy:
      loadBalancer:
        simple: LEAST_REQUEST
      outlierDetection:
        consecutive5xxErrors: 5

  authorizationPolicy:
    enabled: true

  telemetry:
    enabled: true
    tracingSamplingPercentage: 100.0
```

### Example 4: AWS EKS with IRSA

```yaml
# values-eks.yaml
serviceAccount:
  create: true
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/db-backup-role

config:
  storage:
    type: s3
    region: us-east-1
    bucket: db-backups-production

# Don't set AWS credentials - use IRSA
secrets:
  awsAccessKeyId: ""
  awsSecretAccessKey: ""
```

## Upgrading

### To 1.1.0

- New parameter `config.deltaBackups.enabled` added (default: `true`)
- New CronJob for backup verification added

```bash
helm upgrade my-backup db-backup/db-backup \
  --namespace db-backup \
  --reuse-values
```

## Troubleshooting

### Pods not starting

Check pod events:
```bash
kubectl -n db-backup describe pod -l app.kubernetes.io/name=db-backup
```

Check logs:
```bash
kubectl -n db-backup logs -l app.kubernetes.io/name=db-backup
```

### Ingress not working

Verify ingress controller is installed:
```bash
kubectl get ingressclass
```

Check ingress status:
```bash
kubectl -n db-backup get ingress
kubectl -n db-backup describe ingress db-backup
```

### HPA not scaling

Verify metrics server is installed:
```bash
kubectl top nodes
kubectl top pods -n db-backup
```

Check HPA status:
```bash
kubectl -n db-backup get hpa
kubectl -n db-backup describe hpa db-backup
```

### Persistence issues

Check PVC status:
```bash
kubectl -n db-backup get pvc
kubectl -n db-backup describe pvc db-backup-storage
```

Check storage class:
```bash
kubectl get storageclass
```

## Support

- Documentation: https://backup.example.com/docs
- Issues: https://github.com/your-org/db-backup/issues
- Slack: https://slack.example.com/db-backup

## License

MIT License - see LICENSE file for details
