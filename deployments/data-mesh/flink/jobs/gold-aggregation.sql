-- Gold Layer Aggregation: Business Metrics and KPIs
-- This job creates curated datasets for analytics and dashboards

USE CATALOG iceberg_catalog;

-- Create Gold database
CREATE DATABASE IF NOT EXISTS gold;

-- Order Fulfillment Metrics (Daily)
CREATE TABLE IF NOT EXISTS gold.order_fulfillment_daily (
    `date` DATE,
    `priority` STRING,
    `total_orders` BIGINT,
    `completed_orders` BIGINT,
    `cancelled_orders` BIGINT,
    `in_progress_orders` BIGINT,
    `completion_rate` DOUBLE,
    `avg_fulfillment_time_hours` DOUBLE,
    `p50_fulfillment_time_hours` DOUBLE,
    `p95_fulfillment_time_hours` DOUBLE,
    `total_items` BIGINT,
    `total_revenue` DOUBLE,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`date`, `priority`) NOT ENFORCED
) PARTITIONED BY (`date`)
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Populate Order Fulfillment Metrics
INSERT INTO gold.order_fulfillment_daily
SELECT
    CAST(o.created_at AS DATE) AS `date`,
    o.priority,
    COUNT(*) AS total_orders,
    COUNT(CASE WHEN o.status = 'completed' THEN 1 END) AS completed_orders,
    COUNT(CASE WHEN o.status = 'cancelled' THEN 1 END) AS cancelled_orders,
    COUNT(CASE WHEN o.status NOT IN ('completed', 'cancelled') THEN 1 END) AS in_progress_orders,
    CAST(COUNT(CASE WHEN o.status = 'completed' THEN 1 END) AS DOUBLE) / COUNT(*) AS completion_rate,
    AVG(TIMESTAMPDIFF(HOUR, o.created_at, s.shipped_at)) AS avg_fulfillment_time_hours,
    PERCENTILE(TIMESTAMPDIFF(HOUR, o.created_at, s.shipped_at), 0.5) AS p50_fulfillment_time_hours,
    PERCENTILE(TIMESTAMPDIFF(HOUR, o.created_at, s.shipped_at), 0.95) AS p95_fulfillment_time_hours,
    SUM(CARDINALITY(o.items)) AS total_items,
    SUM(o.total_amount) AS total_revenue,
    CURRENT_TIMESTAMP AS processing_time
FROM silver.orders_current o
LEFT JOIN silver.shipments_current s ON o.order_id = s.order_id
WHERE o.is_deleted = FALSE
GROUP BY CAST(o.created_at AS DATE), o.priority;

-- Inventory Turnover Metrics
CREATE TABLE IF NOT EXISTS gold.inventory_metrics_daily (
    `date` DATE,
    `sku` STRING,
    `product_name` STRING,
    `starting_quantity` INT,
    `ending_quantity` INT,
    `quantity_received` INT,
    `quantity_picked` INT,
    `quantity_adjusted` INT,
    `turnover_rate` DOUBLE,
    `days_of_supply` INT,
    `is_low_stock` BOOLEAN,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`date`, `sku`) NOT ENFORCED
) PARTITIONED BY (`date`)
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Labor Productivity Metrics (Daily)
CREATE TABLE IF NOT EXISTS gold.labor_productivity_daily (
    `date` DATE,
    `worker_id` STRING,
    `worker_name` STRING,
    `zone` STRING,
    `shift_type` STRING,
    `tasks_completed` INT,
    `items_processed` INT,
    `total_work_hours` DOUBLE,
    `tasks_per_hour` DOUBLE,
    `items_per_hour` DOUBLE,
    `accuracy_rate` DOUBLE,
    `exceptions_count` INT,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`date`, `worker_id`) NOT ENFORCED
) PARTITIONED BY (`date`)
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Populate Labor Productivity
INSERT INTO gold.labor_productivity_daily
SELECT
    CAST(p.started_at AS DATE) AS `date`,
    p.picker_id AS worker_id,
    w.name AS worker_name,
    p.zone,
    w.shift_type,
    COUNT(*) AS tasks_completed,
    SUM(p.items_picked) AS items_processed,
    SUM(p.duration_seconds) / 3600.0 AS total_work_hours,
    COUNT(*) / (SUM(p.duration_seconds) / 3600.0) AS tasks_per_hour,
    SUM(p.items_picked) / (SUM(p.duration_seconds) / 3600.0) AS items_per_hour,
    w.accuracy_rate,
    0 AS exceptions_count,  -- TODO: Join with exceptions table
    CURRENT_TIMESTAMP AS processing_time
FROM silver.pick_tasks_enriched p
JOIN silver.workers_current w ON p.picker_id = w.worker_id
WHERE p.status = 'completed'
GROUP BY CAST(p.started_at AS DATE), p.picker_id, w.name, p.zone, w.shift_type, w.accuracy_rate;

-- Shipping Performance by Carrier
CREATE TABLE IF NOT EXISTS gold.shipping_performance_daily (
    `date` DATE,
    `carrier_code` STRING,
    `service_type` STRING,
    `shipments_count` BIGINT,
    `on_time_count` BIGINT,
    `delayed_count` BIGINT,
    `on_time_rate` DOUBLE,
    `avg_transit_days` DOUBLE,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`date`, `carrier_code`, `service_type`) NOT ENFORCED
) PARTITIONED BY (`date`)
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Wave Performance Metrics
CREATE TABLE IF NOT EXISTS gold.wave_performance_daily (
    `date` DATE,
    `wave_type` STRING,
    `waves_created` BIGINT,
    `waves_completed` BIGINT,
    `waves_cancelled` BIGINT,
    `avg_orders_per_wave` DOUBLE,
    `avg_items_per_wave` DOUBLE,
    `avg_completion_time_hours` DOUBLE,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`date`, `wave_type`) NOT ENFORCED
) PARTITIONED BY (`date`)
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Real-time Dashboard View: Active Operations
CREATE TABLE IF NOT EXISTS gold.active_operations_snapshot (
    `snapshot_time` TIMESTAMP(3),
    `active_orders` BIGINT,
    `active_waves` BIGINT,
    `active_pick_tasks` BIGINT,
    `active_pack_tasks` BIGINT,
    `pending_shipments` BIGINT,
    `active_workers` BIGINT,
    `orders_last_hour` BIGINT,
    `shipments_last_hour` BIGINT,
    PRIMARY KEY (`snapshot_time`) NOT ENFORCED
)
WITH (
    'format-version' = '2'
);

-- Hourly snapshot insert
INSERT INTO gold.active_operations_snapshot
SELECT
    CURRENT_TIMESTAMP AS snapshot_time,
    (SELECT COUNT(*) FROM silver.orders_current WHERE status NOT IN ('completed', 'cancelled')) AS active_orders,
    (SELECT COUNT(*) FROM silver.waves_current WHERE status IN ('created', 'released')) AS active_waves,
    (SELECT COUNT(*) FROM silver.pick_tasks_enriched WHERE status = 'in_progress') AS active_pick_tasks,
    (SELECT COUNT(*) FROM silver.pack_tasks_current WHERE status = 'in_progress') AS active_pack_tasks,
    (SELECT COUNT(*) FROM silver.shipments_current WHERE status = 'created') AS pending_shipments,
    (SELECT COUNT(*) FROM silver.workers_current WHERE status = 'working') AS active_workers,
    (SELECT COUNT(*) FROM silver.orders_current WHERE created_at >= CURRENT_TIMESTAMP - INTERVAL '1' HOUR) AS orders_last_hour,
    (SELECT COUNT(*) FROM silver.shipments_current WHERE shipped_at >= CURRENT_TIMESTAMP - INTERVAL '1' HOUR) AS shipments_last_hour;

-- ============================================
-- PICKING DATA PRODUCT - Gold Layer
-- ============================================

-- Picking Metrics Daily
CREATE TABLE IF NOT EXISTS gold.picking_metrics_daily (
    `date` DATE,
    `zone` STRING,
    `method` STRING,
    `total_tasks` BIGINT,
    `completed_tasks` BIGINT,
    `cancelled_tasks` BIGINT,
    `exception_tasks` BIGINT,
    `completion_rate` DOUBLE,
    `total_items_picked` BIGINT,
    `avg_items_per_task` DOUBLE,
    `avg_duration_minutes` DOUBLE,
    `avg_pick_rate` DOUBLE,
    `p50_duration_minutes` DOUBLE,
    `p95_duration_minutes` DOUBLE,
    `exception_rate` DOUBLE,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`date`, `zone`, `method`) NOT ENFORCED
) PARTITIONED BY (`date`)
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Populate Picking Metrics
INSERT INTO gold.picking_metrics_daily
SELECT
    CAST(p.created_at AS DATE) AS `date`,
    COALESCE(p.zone, 'UNKNOWN') AS zone,
    p.method,
    COUNT(*) AS total_tasks,
    COUNT(CASE WHEN p.status = 'completed' THEN 1 END) AS completed_tasks,
    COUNT(CASE WHEN p.status = 'cancelled' THEN 1 END) AS cancelled_tasks,
    COUNT(CASE WHEN p.status = 'exception' THEN 1 END) AS exception_tasks,
    CAST(COUNT(CASE WHEN p.status = 'completed' THEN 1 END) AS DOUBLE) / NULLIF(COUNT(*), 0) AS completion_rate,
    SUM(p.picked_items) AS total_items_picked,
    AVG(CAST(p.picked_items AS DOUBLE)) AS avg_items_per_task,
    AVG(p.duration_seconds / 60.0) AS avg_duration_minutes,
    AVG(p.pick_rate) AS avg_pick_rate,
    PERCENTILE(p.duration_seconds / 60.0, 0.5) AS p50_duration_minutes,
    PERCENTILE(p.duration_seconds / 60.0, 0.95) AS p95_duration_minutes,
    CAST(COUNT(CASE WHEN p.exception_count > 0 THEN 1 END) AS DOUBLE) / NULLIF(COUNT(*), 0) AS exception_rate,
    CURRENT_TIMESTAMP AS processing_time
FROM silver.pick_tasks_current p
WHERE p.is_deleted = FALSE
GROUP BY CAST(p.created_at AS DATE), COALESCE(p.zone, 'UNKNOWN'), p.method;

-- Picker Performance Daily
CREATE TABLE IF NOT EXISTS gold.picker_performance_daily (
    `date` DATE,
    `picker_id` STRING,
    `zone` STRING,
    `tasks_completed` INT,
    `items_picked` INT,
    `total_work_minutes` DOUBLE,
    `tasks_per_hour` DOUBLE,
    `items_per_hour` DOUBLE,
    `avg_pick_rate` DOUBLE,
    `exception_count` INT,
    `accuracy_rate` DOUBLE,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`date`, `picker_id`) NOT ENFORCED
) PARTITIONED BY (`date`)
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Populate Picker Performance
INSERT INTO gold.picker_performance_daily
SELECT
    CAST(p.completed_at AS DATE) AS `date`,
    p.picker_id,
    p.zone,
    COUNT(*) AS tasks_completed,
    SUM(p.picked_items) AS items_picked,
    SUM(p.duration_seconds) / 60.0 AS total_work_minutes,
    COUNT(*) / NULLIF(SUM(p.duration_seconds) / 3600.0, 0) AS tasks_per_hour,
    SUM(p.picked_items) / NULLIF(SUM(p.duration_seconds) / 3600.0, 0) AS items_per_hour,
    AVG(p.pick_rate) AS avg_pick_rate,
    SUM(p.exception_count) AS exception_count,
    1.0 - (CAST(SUM(p.exception_count) AS DOUBLE) / NULLIF(SUM(p.total_items), 0)) AS accuracy_rate,
    CURRENT_TIMESTAMP AS processing_time
FROM silver.pick_tasks_current p
WHERE p.status = 'completed'
  AND p.picker_id IS NOT NULL
  AND p.is_deleted = FALSE
GROUP BY CAST(p.completed_at AS DATE), p.picker_id, p.zone;

-- ============================================
-- CONSOLIDATION DATA PRODUCT - Gold Layer
-- ============================================

CREATE TABLE IF NOT EXISTS gold.consolidation_metrics_daily (
    `date` DATE,
    `station` STRING,
    `strategy` STRING,
    `total_units` BIGINT,
    `completed_units` BIGINT,
    `cancelled_units` BIGINT,
    `completion_rate` DOUBLE,
    `total_items_consolidated` BIGINT,
    `avg_items_per_unit` DOUBLE,
    `avg_totes_per_unit` DOUBLE,
    `avg_duration_minutes` DOUBLE,
    `p50_duration_minutes` DOUBLE,
    `p95_duration_minutes` DOUBLE,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`date`, `station`, `strategy`) NOT ENFORCED
) PARTITIONED BY (`date`)
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

INSERT INTO gold.consolidation_metrics_daily
SELECT
    CAST(c.created_at AS DATE) AS `date`,
    COALESCE(c.station, 'UNKNOWN') AS station,
    c.strategy,
    COUNT(*) AS total_units,
    COUNT(CASE WHEN c.status = 'completed' THEN 1 END) AS completed_units,
    COUNT(CASE WHEN c.status = 'cancelled' THEN 1 END) AS cancelled_units,
    CAST(COUNT(CASE WHEN c.status = 'completed' THEN 1 END) AS DOUBLE) / NULLIF(COUNT(*), 0) AS completion_rate,
    SUM(c.total_consolidated) AS total_items_consolidated,
    AVG(CAST(c.total_consolidated AS DOUBLE)) AS avg_items_per_unit,
    AVG(CAST(c.source_tote_count AS DOUBLE)) AS avg_totes_per_unit,
    AVG(c.duration_seconds / 60.0) AS avg_duration_minutes,
    PERCENTILE(c.duration_seconds / 60.0, 0.5) AS p50_duration_minutes,
    PERCENTILE(c.duration_seconds / 60.0, 0.95) AS p95_duration_minutes,
    CURRENT_TIMESTAMP AS processing_time
FROM silver.consolidations_current c
WHERE c.is_deleted = FALSE
GROUP BY CAST(c.created_at AS DATE), COALESCE(c.station, 'UNKNOWN'), c.strategy;

-- ============================================
-- PACKING DATA PRODUCT - Gold Layer
-- ============================================

CREATE TABLE IF NOT EXISTS gold.packing_metrics_daily (
    `date` DATE,
    `station` STRING,
    `package_type` STRING,
    `carrier` STRING,
    `total_tasks` BIGINT,
    `completed_tasks` BIGINT,
    `completion_rate` DOUBLE,
    `total_packages` BIGINT,
    `total_weight_kg` DOUBLE,
    `avg_weight_kg` DOUBLE,
    `avg_items_per_package` DOUBLE,
    `avg_packing_minutes` DOUBLE,
    `avg_labeling_minutes` DOUBLE,
    `avg_total_minutes` DOUBLE,
    `p50_total_minutes` DOUBLE,
    `p95_total_minutes` DOUBLE,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`date`, `station`, `package_type`, `carrier`) NOT ENFORCED
) PARTITIONED BY (`date`)
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

INSERT INTO gold.packing_metrics_daily
SELECT
    CAST(p.created_at AS DATE) AS `date`,
    COALESCE(p.station, 'UNKNOWN') AS station,
    COALESCE(p.package_type, 'UNKNOWN') AS package_type,
    COALESCE(p.carrier, 'UNKNOWN') AS carrier,
    COUNT(*) AS total_tasks,
    COUNT(CASE WHEN p.status = 'completed' THEN 1 END) AS completed_tasks,
    CAST(COUNT(CASE WHEN p.status = 'completed' THEN 1 END) AS DOUBLE) / NULLIF(COUNT(*), 0) AS completion_rate,
    COUNT(CASE WHEN p.package_sealed = TRUE THEN 1 END) AS total_packages,
    SUM(p.total_weight) AS total_weight_kg,
    AVG(p.total_weight) AS avg_weight_kg,
    AVG(CAST(p.item_count AS DOUBLE)) AS avg_items_per_package,
    AVG(p.packing_duration_seconds / 60.0) AS avg_packing_minutes,
    AVG(p.labeling_duration_seconds / 60.0) AS avg_labeling_minutes,
    AVG(p.total_duration_seconds / 60.0) AS avg_total_minutes,
    PERCENTILE(p.total_duration_seconds / 60.0, 0.5) AS p50_total_minutes,
    PERCENTILE(p.total_duration_seconds / 60.0, 0.95) AS p95_total_minutes,
    CURRENT_TIMESTAMP AS processing_time
FROM silver.pack_tasks_current p
WHERE p.is_deleted = FALSE
GROUP BY CAST(p.created_at AS DATE), COALESCE(p.station, 'UNKNOWN'),
         COALESCE(p.package_type, 'UNKNOWN'), COALESCE(p.carrier, 'UNKNOWN');

-- Packer Performance Daily
CREATE TABLE IF NOT EXISTS gold.packer_performance_daily (
    `date` DATE,
    `packer_id` STRING,
    `station` STRING,
    `tasks_completed` INT,
    `packages_sealed` INT,
    `total_work_minutes` DOUBLE,
    `packages_per_hour` DOUBLE,
    `avg_pack_time_minutes` DOUBLE,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`date`, `packer_id`) NOT ENFORCED
) PARTITIONED BY (`date`)
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

INSERT INTO gold.packer_performance_daily
SELECT
    CAST(p.completed_at AS DATE) AS `date`,
    p.packer_id,
    p.station,
    COUNT(*) AS tasks_completed,
    COUNT(CASE WHEN p.package_sealed = TRUE THEN 1 END) AS packages_sealed,
    SUM(p.total_duration_seconds) / 60.0 AS total_work_minutes,
    COUNT(*) / NULLIF(SUM(p.total_duration_seconds) / 3600.0, 0) AS packages_per_hour,
    AVG(p.total_duration_seconds / 60.0) AS avg_pack_time_minutes,
    CURRENT_TIMESTAMP AS processing_time
FROM silver.pack_tasks_current p
WHERE p.status = 'completed'
  AND p.packer_id IS NOT NULL
  AND p.is_deleted = FALSE
GROUP BY CAST(p.completed_at AS DATE), p.packer_id, p.station;

-- ============================================
-- ORDERS BY REQUIREMENTS - Gold Layer
-- For Orders Dashboard Bar Chart
-- ============================================

CREATE TABLE IF NOT EXISTS gold.orders_by_requirements_daily (
    `date` DATE,
    `requirement` STRING,
    `order_count` BIGINT,
    `percentage_of_total` DOUBLE,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`date`, `requirement`) NOT ENFORCED
) PARTITIONED BY (`date`)
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Note: This query assumes orders have a process_requirements array field
-- The actual implementation depends on how requirements are stored in silver.orders_current
-- This is a simplified version - adjust JSON parsing based on actual schema
INSERT INTO gold.orders_by_requirements_daily
WITH daily_totals AS (
    SELECT
        CAST(o.created_at AS DATE) AS order_date,
        COUNT(*) AS total_orders
    FROM silver.orders_current o
    WHERE o.is_deleted = FALSE
    GROUP BY CAST(o.created_at AS DATE)
),
requirement_counts AS (
    -- Gift Wrap orders
    SELECT
        CAST(o.created_at AS DATE) AS order_date,
        'gift_wrap' AS requirement,
        COUNT(*) AS order_count
    FROM silver.orders_current o
    WHERE o.is_deleted = FALSE
      AND JSON_VALUE(o.gift_wrap, '$') = 'true'
    GROUP BY CAST(o.created_at AS DATE)

    UNION ALL

    -- Multi-item orders (more than 1 item type or quantity > 1)
    SELECT
        CAST(o.created_at AS DATE) AS order_date,
        'multi_item' AS requirement,
        COUNT(*) AS order_count
    FROM silver.orders_current o
    WHERE o.is_deleted = FALSE
      AND CARDINALITY(o.items) > 1
    GROUP BY CAST(o.created_at AS DATE)

    UNION ALL

    -- Single-item orders
    SELECT
        CAST(o.created_at AS DATE) AS order_date,
        'single_item' AS requirement,
        COUNT(*) AS order_count
    FROM silver.orders_current o
    WHERE o.is_deleted = FALSE
      AND CARDINALITY(o.items) = 1
    GROUP BY CAST(o.created_at AS DATE)
)
SELECT
    r.order_date AS `date`,
    r.requirement,
    r.order_count,
    CAST(r.order_count AS DOUBLE) / dt.total_orders * 100 AS percentage_of_total,
    CURRENT_TIMESTAMP AS processing_time
FROM requirement_counts r
JOIN daily_totals dt ON r.order_date = dt.order_date;

-- ============================================
-- ORDER FLOW TRACKING - Gold Layer
-- For Order Flow Tracker Dashboard
-- ============================================

CREATE TABLE IF NOT EXISTS gold.order_flow_summary (
    `order_id` STRING,
    `workflow_id` STRING,
    `customer_id` STRING,
    `priority` STRING,
    `current_status` STRING,
    `current_stage` STRING,
    -- Stage timestamps
    `order_received_at` TIMESTAMP(3),
    `order_validated_at` TIMESTAMP(3),
    `wave_assigned_at` TIMESTAMP(3),
    `wave_id` STRING,
    `picking_started_at` TIMESTAMP(3),
    `picking_completed_at` TIMESTAMP(3),
    `pick_task_id` STRING,
    `picker_id` STRING,
    `consolidation_started_at` TIMESTAMP(3),
    `consolidation_completed_at` TIMESTAMP(3),
    `consolidation_id` STRING,
    `packing_started_at` TIMESTAMP(3),
    `packing_completed_at` TIMESTAMP(3),
    `pack_task_id` STRING,
    `packer_id` STRING,
    `shipping_created_at` TIMESTAMP(3),
    `shipped_at` TIMESTAMP(3),
    `shipment_id` STRING,
    `tracking_number` STRING,
    `carrier` STRING,
    -- Durations (minutes)
    `validation_duration_min` DOUBLE,
    `wave_wait_duration_min` DOUBLE,
    `picking_duration_min` DOUBLE,
    `consolidation_duration_min` DOUBLE,
    `packing_duration_min` DOUBLE,
    `shipping_duration_min` DOUBLE,
    `total_fulfillment_duration_min` DOUBLE,
    -- Flags
    `has_exceptions` BOOLEAN,
    `exception_count` INT,
    `is_complete` BOOLEAN,
    `is_cancelled` BOOLEAN,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`order_id`) NOT ENFORCED
) PARTITIONED BY (days(`order_received_at`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Populate Order Flow Summary by joining all Silver tables
INSERT INTO gold.order_flow_summary
SELECT
    o.order_id,
    CONCAT('order-fulfillment-', o.order_id) AS workflow_id,
    o.customer_id,
    o.priority,
    o.status AS current_status,
    CASE
        WHEN s.shipped_at IS NOT NULL THEN 'shipped'
        WHEN pk.completed_at IS NOT NULL THEN 'packing_complete'
        WHEN pk.started_at IS NOT NULL THEN 'packing'
        WHEN c.completed_at IS NOT NULL THEN 'consolidation_complete'
        WHEN c.started_at IS NOT NULL THEN 'consolidating'
        WHEN p.completed_at IS NOT NULL THEN 'picking_complete'
        WHEN p.started_at IS NOT NULL THEN 'picking'
        WHEN o.wave_id IS NOT NULL THEN 'wave_assigned'
        WHEN o.status = 'validated' THEN 'validated'
        ELSE 'received'
    END AS current_stage,
    o.created_at AS order_received_at,
    CASE WHEN o.status != 'received' THEN o.updated_at ELSE NULL END AS order_validated_at,
    CASE WHEN o.wave_id IS NOT NULL THEN o.updated_at ELSE NULL END AS wave_assigned_at,
    o.wave_id,
    p.started_at AS picking_started_at,
    p.completed_at AS picking_completed_at,
    p.task_id AS pick_task_id,
    p.picker_id,
    c.started_at AS consolidation_started_at,
    c.completed_at AS consolidation_completed_at,
    c.consolidation_id,
    pk.started_at AS packing_started_at,
    pk.completed_at AS packing_completed_at,
    pk.task_id AS pack_task_id,
    pk.packer_id,
    s.created_at AS shipping_created_at,
    s.shipped_at,
    s.shipment_id,
    s.tracking_number,
    s.carrier_code AS carrier,
    -- Duration calculations (in minutes)
    TIMESTAMPDIFF(MINUTE, o.created_at, o.updated_at) AS validation_duration_min,
    TIMESTAMPDIFF(MINUTE, o.updated_at, p.started_at) AS wave_wait_duration_min,
    TIMESTAMPDIFF(MINUTE, p.started_at, p.completed_at) AS picking_duration_min,
    TIMESTAMPDIFF(MINUTE, c.started_at, c.completed_at) AS consolidation_duration_min,
    TIMESTAMPDIFF(MINUTE, pk.started_at, pk.completed_at) AS packing_duration_min,
    TIMESTAMPDIFF(MINUTE, s.created_at, s.shipped_at) AS shipping_duration_min,
    TIMESTAMPDIFF(MINUTE, o.created_at, COALESCE(s.shipped_at, CURRENT_TIMESTAMP)) AS total_fulfillment_duration_min,
    COALESCE(p.exception_count > 0, FALSE) AS has_exceptions,
    COALESCE(p.exception_count, 0) AS exception_count,
    s.shipped_at IS NOT NULL AS is_complete,
    o.status = 'cancelled' AS is_cancelled,
    CURRENT_TIMESTAMP AS processing_time
FROM silver.orders_current o
LEFT JOIN silver.pick_tasks_current p ON o.order_id = p.order_id
LEFT JOIN silver.consolidations_current c ON o.order_id = c.order_id
LEFT JOIN silver.pack_tasks_current pk ON o.order_id = pk.order_id
LEFT JOIN silver.shipments_current s ON o.order_id = s.order_id
WHERE o.is_deleted = FALSE;

-- ============================================
-- ROUTING DATA PRODUCT - Gold Layer
-- ============================================

-- Route Performance by Zone (Daily)
CREATE TABLE IF NOT EXISTS gold.route_performance_by_zone_daily (
    `date` DATE,
    `zone` STRING,
    `total_routes` BIGINT,
    `completed_routes` BIGINT,
    `cancelled_routes` BIGINT,
    `in_progress_routes` BIGINT,
    `completion_rate` DOUBLE,
    `total_items_picked` BIGINT,
    `avg_items_per_route` DOUBLE,
    `avg_stops_per_route` DOUBLE,
    `total_distance_m` DOUBLE,
    `avg_distance_per_route_m` DOUBLE,
    `total_duration_seconds` BIGINT,
    `avg_duration_minutes` DOUBLE,
    `p50_duration_minutes` DOUBLE,
    `p95_duration_minutes` DOUBLE,
    `avg_pick_rate` DOUBLE,
    `unique_pickers` BIGINT,
    `routes_per_picker` DOUBLE,
    `avg_efficiency_ratio` DOUBLE,
    `multi_route_count` BIGINT,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`date`, `zone`) NOT ENFORCED
) PARTITIONED BY (`date`)
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Populate Route Performance by Zone
INSERT INTO gold.route_performance_by_zone_daily
SELECT
    CAST(r.created_at AS DATE) AS `date`,
    COALESCE(r.zone, 'UNKNOWN') AS zone,
    COUNT(*) AS total_routes,
    COUNT(CASE WHEN r.status = 'completed' THEN 1 END) AS completed_routes,
    COUNT(CASE WHEN r.status = 'cancelled' THEN 1 END) AS cancelled_routes,
    COUNT(CASE WHEN r.status = 'in_progress' THEN 1 END) AS in_progress_routes,
    CAST(COUNT(CASE WHEN r.status = 'completed' THEN 1 END) AS DOUBLE) / NULLIF(COUNT(*), 0) AS completion_rate,
    SUM(r.picked_items) AS total_items_picked,
    AVG(CAST(r.total_items AS DOUBLE)) AS avg_items_per_route,
    AVG(CAST(r.stop_count AS DOUBLE)) AS avg_stops_per_route,
    SUM(r.actual_distance_m) AS total_distance_m,
    AVG(r.actual_distance_m) AS avg_distance_per_route_m,
    SUM(r.duration_seconds) AS total_duration_seconds,
    AVG(r.duration_seconds / 60.0) AS avg_duration_minutes,
    PERCENTILE(r.duration_seconds / 60.0, 0.5) AS p50_duration_minutes,
    PERCENTILE(r.duration_seconds / 60.0, 0.95) AS p95_duration_minutes,
    AVG(r.pick_rate) AS avg_pick_rate,
    COUNT(DISTINCT r.picker_id) AS unique_pickers,
    CAST(COUNT(*) AS DOUBLE) / NULLIF(COUNT(DISTINCT r.picker_id), 0) AS routes_per_picker,
    AVG(r.efficiency_ratio) AS avg_efficiency_ratio,
    COUNT(CASE WHEN r.is_multi_route = TRUE THEN 1 END) AS multi_route_count,
    CURRENT_TIMESTAMP AS processing_time
FROM silver.routes_current r
WHERE r.is_deleted = FALSE
GROUP BY CAST(r.created_at AS DATE), COALESCE(r.zone, 'UNKNOWN');

-- Picker Route Performance (Daily)
CREATE TABLE IF NOT EXISTS gold.picker_route_performance_daily (
    `date` DATE,
    `picker_id` STRING,
    `zone` STRING,
    `routes_completed` INT,
    `total_items_picked` INT,
    `total_stops_completed` INT,
    `total_distance_m` DOUBLE,
    `total_work_minutes` DOUBLE,
    `routes_per_hour` DOUBLE,
    `items_per_hour` DOUBLE,
    `avg_route_duration_minutes` DOUBLE,
    `avg_pick_rate` DOUBLE,
    `avg_efficiency_ratio` DOUBLE,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`date`, `picker_id`) NOT ENFORCED
) PARTITIONED BY (`date`)
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Populate Picker Route Performance
INSERT INTO gold.picker_route_performance_daily
SELECT
    CAST(r.completed_at AS DATE) AS `date`,
    r.picker_id,
    r.zone,
    COUNT(*) AS routes_completed,
    SUM(r.picked_items) AS total_items_picked,
    SUM(r.stop_count) AS total_stops_completed,
    SUM(r.actual_distance_m) AS total_distance_m,
    SUM(r.duration_seconds) / 60.0 AS total_work_minutes,
    COUNT(*) / NULLIF(SUM(r.duration_seconds) / 3600.0, 0) AS routes_per_hour,
    SUM(r.picked_items) / NULLIF(SUM(r.duration_seconds) / 3600.0, 0) AS items_per_hour,
    AVG(r.duration_seconds / 60.0) AS avg_route_duration_minutes,
    AVG(r.pick_rate) AS avg_pick_rate,
    AVG(r.efficiency_ratio) AS avg_efficiency_ratio,
    CURRENT_TIMESTAMP AS processing_time
FROM silver.routes_current r
WHERE r.status = 'completed'
  AND r.picker_id IS NOT NULL
  AND r.is_deleted = FALSE
GROUP BY CAST(r.completed_at AS DATE), r.picker_id, r.zone;

-- Route Optimization Metrics (Daily)
CREATE TABLE IF NOT EXISTS gold.route_optimization_daily (
    `date` DATE,
    `route_complexity` STRING,
    `total_routes` BIGINT,
    `avg_stops` DOUBLE,
    `avg_zones_visited` DOUBLE,
    `avg_distance_m` DOUBLE,
    `avg_estimated_time_min` DOUBLE,
    `avg_actual_time_min` DOUBLE,
    `avg_efficiency_ratio` DOUBLE,
    `routes_exceeding_time_estimate` BIGINT,
    `routes_with_multi_zone` BIGINT,
    `avg_items_per_stop` DOUBLE,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`date`, `route_complexity`) NOT ENFORCED
) PARTITIONED BY (`date`)
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Populate Route Optimization Metrics
INSERT INTO gold.route_optimization_daily
SELECT
    CAST(r.created_at AS DATE) AS `date`,
    CASE
        WHEN r.stop_count <= 5 AND r.zone_count = 1 THEN 'simple'
        WHEN r.stop_count <= 15 AND r.zone_count <= 2 THEN 'moderate'
        ELSE 'complex'
    END AS route_complexity,
    COUNT(*) AS total_routes,
    AVG(CAST(r.stop_count AS DOUBLE)) AS avg_stops,
    AVG(CAST(r.zone_count AS DOUBLE)) AS avg_zones_visited,
    AVG(r.estimated_distance_m) AS avg_distance_m,
    AVG(r.estimated_time_seconds / 60.0) AS avg_estimated_time_min,
    AVG(r.actual_time_seconds / 60.0) AS avg_actual_time_min,
    AVG(r.efficiency_ratio) AS avg_efficiency_ratio,
    COUNT(CASE WHEN r.efficiency_ratio > 1.2 THEN 1 END) AS routes_exceeding_time_estimate,
    COUNT(CASE WHEN r.zone_count > 1 THEN 1 END) AS routes_with_multi_zone,
    AVG(CAST(r.total_items AS DOUBLE) / NULLIF(r.stop_count, 0)) AS avg_items_per_stop,
    CURRENT_TIMESTAMP AS processing_time
FROM silver.routes_current r
WHERE r.is_deleted = FALSE
GROUP BY CAST(r.created_at AS DATE),
    CASE
        WHEN r.stop_count <= 5 AND r.zone_count = 1 THEN 'simple'
        WHEN r.stop_count <= 15 AND r.zone_count <= 2 THEN 'moderate'
        ELSE 'complex'
    END;

-- Individual Route Optimization Candidates
CREATE TABLE IF NOT EXISTS gold.route_optimization_candidates (
    `route_id` STRING,
    `order_id` STRING,
    `date` DATE,
    `zone` STRING,
    `strategy` STRING,
    `stop_count` INT,
    `zone_count` INT,
    `estimated_distance_m` DOUBLE,
    `actual_distance_m` DOUBLE,
    `estimated_time_min` DOUBLE,
    `actual_time_min` DOUBLE,
    `efficiency_ratio` DOUBLE,
    `distance_accuracy` DOUBLE,
    `items_per_stop` DOUBLE,
    `has_high_zone_count` BOOLEAN,
    `is_inefficient` BOOLEAN,
    `is_very_inefficient` BOOLEAN,
    `is_long_distance` BOOLEAN,
    `has_many_stops` BOOLEAN,
    `distance_exceeded` BOOLEAN,
    `optimization_score` INT,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`route_id`) NOT ENFORCED
) PARTITIONED BY (`date`)
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Populate Route Optimization Candidates
INSERT INTO gold.route_optimization_candidates
SELECT
    r.route_id,
    r.order_id,
    CAST(r.created_at AS DATE) AS `date`,
    r.zone,
    r.strategy,
    r.stop_count,
    r.zone_count,
    r.estimated_distance_m,
    r.actual_distance_m,
    r.estimated_time_seconds / 60.0 AS estimated_time_min,
    r.actual_time_seconds / 60.0 AS actual_time_min,
    r.efficiency_ratio,
    r.distance_accuracy,
    CAST(r.total_items AS DOUBLE) / NULLIF(r.stop_count, 0) AS items_per_stop,
    -- Optimization flags
    r.zone_count > 2 AS has_high_zone_count,
    r.efficiency_ratio > 1.2 AS is_inefficient,
    r.efficiency_ratio > 1.5 AS is_very_inefficient,
    r.estimated_distance_m > 500 AS is_long_distance,
    r.stop_count > 20 AS has_many_stops,
    r.distance_accuracy > 1.3 AS distance_exceeded,
    -- Calculate optimization score
    (CASE WHEN r.zone_count > 2 THEN 3 ELSE 0 END) +
    (CASE WHEN r.efficiency_ratio > 1.5 THEN 3 WHEN r.efficiency_ratio > 1.2 THEN 1 ELSE 0 END) +
    (CASE WHEN r.estimated_distance_m > 500 THEN 2 ELSE 0 END) +
    (CASE WHEN r.stop_count > 20 THEN 2 ELSE 0 END) AS optimization_score,
    CURRENT_TIMESTAMP AS processing_time
FROM silver.routes_current r
WHERE r.status = 'completed'
  AND r.is_deleted = FALSE;
