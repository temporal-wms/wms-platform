---
sidebar_position: 1
---

# Infrastructure Topology

This document describes the infrastructure topology of the WMS Platform, including all components and their interactions.

## Infrastructure Overview

```mermaid
graph TB
    subgraph "External"
        Internet[Internet]
        Carriers[Carrier APIs]
    end

    subgraph "Kubernetes Cluster"
        subgraph "Ingress Layer"
            Ingress[Ingress Controller]
            WAF[Web Application Firewall]
        end

        subgraph "Application Layer"
            subgraph "wms-platform namespace"
                OrderSvc[Order Service]
                WavingSvc[Waving Service]
                RoutingSvc[Routing Service]
                PickingSvc[Picking Service]
                ConsolidationSvc[Consolidation Service]
                PackingSvc[Packing Service]
                ShippingSvc[Shipping Service]
                InventorySvc[Inventory Service]
                LaborSvc[Labor Service]
                Orchestrator[Orchestrator]
            end
        end

        subgraph "Workflow Layer"
            subgraph "temporal namespace"
                TemporalFrontend[Temporal Frontend]
                TemporalHistory[Temporal History]
                TemporalMatching[Temporal Matching]
                TemporalWorker[Temporal Worker]
            end
        end

        subgraph "Message Layer"
            subgraph "kafka namespace"
                Kafka1[Kafka Broker 1]
                Kafka2[Kafka Broker 2]
                Kafka3[Kafka Broker 3]
                Zookeeper[Zookeeper]
            end
        end

        subgraph "Data Layer"
            subgraph "mongodb namespace"
                MongoDB1[(MongoDB Primary)]
                MongoDB2[(MongoDB Secondary)]
                MongoDB3[(MongoDB Secondary)]
            end
        end

        subgraph "Observability Layer"
            subgraph "observability namespace"
                Prometheus[Prometheus]
                Grafana[Grafana]
                Tempo[Tempo]
                Loki[Loki]
                OTEL[OTel Collector]
            end
        end
    end

    Internet --> Ingress
    Ingress --> WAF
    WAF --> OrderSvc
    WAF --> WavingSvc
    WAF --> PickingSvc

    OrderSvc --> Kafka1
    OrderSvc --> MongoDB1
    Orchestrator --> TemporalFrontend
    ShippingSvc --> Carriers

    OrderSvc --> OTEL
    OTEL --> Tempo
    OTEL --> Loki
    Prometheus --> Grafana
```

## Network Architecture

```mermaid
graph TB
    subgraph "External Network"
        LB[Load Balancer<br/>AWS ALB / GCP LB]
    end

    subgraph "DMZ"
        Ingress[Ingress Controller]
    end

    subgraph "Service Network"
        subgraph "Public Services"
            OrderAPI[Order API]
            WavingAPI[Waving API]
        end

        subgraph "Internal Services"
            Orchestrator[Orchestrator]
            PickingAPI[Picking API]
            PackingAPI[Packing API]
        end
    end

    subgraph "Data Network"
        MongoDB[(MongoDB)]
        Kafka[Kafka]
        Temporal[Temporal]
    end

    LB --> Ingress
    Ingress --> OrderAPI
    Ingress --> WavingAPI
    OrderAPI --> Orchestrator
    Orchestrator --> PickingAPI
    OrderAPI --> MongoDB
    OrderAPI --> Kafka
    Orchestrator --> Temporal
```

## Component Specifications

### Application Services

| Service | Replicas | CPU Request | Memory Request | CPU Limit | Memory Limit |
|---------|----------|-------------|----------------|-----------|--------------|
| Order Service | 2 | 100m | 128Mi | 500m | 512Mi |
| Waving Service | 2 | 100m | 128Mi | 500m | 512Mi |
| Routing Service | 2 | 100m | 128Mi | 500m | 512Mi |
| Picking Service | 2 | 100m | 128Mi | 500m | 512Mi |
| Consolidation Service | 2 | 100m | 128Mi | 500m | 512Mi |
| Packing Service | 2 | 100m | 128Mi | 500m | 512Mi |
| Shipping Service | 2 | 100m | 128Mi | 500m | 512Mi |
| Inventory Service | 2 | 100m | 128Mi | 500m | 512Mi |
| Labor Service | 2 | 100m | 128Mi | 500m | 512Mi |
| Orchestrator | 3 | 200m | 256Mi | 1000m | 1Gi |

### Infrastructure Components

| Component | Configuration | Storage | High Availability |
|-----------|--------------|---------|-------------------|
| MongoDB | 3-node ReplicaSet | 100Gi SSD | Primary + 2 Secondary |
| Kafka | 3 brokers | 50Gi SSD each | Replication factor 3 |
| Zookeeper | 3 nodes | 10Gi SSD | Quorum-based |
| Temporal | 4 services | 20Gi SSD | Multi-replica |
| Prometheus | 1 replica | 100Gi | N/A (stateful) |
| Tempo | 1 replica | 100Gi | N/A (stateful) |

## Service Mesh

```mermaid
graph TB
    subgraph "Service Mesh (Optional)"
        subgraph "Control Plane"
            Istiod[Istiod]
        end

        subgraph "Data Plane"
            subgraph "Order Pod"
                OrderApp[Order Service]
                OrderProxy[Envoy Sidecar]
            end

            subgraph "Picking Pod"
                PickingApp[Picking Service]
                PickingProxy[Envoy Sidecar]
            end
        end
    end

    Istiod --> OrderProxy
    Istiod --> PickingProxy
    OrderProxy <--> PickingProxy
```

## Storage Architecture

```mermaid
graph TB
    subgraph "Persistent Storage"
        subgraph "MongoDB Storage"
            PV1[PersistentVolume<br/>100Gi]
            PV2[PersistentVolume<br/>100Gi]
            PV3[PersistentVolume<br/>100Gi]
        end

        subgraph "Kafka Storage"
            PV4[PersistentVolume<br/>50Gi]
            PV5[PersistentVolume<br/>50Gi]
            PV6[PersistentVolume<br/>50Gi]
        end

        subgraph "Observability Storage"
            PV7[Prometheus PV<br/>100Gi]
            PV8[Tempo PV<br/>100Gi]
            PV9[Loki PV<br/>100Gi]
        end
    end

    subgraph "Storage Class"
        SSD[SSD Storage Class<br/>gp3/pd-ssd]
    end

    PV1 --> SSD
    PV2 --> SSD
    PV3 --> SSD
    PV4 --> SSD
```

## Security Architecture

```mermaid
graph TB
    subgraph "Security Layers"
        subgraph "Network Security"
            NetworkPolicy[Network Policies]
            PodSecurity[Pod Security Standards]
        end

        subgraph "Authentication"
            OIDC[OIDC Provider]
            ServiceAccounts[Service Accounts]
        end

        subgraph "Authorization"
            RBAC[RBAC Policies]
            PodSecurityPolicy[Pod Security]
        end

        subgraph "Secrets Management"
            K8sSecrets[Kubernetes Secrets]
            ExternalSecrets[External Secrets<br/>Vault/AWS SM]
        end

        subgraph "Communication"
            mTLS[Mutual TLS]
            CertManager[Cert Manager]
        end
    end
```

## High Availability

### MongoDB ReplicaSet

```mermaid
graph LR
    subgraph "MongoDB ReplicaSet"
        Primary[(Primary)]
        Secondary1[(Secondary 1)]
        Secondary2[(Secondary 2)]
    end

    App[Application] --> Primary
    Primary --> Secondary1
    Primary --> Secondary2

    Secondary1 -.->|Failover| Primary
    Secondary2 -.->|Failover| Primary
```

### Kafka Cluster

```mermaid
graph TB
    subgraph "Kafka Cluster"
        Broker1[Broker 1]
        Broker2[Broker 2]
        Broker3[Broker 3]
    end

    subgraph "Topic: wms.orders.events"
        P0[Partition 0]
        P1[Partition 1]
        P2[Partition 2]
    end

    Broker1 --> P0
    Broker2 --> P1
    Broker3 --> P2

    P0 -.->|Replica| Broker2
    P1 -.->|Replica| Broker3
    P2 -.->|Replica| Broker1
```

## Disaster Recovery

| Component | RPO | RTO | Backup Strategy |
|-----------|-----|-----|-----------------|
| MongoDB | 1 hour | 15 min | Continuous backup + Snapshots |
| Kafka | 0 (replication) | 5 min | Multi-broker replication |
| Temporal | 1 hour | 15 min | Database backup |
| Configuration | Real-time | 5 min | GitOps |

## Related Diagrams

- [Deployment](./deployment) - Kubernetes resources
- [Data Flow](./data-flow) - Data movement
- [Observability](/infrastructure/observability) - Monitoring stack
