import React from 'react';
import ComponentCreator from '@docusaurus/ComponentCreator';

export default [
  {
    path: '/wms-platform/',
    component: ComponentCreator('/wms-platform/', 'ec0'),
    routes: [
      {
        path: '/wms-platform/',
        component: ComponentCreator('/wms-platform/', 'b90'),
        routes: [
          {
            path: '/wms-platform/',
            component: ComponentCreator('/wms-platform/', '24f'),
            routes: [
              {
                path: '/wms-platform/api/catalog',
                component: ComponentCreator('/wms-platform/api/catalog', '15a'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/events-api',
                component: ComponentCreator('/wms-platform/api/events-api', '382'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/order-service/cancel-order',
                component: ComponentCreator('/wms-platform/api/order-service/cancel-order', '1c7'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/order-service/create-a-new-order',
                component: ComponentCreator('/wms-platform/api/order-service/create-a-new-order', 'a95'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/order-service/get-dlq-statistics',
                component: ComponentCreator('/wms-platform/api/order-service/get-dlq-statistics', '9ae'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/order-service/get-order-by-id',
                component: ComponentCreator('/wms-platform/api/order-service/get-order-by-id', 'dd7'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/order-service/get-orders-eligible-for-retry',
                component: ComponentCreator('/wms-platform/api/order-service/get-orders-eligible-for-retry', 'da8'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/order-service/get-retry-metadata-for-order',
                component: ComponentCreator('/wms-platform/api/order-service/get-retry-metadata-for-order', 'bec'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/order-service/get-specific-dlq-entry',
                component: ComponentCreator('/wms-platform/api/order-service/get-specific-dlq-entry', 'd10'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/order-service/health-check',
                component: ComponentCreator('/wms-platform/api/order-service/health-check', '1e3'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/order-service/increment-retry-count',
                component: ComponentCreator('/wms-platform/api/order-service/increment-retry-count', '7ad'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/order-service/list-dead-letter-queue-entries',
                component: ComponentCreator('/wms-platform/api/order-service/list-dead-letter-queue-entries', '255'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/order-service/list-orders',
                component: ComponentCreator('/wms-platform/api/order-service/list-orders', 'b1e'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/order-service/move-order-to-dead-letter-queue',
                component: ComponentCreator('/wms-platform/api/order-service/move-order-to-dead-letter-queue', '437'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/order-service/order-service-api',
                component: ComponentCreator('/wms-platform/api/order-service/order-service-api', '87a'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/order-service/reset-order-for-retry',
                component: ComponentCreator('/wms-platform/api/order-service/reset-order-for-retry', '94a'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/order-service/resolve-dlq-entry',
                component: ComponentCreator('/wms-platform/api/order-service/resolve-dlq-entry', '341'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/order-service/validate-order',
                component: ComponentCreator('/wms-platform/api/order-service/validate-order', 'c58'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/rest-api',
                component: ComponentCreator('/wms-platform/api/rest-api', '300'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/specs/asyncapi',
                component: ComponentCreator('/wms-platform/api/specs/asyncapi', 'f8e'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/api/specs/openapi',
                component: ComponentCreator('/wms-platform/api/specs/openapi', '700'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/architecture/c4-diagrams/code',
                component: ComponentCreator('/wms-platform/architecture/c4-diagrams/code', 'c47'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/architecture/c4-diagrams/components',
                component: ComponentCreator('/wms-platform/architecture/c4-diagrams/components', '22a'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/architecture/c4-diagrams/containers',
                component: ComponentCreator('/wms-platform/architecture/c4-diagrams/containers', '791'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/architecture/c4-diagrams/context',
                component: ComponentCreator('/wms-platform/architecture/c4-diagrams/context', '46f'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/architecture/overview',
                component: ComponentCreator('/wms-platform/architecture/overview', 'a5d'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/architecture/sequence-diagrams/order-cancellation',
                component: ComponentCreator('/wms-platform/architecture/sequence-diagrams/order-cancellation', '8da'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/architecture/sequence-diagrams/order-fulfillment',
                component: ComponentCreator('/wms-platform/architecture/sequence-diagrams/order-fulfillment', '2a6'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/architecture/sequence-diagrams/packing-workflow',
                component: ComponentCreator('/wms-platform/architecture/sequence-diagrams/packing-workflow', '5a6'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/architecture/sequence-diagrams/picking-workflow',
                component: ComponentCreator('/wms-platform/architecture/sequence-diagrams/picking-workflow', '37d'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/architecture/sequence-diagrams/receiving-workflow',
                component: ComponentCreator('/wms-platform/architecture/sequence-diagrams/receiving-workflow', '132'),
                exact: true
              },
              {
                path: '/wms-platform/architecture/sequence-diagrams/reprocessing-workflow',
                component: ComponentCreator('/wms-platform/architecture/sequence-diagrams/reprocessing-workflow', '940'),
                exact: true
              },
              {
                path: '/wms-platform/architecture/sequence-diagrams/shipping-workflow',
                component: ComponentCreator('/wms-platform/architecture/sequence-diagrams/shipping-workflow', '96b'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/architecture/sequence-diagrams/unit-tracking',
                component: ComponentCreator('/wms-platform/architecture/sequence-diagrams/unit-tracking', '544'),
                exact: true
              },
              {
                path: '/wms-platform/architecture/sequence-diagrams/walling-workflow',
                component: ComponentCreator('/wms-platform/architecture/sequence-diagrams/walling-workflow', 'b63'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/architecture/sequence-diagrams/wes-execution',
                component: ComponentCreator('/wms-platform/architecture/sequence-diagrams/wes-execution', '9bc'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/architecture/system-diagrams/data-flow',
                component: ComponentCreator('/wms-platform/architecture/system-diagrams/data-flow', '58d'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/architecture/system-diagrams/deployment',
                component: ComponentCreator('/wms-platform/architecture/system-diagrams/deployment', '655'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/architecture/system-diagrams/infrastructure',
                component: ComponentCreator('/wms-platform/architecture/system-diagrams/infrastructure', 'b9f'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/domain-driven-design/aggregates/consolidation-unit',
                component: ComponentCreator('/wms-platform/domain-driven-design/aggregates/consolidation-unit', 'a78'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/domain-driven-design/aggregates/inbound-shipment',
                component: ComponentCreator('/wms-platform/domain-driven-design/aggregates/inbound-shipment', '26a'),
                exact: true
              },
              {
                path: '/wms-platform/domain-driven-design/aggregates/inventory-item',
                component: ComponentCreator('/wms-platform/domain-driven-design/aggregates/inventory-item', '72a'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/domain-driven-design/aggregates/order',
                component: ComponentCreator('/wms-platform/domain-driven-design/aggregates/order', 'c92'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/domain-driven-design/aggregates/pack-task',
                component: ComponentCreator('/wms-platform/domain-driven-design/aggregates/pack-task', '65b'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/domain-driven-design/aggregates/pick-route',
                component: ComponentCreator('/wms-platform/domain-driven-design/aggregates/pick-route', '60b'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/domain-driven-design/aggregates/pick-task',
                component: ComponentCreator('/wms-platform/domain-driven-design/aggregates/pick-task', 'b1f'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/domain-driven-design/aggregates/process-path',
                component: ComponentCreator('/wms-platform/domain-driven-design/aggregates/process-path', 'b03'),
                exact: true
              },
              {
                path: '/wms-platform/domain-driven-design/aggregates/putaway-task',
                component: ComponentCreator('/wms-platform/domain-driven-design/aggregates/putaway-task', 'f2c'),
                exact: true
              },
              {
                path: '/wms-platform/domain-driven-design/aggregates/shipment',
                component: ComponentCreator('/wms-platform/domain-driven-design/aggregates/shipment', '64d'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/domain-driven-design/aggregates/sortation-batch',
                component: ComponentCreator('/wms-platform/domain-driven-design/aggregates/sortation-batch', 'f33'),
                exact: true
              },
              {
                path: '/wms-platform/domain-driven-design/aggregates/station',
                component: ComponentCreator('/wms-platform/domain-driven-design/aggregates/station', '690'),
                exact: true
              },
              {
                path: '/wms-platform/domain-driven-design/aggregates/task-route',
                component: ComponentCreator('/wms-platform/domain-driven-design/aggregates/task-route', 'd61'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/domain-driven-design/aggregates/unit',
                component: ComponentCreator('/wms-platform/domain-driven-design/aggregates/unit', 'fdb'),
                exact: true
              },
              {
                path: '/wms-platform/domain-driven-design/aggregates/walling-task',
                component: ComponentCreator('/wms-platform/domain-driven-design/aggregates/walling-task', '194'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/domain-driven-design/aggregates/wave',
                component: ComponentCreator('/wms-platform/domain-driven-design/aggregates/wave', 'bd7'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/domain-driven-design/aggregates/worker',
                component: ComponentCreator('/wms-platform/domain-driven-design/aggregates/worker', '5c2'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/domain-driven-design/bounded-contexts',
                component: ComponentCreator('/wms-platform/domain-driven-design/bounded-contexts', 'a46'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/domain-driven-design/context-map',
                component: ComponentCreator('/wms-platform/domain-driven-design/context-map', 'c7d'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/domain-driven-design/domain-events',
                component: ComponentCreator('/wms-platform/domain-driven-design/domain-events', '56a'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/domain-driven-design/overview',
                component: ComponentCreator('/wms-platform/domain-driven-design/overview', '313'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/domain-driven-design/value-objects',
                component: ComponentCreator('/wms-platform/domain-driven-design/value-objects', '458'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/infrastructure/kafka',
                component: ComponentCreator('/wms-platform/infrastructure/kafka', '953'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/infrastructure/mongodb',
                component: ComponentCreator('/wms-platform/infrastructure/mongodb', '2b9'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/infrastructure/observability',
                component: ComponentCreator('/wms-platform/infrastructure/observability', '1d0'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/infrastructure/overview',
                component: ComponentCreator('/wms-platform/infrastructure/overview', 'a8c'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/infrastructure/temporal',
                component: ComponentCreator('/wms-platform/infrastructure/temporal', '659'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/services/billing-service',
                component: ComponentCreator('/wms-platform/services/billing-service', '05e'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/services/channel-service',
                component: ComponentCreator('/wms-platform/services/channel-service', '9d3'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/services/consolidation-service',
                component: ComponentCreator('/wms-platform/services/consolidation-service', 'fea'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/services/facility-service',
                component: ComponentCreator('/wms-platform/services/facility-service', 'a9e'),
                exact: true
              },
              {
                path: '/wms-platform/services/inventory-service',
                component: ComponentCreator('/wms-platform/services/inventory-service', '8c0'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/services/labor-service',
                component: ComponentCreator('/wms-platform/services/labor-service', '5af'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/services/orchestrator',
                component: ComponentCreator('/wms-platform/services/orchestrator', '439'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/services/order-service',
                component: ComponentCreator('/wms-platform/services/order-service', '3dd'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/services/packing-service',
                component: ComponentCreator('/wms-platform/services/packing-service', '576'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/services/picking-service',
                component: ComponentCreator('/wms-platform/services/picking-service', '69b'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/services/process-path-service',
                component: ComponentCreator('/wms-platform/services/process-path-service', '3ef'),
                exact: true
              },
              {
                path: '/wms-platform/services/receiving-service',
                component: ComponentCreator('/wms-platform/services/receiving-service', '139'),
                exact: true
              },
              {
                path: '/wms-platform/services/routing-service',
                component: ComponentCreator('/wms-platform/services/routing-service', 'a75'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/services/seller-portal',
                component: ComponentCreator('/wms-platform/services/seller-portal', '10a'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/services/seller-service',
                component: ComponentCreator('/wms-platform/services/seller-service', '745'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/services/shipping-service',
                component: ComponentCreator('/wms-platform/services/shipping-service', '5ec'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/services/sortation-service',
                component: ComponentCreator('/wms-platform/services/sortation-service', '8bd'),
                exact: true
              },
              {
                path: '/wms-platform/services/stow-service',
                component: ComponentCreator('/wms-platform/services/stow-service', 'cea'),
                exact: true
              },
              {
                path: '/wms-platform/services/unit-service',
                component: ComponentCreator('/wms-platform/services/unit-service', '0c4'),
                exact: true
              },
              {
                path: '/wms-platform/services/walling-service',
                component: ComponentCreator('/wms-platform/services/walling-service', 'e83'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/services/waving-service',
                component: ComponentCreator('/wms-platform/services/waving-service', '9ec'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/services/wes-service',
                component: ComponentCreator('/wms-platform/services/wes-service', 'b65'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/activities/consolidation-activities',
                component: ComponentCreator('/wms-platform/temporal/activities/consolidation-activities', '88e'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/activities/giftwrap-activities',
                component: ComponentCreator('/wms-platform/temporal/activities/giftwrap-activities', '18a'),
                exact: true
              },
              {
                path: '/wms-platform/temporal/activities/inventory-activities',
                component: ComponentCreator('/wms-platform/temporal/activities/inventory-activities', '728'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/activities/order-activities',
                component: ComponentCreator('/wms-platform/temporal/activities/order-activities', 'fb8'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/activities/overview',
                component: ComponentCreator('/wms-platform/temporal/activities/overview', '388'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/activities/packing-activities',
                component: ComponentCreator('/wms-platform/temporal/activities/packing-activities', '4b7'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/activities/picking-activities',
                component: ComponentCreator('/wms-platform/temporal/activities/picking-activities', '8c2'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/activities/process-path-activities',
                component: ComponentCreator('/wms-platform/temporal/activities/process-path-activities', '5a2'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/activities/receiving-activities',
                component: ComponentCreator('/wms-platform/temporal/activities/receiving-activities', 'a9b'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/activities/reprocessing-activities',
                component: ComponentCreator('/wms-platform/temporal/activities/reprocessing-activities', 'bbe'),
                exact: true
              },
              {
                path: '/wms-platform/temporal/activities/routing-activities',
                component: ComponentCreator('/wms-platform/temporal/activities/routing-activities', '5be'),
                exact: true
              },
              {
                path: '/wms-platform/temporal/activities/shipping-activities',
                component: ComponentCreator('/wms-platform/temporal/activities/shipping-activities', '2b9'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/activities/slam-activities',
                component: ComponentCreator('/wms-platform/temporal/activities/slam-activities', 'd7b'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/activities/sortation-activities',
                component: ComponentCreator('/wms-platform/temporal/activities/sortation-activities', 'f0d'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/activities/stow-activities',
                component: ComponentCreator('/wms-platform/temporal/activities/stow-activities', '7f3'),
                exact: true
              },
              {
                path: '/wms-platform/temporal/activities/unit-activities',
                component: ComponentCreator('/wms-platform/temporal/activities/unit-activities', '3ed'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/diagrams/order-flow',
                component: ComponentCreator('/wms-platform/temporal/diagrams/order-flow', 'c0d'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/diagrams/signal-flow',
                component: ComponentCreator('/wms-platform/temporal/diagrams/signal-flow', '119'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/diagrams/workflow-hierarchy',
                component: ComponentCreator('/wms-platform/temporal/diagrams/workflow-hierarchy', '474'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/overview',
                component: ComponentCreator('/wms-platform/temporal/overview', '96e'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/retry-policies',
                component: ComponentCreator('/wms-platform/temporal/retry-policies', '0d9'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/signals-queries',
                component: ComponentCreator('/wms-platform/temporal/signals-queries', 'e42'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/task-queues',
                component: ComponentCreator('/wms-platform/temporal/task-queues', 'ca7'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/cancellation',
                component: ComponentCreator('/wms-platform/temporal/workflows/cancellation', '488'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/consolidation',
                component: ComponentCreator('/wms-platform/temporal/workflows/consolidation', 'd63'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/gift-wrap',
                component: ComponentCreator('/wms-platform/temporal/workflows/gift-wrap', '79f'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/inbound-fulfillment',
                component: ComponentCreator('/wms-platform/temporal/workflows/inbound-fulfillment', 'ff4'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/order-fulfillment',
                component: ComponentCreator('/wms-platform/temporal/workflows/order-fulfillment', '563'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/packing',
                component: ComponentCreator('/wms-platform/temporal/workflows/packing', '2c7'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/picking',
                component: ComponentCreator('/wms-platform/temporal/workflows/picking', 'da1'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/planning',
                component: ComponentCreator('/wms-platform/temporal/workflows/planning', 'aa9'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/reprocessing',
                component: ComponentCreator('/wms-platform/temporal/workflows/reprocessing', '626'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/service-consolidation',
                component: ComponentCreator('/wms-platform/temporal/workflows/service-consolidation', '927'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/service-packing',
                component: ComponentCreator('/wms-platform/temporal/workflows/service-packing', 'b03'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/service-picking',
                component: ComponentCreator('/wms-platform/temporal/workflows/service-picking', 'a0f'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/service-shipping',
                component: ComponentCreator('/wms-platform/temporal/workflows/service-shipping', '2b7'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/service-wes',
                component: ComponentCreator('/wms-platform/temporal/workflows/service-wes', '32a'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/shipping',
                component: ComponentCreator('/wms-platform/temporal/workflows/shipping', 'dd6'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/sortation',
                component: ComponentCreator('/wms-platform/temporal/workflows/sortation', 'f2f'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/stock-shortage',
                component: ComponentCreator('/wms-platform/temporal/workflows/stock-shortage', '23d'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/temporal/workflows/wes-execution',
                component: ComponentCreator('/wms-platform/temporal/workflows/wes-execution', 'd81'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/wms-platform/',
                component: ComponentCreator('/wms-platform/', 'c5c'),
                exact: true,
                sidebar: "docsSidebar"
              }
            ]
          }
        ]
      }
    ]
  },
  {
    path: '*',
    component: ComponentCreator('*'),
  },
];
