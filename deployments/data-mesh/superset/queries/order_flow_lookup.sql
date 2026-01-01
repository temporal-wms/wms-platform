-- Order Flow Lookup Queries for Apache Superset
-- For Order Flow Tracker Dashboard
-- Replace {{ order_id }} with Jinja template parameter in Superset

-- ============================================
-- Query 1: Order Flow Summary
-- Returns full flow details for a specific order
-- ============================================
SELECT
    order_id,
    workflow_id,
    customer_id,
    priority,
    current_status,
    current_stage,
    -- Timestamps
    order_received_at,
    order_validated_at,
    wave_assigned_at,
    wave_id,
    picking_started_at,
    picking_completed_at,
    pick_task_id,
    picker_id,
    consolidation_started_at,
    consolidation_completed_at,
    consolidation_id,
    packing_started_at,
    packing_completed_at,
    pack_task_id,
    packer_id,
    shipping_created_at,
    shipped_at,
    shipment_id,
    tracking_number,
    carrier,
    -- Durations
    validation_duration_min,
    wave_wait_duration_min,
    picking_duration_min,
    consolidation_duration_min,
    packing_duration_min,
    shipping_duration_min,
    total_fulfillment_duration_min,
    -- Flags
    has_exceptions,
    exception_count,
    is_complete,
    is_cancelled
FROM iceberg.gold.order_flow_summary
WHERE order_id = '{{ order_id }}';


-- ============================================
-- Query 2: Visual Flow Stages
-- Returns stage statuses for visual flow diagram
-- ============================================
SELECT
    order_id,
    CASE WHEN order_received_at IS NOT NULL THEN 'COMPLETE' ELSE 'PENDING' END AS stage_1_received,
    CASE WHEN order_validated_at IS NOT NULL THEN 'COMPLETE' ELSE 'PENDING' END AS stage_2_validated,
    CASE WHEN wave_assigned_at IS NOT NULL THEN 'COMPLETE' ELSE 'PENDING' END AS stage_3_wave,
    CASE
        WHEN picking_completed_at IS NOT NULL THEN 'COMPLETE'
        WHEN picking_started_at IS NOT NULL THEN 'IN_PROGRESS'
        ELSE 'PENDING'
    END AS stage_4_picking,
    CASE
        WHEN consolidation_completed_at IS NOT NULL THEN 'COMPLETE'
        WHEN consolidation_started_at IS NOT NULL THEN 'IN_PROGRESS'
        ELSE 'PENDING'
    END AS stage_5_consolidation,
    CASE
        WHEN packing_completed_at IS NOT NULL THEN 'COMPLETE'
        WHEN packing_started_at IS NOT NULL THEN 'IN_PROGRESS'
        ELSE 'PENDING'
    END AS stage_6_packing,
    CASE
        WHEN shipped_at IS NOT NULL THEN 'COMPLETE'
        WHEN shipping_created_at IS NOT NULL THEN 'IN_PROGRESS'
        ELSE 'PENDING'
    END AS stage_7_shipped
FROM iceberg.gold.order_flow_summary
WHERE order_id = '{{ order_id }}';


-- ============================================
-- Query 3: Stage Durations for Bar Chart
-- Returns duration per stage for visualization
-- ============================================
SELECT stage, duration_minutes
FROM (
    SELECT 'Validation' AS stage, validation_duration_min AS duration_minutes, 1 AS sort_order
    FROM iceberg.gold.order_flow_summary WHERE order_id = '{{ order_id }}'
    UNION ALL
    SELECT 'Wave Wait', wave_wait_duration_min, 2
    FROM iceberg.gold.order_flow_summary WHERE order_id = '{{ order_id }}'
    UNION ALL
    SELECT 'Picking', picking_duration_min, 3
    FROM iceberg.gold.order_flow_summary WHERE order_id = '{{ order_id }}'
    UNION ALL
    SELECT 'Consolidation', consolidation_duration_min, 4
    FROM iceberg.gold.order_flow_summary WHERE order_id = '{{ order_id }}'
    UNION ALL
    SELECT 'Packing', packing_duration_min, 5
    FROM iceberg.gold.order_flow_summary WHERE order_id = '{{ order_id }}'
    UNION ALL
    SELECT 'Shipping', shipping_duration_min, 6
    FROM iceberg.gold.order_flow_summary WHERE order_id = '{{ order_id }}'
) stages
WHERE duration_minutes IS NOT NULL
ORDER BY sort_order;


-- ============================================
-- Query 4: Order Timeline Events
-- Alternative real-time query from MongoDB
-- ============================================
/*
WITH order_events AS (
    -- Order events
    SELECT
        order_id,
        'order' AS event_stage,
        CASE
            WHEN status = 'received' THEN 'wms.order.received'
            WHEN status = 'validated' THEN 'wms.order.validated'
            WHEN wave_id IS NOT NULL THEN 'wms.order.wave-assigned'
            ELSE 'wms.order.received'
        END AS event_type,
        created_at AS event_timestamp,
        status
    FROM mongodb.wms.orders
    WHERE order_id = '{{ order_id }}'

    UNION ALL

    -- Pick task events
    SELECT
        order_id,
        'picking' AS event_stage,
        CASE
            WHEN status = 'completed' THEN 'wms.picking.task-completed'
            WHEN status = 'in_progress' THEN 'wms.picking.in-progress'
            WHEN status = 'assigned' THEN 'wms.picking.task-assigned'
            ELSE 'wms.picking.task-created'
        END AS event_type,
        COALESCE(completed_at, started_at, assigned_at, created_at) AS event_timestamp,
        status
    FROM mongodb.wms.pick_tasks
    WHERE order_id = '{{ order_id }}'

    UNION ALL

    -- Consolidation events
    SELECT
        order_id,
        'consolidation' AS event_stage,
        CASE
            WHEN status = 'completed' THEN 'wms.consolidation.completed'
            WHEN status = 'in_progress' THEN 'wms.consolidation.in-progress'
            ELSE 'wms.consolidation.started'
        END AS event_type,
        COALESCE(completed_at, started_at, created_at) AS event_timestamp,
        status
    FROM mongodb.wms.consolidations
    WHERE order_id = '{{ order_id }}'

    UNION ALL

    -- Pack task events
    SELECT
        order_id,
        'packing' AS event_stage,
        CASE
            WHEN status = 'completed' THEN 'wms.packing.task-completed'
            WHEN status = 'labeled' THEN 'wms.packing.label-applied'
            WHEN status = 'packed' THEN 'wms.packing.package-sealed'
            ELSE 'wms.packing.task-created'
        END AS event_type,
        COALESCE(completed_at, labeled_at, packed_at, started_at, created_at) AS event_timestamp,
        status
    FROM mongodb.wms.pack_tasks
    WHERE order_id = '{{ order_id }}'

    UNION ALL

    -- Shipment events
    SELECT
        order_id,
        'shipping' AS event_stage,
        CASE
            WHEN status = 'shipped' THEN 'wms.shipping.confirmed'
            WHEN status = 'manifested' THEN 'wms.shipping.manifested'
            WHEN status = 'label_generated' THEN 'wms.shipping.label-generated'
            ELSE 'wms.shipping.shipment-created'
        END AS event_type,
        COALESCE(shipped_at, created_at) AS event_timestamp,
        status
    FROM mongodb.wms.shipments
    WHERE order_id = '{{ order_id }}'
)
SELECT
    order_id,
    ROW_NUMBER() OVER (ORDER BY event_timestamp) AS event_sequence,
    event_type,
    event_stage,
    event_timestamp,
    status,
    TIMESTAMPDIFF(MINUTE, LAG(event_timestamp) OVER (ORDER BY event_timestamp), event_timestamp) AS duration_from_prev_minutes
FROM order_events
ORDER BY event_timestamp ASC;
*/
