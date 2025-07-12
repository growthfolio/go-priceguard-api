#!/bin/bash

# PriceGuard Disaster Recovery Script
# Usage: ./disaster-recovery.sh [environment] [backup_timestamp]

set -euo pipefail

ENVIRONMENT=${1:-production}
BACKUP_TIMESTAMP=${2:-latest}

echo "🚨 PriceGuard Disaster Recovery Procedure"
echo "Environment: $ENVIRONMENT"
echo "Backup: $BACKUP_TIMESTAMP"
echo "Started at: $(date)"
echo "=========================================="

# Configuration
case $ENVIRONMENT in
    "production")
        NAMESPACE="priceguard"
        BACKUP_BUCKET="priceguard-backups-prod"
        ;;
    "staging")
        NAMESPACE="priceguard-staging"
        BACKUP_BUCKET="priceguard-backups-staging"
        ;;
    *)
        echo "Unknown environment: $ENVIRONMENT"
        exit 1
        ;;
esac

# Functions
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

send_notification() {
    if [ ! -z "${SLACK_WEBHOOK_URL:-}" ]; then
        curl -s -X POST -H 'Content-type: application/json' \
            --data "{\"text\":\"🚨 $1\"}" \
            $SLACK_WEBHOOK_URL
    fi
}

check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check kubectl access
    if ! kubectl get ns $NAMESPACE >/dev/null 2>&1; then
        log "❌ Cannot access Kubernetes namespace: $NAMESPACE"
        exit 1
    fi
    
    # Check AWS CLI (for S3 backups)
    if ! command -v aws &> /dev/null; then
        log "⚠️  AWS CLI not found - S3 backup restore not available"
    fi
    
    log "✅ Prerequisites checked"
}

download_backup() {
    log "Downloading backup from S3..."
    
    BACKUP_DIR="/tmp/disaster-recovery"
    mkdir -p $BACKUP_DIR
    
    if [ "$BACKUP_TIMESTAMP" = "latest" ]; then
        # Find latest backup
        BACKUP_FILE=$(aws s3 ls s3://$BACKUP_BUCKET/$ENVIRONMENT/ | sort | tail -n 1 | awk '{print $4}')
    else
        BACKUP_FILE="priceguard_${ENVIRONMENT}_${BACKUP_TIMESTAMP}.sql.gz"
    fi
    
    if [ -z "$BACKUP_FILE" ]; then
        log "❌ No backup found"
        exit 1
    fi
    
    log "Downloading: $BACKUP_FILE"
    aws s3 cp "s3://$BACKUP_BUCKET/$ENVIRONMENT/$BACKUP_FILE" "$BACKUP_DIR/$BACKUP_FILE"
    
    echo "$BACKUP_DIR/$BACKUP_FILE"
}

deploy_infrastructure() {
    log "Deploying infrastructure..."
    
    # Apply Kubernetes manifests
    kubectl apply -f k8s/namespace.yaml
    kubectl apply -f k8s/configmap.yaml
    kubectl apply -f k8s/secrets.yaml
    kubectl apply -f k8s/database.yaml
    kubectl apply -f k8s/services.yaml
    kubectl apply -f k8s/pdb.yaml
    kubectl apply -f k8s/network-policy.yaml
    
    # Wait for database to be ready
    log "Waiting for database to be ready..."
    kubectl wait --for=condition=ready pod -l app=postgres -n $NAMESPACE --timeout=300s
    
    # Wait for Redis to be ready
    log "Waiting for Redis to be ready..."
    kubectl wait --for=condition=ready pod -l app=redis -n $NAMESPACE --timeout=300s
    
    log "✅ Infrastructure deployed"
}

restore_database() {
    local backup_file=$1
    log "Restoring database from backup..."
    
    # Run restore script
    ./scripts/restore-database.sh "$backup_file" "$ENVIRONMENT"
    
    log "✅ Database restored"
}

deploy_application() {
    log "Deploying application..."
    
    kubectl apply -f k8s/deployment.yaml
    kubectl apply -f k8s/hpa.yaml
    kubectl apply -f k8s/ingress.yaml
    
    # Wait for application to be ready
    log "Waiting for application to be ready..."
    kubectl wait --for=condition=ready pod -l app=priceguard-api -n $NAMESPACE --timeout=300s
    
    log "✅ Application deployed"
}

verify_recovery() {
    log "Verifying disaster recovery..."
    
    # Get service endpoint
    API_ENDPOINT=$(kubectl get svc priceguard-api-service -n $NAMESPACE -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
    if [ -z "$API_ENDPOINT" ]; then
        API_ENDPOINT="localhost:8080"
        kubectl port-forward svc/priceguard-api-service -n $NAMESPACE 8080:80 &
        PORT_FORWARD_PID=$!
        sleep 5
    fi
    
    # Health check
    if curl -s "http://$API_ENDPOINT/health" | grep -q "healthy"; then
        log "✅ API health check passed"
    else
        log "❌ API health check failed"
        return 1
    fi
    
    # Database connectivity check
    if curl -s "http://$API_ENDPOINT/api/health/db" | grep -q "healthy"; then
        log "✅ Database connectivity check passed"
    else
        log "❌ Database connectivity check failed"
        return 1
    fi
    
    # Clean up port forward if created
    if [ ! -z "${PORT_FORWARD_PID:-}" ]; then
        kill $PORT_FORWARD_PID 2>/dev/null || true
    fi
    
    log "✅ Recovery verification completed"
}

# Main disaster recovery procedure
main() {
    send_notification "Disaster recovery started for $ENVIRONMENT environment"
    
    check_prerequisites
    
    # Download backup if using S3
    if [ "$BACKUP_TIMESTAMP" != "local" ] && command -v aws &> /dev/null; then
        BACKUP_FILE=$(download_backup)
    else
        BACKUP_FILE="/backups/priceguard/priceguard_${ENVIRONMENT}_${BACKUP_TIMESTAMP}.sql.gz"
        if [ ! -f "$BACKUP_FILE" ]; then
            log "❌ Local backup file not found: $BACKUP_FILE"
            exit 1
        fi
    fi
    
    deploy_infrastructure
    restore_database "$BACKUP_FILE"
    deploy_application
    
    if verify_recovery; then
        log "🎉 Disaster recovery completed successfully!"
        send_notification "✅ Disaster recovery completed successfully for $ENVIRONMENT environment"
    else
        log "❌ Disaster recovery verification failed"
        send_notification "❌ Disaster recovery failed for $ENVIRONMENT environment"
        exit 1
    fi
    
    log "Recovery summary:"
    log "- Environment: $ENVIRONMENT"
    log "- Backup used: $(basename $BACKUP_FILE)"
    log "- Completed at: $(date)"
}

# Run main function
main "$@"
