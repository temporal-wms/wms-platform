import type { SidebarsConfig } from "@docusaurus/plugin-content-docs";

const sidebar: SidebarsConfig = {
  apisidebar: [
    {
      type: "doc",
      id: "api/order-service/order-service-api",
    },
    {
      type: "category",
      label: "Orders",
      items: [
        {
          type: "doc",
          id: "api/order-service/create-a-new-order",
          label: "Create a new order",
          className: "api-method post",
        },
        {
          type: "doc",
          id: "api/order-service/list-orders",
          label: "List orders",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api/order-service/get-order-by-id",
          label: "Get order by ID",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api/order-service/validate-order",
          label: "Validate order",
          className: "api-method put",
        },
        {
          type: "doc",
          id: "api/order-service/cancel-order",
          label: "Cancel order",
          className: "api-method put",
        },
      ],
    },
    {
      type: "category",
      label: "Reprocessing",
      items: [
        {
          type: "doc",
          id: "api/order-service/get-orders-eligible-for-retry",
          label: "Get orders eligible for retry",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api/order-service/get-retry-metadata-for-order",
          label: "Get retry metadata for order",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api/order-service/increment-retry-count",
          label: "Increment retry count",
          className: "api-method post",
        },
        {
          type: "doc",
          id: "api/order-service/reset-order-for-retry",
          label: "Reset order for retry",
          className: "api-method post",
        },
        {
          type: "doc",
          id: "api/order-service/move-order-to-dead-letter-queue",
          label: "Move order to dead letter queue",
          className: "api-method post",
        },
      ],
    },
    {
      type: "category",
      label: "Dead Letter Queue",
      items: [
        {
          type: "doc",
          id: "api/order-service/list-dead-letter-queue-entries",
          label: "List dead letter queue entries",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api/order-service/get-dlq-statistics",
          label: "Get DLQ statistics",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api/order-service/get-specific-dlq-entry",
          label: "Get specific DLQ entry",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "api/order-service/resolve-dlq-entry",
          label: "Resolve DLQ entry",
          className: "api-method patch",
        },
      ],
    },
    {
      type: "category",
      label: "Health",
      items: [
        {
          type: "doc",
          id: "api/order-service/health-check",
          label: "Health check",
          className: "api-method get",
        },
      ],
    },
  ],
};

export default sidebar.apisidebar;
