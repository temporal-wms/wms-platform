-- Silver and Gold Layer Transformations for New Data Products
-- Aggregations and metrics for analytics dashboards

-- ============================================
-- GOLD LAYER: BILLING METRICS
-- ============================================

-- Create Gold database if not exists
CREATE DATABASE IF NOT EXISTS iceberg_catalog.gold;

-- Daily billing metrics by seller and activity type
CREATE TABLE IF NOT EXISTS iceberg_catalog.gold.billing_metrics_daily (
    `date` DATE,
    `seller_id` STRING,
    `seller_name` STRING,
    `activity_type` STRING,
    `total_activities` BIGINT,
    `total_quantity` DOUBLE,
    `total_revenue` DOUBLE,
    `avg_unit_price` DOUBLE,
    `invoiced_amount` DOUBLE,
    `uninvoiced_amount` DOUBLE,
    PRIMARY KEY (`date`, `seller_id`, `activity_type`) NOT ENFORCED
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Populate billing metrics
INSERT INTO iceberg_catalog.gold.billing_metrics_daily
SELECT
    CAST(activity_date AS DATE) AS `date`,
    seller_id,
    MAX(seller_id) AS seller_name,  -- Will be enriched from sellers table
    type AS activity_type,
    COUNT(*) AS total_activities,
    SUM(quantity) AS total_quantity,
    SUM(amount) AS total_revenue,
    AVG(unit_price) AS avg_unit_price,
    SUM(CASE WHEN invoiced = true THEN amount ELSE 0 END) AS invoiced_amount,
    SUM(CASE WHEN invoiced = false THEN amount ELSE 0 END) AS uninvoiced_amount
FROM iceberg_catalog.bronze.billing_activities_raw
WHERE cdc_operation != 'd'
GROUP BY CAST(activity_date AS DATE), seller_id, type;

-- Invoice summary metrics
CREATE TABLE IF NOT EXISTS iceberg_catalog.gold.invoice_metrics_daily (
    `date` DATE,
    `status` STRING,
    `total_invoices` BIGINT,
    `total_amount` DOUBLE,
    `avg_invoice_amount` DOUBLE,
    `overdue_count` BIGINT,
    `paid_count` BIGINT,
    PRIMARY KEY (`date`, `status`) NOT ENFORCED
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- ============================================
-- GOLD LAYER: SELLER PERFORMANCE
-- ============================================

-- Daily seller performance metrics
CREATE TABLE IF NOT EXISTS iceberg_catalog.gold.seller_performance_daily (
    `date` DATE,
    `seller_id` STRING,
    `seller_name` STRING,
    `status` STRING,
    `tier` STRING,
    `total_orders` BIGINT,
    `total_revenue` DOUBLE,
    `avg_order_value` DOUBLE,
    `storage_fees` DOUBLE,
    `fulfillment_fees` DOUBLE,
    `channels_connected` INT,
    PRIMARY KEY (`date`, `seller_id`) NOT ENFORCED
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Current sellers snapshot
CREATE TABLE IF NOT EXISTS iceberg_catalog.silver.sellers_current (
    `seller_id` STRING,
    `tenant_id` STRING,
    `name` STRING,
    `email` STRING,
    `phone` STRING,
    `status` STRING,
    `tier` STRING,
    `contract_start_date` TIMESTAMP(3),
    `contract_end_date` TIMESTAMP(3),
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    PRIMARY KEY (`seller_id`) NOT ENFORCED
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Populate sellers current
INSERT INTO iceberg_catalog.silver.sellers_current
SELECT
    seller_id,
    tenant_id,
    name,
    email,
    phone,
    status,
    tier,
    contract_start_date,
    contract_end_date,
    created_at,
    updated_at
FROM iceberg_catalog.bronze.sellers_raw
WHERE cdc_operation != 'd';

-- ============================================
-- GOLD LAYER: CHANNEL METRICS
-- ============================================

-- Daily channel sync metrics
CREATE TABLE IF NOT EXISTS iceberg_catalog.gold.channel_sync_metrics_daily (
    `date` DATE,
    `platform` STRING,
    `total_channels` BIGINT,
    `connected_channels` BIGINT,
    `error_channels` BIGINT,
    `total_orders_imported` BIGINT,
    `total_products_synced` BIGINT,
    `sync_success_rate` DOUBLE,
    PRIMARY KEY (`date`, `platform`) NOT ENFORCED
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Current channels snapshot
CREATE TABLE IF NOT EXISTS iceberg_catalog.silver.channels_current (
    `channel_id` STRING,
    `seller_id` STRING,
    `tenant_id` STRING,
    `platform` STRING,
    `store_name` STRING,
    `status` STRING,
    `last_sync_at` TIMESTAMP(3),
    `last_sync_status` STRING,
    `orders_imported` BIGINT,
    `products_synced` BIGINT,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    PRIMARY KEY (`channel_id`) NOT ENFORCED
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Populate channels current
INSERT INTO iceberg_catalog.silver.channels_current
SELECT
    channel_id,
    seller_id,
    tenant_id,
    platform,
    store_name,
    status,
    last_sync_at,
    last_sync_status,
    orders_imported,
    products_synced,
    created_at,
    updated_at
FROM iceberg_catalog.bronze.channels_raw
WHERE cdc_operation != 'd';

-- ============================================
-- GOLD LAYER: FACILITY UTILIZATION
-- ============================================

-- Daily facility utilization metrics
CREATE TABLE IF NOT EXISTS iceberg_catalog.gold.facility_utilization_daily (
    `date` DATE,
    `facility_id` STRING,
    `zone` STRING,
    `station_type` STRING,
    `total_stations` BIGINT,
    `active_stations` BIGINT,
    `maintenance_stations` BIGINT,
    `avg_utilization_pct` DOUBLE,
    `total_tasks_processed` BIGINT,
    `tasks_per_station` DOUBLE,
    PRIMARY KEY (`date`, `facility_id`, `zone`, `station_type`) NOT ENFORCED
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Current stations snapshot
CREATE TABLE IF NOT EXISTS iceberg_catalog.silver.stations_current (
    `station_id` STRING,
    `tenant_id` STRING,
    `facility_id` STRING,
    `warehouse_id` STRING,
    `name` STRING,
    `zone` STRING,
    `station_type` STRING,
    `status` STRING,
    `capabilities` STRING,
    `max_concurrent_tasks` INT,
    `current_tasks` INT,
    `assigned_worker_id` STRING,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    PRIMARY KEY (`station_id`) NOT ENFORCED
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Populate stations current
INSERT INTO iceberg_catalog.silver.stations_current
SELECT
    station_id,
    tenant_id,
    facility_id,
    warehouse_id,
    name,
    zone,
    station_type,
    status,
    capabilities,
    max_concurrent_tasks,
    current_tasks,
    assigned_worker_id,
    created_at,
    updated_at
FROM iceberg_catalog.bronze.stations_raw
WHERE cdc_operation != 'd';

-- ============================================
-- GOLD LAYER: SORTATION METRICS
-- ============================================

-- Daily sortation performance metrics
CREATE TABLE IF NOT EXISTS iceberg_catalog.gold.sortation_metrics_daily (
    `date` DATE,
    `sortation_center` STRING,
    `carrier_id` STRING,
    `total_batches` BIGINT,
    `dispatched_batches` BIGINT,
    `total_packages` BIGINT,
    `sorted_packages` BIGINT,
    `total_weight` DOUBLE,
    `sort_completion_rate` DOUBLE,
    `avg_packages_per_batch` DOUBLE,
    `avg_sort_time_minutes` DOUBLE,
    PRIMARY KEY (`date`, `sortation_center`, `carrier_id`) NOT ENFORCED
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Current sortation batches snapshot
CREATE TABLE IF NOT EXISTS iceberg_catalog.silver.sortation_batches_current (
    `batch_id` STRING,
    `sortation_center` STRING,
    `destination_group` STRING,
    `carrier_id` STRING,
    `status` STRING,
    `total_packages` INT,
    `sorted_count` INT,
    `total_weight` DOUBLE,
    `trailer_id` STRING,
    `dispatch_dock` STRING,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `dispatched_at` TIMESTAMP(3),
    PRIMARY KEY (`batch_id`) NOT ENFORCED
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Populate sortation batches current
INSERT INTO iceberg_catalog.silver.sortation_batches_current
SELECT
    batch_id,
    sortation_center,
    destination_group,
    carrier_id,
    status,
    total_packages,
    sorted_count,
    total_weight,
    trailer_id,
    dispatch_dock,
    created_at,
    updated_at,
    dispatched_at
FROM iceberg_catalog.bronze.sortation_batches_raw
WHERE cdc_operation != 'd';

-- ============================================
-- GOLD LAYER: WALLING METRICS
-- ============================================

-- Daily walling performance metrics
CREATE TABLE IF NOT EXISTS iceberg_catalog.gold.walling_metrics_daily (
    `date` DATE,
    `wall_id` STRING,
    `zone` STRING,
    `total_assignments` BIGINT,
    `completed_assignments` BIGINT,
    `total_items_expected` BIGINT,
    `total_items_placed` BIGINT,
    `completion_rate` DOUBLE,
    `avg_items_per_assignment` DOUBLE,
    PRIMARY KEY (`date`, `wall_id`, `zone`) NOT ENFORCED
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- ============================================
-- GOLD LAYER: UNIT TRACKING
-- ============================================

-- Daily unit movement metrics
CREATE TABLE IF NOT EXISTS iceberg_catalog.gold.unit_movement_daily (
    `date` DATE,
    `status` STRING,
    `total_units` BIGINT,
    `units_received` BIGINT,
    `units_shipped` BIGINT,
    `units_damaged` BIGINT,
    PRIMARY KEY (`date`, `status`) NOT ENFORCED
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Current units snapshot
CREATE TABLE IF NOT EXISTS iceberg_catalog.silver.units_current (
    `unit_id` STRING,
    `sku` STRING,
    `seller_id` STRING,
    `tenant_id` STRING,
    `location` STRING,
    `status` STRING,
    `condition` STRING,
    `last_movement_at` TIMESTAMP(3),
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    PRIMARY KEY (`unit_id`) NOT ENFORCED
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Populate units current
INSERT INTO iceberg_catalog.silver.units_current
SELECT
    unit_id,
    sku,
    seller_id,
    tenant_id,
    location,
    status,
    `condition`,
    last_movement_at,
    created_at,
    updated_at
FROM iceberg_catalog.bronze.units_raw
WHERE cdc_operation != 'd';

-- ============================================
-- GOLD LAYER: PROCESS PATH METRICS
-- ============================================

-- Daily process path distribution
CREATE TABLE IF NOT EXISTS iceberg_catalog.gold.process_path_metrics_daily (
    `date` DATE,
    `path_type` STRING,
    `total_orders` BIGINT,
    `completed_orders` BIGINT,
    `avg_duration_minutes` DOUBLE,
    `overridden_count` BIGINT,
    PRIMARY KEY (`date`, `path_type`) NOT ENFORCED
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- ============================================
-- GOLD LAYER: WES METRICS
-- ============================================

-- Daily WES stage performance
CREATE TABLE IF NOT EXISTS iceberg_catalog.gold.wes_stage_metrics_daily (
    `date` DATE,
    `stage_type` STRING,
    `total_stages` BIGINT,
    `completed_stages` BIGINT,
    `failed_stages` BIGINT,
    `avg_duration_minutes` DOUBLE,
    `retry_rate` DOUBLE,
    `completion_rate` DOUBLE,
    PRIMARY KEY (`date`, `stage_type`) NOT ENFORCED
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- ============================================
-- GOLD LAYER: WMS BENCHMARKS AGGREGATION
-- ============================================

-- Combined WMS KPI metrics for benchmarking dashboard
CREATE TABLE IF NOT EXISTS iceberg_catalog.gold.wms_benchmarks_daily (
    `date` DATE,
    -- Operational KPIs
    `picks_per_hour` DOUBLE,
    `order_accuracy_pct` DOUBLE,
    `on_time_shipment_pct` DOUBLE,
    `dock_to_stock_hours` DOUBLE,
    `space_utilization_pct` DOUBLE,
    -- Financial KPIs
    `cost_per_order` DOUBLE,
    `labor_cost_ratio` DOUBLE,
    `fulfillment_revenue_pct` DOUBLE,
    `total_revenue` DOUBLE,
    `total_orders` BIGINT,
    -- Volume metrics
    `total_picks` BIGINT,
    `total_shipments` BIGINT,
    `active_workers` BIGINT,
    PRIMARY KEY (`date`) NOT ENFORCED
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Benchmark targets reference table
CREATE TABLE IF NOT EXISTS iceberg_catalog.gold.wms_benchmark_targets (
    `metric_name` STRING,
    `target_value` DOUBLE,
    `best_in_class_value` DOUBLE,
    `unit` STRING,
    `direction` STRING,  -- 'higher_better' or 'lower_better'
    PRIMARY KEY (`metric_name`) NOT ENFORCED
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert benchmark targets based on industry research
INSERT INTO iceberg_catalog.gold.wms_benchmark_targets VALUES
    ('picks_per_hour', 175, 250, 'picks/hour', 'higher_better'),
    ('order_accuracy_pct', 99.0, 99.9, 'percent', 'higher_better'),
    ('on_time_shipment_pct', 98.0, 99.0, 'percent', 'higher_better'),
    ('dock_to_stock_hours', 4.0, 2.0, 'hours', 'lower_better'),
    ('space_utilization_pct', 85.0, 90.0, 'percent', 'higher_better'),
    ('cost_per_order', 5.0, 3.0, 'USD', 'lower_better'),
    ('labor_cost_ratio', 60.0, 50.0, 'percent', 'lower_better'),
    ('fulfillment_revenue_pct', 8.0, 5.0, 'percent', 'lower_better');
