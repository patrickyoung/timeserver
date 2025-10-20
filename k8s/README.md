# Kubernetes Deployment Guide

This directory contains Kubernetes manifests for deploying the Time Service with database persistence.

## Overview

The Time Service uses a **StatefulSet** instead of a Deployment to ensure stable storage for the SQLite database. Each pod gets its own PersistentVolumeClaim (PVC) that persists across pod restarts and rescheduling.

## Architecture

```
┌─────────────────────────────────────┐
│         StatefulSet                 │
│  (timeservice-0)                    │
│  ┌───────────────────────────────┐  │
│  │  Container: timeservice       │  │
│  │  Port: 8080                   │  │
│  │  Volume: /app/data            │  │
│  └───────────────┬───────────────┘  │
│                  │                   │
└──────────────────┼───────────────────┘
                   │
                   ▼
    ┌──────────────────────────────┐
    │  PersistentVolumeClaim       │
    │  data-timeservice-0          │
    │  Size: 1Gi                   │
    │  AccessMode: ReadWriteOnce   │
    └──────────────┬───────────────┘
                   │
                   ▼
    ┌──────────────────────────────┐
    │  PersistentVolume            │
    │  (Provisioned by storage     │
    │   class or statically bound) │
    └──────────────────────────────┘
```

## Prerequisites

- Kubernetes cluster (v1.19+)
- kubectl configured to access your cluster
- StorageClass configured for dynamic PVC provisioning (or manual PV creation)
- Container registry with the `timeservice:v1.0.0` image

## Quick Start

### 1. Deploy the Service

```bash
# Apply all manifests
kubectl apply -f k8s/deployment.yaml

# Check deployment status
kubectl get statefulset timeservice
kubectl get pods -l app=timeservice
kubectl get pvc -l app=timeservice
kubectl get svc timeservice
```

### 2. Verify the Deployment

```bash
# Check pod logs
kubectl logs -f timeservice-0

# Check database file
kubectl exec timeservice-0 -- ls -lh /app/data

# Port forward to access locally
kubectl port-forward timeservice-0 8080:8080

# Test the service
curl http://localhost:8080/health
curl http://localhost:8080/api/time
```

### 3. Create a Test Location

```bash
# Create a location
curl -X POST http://localhost:8080/api/locations \
  -H "Content-Type: application/json" \
  -d '{"name":"headquarters","timezone":"America/New_York","description":"Company HQ"}'

# List locations
curl http://localhost:8080/api/locations
```

## StatefulSet vs Deployment

### Why StatefulSet?

The Time Service uses **SQLite** as its database, which is file-based and requires:

1. **Stable Storage**: The database file must persist across pod restarts
2. **Single Writer**: SQLite uses file-level locking, so only one pod can write at a time
3. **Stable Network Identity**: StatefulSet provides predictable pod names (`timeservice-0`)

### Limitations

- **No Horizontal Scaling**: SQLite = single instance only (`replicas: 1`)
- **No High Availability**: Single point of failure
- **Limited Concurrency**: File-based locking

### Migration Path

For production workloads requiring:
- Multiple replicas
- High availability
- Better concurrency

Consider migrating to:
- **PostgreSQL** (recommended for production)
- **MySQL/MariaDB**
- **CockroachDB** (for distributed deployments)

## Storage Configuration

### Dynamic Provisioning (Recommended)

The StatefulSet uses `volumeClaimTemplates` for automatic PVC creation:

```yaml
volumeClaimTemplates:
- metadata:
    name: data
  spec:
    accessModes: [ "ReadWriteOnce" ]
    resources:
      requests:
        storage: 1Gi
```

The cluster's default StorageClass will provision PersistentVolumes automatically.

### Custom StorageClass

To use a specific storage class (e.g., fast SSD):

```yaml
volumeClaimTemplates:
- metadata:
    name: data
  spec:
    accessModes: [ "ReadWriteOnce" ]
    storageClassName: "fast-ssd"  # Add this line
    resources:
      requests:
        storage: 1Gi
```

### Static Provisioning

For manual PV creation:

1. Create a PersistentVolume:

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: timeservice-pv
spec:
  capacity:
    storage: 1Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  hostPath:
    path: /mnt/data/timeservice
```

2. The PVC will automatically bind to it

## Database Configuration

Database settings are configured via environment variables:

```yaml
env:
- name: DB_PATH
  value: "/app/data/timeservice.db"
- name: DB_MAX_OPEN_CONNS
  value: "25"
- name: DB_MAX_IDLE_CONNS
  value: "5"
- name: DB_CACHE_SIZE_KB
  value: "64000"
- name: DB_WAL_MODE
  value: "true"
```

### Tuning for Performance

For high-traffic workloads:

```yaml
- name: DB_MAX_OPEN_CONNS
  value: "50"
- name: DB_CACHE_SIZE_KB
  value: "128000"  # 128MB cache
```

For low-memory environments:

```yaml
- name: DB_MAX_OPEN_CONNS
  value: "10"
- name: DB_CACHE_SIZE_KB
  value: "32000"  # 32MB cache
```

## Backup and Restore

### Manual Backup

Use the provided backup script:

```bash
# Port-forward to access the pod
kubectl port-forward timeservice-0 8080:8080 &

# Copy database from pod
kubectl cp timeservice-0:/app/data/timeservice.db ./timeservice-backup.db

# Or use exec
kubectl exec timeservice-0 -- cat /app/data/timeservice.db > timeservice-backup.db
```

### Automated Backups with CronJob

Create a backup CronJob:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: timeservice-backup
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: alpine:3.21
            command:
            - /bin/sh
            - -c
            - |
              apk add --no-cache sqlite
              TIMESTAMP=$(date +%Y%m%d_%H%M%S)
              sqlite3 /data/timeservice.db "VACUUM INTO '/backup/timeservice_$TIMESTAMP.db'"
              # Delete backups older than 7 days
              find /backup -name "timeservice_*.db" -mtime +7 -delete
            volumeMounts:
            - name: data
              mountPath: /data
              readOnly: true
            - name: backup
              mountPath: /backup
          volumes:
          - name: data
            persistentVolumeClaim:
              claimName: data-timeservice-0
          - name: backup
            persistentVolumeClaim:
              claimName: backup-storage
          restartPolicy: OnFailure
```

### Restore from Backup

```bash
# Copy backup to pod
kubectl cp timeservice-backup.db timeservice-0:/app/data/timeservice.db

# Restart pod to reload
kubectl rollout restart statefulset timeservice
```

## Scaling Considerations

### Current Setup (SQLite)

- **Replicas**: 1 (cannot scale horizontally)
- **Availability**: Single pod (no HA)
- **Concurrency**: Limited by file locking

### To Scale Horizontally

You must migrate from SQLite to a client-server database:

1. **PostgreSQL** (Recommended):
   ```yaml
   replicas: 3  # Can now scale
   env:
   - name: DB_TYPE
     value: "postgres"
   - name: DB_HOST
     value: "postgres-service"
   ```

2. **Update Connection Pooling**:
   ```yaml
   - name: DB_MAX_OPEN_CONNS
     value: "100"  # Higher for networked DB
   ```

3. **Remove PVC** (database is external)

## Monitoring

### Prometheus Metrics

The service exposes metrics at `/metrics`:

- `timeservice_db_query_duration_seconds` - Query latency
- `timeservice_db_queries_total` - Query counts
- `timeservice_db_connections_open` - Open connections
- `timeservice_db_connections_idle` - Idle connections
- `timeservice_db_errors_total` - Database errors

### ServiceMonitor

If using Prometheus Operator:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: timeservice
spec:
  selector:
    matchLabels:
      app: timeservice
  endpoints:
  - port: http
    path: /metrics
```

### Grafana Dashboard

Key metrics to monitor:

- Query duration (p50, p95, p99)
- Connection pool utilization
- Database file size
- PVC usage

## Troubleshooting

### Pod Won't Start

```bash
# Check pod events
kubectl describe pod timeservice-0

# Check PVC status
kubectl get pvc data-timeservice-0
kubectl describe pvc data-timeservice-0

# Check logs
kubectl logs timeservice-0
```

### Database Locked Errors

SQLite uses file-level locking. If you see "database is locked" errors:

1. Ensure `replicas: 1` (only one pod)
2. Check for stale locks:
   ```bash
   kubectl exec timeservice-0 -- ls -la /app/data
   # Look for .db-shm and .db-wal files
   ```

### PVC Not Binding

```bash
# Check PV/PVC status
kubectl get pv,pvc

# Check StorageClass
kubectl get storageclass

# If no default StorageClass, create one or use static provisioning
```

### Out of Disk Space

```bash
# Check PVC size
kubectl exec timeservice-0 -- df -h /app/data

# Resize PVC (if StorageClass supports it)
kubectl patch pvc data-timeservice-0 -p '{"spec":{"resources":{"requests":{"storage":"5Gi"}}}}'
```

## Cleanup

### Delete Everything

```bash
# Delete StatefulSet and Service
kubectl delete -f k8s/deployment.yaml

# Check PVC (may need manual deletion)
kubectl get pvc -l app=timeservice

# Delete PVC (WARNING: Deletes all data!)
kubectl delete pvc data-timeservice-0
```

### Preserve Data

To delete the StatefulSet but keep data:

```bash
# Delete StatefulSet but not PVC
kubectl delete statefulset timeservice --cascade=orphan

# PVC remains - redeploy to reconnect
kubectl apply -f k8s/deployment.yaml
```

## Security Best Practices

1. **RBAC**: The service uses a dedicated ServiceAccount with `automountServiceAccountToken: false`
2. **Network Policies**: Consider adding NetworkPolicy for ingress/egress control
3. **Pod Security Standards**: The pod runs with:
   - Non-root user (UID 10001)
   - Read-only root filesystem
   - No capabilities
   - seccomp profile

4. **CORS**: Update `ALLOWED_ORIGINS` in the deployment
5. **Authentication**: Enable OIDC auth for production (see deployment.yaml comments)

## Support

For issues or questions:
- GitHub Issues: https://github.com/yourorg/timeservice/issues
- Docs: https://docs.example.com/timeservice
