#!/bin/bash

# PriceGuard Database Restore Script
# Usage: ./restore-database.sh [backup_file] [environment]

set -euo pipefail

BACKUP_FILE=${1:-}
ENVIRONMENT=${2:-production}

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file> [environment]"
    echo "Example: $0 /backups/priceguard_production_20240101_120000.sql.gz production"
    exit 1
fi

if [ ! -f "$BACKUP_FILE" ]; then
    echo "Backup file not found: $BACKUP_FILE"
    exit 1
fi

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

# Get database password from Kubernetes secret
DB_PASSWORD=$(kubectl get secret priceguard-secrets -n $NAMESPACE -o jsonpath='{.data.db-password}' | base64 -d)

echo "⚠️  WARNING: This will REPLACE the current database in $ENVIRONMENT environment!"
echo "Backup file: $BACKUP_FILE"
echo "Target database: $DB_NAME on $DB_HOST"
echo ""
read -p "Are you sure you want to continue? (yes/no): " confirmation

if [ "$confirmation" != "yes" ]; then
    echo "Restore cancelled."
    exit 0
fi

# Create temporary restore directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

echo "Extracting backup file..."
if [[ $BACKUP_FILE == *.gz ]]; then
    gunzip -c "$BACKUP_FILE" > "$TEMP_DIR/restore.sql"
    RESTORE_FILE="$TEMP_DIR/restore.sql"
else
    RESTORE_FILE="$BACKUP_FILE"
fi

echo "Starting database restore for $ENVIRONMENT environment..."

# Set environment variables
export PGPASSWORD=$DB_PASSWORD

# Stop application pods to prevent connections during restore
echo "Scaling down application pods..."
kubectl scale deployment priceguard-api -n $NAMESPACE --replicas=0

# Wait for pods to terminate
kubectl wait --for=delete pod -l app=priceguard-api -n $NAMESPACE --timeout=60s

# Create a new database for restore (optional - uncomment if needed)
# psql -h $DB_HOST -U $DB_USER -d postgres -c "DROP DATABASE IF EXISTS ${DB_NAME}_restore;"
# psql -h $DB_HOST -U $DB_USER -d postgres -c "CREATE DATABASE ${DB_NAME}_restore;"

# Restore database
echo "Restoring database..."
pg_restore -h $DB_HOST -U $DB_USER -d $DB_NAME \
    --verbose \
    --clean \
    --if-exists \
    --no-owner \
    --no-privileges \
    "$RESTORE_FILE"

# Run post-restore tasks
echo "Running post-restore tasks..."

# Update sequences (in case of data conflicts)
psql -h $DB_HOST -U $DB_USER -d $DB_NAME << 'EOF'
-- Reset sequences to avoid primary key conflicts
SELECT 'SELECT SETVAL(' || quote_literal(quote_ident(PGT.schemaname)||'.'||quote_ident(S.relname)) ||
       ', COALESCE(MAX(' ||quote_ident(C.attname)|| '), 1) ) FROM ' ||
       quote_ident(PGT.schemaname)||'.'||quote_ident(T.relname)|| ';'
FROM pg_class AS S,
     pg_depend AS D,
     pg_class AS T,
     pg_attribute AS C,
     pg_tables AS PGT
WHERE S.relkind = 'S'
    AND S.oid = D.objid
    AND D.refobjid = T.oid
    AND D.refobjid = C.attrelid
    AND D.refobjsubid = C.attnum
    AND T.relname = PGT.tablename
    AND PGT.schemaname = 'public'
ORDER BY S.relname;
EOF

# Verify restore
echo "Verifying restore..."
TABLE_COUNT=$(psql -h $DB_HOST -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';")
echo "Tables restored: $TABLE_COUNT"

USER_COUNT=$(psql -h $DB_HOST -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM users;" 2>/dev/null || echo "0")
echo "Users in database: $USER_COUNT"

# Scale application back up
echo "Scaling application back up..."
kubectl scale deployment priceguard-api -n $NAMESPACE --replicas=3

# Wait for pods to be ready
kubectl wait --for=condition=ready pod -l app=priceguard-api -n $NAMESPACE --timeout=120s

echo "✅ Database restore completed successfully!"

# Send notification (if configured)
if [ ! -z "${SLACK_WEBHOOK_URL:-}" ]; then
    curl -X POST -H 'Content-type: application/json' \
        --data "{\"text\":\"✅ Database restore completed for $ENVIRONMENT environment from backup: $(basename $BACKUP_FILE)\"}" \
        $SLACK_WEBHOOK_URL
fi

unset PGPASSWORD
