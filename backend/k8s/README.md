# Kubernetes Deployment Guide

This directory contains Kubernetes manifests for deploying the Database Backup Utility.

## Directory Structure

```
k8s/
├── README.md                    # This file
├── namespace.yaml               # Namespace definition
├── configmap.yaml               # Application configuration
├── secret.yaml                  # Secrets (credentials, tokens)
├── deployment.yaml              # Main application deployment
├── service.yaml                 # Services (ClusterIP, LoadBalancer)
├── serviceaccount.yaml          # RBAC (ServiceAccount, Role, RoleBinding)
├── pvc.yaml                     # Persistent storage
├── hpa.yaml                     # Horizontal Pod Autoscaler
├── pdb.yaml                     # Pod Disruption Budget
├── ingress.yaml                 # Ingress configuration
├── networkpolicy.yaml           # Network policies
├── cronjob.yaml                 # CronJobs for cleanup and verification
└── istio/                       # Istio service mesh integration
    ├── virtualservice.yaml      # Traffic routing
    ├── gateway.yaml             # Ingress gateway
    ├── destinationrule.yaml     # Load balancing, circuit breaking
    ├── peerauthentication.yaml  # mTLS configuration
    ├── authorizationpolicy.yaml # Access control
    ├── serviceentry.yaml        # External service access
    └── telemetry.yaml           # Observability configuration
```

## Prerequisites

- Kubernetes cluster 1.24+
- kubectl configured
- Storage provisioner (for PVC)
- Ingress controller (nginx, ALB, etc.)
- (Optional) Istio 1.16+ for service mesh

## Quick Start

### 1. Create namespace

```bash
kubectl apply -f namespace.yaml
```

### 2. Update secrets

Edit `secret.yaml` with your actual credentials:

```bash
# Generate secure JWT secret
openssl rand -base64 32

# Edit secret.yaml
kubectl apply -f secret.yaml
```

### 3. Update configuration

Edit `configmap.yaml` with your settings:

```bash
kubectl apply -f configmap.yaml
```

### 4. Deploy storage

```bash
kubectl apply -f pvc.yaml
```

### 5. Deploy RBAC

```bash
kubectl apply -f serviceaccount.yaml
```

### 6. Deploy application

```bash
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

### 7. Deploy autoscaling

```bash
kubectl apply -f hpa.yaml
kubectl apply -f pdb.yaml
```

### 8. Deploy ingress

```bash
kubectl apply -f ingress.yaml
```

### 9. (Optional) Deploy network policies

```bash
kubectl apply -f networkpolicy.yaml
```

### 10. (Optional) Deploy CronJobs

```bash
kubectl apply -f cronjob.yaml
```

## Deploy All

```bash
# Deploy all manifests at once
kubectl apply -f namespace.yaml
kubectl apply -f secret.yaml
kubectl apply -f configmap.yaml
kubectl apply -f pvc.yaml
kubectl apply -f serviceaccount.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f hpa.yaml
kubectl apply -f pdb.yaml
kubectl apply -f ingress.yaml
kubectl apply -f networkpolicy.yaml
kubectl apply -f cronjob.yaml
```

Or use kustomize:

```bash
kubectl apply -k .
```

## Istio Service Mesh

If using Istio, deploy service mesh resources:

```bash
# Label namespace for Istio injection
kubectl label namespace db-backup istio-injection=enabled

# Deploy Istio resources
kubectl apply -f istio/
```

## Verification

### Check deployment status

```bash
kubectl -n db-backup get all
```

### Check pod logs

```bash
kubectl -n db-backup logs -l app.kubernetes.io/name=db-backup
```

### Check pod health

```bash
kubectl -n db-backup describe pod -l app.kubernetes.io/name=db-backup
```

### Test health endpoint

```bash
kubectl -n db-backup port-forward svc/db-backup 8080:8080
curl http://localhost:8080/health
```

### Check HPA status

```bash
kubectl -n db-backup get hpa
kubectl -n db-backup describe hpa db-backup
```

### Check ingress

```bash
kubectl -n db-backup get ingress
kubectl -n db-backup describe ingress db-backup
```

## Updating

### Update configuration

```bash
kubectl edit configmap -n db-backup db-backup-config
```

### Update secrets

```bash
kubectl edit secret -n db-backup db-backup-secrets
```

### Rolling update

```bash
# Update image
kubectl set image deployment/db-backup \
  db-backup=db-backup:1.1.0 \
  --namespace=db-backup

# Check rollout status
kubectl rollout status deployment/db-backup -n db-backup

# Rollback if needed
kubectl rollout undo deployment/db-backup -n db-backup
```

## Scaling

### Manual scaling

```bash
kubectl scale deployment db-backup --replicas=5 -n db-backup
```

### Check autoscaling

```bash
kubectl -n db-backup get hpa
```

## Troubleshooting

### Pods not starting

```bash
# Check events
kubectl -n db-backup get events --sort-by='.lastTimestamp'

# Describe pod
kubectl -n db-backup describe pod -l app.kubernetes.io/name=db-backup

# Check logs
kubectl -n db-backup logs -l app.kubernetes.io/name=db-backup --tail=100
```

### Connection issues

```bash
# Test from within cluster
kubectl -n db-backup run -it --rm debug \
  --image=nicolaka/netshoot \
  --restart=Never \
  -- curl http://db-backup:8080/health
```

### Storage issues

```bash
# Check PVC
kubectl -n db-backup get pvc
kubectl -n db-backup describe pvc db-backup-storage

# Check PV
kubectl get pv
```

### Network policy issues

```bash
# Temporarily disable network policies
kubectl -n db-backup delete networkpolicy --all

# Re-apply after debugging
kubectl apply -f networkpolicy.yaml
```

## Monitoring

### Metrics

```bash
# Access Prometheus metrics
kubectl -n db-backup port-forward svc/db-backup 9090:9090
curl http://localhost:9090/metrics
```

### Istio Metrics

```bash
# Access Kiali dashboard
istioctl dashboard kiali

# Access Jaeger for tracing
istioctl dashboard jaeger

# Access Grafana
istioctl dashboard grafana
```

## Cleanup

### Delete all resources

```bash
kubectl delete namespace db-backup
```

### Delete specific resources

```bash
kubectl delete -f deployment.yaml -n db-backup
kubectl delete -f service.yaml -n db-backup
```

## Production Checklist

- [ ] Update all secrets with secure values
- [ ] Configure TLS certificates
- [ ] Set up persistent storage with backups
- [ ] Configure resource limits appropriately
- [ ] Enable network policies
- [ ] Configure monitoring and alerting
- [ ] Set up log aggregation
- [ ] Test disaster recovery procedures
- [ ] Configure backup verification CronJob
- [ ] Review and adjust HPA settings
- [ ] Configure PDB for high availability
- [ ] Set up external DNS
- [ ] Configure rate limiting
- [ ] Enable audit logging

## Security Best Practices

1. **Never commit secrets to version control**
   - Use sealed-secrets, external-secrets, or Vault
   - Rotate credentials regularly

2. **Enable network policies**
   - Restrict ingress and egress traffic
   - Follow principle of least privilege

3. **Use RBAC**
   - Limit service account permissions
   - Regular audit of permissions

4. **Enable Pod Security Standards**
   - Use restricted pod security policy
   - Run as non-root user

5. **Keep images updated**
   - Regular security scanning
   - Automated updates with testing

## Support

- Documentation: https://backup.example.com/docs
- Kubernetes Guide: https://backup.example.com/docs/kubernetes
- Issues: https://github.com/your-org/db-backup/issues
