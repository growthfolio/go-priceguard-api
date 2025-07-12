#!/bin/bash

# PriceGuard Database Backup Script
# Usage: ./backup-database.sh [environment]

set -euo pipefail

ENVIRONMENT=${1:-production}
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups/priceguard"
S3_BUCKET="priceguard-backups"

# Configuration based on environment
case $ENVIRONMENT in
    "production")
        DB_HOST="postgres-service.priceguard.svc.cluster.local"
        DB_NAME="priceguard"
        DB_USER="postgres"
        NAMESPACE="priceguard"
        ;;
    "staging")
        DB_HOST="postgres-service.priceguard-staging.svc.cluster.local"
        DB_NAME="priceguard_staging"
        DB_USER="postgres"
        NAMESPACE="priceguard-staging"
        ;;
    *)
        echo "Unknown environment: $ENVIRONMENT"
        exit 1
        ;;
esac

# Create backup directory
mkdir -p $BACKUP_DIR

# Get database password from Kubernetes secret
DB_PASSWORD=$(kubectl get secret priceguard-secrets -n $NAMESPACE -o jsonpath='{.data.db-password}' | base64 -d)

# Backup filename
BACKUP_FILE="$BACKUP_DIR/priceguard_${ENVIRONMENT}_${TIMESTAMP}.sql"
BACKUP_FILE_COMPRESSED="$BACKUP_FILE.gz"

echo "Starting database backup for $ENVIRONMENT environment..."
echo "Backup file: $BACKUP_FILE_COMPRESSED"

# Create database dump
export PGPASSWORD=$DB_PASSWORD
pg_dump -h $DB_HOST -U $DB_USER -d $DB_NAME \
    --verbose \
    --no-owner \
    --no-privileges \
    --format=custom \
    --file=$BACKUP_FILE

# Compress backup
gzip $BACKUP_FILE

# Upload to S3 if configured
if command -v aws &> /dev/null && [ ! -z "${AWS_ACCESS_KEY_ID:-}" ]; then
    echo "Uploading backup to S3..."
    aws s3 cp $BACKUP_FILE_COMPRESSED "s3://$S3_BUCKET/$ENVIRONMENT/"
    
    # Set lifecycle policy for old backups
    aws s3api put-object-tagging \
        --bucket $S3_BUCKET \
        --key "$ENVIRONMENT/$(basename $BACKUP_FILE_COMPRESSED)" \
        --tagging 'TagSet=[{Key=backup-type,Value=database},{Key=environment,Value='$ENVIRONMENT'}]'
fi

# Keep only last 7 local backups
find $BACKUP_DIR -name "priceguard_${ENVIRONMENT}_*.sql.gz" -type f -mtime +7 -delete

# Verify backup integrity
echo "Verifying backup integrity..."
if gzip -t $BACKUP_FILE_COMPRESSED; then
    echo "✅ Backup created successfully: $BACKUP_FILE_COMPRESSED"
    
    # Send notification (if configured)
    if [ ! -z "${SLACK_WEBHOOK_URL:-}" ]; then
        curl -X POST -H 'Content-type: application/json' \
            --data "{\"text\":\"✅ Database backup completed for $ENVIRONMENT environment\"}" \
            $SLACK_WEBHOOK_URL
    fi
else
    echo "❌ Backup verification failed!"
    exit 1
fi

# Display backup size and location
echo "Backup size: $(du -h $BACKUP_FILE_COMPRESSED | cut -f1)"
echo "Backup location: $BACKUP_FILE_COMPRESSED"

unset PGPASSWORD
