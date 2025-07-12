#!/bin/bash

# PriceGuard Deployment Script
# Usage: ./deploy.sh [environment] [version]

set -euo pipefail

ENVIRONMENT=${1:-staging}
VERSION=${2:-latest}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
case $ENVIRONMENT in
    "production")
        NAMESPACE="priceguard"
        REPLICAS=3
        DOMAIN="api.priceguard.growthfolio.com"
        ;;
    "staging")
        NAMESPACE="priceguard-staging"
        REPLICAS=1
        DOMAIN="api-staging.priceguard.growthfolio.com"
        ;;
    *)
        echo -e "${RED}Unknown environment: $ENVIRONMENT${NC}"
        exit 1
        ;;
esac

# Functions
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
}

check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        error "kubectl is not installed"
        exit 1
    fi
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        error "Docker is not installed"
        exit 1
    fi
    
    # Check cluster access
    if ! kubectl cluster-info &> /dev/null; then
        error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    success "Prerequisites checked"
}

build_image() {
    log "Building Docker image..."
    
    # Build image
    docker build -t "priceguard-api:$VERSION" .
    
    # Tag for registry (replace with your registry)
    docker tag "priceguard-api:$VERSION" "ghcr.io/growthfolio/priceguard-api:$VERSION"
    
    success "Docker image built: priceguard-api:$VERSION"
}

push_image() {
    log "Pushing Docker image to registry..."
    
    # Login to registry (GitHub Container Registry example)
    echo $GITHUB_TOKEN | docker login ghcr.io -u $GITHUB_USERNAME --password-stdin
    
    # Push image
    docker push "ghcr.io/growthfolio/priceguard-api:$VERSION"
    
    success "Docker image pushed"
}

create_namespace() {
    log "Creating namespace if not exists..."
    
    kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    
    success "Namespace $NAMESPACE ready"
}

deploy_secrets() {
    log "Deploying secrets..."
    
    # Check if secrets exist
    if kubectl get secret priceguard-secrets -n $NAMESPACE &> /dev/null; then
        warning "Secrets already exist, skipping..."
    else
        # Apply secrets (make sure to update with real values)
        kubectl apply -f k8s/secrets.yaml -n $NAMESPACE
        success "Secrets deployed"
    fi
}

deploy_configmaps() {
    log "Deploying configmaps..."
    
    # Apply configmaps
    kubectl apply -f k8s/configmap.yaml
    
    success "ConfigMaps deployed"
}

deploy_database() {
    log "Deploying database..."
    
    # Apply database manifests
    kubectl apply -f k8s/database.yaml
    
    # Wait for database to be ready
    log "Waiting for database to be ready..."
    kubectl wait --for=condition=ready pod -l app=postgres -n $NAMESPACE --timeout=300s
    kubectl wait --for=condition=ready pod -l app=redis -n $NAMESPACE --timeout=300s
    
    success "Database deployed and ready"
}

run_migrations() {
    log "Running database migrations..."
    
    # Create migration job
    cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: priceguard-migration-$VERSION
  namespace: $NAMESPACE
spec:
  template:
    spec:
      containers:
      - name: migration
        image: ghcr.io/growthfolio/priceguard-api:$VERSION
        command: ["/app/migrate"]
        env:
        - name: DB_HOST
          valueFrom:
            configMapKeyRef:
              name: priceguard-config
              key: DB_HOST
        - name: DB_NAME
          valueFrom:
            configMapKeyRef:
              name: priceguard-config
              key: DB_NAME
        - name: DB_USER
          valueFrom:
            configMapKeyRef:
              name: priceguard-config
              key: DB_USER
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: priceguard-secrets
              key: db-password
      restartPolicy: Never
EOF
    
    # Wait for migration to complete
    kubectl wait --for=condition=complete job/priceguard-migration-$VERSION -n $NAMESPACE --timeout=300s
    
    success "Database migrations completed"
}

deploy_application() {
    log "Deploying application..."
    
    # Update deployment with new image
    kubectl set image deployment/priceguard-api priceguard-api=ghcr.io/growthfolio/priceguard-api:$VERSION -n $NAMESPACE
    
    # Apply other manifests
    kubectl apply -f k8s/deployment.yaml
    kubectl apply -f k8s/services.yaml
    kubectl apply -f k8s/hpa.yaml
    kubectl apply -f k8s/pdb.yaml
    kubectl apply -f k8s/network-policy.yaml
    kubectl apply -f k8s/ingress.yaml
    
    # Wait for rollout to complete
    log "Waiting for deployment rollout..."
    kubectl rollout status deployment/priceguard-api -n $NAMESPACE --timeout=600s
    
    success "Application deployed"
}

run_health_check() {
    log "Running health checks..."
    
    # Get service endpoint
    if kubectl get ingress priceguard-ingress -n $NAMESPACE &> /dev/null; then
        ENDPOINT="https://$DOMAIN"
    else
        # Use port-forward for testing
        kubectl port-forward svc/priceguard-api-service -n $NAMESPACE 8080:80 &
        PORT_FORWARD_PID=$!
        sleep 5
        ENDPOINT="http://localhost:8080"
    fi
    
    # Health check
    for i in {1..10}; do
        if curl -s "$ENDPOINT/health" | grep -q "healthy"; then
            success "Health check passed"
            break
        else
            if [ $i -eq 10 ]; then
                error "Health check failed after 10 attempts"
                return 1
            fi
            log "Health check attempt $i failed, retrying..."
            sleep 10
        fi
    done
    
    # API endpoint test
    if curl -s "$ENDPOINT/api/health" | grep -q "healthy"; then
        success "API endpoint test passed"
    else
        warning "API endpoint test failed"
    fi
    
    # Clean up port forward if created
    if [ ! -z "${PORT_FORWARD_PID:-}" ]; then
        kill $PORT_FORWARD_PID 2>/dev/null || true
    fi
}

rollback() {
    local previous_version=$1
    error "Deployment failed, rolling back to $previous_version"
    
    kubectl set image deployment/priceguard-api priceguard-api=ghcr.io/growthfolio/priceguard-api:$previous_version -n $NAMESPACE
    kubectl rollout status deployment/priceguard-api -n $NAMESPACE --timeout=300s
    
    warning "Rollback completed to version $previous_version"
}

main() {
    log "Starting deployment of PriceGuard API"
    log "Environment: $ENVIRONMENT"
    log "Version: $VERSION"
    log "Namespace: $NAMESPACE"
    echo ""
    
    # Get current version for potential rollback
    CURRENT_VERSION=$(kubectl get deployment priceguard-api -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null | cut -d':' -f2 || echo "none")
    
    check_prerequisites
    
    # Build and push image (skip if version is 'latest' and in staging)
    if [ "$VERSION" != "latest" ] || [ "$ENVIRONMENT" = "production" ]; then
        build_image
        push_image
    fi
    
    create_namespace
    deploy_secrets
    deploy_configmaps
    deploy_database
    
    # Run migrations only if not rollback
    if [ "$VERSION" != "$CURRENT_VERSION" ]; then
        run_migrations
    fi
    
    deploy_application
    
    # Health check with rollback on failure
    if ! run_health_check; then
        if [ "$CURRENT_VERSION" != "none" ]; then
            rollback "$CURRENT_VERSION"
        else
            error "Deployment failed and no previous version to rollback to"
            exit 1
        fi
    else
        success "Deployment completed successfully!"
        log "Application is running at: https://$DOMAIN"
    fi
}

# Handle script arguments
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Usage: $0 [environment] [version]"
    echo ""
    echo "Arguments:"
    echo "  environment  Target environment (staging|production) [default: staging]"
    echo "  version      Docker image version [default: latest]"
    echo ""
    echo "Examples:"
    echo "  $0 staging v1.0.0"
    echo "  $0 production v1.0.1"
    echo ""
    exit 0
fi

# Run main deployment
main "$@"
