/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  docsSidebar: [
    'intro',
    {
      type: 'category',
      label: 'Architecture',
      collapsed: false,
      items: [
        'architecture/overview',
        {
          type: 'category',
          label: 'C4 Diagrams',
          items: [
            'architecture/c4-diagrams/context',
            'architecture/c4-diagrams/containers',
            'architecture/c4-diagrams/components',
            'architecture/c4-diagrams/code',
          ],
        },
        {
          type: 'category',
          label: 'System Diagrams',
          items: [
            'architecture/system-diagrams/infrastructure',
            'architecture/system-diagrams/deployment',
            'architecture/system-diagrams/data-flow',
          ],
        },
        {
          type: 'category',
          label: 'Sequence Diagrams',
          items: [
            'architecture/sequence-diagrams/order-fulfillment',
            'architecture/sequence-diagrams/order-cancellation',
            'architecture/sequence-diagrams/wes-execution',
            'architecture/sequence-diagrams/walling-workflow',
            'architecture/sequence-diagrams/picking-workflow',
            'architecture/sequence-diagrams/packing-workflow',
            'architecture/sequence-diagrams/shipping-workflow',
          ],
        },
      ],
    },
    {
      type: 'category',
      label: 'Domain-Driven Design',
      collapsed: false,
      items: [
        'domain-driven-design/overview',
        'domain-driven-design/bounded-contexts',
        'domain-driven-design/context-map',
        {
          type: 'category',
          label: 'Aggregates',
          items: [
            'domain-driven-design/aggregates/order',
            'domain-driven-design/aggregates/wave',
            'domain-driven-design/aggregates/task-route',
            'domain-driven-design/aggregates/walling-task',
            'domain-driven-design/aggregates/pick-task',
            'domain-driven-design/aggregates/pick-route',
            'domain-driven-design/aggregates/consolidation-unit',
            'domain-driven-design/aggregates/pack-task',
            'domain-driven-design/aggregates/shipment',
            'domain-driven-design/aggregates/inventory-item',
            'domain-driven-design/aggregates/worker',
          ],
        },
        'domain-driven-design/domain-events',
        'domain-driven-design/value-objects',
      ],
    },
    {
      type: 'category',
      label: 'Services',
      collapsed: false,
      items: [
        'services/order-service',
        'services/waving-service',
        'services/wes-service',
        'services/walling-service',
        'services/routing-service',
        'services/picking-service',
        'services/consolidation-service',
        'services/packing-service',
        'services/shipping-service',
        'services/inventory-service',
        'services/labor-service',
        'services/orchestrator',
      ],
    },
    {
      type: 'category',
      label: 'API Reference',
      items: [
        'api/rest-api',
        'api/events-api',
      ],
    },
    {
      type: 'category',
      label: 'Infrastructure',
      items: [
        'infrastructure/overview',
        'infrastructure/mongodb',
        'infrastructure/kafka',
        'infrastructure/temporal',
        'infrastructure/observability',
      ],
    },
  ],
};

export default sidebars;
