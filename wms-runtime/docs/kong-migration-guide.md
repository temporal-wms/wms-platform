# Kong API Gateway Migration Guide

This guide helps you migrate from kubectl port-forward to Kong API Gateway for the WMS emulator.

## Table of Contents

- [Overview](#overview)
- [Benefits](#benefits)
- [URL Mapping Reference](#url-mapping-reference)
- [Migration Steps](#migration-steps)
- [Configuration](#configuration)
- [Testing the Migration](#testing-the-migration)
- [Troubleshooting](#troubleshooting)
- [Rollback Instructions](#rollback-instructions)

## Overview

### Before (Port-Forward Approach)

**Requirements:**
- Multiple terminal windows running kubectl port-forward
- Each service has its own localhost port
- Manual management of port-forward processes

**Setup:**
```bash
# Terminal 1
kubectl port-forward svc/order-service -n wms-platform 8001:8001

# Terminal 2
kubectl port-forward svc/inventory-service -n wms-platform 8008:8008

# Terminal 3
kubectl port-forward svc/labor-service -n wms-platform 8009:8009

# Terminal 4
kubectl port-forward svc/orchestrator -n wms-platform 8080:8080

# ... repeat for 17 more services

# Terminal N
k6 run scripts/setup.js
```

### After (Kong Gateway Approach)

**Requirements:**
- Kong Gateway deployed (already done)
- Single entry point at `http://localhost:8888`
- No manual port-forwarding needed

**Setup:**
```bash
# Just run!
k6 run scripts/setup.js
```

## Benefits

| Aspect | Port-Forward | Kong Gateway |
|--------|--------------|--------------|
| **Setup Complexity** | High - Multiple terminal windows | Low - No setup needed |
| **Terminal Windows** | 20+ terminals | 1 terminal |
| **Port Management** | Manual - Easy conflicts | Automatic - No conflicts |
| **Service Discovery** | Manual URL configuration | Automatic routing |
| **Production Similarity** | Low - Direct service access | High - Same as production |
| **Parallel Testing** | Hard - Port conflicts | Easy - No conflicts |
| **Maintenance** | High - Process management | Low - Gateway handles routing |

## URL Mapping Reference

### Complete Service Mapping

| Service | Port-Forward URL | Kong Gateway URL | Port |
|---------|------------------|------------------|------|
| **Order Service** | `http://localhost:8001` | `http://localhost:8888/order-service` | 8001 |
| **Waving Service** | `http://localhost:8002` | `http://localhost:8888/waving-service` | 8002 |
| **Routing Service** | `http://localhost:8003` | `http://localhost:8888/routing-service` | 8003 |
| **Picking Service** | `http://localhost:8004` | `http://localhost:8888/picking-service` | 8004 |
| **Consolidation Service** | `http://localhost:8005` | `http://localhost:8888/consolidation-service` | 8005 |
| **Packing Service** | `http://localhost:8006` | `http://localhost:8888/packing-service` | 8006 |
| **Shipping Service** | `http://localhost:8007` | `http://localhost:8888/shipping-service` | 8007 |
| **Inventory Service** | `http://localhost:8008` | `http://localhost:8888/inventory-service` | 8008 |
| **Labor Service** | `http://localhost:8009` | `http://localhost:8888/labor-service` | 8009 |
| **Facility Service** | `http://localhost:8010` | `http://localhost:8888/facility-service` | 8010 |
| **Stow Service** | `http://localhost:8011` | `http://localhost:8888/stow-service` | 8011 |
| **Sortation Service** | `http://localhost:8012` | `http://localhost:8888/sortation-service` | 8012 |
| **Receiving Service** | `http://localhost:8013` | `http://localhost:8888/receiving-service` | 8013 |
| **Unit Service** | `http://localhost:8014` | `http://localhost:8888/unit-service` | 8014 |
| **Process Path Service** | `http://localhost:8015` | `http://localhost:8888/process-path-service` | 8015 |
| **WES Service** | `http://localhost:8016` | `http://localhost:8888/wes-service` | 8016 |
| **Walling Service** | `http://localhost:8017` | `http://localhost:8888/walling-service` | 8017 |
| **Billing Service** | `http://localhost:8018` | `http://localhost:8888/billing-service` | 8018 |
| **Channel Service** | `http://localhost:8019` | `http://localhost:8888/channel-service` | 8019 |
| **Seller Service** | `http://localhost:8020` | `http://localhost:8888/seller-service` | 8020 |
| **Seller Portal** | `http://localhost:8021` | `http://localhost:8888/seller-portal` | 8021 |
| **Orchestrator** | `http://localhost:8080` | `http://localhost:8888/orchestrator` | 8080 |

### API Path Examples

Kong Gateway uses path-based routing with automatic prefix stripping:

| Example API Call | Port-Forward | Kong Gateway |
|------------------|--------------|--------------|
| Create Order | `POST http://localhost:8001/api/v1/orders` | `POST http://localhost:8888/order-service/api/v1/orders` |
| Get Inventory | `GET http://localhost:8008/api/v1/stock` | `GET http://localhost:8888/inventory-service/api/v1/stock` |
| Create Worker | `POST http://localhost:8009/api/v1/workers` | `POST http://localhost:8888/labor-service/api/v1/workers` |
| Get Pick Tasks | `GET http://localhost:8004/api/v1/tasks` | `GET http://localhost:8888/picking-service/api/v1/tasks` |
| Send Signal | `POST http://localhost:8080/api/v1/signals/pick-completed` | `POST http://localhost:8888/orchestrator/api/v1/signals/pick-completed` |

**Note:** Kong strips the `/{service-name}` prefix before forwarding to the backend service, so the service receives the same path structure.

## Migration Steps

### Step 1: Verify Kong Gateway is Running

```bash
# Check Kong pods
kubectl get pods -n kong

# Expected output:
# NAME                            READY   STATUS    RESTARTS   AGE
# kong-gateway-xxxxx-xxxxx        1/1     Running   0          10m
```

### Step 2: Test Kong Gateway Connectivity

```bash
# Port-forward Kong Gateway (if not already accessible)
kubectl port-forward -n kong svc/kong-gateway 8888:80

# Test a service through Kong
curl http://localhost:8888/order-service/health

# Expected: {"status": "ok"} or similar
```

### Step 3: Stop Existing Port-Forwards (Optional)

```bash
# If you have the port-forward script running
cd ../network-utilities
./port-forward.sh stop

# Or kill individual port-forwards
pkill -f "kubectl port-forward"
```

### Step 4: Update Your Environment (If Needed)

The emulator now uses Kong Gateway by default, so no environment changes are needed!

**Default behavior (Kong Gateway):**
```bash
k6 run scripts/setup.js
```

**Custom Kong Gateway URL:**
```bash
export KONG_GATEWAY_URL=http://custom-host:8888
k6 run scripts/setup.js
```

### Step 5: Run Your First Test

```bash
cd wms-runtime

# Run setup script
k6 run scripts/setup.js

# Run smoke test
k6 run scripts/scenarios/smoke.js
```

### Step 6: Verify Success

Check the k6 output for successful HTTP requests:

```
✓ create facility status 201
✓ create inventory status 201
✓ create worker status 201
✓ create order status 201
```

All requests should succeed with Kong Gateway routing.

## Configuration

### Environment Variables

The WMS emulator supports these Kong-related environment variables:

| Variable | Default | Purpose |
|----------|---------|---------|
| `KONG_GATEWAY_URL` | `http://localhost:8888` | Kong Gateway base URL |
| `USE_KONG` | `true` | Enable/disable Kong Gateway routing |

### Usage Modes

#### 1. Default Mode (Kong Gateway)

**No configuration needed:**
```bash
k6 run scripts/scenarios/load.js
```

**With custom Kong URL:**
```bash
k6 run -e KONG_GATEWAY_URL=http://staging-kong:8888 scripts/scenarios/load.js
```

#### 2. Legacy Mode (Port-Forward)

**Disable Kong and use direct service connections:**
```bash
k6 run -e USE_KONG=false scripts/scenarios/load.js
```

**Then set up port-forwards manually:**
```bash
kubectl port-forward svc/order-service -n wms-platform 8001:8001
kubectl port-forward svc/inventory-service -n wms-platform 8008:8008
# ... etc
```

#### 3. Hybrid Mode

**Use Kong for most services, but override specific ones:**
```bash
# Use Kong for everything except order service
k6 run -e ORDER_SERVICE_URL=http://localhost:9001 scripts/scenarios/load.js
```

This is useful for:
- Debugging specific services locally
- Testing against different service versions
- Bypassing Kong for troubleshooting

## Testing the Migration

### Basic Connectivity Test

```bash
# Test each service via Kong Gateway
curl http://localhost:8888/order-service/health
curl http://localhost:8888/inventory-service/health
curl http://localhost:8888/labor-service/health
curl http://localhost:8888/picking-service/health
curl http://localhost:8888/orchestrator/health
```

### Smoke Test

```bash
cd wms-runtime
k6 run scripts/scenarios/smoke.js
```

**Expected output:**
```
✓ orders created: 5
✓ order_success_rate > 90%
✓ http_req_duration p(95) < 500ms
```

### Full Setup Test

```bash
k6 run scripts/setup.js
```

**Expected output:**
```
✓ Facilities created
✓ Inventory items created
✓ Workers created
✓ Shifts started
```

### Load Test

```bash
k6 run --duration 1m scripts/scenarios/load.js
```

**Expected output:**
```
✓ orders created: 100+
✓ order_success_rate > 90%
✓ http_req_failed < 1%
```

## Troubleshooting

### Issue: "Connection Refused" on Kong Gateway

**Symptoms:**
```
ERRO[0000] Connection refused on http://localhost:8888/order-service/api/v1/orders
```

**Solutions:**

1. **Check if Kong Gateway is running:**
   ```bash
   kubectl get pods -n kong
   ```

2. **Verify Kong Gateway port-forward:**
   ```bash
   kubectl port-forward -n kong svc/kong-gateway 8888:80
   ```

3. **Check HTTPRoutes are configured:**
   ```bash
   kubectl get httproutes -n wms-platform-dev
   ```

4. **Fall back to legacy mode:**
   ```bash
   k6 run -e USE_KONG=false scripts/setup.js
   ```

### Issue: "404 Not Found" on Service Routes

**Symptoms:**
```
ERRO[0000] HTTP 404: Not Found on http://localhost:8888/order-service/api/v1/orders
```

**Solutions:**

1. **Verify HTTPRoute exists for the service:**
   ```bash
   kubectl get httproute order-service-route -n wms-platform-dev -o yaml
   ```

2. **Check service path configuration:**
   ```bash
   # Ensure path matches HTTPRoute spec
   curl -v http://localhost:8888/order-service/api/v1/health
   ```

3. **Check Kong Gateway logs:**
   ```bash
   kubectl logs -n kong -l app=kong-gateway
   ```

### Issue: Services Work via Port-Forward but Not Kong

**Symptoms:**
- Direct port-forward: ✅ Works
- Kong Gateway: ❌ Fails

**Solutions:**

1. **Check ReferenceGrant (cross-namespace access):**
   ```bash
   kubectl get referencegrants -n kong
   ```

2. **Verify service is in correct namespace:**
   ```bash
   kubectl get svc -n wms-platform-dev | grep order-service
   ```

3. **Test service directly from Kong pod:**
   ```bash
   kubectl exec -n kong deploy/kong-gateway -- curl http://order-service.wms-platform-dev:8001/health
   ```

### Issue: Environment Variables Not Working

**Symptoms:**
```
Still using http://localhost:8001 instead of Kong Gateway
```

**Solutions:**

1. **Verify environment variable syntax:**
   ```bash
   # Correct
   k6 run -e KONG_GATEWAY_URL=http://localhost:8888 scripts/setup.js

   # Incorrect (missing -e flag)
   k6 run KONG_GATEWAY_URL=http://localhost:8888 scripts/setup.js
   ```

2. **Check if USE_KONG is explicitly disabled:**
   ```bash
   # This will disable Kong
   k6 run -e USE_KONG=false scripts/setup.js

   # This will enable Kong (default)
   k6 run scripts/setup.js
   ```

3. **Clear any conflicting environment variables:**
   ```bash
   unset ORDER_SERVICE_URL INVENTORY_SERVICE_URL
   k6 run scripts/setup.js
   ```

### Issue: High Latency Through Kong

**Symptoms:**
```
p(95) response time: 2000ms (was 200ms with port-forward)
```

**Solutions:**

1. **Check Kong Gateway resource limits:**
   ```bash
   kubectl describe pod -n kong -l app=kong-gateway | grep -A 5 Limits
   ```

2. **Monitor Kong Gateway metrics:**
   ```bash
   kubectl port-forward -n kong svc/kong-gateway 8001:8001
   curl http://localhost:8001/metrics
   ```

3. **Check if Kong plugins are adding overhead:**
   ```bash
   kubectl get kongplugins -n kong
   ```

4. **Use direct port-forward for performance testing:**
   ```bash
   k6 run -e USE_KONG=false scripts/scenarios/stress.js
   ```

## Rollback Instructions

If you need to revert to the old port-forward approach:

### Option 1: Use Legacy Mode Flag

**Quickest method - no code changes:**

```bash
# Set USE_KONG=false for all tests
export USE_KONG=false

k6 run scripts/setup.js
k6 run scripts/scenarios/load.js
```

**Don't forget to set up port-forwards:**
```bash
cd ../network-utilities
./port-forward.sh start
```

### Option 2: Override Service URLs

**Override specific services:**

```bash
k6 run \
  -e ORDER_SERVICE_URL=http://localhost:8001 \
  -e INVENTORY_SERVICE_URL=http://localhost:8008 \
  -e LABOR_SERVICE_URL=http://localhost:8009 \
  scripts/setup.js
```

### Option 3: Temporary Config Modification

**Edit config.js temporarily:**

```javascript
// In scripts/lib/config.js
const USE_KONG = false; // Force disable Kong

export const BASE_URLS = {
  orders: 'http://localhost:8001',
  inventory: 'http://localhost:8008',
  // ... etc
};
```

**⚠️ Note:** This is not recommended for long-term use. Use environment variables instead.

### Option 4: Full Revert (Git)

**If you have uncommitted changes:**

```bash
cd wms-runtime
git checkout scripts/lib/config.js
git checkout docs/
git checkout README.md
```

**Then use port-forward approach:**
```bash
cd ../network-utilities
./port-forward.sh start

cd ../wms-runtime
k6 run scripts/setup.js
```

## Migration Checklist

Use this checklist to track your migration progress:

- [ ] Kong Gateway is deployed and running
- [ ] Kong Gateway is accessible at `http://localhost:8888`
- [ ] Tested basic connectivity to services via Kong
- [ ] Ran smoke test successfully with Kong Gateway
- [ ] Ran full setup script with Kong Gateway
- [ ] Updated any custom scripts or documentation
- [ ] Stopped unnecessary port-forward processes
- [ ] Verified all team members are aware of the change
- [ ] Documented any custom configurations or overrides
- [ ] Created backup plan / rollback procedure

## Best Practices

### 1. Default to Kong Gateway

Always use Kong Gateway unless you have a specific reason not to:

**✅ Do:**
```bash
k6 run scripts/scenarios/load.js
```

**❌ Don't:**
```bash
k6 run -e USE_KONG=false scripts/scenarios/load.js  # Unless necessary
```

### 2. Use Environment Variables for Overrides

Don't hardcode URLs - use environment variables:

**✅ Do:**
```bash
k6 run -e KONG_GATEWAY_URL=http://staging:8888 scripts/scenarios/load.js
```

**❌ Don't:**
```javascript
// Don't hardcode in scripts
const BASE_URL = 'http://staging:8888';
```

### 3. Document Custom Configurations

If you need custom configurations, document them:

```bash
# Create a .env file or script
cat > run-staging-tests.sh <<EOF
#!/bin/bash
export KONG_GATEWAY_URL=http://staging-kong:8888
export MAX_ORDERS_PER_RUN=100
k6 run scripts/scenarios/load.js
EOF

chmod +x run-staging-tests.sh
```

### 4. Monitor Kong Gateway Health

Regularly check Kong Gateway health:

```bash
# Add to monitoring scripts
curl http://localhost:8888/order-service/health
kubectl get pods -n kong
```

### 5. Keep Rollback Plan Ready

Always have a rollback plan:

```bash
# Quick rollback script
cat > rollback-to-portforward.sh <<EOF
#!/bin/bash
echo "Rolling back to port-forward mode..."
export USE_KONG=false
cd network-utilities && ./port-forward.sh start
cd ../wms-runtime
echo "Ready to run tests with port-forward mode"
EOF

chmod +x rollback-to-portforward.sh
```

## Additional Resources

- [WMS Runtime README](../README.md) - Main emulator documentation
- [Picker Simulator Guide](picker-simulator.md) - Picker workflow simulation
- [Kong Gateway Documentation](https://docs.konghq.com/) - Official Kong docs
- [HTTPRoute Specification](https://gateway-api.sigs.k8s.io/references/spec/#gateway.networking.k8s.io/v1.HTTPRoute) - Kubernetes Gateway API

## Support

If you encounter issues not covered in this guide:

1. Check Kong Gateway logs:
   ```bash
   kubectl logs -n kong -l app=kong-gateway --tail=100
   ```

2. Check WMS service logs:
   ```bash
   kubectl logs -n wms-platform-dev deployment/order-service --tail=100
   ```

3. Review HTTPRoute configurations:
   ```bash
   kubectl get httproutes -n wms-platform-dev -o yaml
   ```

4. Fall back to legacy mode and report the issue:
   ```bash
   k6 run -e USE_KONG=false scripts/scenarios/smoke.js
   ```
