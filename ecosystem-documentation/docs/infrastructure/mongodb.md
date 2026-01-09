---
sidebar_position: 2
---

# MongoDB

MongoDB is used as the primary database for all WMS Platform services.

## Configuration

### Connection String

```
mongodb://wmsuser:wmspassword@mongodb-0.mongodb-headless:27017/wms?authSource=wms&replicaSet=rs0
```

### Databases

| Service | Database |
|---------|----------|
| Order Service | orders_db |
| Waving Service | waves_db |
| Routing Service | routing_db |
| Picking Service | picking_db |
| Consolidation Service | consolidation_db |
| Packing Service | packing_db |
| Shipping Service | shipping_db |
| Inventory Service | inventory_db |
| Labor Service | labor_db |

## Deployment

### Kubernetes (Bitnami Helm Chart)

```yaml
# mongodb-values.yaml
architecture: replicaset
replicaCount: 3

auth:
  enabled: true
  rootUser: root
  rootPassword: rootpassword
  databases:
    - wms
  usernames:
    - wmsuser
  passwords:
    - wmspassword

persistence:
  enabled: true
  size: 50Gi
  storageClass: standard
```

```bash
helm install mongodb bitnami/mongodb \
  -f mongodb-values.yaml \
  -n mongodb --create-namespace
```

## Indexes

### Orders Collection

```javascript
db.orders.createIndex({ "customerId": 1 })
db.orders.createIndex({ "status": 1 })
db.orders.createIndex({ "createdAt": -1 })
db.orders.createIndex({ "waveId": 1 })
```

### Outbox Collection

```javascript
db.outbox.createIndex({ "published": 1, "createdAt": 1 })
db.outbox.createIndex({ "eventType": 1 })
```

### Inventory Collection

```javascript
db.inventory_items.createIndex({ "sku": 1 })
db.inventory_items.createIndex({ "location.zone": 1, "location.aisle": 1 })
```

## Replica Set

```mermaid
graph LR
    App[Application] --> Primary[(Primary)]
    Primary --> Secondary1[(Secondary 1)]
    Primary --> Secondary2[(Secondary 2)]

    Secondary1 -.->|Failover| Primary
    Secondary2 -.->|Failover| Primary
```

## Backup Strategy

```bash
# Create backup
mongodump --uri="mongodb://..." --out=/backup/$(date +%Y%m%d)

# Restore
mongorestore --uri="mongodb://..." /backup/20240115
```

## Monitoring

Key metrics:
- `mongodb_connections_current`
- `mongodb_op_counters_total`
- `mongodb_mem_resident`
- `mongodb_repl_lag_seconds`

## Related Documentation

- [Infrastructure Overview](./overview) - System topology
- [Services](/services/order-service) - Service configuration
