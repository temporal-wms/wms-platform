import React from 'react';
import ComponentCreator from '@docusaurus/ComponentCreator';

export default [
  {
    path: '/__docusaurus/debug',
    component: ComponentCreator('/__docusaurus/debug', '5ff'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/config',
    component: ComponentCreator('/__docusaurus/debug/config', '5ba'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/content',
    component: ComponentCreator('/__docusaurus/debug/content', 'a2b'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/globalData',
    component: ComponentCreator('/__docusaurus/debug/globalData', 'c3c'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/metadata',
    component: ComponentCreator('/__docusaurus/debug/metadata', '156'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/registry',
    component: ComponentCreator('/__docusaurus/debug/registry', '88c'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/routes',
    component: ComponentCreator('/__docusaurus/debug/routes', '000'),
    exact: true
  },
  {
    path: '/',
    component: ComponentCreator('/', 'bab'),
    routes: [
      {
        path: '/',
        component: ComponentCreator('/', '6c5'),
        routes: [
          {
            path: '/',
            component: ComponentCreator('/', '4af'),
            routes: [
              {
                path: '/api/catalog',
                component: ComponentCreator('/api/catalog', 'f82'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/events-api',
                component: ComponentCreator('/api/events-api', 'd0f'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/order-service/cancel-order',
                component: ComponentCreator('/api/order-service/cancel-order', '61b'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/order-service/create-a-new-order',
                component: ComponentCreator('/api/order-service/create-a-new-order', '719'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/order-service/get-dlq-statistics',
                component: ComponentCreator('/api/order-service/get-dlq-statistics', '712'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/order-service/get-order-by-id',
                component: ComponentCreator('/api/order-service/get-order-by-id', 'c68'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/order-service/get-orders-eligible-for-retry',
                component: ComponentCreator('/api/order-service/get-orders-eligible-for-retry', '5a5'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/order-service/get-retry-metadata-for-order',
                component: ComponentCreator('/api/order-service/get-retry-metadata-for-order', 'a9b'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/order-service/get-specific-dlq-entry',
                component: ComponentCreator('/api/order-service/get-specific-dlq-entry', '004'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/order-service/health-check',
                component: ComponentCreator('/api/order-service/health-check', '401'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/order-service/increment-retry-count',
                component: ComponentCreator('/api/order-service/increment-retry-count', '650'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/order-service/list-dead-letter-queue-entries',
                component: ComponentCreator('/api/order-service/list-dead-letter-queue-entries', '3af'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/order-service/list-orders',
                component: ComponentCreator('/api/order-service/list-orders', '3d6'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/order-service/move-order-to-dead-letter-queue',
                component: ComponentCreator('/api/order-service/move-order-to-dead-letter-queue', '975'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/order-service/order-service-api',
                component: ComponentCreator('/api/order-service/order-service-api', '65d'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/order-service/reset-order-for-retry',
                component: ComponentCreator('/api/order-service/reset-order-for-retry', '116'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/order-service/resolve-dlq-entry',
                component: ComponentCreator('/api/order-service/resolve-dlq-entry', '190'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/order-service/validate-order',
                component: ComponentCreator('/api/order-service/validate-order', '9fd'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/rest-api',
                component: ComponentCreator('/api/rest-api', '7c4'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/specs/asyncapi/',
                component: ComponentCreator('/api/specs/asyncapi/', '515'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/api/specs/openapi/',
                component: ComponentCreator('/api/specs/openapi/', '7c1'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/architecture/c4-diagrams/code',
                component: ComponentCreator('/architecture/c4-diagrams/code', '82f'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/architecture/c4-diagrams/components',
                component: ComponentCreator('/architecture/c4-diagrams/components', 'c6d'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/architecture/c4-diagrams/containers',
                component: ComponentCreator('/architecture/c4-diagrams/containers', 'd94'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/architecture/c4-diagrams/context',
                component: ComponentCreator('/architecture/c4-diagrams/context', '6a3'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/architecture/overview',
                component: ComponentCreator('/architecture/overview', 'c05'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/architecture/sequence-diagrams/order-cancellation',
                component: ComponentCreator('/architecture/sequence-diagrams/order-cancellation', '636'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/architecture/sequence-diagrams/order-fulfillment',
                component: ComponentCreator('/architecture/sequence-diagrams/order-fulfillment', '167'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/architecture/sequence-diagrams/packing-workflow',
                component: ComponentCreator('/architecture/sequence-diagrams/packing-workflow', '69f'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/architecture/sequence-diagrams/picking-workflow',
                component: ComponentCreator('/architecture/sequence-diagrams/picking-workflow', '097'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/architecture/sequence-diagrams/shipping-workflow',
                component: ComponentCreator('/architecture/sequence-diagrams/shipping-workflow', 'd01'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/architecture/sequence-diagrams/walling-workflow',
                component: ComponentCreator('/architecture/sequence-diagrams/walling-workflow', 'a45'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/architecture/sequence-diagrams/wes-execution',
                component: ComponentCreator('/architecture/sequence-diagrams/wes-execution', '2bb'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/architecture/system-diagrams/data-flow',
                component: ComponentCreator('/architecture/system-diagrams/data-flow', '0ca'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/architecture/system-diagrams/deployment',
                component: ComponentCreator('/architecture/system-diagrams/deployment', '785'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/architecture/system-diagrams/infrastructure',
                component: ComponentCreator('/architecture/system-diagrams/infrastructure', 'db0'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/domain-driven-design/aggregates/consolidation-unit',
                component: ComponentCreator('/domain-driven-design/aggregates/consolidation-unit', '244'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/domain-driven-design/aggregates/inbound-shipment',
                component: ComponentCreator('/domain-driven-design/aggregates/inbound-shipment', 'a33'),
                exact: true
              },
              {
                path: '/domain-driven-design/aggregates/inventory-item',
                component: ComponentCreator('/domain-driven-design/aggregates/inventory-item', 'b95'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/domain-driven-design/aggregates/order',
                component: ComponentCreator('/domain-driven-design/aggregates/order', '0e8'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/domain-driven-design/aggregates/pack-task',
                component: ComponentCreator('/domain-driven-design/aggregates/pack-task', '764'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/domain-driven-design/aggregates/pick-route',
                component: ComponentCreator('/domain-driven-design/aggregates/pick-route', '363'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/domain-driven-design/aggregates/pick-task',
                component: ComponentCreator('/domain-driven-design/aggregates/pick-task', '813'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/domain-driven-design/aggregates/putaway-task',
                component: ComponentCreator('/domain-driven-design/aggregates/putaway-task', 'd45'),
                exact: true
              },
              {
                path: '/domain-driven-design/aggregates/shipment',
                component: ComponentCreator('/domain-driven-design/aggregates/shipment', 'ef8'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/domain-driven-design/aggregates/sortation-batch',
                component: ComponentCreator('/domain-driven-design/aggregates/sortation-batch', '92b'),
                exact: true
              },
              {
                path: '/domain-driven-design/aggregates/station',
                component: ComponentCreator('/domain-driven-design/aggregates/station', 'c27'),
                exact: true
              },
              {
                path: '/domain-driven-design/aggregates/task-route',
                component: ComponentCreator('/domain-driven-design/aggregates/task-route', 'c4f'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/domain-driven-design/aggregates/walling-task',
                component: ComponentCreator('/domain-driven-design/aggregates/walling-task', '183'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/domain-driven-design/aggregates/wave',
                component: ComponentCreator('/domain-driven-design/aggregates/wave', 'a29'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/domain-driven-design/aggregates/worker',
                component: ComponentCreator('/domain-driven-design/aggregates/worker', 'bc1'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/domain-driven-design/bounded-contexts',
                component: ComponentCreator('/domain-driven-design/bounded-contexts', '19e'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/domain-driven-design/context-map',
                component: ComponentCreator('/domain-driven-design/context-map', 'b15'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/domain-driven-design/domain-events',
                component: ComponentCreator('/domain-driven-design/domain-events', '5b4'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/domain-driven-design/overview',
                component: ComponentCreator('/domain-driven-design/overview', '36a'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/domain-driven-design/value-objects',
                component: ComponentCreator('/domain-driven-design/value-objects', 'b19'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/infrastructure/kafka',
                component: ComponentCreator('/infrastructure/kafka', '461'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/infrastructure/mongodb',
                component: ComponentCreator('/infrastructure/mongodb', '184'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/infrastructure/observability',
                component: ComponentCreator('/infrastructure/observability', '4de'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/infrastructure/overview',
                component: ComponentCreator('/infrastructure/overview', 'd07'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/infrastructure/temporal',
                component: ComponentCreator('/infrastructure/temporal', 'd51'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/services/consolidation-service',
                component: ComponentCreator('/services/consolidation-service', 'b90'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/services/facility-service',
                component: ComponentCreator('/services/facility-service', '566'),
                exact: true
              },
              {
                path: '/services/inventory-service',
                component: ComponentCreator('/services/inventory-service', 'a07'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/services/labor-service',
                component: ComponentCreator('/services/labor-service', '267'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/services/orchestrator',
                component: ComponentCreator('/services/orchestrator', 'cd4'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/services/order-service',
                component: ComponentCreator('/services/order-service', '3db'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/services/packing-service',
                component: ComponentCreator('/services/packing-service', 'f29'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/services/picking-service',
                component: ComponentCreator('/services/picking-service', '902'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/services/process-path-service',
                component: ComponentCreator('/services/process-path-service', '51f'),
                exact: true
              },
              {
                path: '/services/receiving-service',
                component: ComponentCreator('/services/receiving-service', '794'),
                exact: true
              },
              {
                path: '/services/routing-service',
                component: ComponentCreator('/services/routing-service', 'a34'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/services/shipping-service',
                component: ComponentCreator('/services/shipping-service', '202'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/services/sortation-service',
                component: ComponentCreator('/services/sortation-service', 'cab'),
                exact: true
              },
              {
                path: '/services/stow-service',
                component: ComponentCreator('/services/stow-service', 'fb0'),
                exact: true
              },
              {
                path: '/services/walling-service',
                component: ComponentCreator('/services/walling-service', '2cc'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/services/waving-service',
                component: ComponentCreator('/services/waving-service', 'e88'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/services/wes-service',
                component: ComponentCreator('/services/wes-service', '1f8'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/activities/consolidation-activities',
                component: ComponentCreator('/temporal/activities/consolidation-activities', 'b59'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/activities/inventory-activities',
                component: ComponentCreator('/temporal/activities/inventory-activities', '7bd'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/activities/order-activities',
                component: ComponentCreator('/temporal/activities/order-activities', 'fea'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/activities/overview',
                component: ComponentCreator('/temporal/activities/overview', '225'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/activities/packing-activities',
                component: ComponentCreator('/temporal/activities/packing-activities', '9e9'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/activities/picking-activities',
                component: ComponentCreator('/temporal/activities/picking-activities', '7c5'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/activities/process-path-activities',
                component: ComponentCreator('/temporal/activities/process-path-activities', 'eb8'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/activities/receiving-activities',
                component: ComponentCreator('/temporal/activities/receiving-activities', '17c'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/activities/shipping-activities',
                component: ComponentCreator('/temporal/activities/shipping-activities', 'd95'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/activities/slam-activities',
                component: ComponentCreator('/temporal/activities/slam-activities', '29a'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/activities/sortation-activities',
                component: ComponentCreator('/temporal/activities/sortation-activities', '798'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/activities/unit-activities',
                component: ComponentCreator('/temporal/activities/unit-activities', '580'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/diagrams/order-flow',
                component: ComponentCreator('/temporal/diagrams/order-flow', 'fa6'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/diagrams/signal-flow',
                component: ComponentCreator('/temporal/diagrams/signal-flow', '832'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/diagrams/workflow-hierarchy',
                component: ComponentCreator('/temporal/diagrams/workflow-hierarchy', '0e4'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/overview',
                component: ComponentCreator('/temporal/overview', 'ff4'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/retry-policies',
                component: ComponentCreator('/temporal/retry-policies', 'f87'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/signals-queries',
                component: ComponentCreator('/temporal/signals-queries', 'e04'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/task-queues',
                component: ComponentCreator('/temporal/task-queues', '37d'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/cancellation',
                component: ComponentCreator('/temporal/workflows/cancellation', 'd8e'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/consolidation',
                component: ComponentCreator('/temporal/workflows/consolidation', 'd9a'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/gift-wrap',
                component: ComponentCreator('/temporal/workflows/gift-wrap', 'fe8'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/inbound-fulfillment',
                component: ComponentCreator('/temporal/workflows/inbound-fulfillment', '032'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/order-fulfillment',
                component: ComponentCreator('/temporal/workflows/order-fulfillment', 'ed4'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/packing',
                component: ComponentCreator('/temporal/workflows/packing', '3f2'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/picking',
                component: ComponentCreator('/temporal/workflows/picking', '039'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/planning',
                component: ComponentCreator('/temporal/workflows/planning', 'c68'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/reprocessing',
                component: ComponentCreator('/temporal/workflows/reprocessing', 'd5a'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/service-consolidation',
                component: ComponentCreator('/temporal/workflows/service-consolidation', 'f78'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/service-packing',
                component: ComponentCreator('/temporal/workflows/service-packing', '7f8'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/service-picking',
                component: ComponentCreator('/temporal/workflows/service-picking', '4ee'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/service-shipping',
                component: ComponentCreator('/temporal/workflows/service-shipping', '7dd'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/service-wes',
                component: ComponentCreator('/temporal/workflows/service-wes', '0b6'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/shipping',
                component: ComponentCreator('/temporal/workflows/shipping', '3cb'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/sortation',
                component: ComponentCreator('/temporal/workflows/sortation', 'ea5'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/stock-shortage',
                component: ComponentCreator('/temporal/workflows/stock-shortage', '898'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/temporal/workflows/wes-execution',
                component: ComponentCreator('/temporal/workflows/wes-execution', '802'),
                exact: true,
                sidebar: "docsSidebar"
              },
              {
                path: '/',
                component: ComponentCreator('/', '774'),
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
