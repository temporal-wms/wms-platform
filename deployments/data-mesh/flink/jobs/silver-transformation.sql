-- Silver Layer Transformation: Cleaned and Deduplicated Data
-- This job reads from Bronze layer and creates Silver tables

-- Use Iceberg catalog
USE CATALOG iceberg_catalog;

-- Create Silver database
CREATE DATABASE IF NOT EXISTS silver;

-- Orders Silver: Deduplicated and with latest state only
CREATE TABLE IF NOT EXISTS silver.orders_current (
    `order_id` STRING,
    `customer_id` STRING,
    `status` STRING,
    `priority` STRING,
    `items` ARRAY<ROW<
        sku STRING,
        quantity INT,
        weight DOUBLE,
        unit_price DOUBLE
    >>,
    `shipping_address` ROW<
        street STRING,
        city STRING,
        state STRING,
        zip_code STRING,
        country STRING
    >,
    `wave_id` STRING,
    `tracking_number` STRING,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `is_deleted` BOOLEAN,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`order_id`) NOT ENFORCED
) PARTITIONED BY (days(`created_at`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Deduplicate orders and parse JSON
INSERT INTO silver.orders_current
SELECT
    order_id,
    customer_id,
    status,
    priority,
    CAST(JSON_QUERY(items, '$[*]' RETURNING ARRAY<ROW<
        sku STRING,
        quantity INT,
        weight DOUBLE,
        unit_price DOUBLE
    >>) AS ARRAY<ROW<sku STRING, quantity INT, weight DOUBLE, unit_price DOUBLE>>) AS items,
    CAST(JSON_QUERY(shipping_address, '$' RETURNING ROW<
        street STRING,
        city STRING,
        state STRING,
        zip_code STRING,
        country STRING
    >) AS ROW<street STRING, city STRING, state STRING, zip_code STRING, country STRING>) AS shipping_address,
    wave_id,
    tracking_number,
    created_at,
    updated_at,
    CASE WHEN cdc_operation = 'd' THEN TRUE ELSE FALSE END AS is_deleted,
    CURRENT_TIMESTAMP AS processing_time
FROM (
    SELECT *,
        ROW_NUMBER() OVER (PARTITION BY order_id ORDER BY cdc_timestamp DESC) as rn
    FROM bronze.orders_raw
)
WHERE rn = 1;

-- Inventory Silver: Current stock levels
CREATE TABLE IF NOT EXISTS silver.inventory_current (
    `sku` STRING,
    `product_name` STRING,
    `total_quantity` INT,
    `reserved_quantity` INT,
    `available_quantity` INT,
    `locations` ARRAY<ROW<
        location_id STRING,
        zone STRING,
        aisle STRING,
        rack STRING,
        level STRING,
        quantity INT
    >>,
    `reorder_point` INT,
    `updated_at` TIMESTAMP(3),
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`sku`) NOT ENFORCED
)
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Pick Tasks Silver: Enriched with order and worker data
CREATE TABLE IF NOT EXISTS silver.pick_tasks_enriched (
    `task_id` STRING,
    `order_id` STRING,
    `wave_id` STRING,
    `route_id` STRING,
    `picker_id` STRING,
    `status` STRING,
    `pick_method` STRING,
    `items` ARRAY<ROW<
        sku STRING,
        quantity INT,
        picked_quantity INT,
        location_id STRING,
        status STRING
    >>,
    `tote_id` STRING,
    `zone` STRING,
    `started_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `duration_seconds` BIGINT,
    `items_picked` INT,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`task_id`) NOT ENFORCED
) PARTITIONED BY (days(`started_at`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Shipments Silver: With carrier details
CREATE TABLE IF NOT EXISTS silver.shipments_current (
    `shipment_id` STRING,
    `order_id` STRING,
    `package_id` STRING,
    `wave_id` STRING,
    `status` STRING,
    `carrier_code` STRING,
    `service_type` STRING,
    `tracking_number` STRING,
    `label_url` STRING,
    `manifest_id` STRING,
    `shipped_at` TIMESTAMP(3),
    `estimated_delivery` TIMESTAMP(3),
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`shipment_id`) NOT ENFORCED
) PARTITIONED BY (days(`created_at`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Workers Silver: Current assignments and performance
CREATE TABLE IF NOT EXISTS silver.workers_current (
    `worker_id` STRING,
    `employee_id` STRING,
    `name` STRING,
    `skills` ARRAY<ROW<
        skill_type STRING,
        level INT,
        certified_at TIMESTAMP(3)
    >>,
    `current_zone` STRING,
    `current_task_id` STRING,
    `status` STRING,
    `shift_type` STRING,
    `shift_started_at` TIMESTAMP(3),
    `avg_tasks_per_hour` DOUBLE,
    `avg_items_per_hour` DOUBLE,
    `accuracy_rate` DOUBLE,
    `updated_at` TIMESTAMP(3),
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`worker_id`) NOT ENFORCED
)
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- ============================================
-- PICKING DATA PRODUCT - Silver Layer
-- ============================================

-- Pick Tasks Current: Full details with computed metrics
CREATE TABLE IF NOT EXISTS silver.pick_tasks_current (
    `task_id` STRING,
    `order_id` STRING,
    `wave_id` STRING,
    `route_id` STRING,
    `picker_id` STRING,
    `status` STRING,
    `method` STRING,
    `items` ARRAY<ROW<
        sku STRING,
        product_name STRING,
        quantity INT,
        picked_qty INT,
        location_id STRING,
        zone STRING,
        status STRING,
        tote_id STRING
    >>,
    `tote_id` STRING,
    `zone` STRING,
    `priority` INT,
    `total_items` INT,
    `picked_items` INT,
    `exception_count` INT,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `assigned_at` TIMESTAMP(3),
    `started_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `duration_seconds` BIGINT,
    `pick_rate` DOUBLE,
    `is_deleted` BOOLEAN,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`task_id`) NOT ENFORCED
) PARTITIONED BY (days(`created_at`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Transform and enrich pick tasks from Bronze
INSERT INTO silver.pick_tasks_current
SELECT
    task_id,
    order_id,
    wave_id,
    route_id,
    picker_id,
    status,
    method,
    CAST(JSON_QUERY(items, '$[*]' RETURNING ARRAY<ROW<
        sku STRING,
        product_name STRING,
        quantity INT,
        picked_qty INT,
        location_id STRING,
        zone STRING,
        status STRING,
        tote_id STRING
    >>) AS ARRAY<ROW<sku STRING, product_name STRING, quantity INT, picked_qty INT, location_id STRING, zone STRING, status STRING, tote_id STRING>>) AS items,
    tote_id,
    zone,
    priority,
    total_items,
    picked_items,
    COALESCE(JSON_ARRAY_LENGTH(exceptions), 0) AS exception_count,
    created_at,
    updated_at,
    assigned_at,
    started_at,
    completed_at,
    CASE
        WHEN completed_at IS NOT NULL AND started_at IS NOT NULL
        THEN TIMESTAMPDIFF(SECOND, started_at, completed_at)
        ELSE NULL
    END AS duration_seconds,
    CASE
        WHEN completed_at IS NOT NULL AND started_at IS NOT NULL AND picked_items > 0
        THEN CAST(picked_items AS DOUBLE) / (TIMESTAMPDIFF(SECOND, started_at, completed_at) / 60.0)
        ELSE NULL
    END AS pick_rate,
    CASE WHEN cdc_operation = 'd' THEN TRUE ELSE FALSE END AS is_deleted,
    CURRENT_TIMESTAMP AS processing_time
FROM (
    SELECT *,
        ROW_NUMBER() OVER (PARTITION BY task_id ORDER BY cdc_timestamp DESC) as rn
    FROM bronze.pick_tasks_raw
)
WHERE rn = 1;

-- ============================================
-- CONSOLIDATION DATA PRODUCT - Silver Layer
-- ============================================

CREATE TABLE IF NOT EXISTS silver.consolidations_current (
    `consolidation_id` STRING,
    `order_id` STRING,
    `wave_id` STRING,
    `status` STRING,
    `strategy` STRING,
    `expected_items` ARRAY<ROW<
        sku STRING,
        product_name STRING,
        quantity INT,
        source_tote_id STRING,
        received INT,
        status STRING
    >>,
    `consolidated_items` ARRAY<ROW<
        sku STRING,
        quantity INT,
        source_tote_id STRING,
        scanned_at TIMESTAMP(3),
        verified_by STRING
    >>,
    `source_totes` ARRAY<STRING>,
    `source_tote_count` INT,
    `destination_bin` STRING,
    `station` STRING,
    `worker_id` STRING,
    `total_expected` INT,
    `total_consolidated` INT,
    `consolidation_rate` DOUBLE,
    `ready_for_packing` BOOLEAN,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `started_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `duration_seconds` BIGINT,
    `is_deleted` BOOLEAN,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`consolidation_id`) NOT ENFORCED
) PARTITIONED BY (days(`created_at`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Transform consolidation data from Bronze
INSERT INTO silver.consolidations_current
SELECT
    consolidation_id,
    order_id,
    wave_id,
    status,
    strategy,
    CAST(JSON_QUERY(expected_items, '$[*]' RETURNING ARRAY<ROW<
        sku STRING, product_name STRING, quantity INT,
        source_tote_id STRING, received INT, status STRING
    >>) AS ARRAY<ROW<sku STRING, product_name STRING, quantity INT, source_tote_id STRING, received INT, status STRING>>) AS expected_items,
    CAST(JSON_QUERY(consolidated_items, '$[*]' RETURNING ARRAY<ROW<
        sku STRING, quantity INT, source_tote_id STRING,
        scanned_at TIMESTAMP(3), verified_by STRING
    >>) AS ARRAY<ROW<sku STRING, quantity INT, source_tote_id STRING, scanned_at TIMESTAMP(3), verified_by STRING>>) AS consolidated_items,
    CAST(JSON_QUERY(source_totes, '$[*]' RETURNING ARRAY<STRING>) AS ARRAY<STRING>) AS source_totes,
    COALESCE(JSON_ARRAY_LENGTH(source_totes), 0) AS source_tote_count,
    destination_bin,
    station,
    worker_id,
    total_expected,
    total_consolidated,
    CASE WHEN total_expected > 0
         THEN CAST(total_consolidated AS DOUBLE) / total_expected * 100
         ELSE 0 END AS consolidation_rate,
    ready_for_packing,
    created_at,
    updated_at,
    started_at,
    completed_at,
    CASE
        WHEN completed_at IS NOT NULL AND started_at IS NOT NULL
        THEN TIMESTAMPDIFF(SECOND, started_at, completed_at)
        ELSE NULL
    END AS duration_seconds,
    CASE WHEN cdc_operation = 'd' THEN TRUE ELSE FALSE END AS is_deleted,
    CURRENT_TIMESTAMP AS processing_time
FROM (
    SELECT *, ROW_NUMBER() OVER (PARTITION BY consolidation_id ORDER BY cdc_timestamp DESC) as rn
    FROM bronze.consolidations_raw
)
WHERE rn = 1;

-- ============================================
-- PACKING DATA PRODUCT - Silver Layer
-- ============================================

CREATE TABLE IF NOT EXISTS silver.pack_tasks_current (
    `task_id` STRING,
    `order_id` STRING,
    `consolidation_id` STRING,
    `wave_id` STRING,
    `packer_id` STRING,
    `status` STRING,
    `items` ARRAY<ROW<
        sku STRING,
        product_name STRING,
        quantity INT,
        weight DOUBLE,
        fragile BOOLEAN,
        verified BOOLEAN
    >>,
    `item_count` INT,
    `package_id` STRING,
    `package_type` STRING,
    `package_suggested_type` STRING,
    `package_length` DOUBLE,
    `package_width` DOUBLE,
    `package_height` DOUBLE,
    `package_weight` DOUBLE,
    `total_weight` DOUBLE,
    `package_sealed` BOOLEAN,
    `materials` ARRAY<STRING>,
    `tracking_number` STRING,
    `carrier` STRING,
    `service_type` STRING,
    `station` STRING,
    `priority` INT,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `started_at` TIMESTAMP(3),
    `packed_at` TIMESTAMP(3),
    `labeled_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `packing_duration_seconds` BIGINT,
    `labeling_duration_seconds` BIGINT,
    `total_duration_seconds` BIGINT,
    `is_deleted` BOOLEAN,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`task_id`) NOT ENFORCED
) PARTITIONED BY (days(`created_at`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Transform pack tasks from Bronze
INSERT INTO silver.pack_tasks_current
SELECT
    task_id,
    order_id,
    consolidation_id,
    wave_id,
    packer_id,
    status,
    CAST(JSON_QUERY(items, '$[*]' RETURNING ARRAY<ROW<
        sku STRING, product_name STRING, quantity INT,
        weight DOUBLE, fragile BOOLEAN, verified BOOLEAN
    >>) AS ARRAY<ROW<sku STRING, product_name STRING, quantity INT, weight DOUBLE, fragile BOOLEAN, verified BOOLEAN>>) AS items,
    COALESCE(JSON_ARRAY_LENGTH(items), 0) AS item_count,
    JSON_VALUE(package, '$.packageId') AS package_id,
    JSON_VALUE(package, '$.type') AS package_type,
    JSON_VALUE(package, '$.suggestedType') AS package_suggested_type,
    CAST(JSON_VALUE(package, '$.dimensions.length') AS DOUBLE) AS package_length,
    CAST(JSON_VALUE(package, '$.dimensions.width') AS DOUBLE) AS package_width,
    CAST(JSON_VALUE(package, '$.dimensions.height') AS DOUBLE) AS package_height,
    CAST(JSON_VALUE(package, '$.weight') AS DOUBLE) AS package_weight,
    CAST(JSON_VALUE(package, '$.totalWeight') AS DOUBLE) AS total_weight,
    CAST(JSON_VALUE(package, '$.sealed') AS BOOLEAN) AS package_sealed,
    CAST(JSON_QUERY(package, '$.materials' RETURNING ARRAY<STRING>) AS ARRAY<STRING>) AS materials,
    JSON_VALUE(shipping_label, '$.trackingNumber') AS tracking_number,
    JSON_VALUE(shipping_label, '$.carrier') AS carrier,
    JSON_VALUE(shipping_label, '$.serviceType') AS service_type,
    station,
    priority,
    created_at,
    updated_at,
    started_at,
    packed_at,
    labeled_at,
    completed_at,
    CASE WHEN packed_at IS NOT NULL AND started_at IS NOT NULL
         THEN TIMESTAMPDIFF(SECOND, started_at, packed_at) ELSE NULL END AS packing_duration_seconds,
    CASE WHEN labeled_at IS NOT NULL AND packed_at IS NOT NULL
         THEN TIMESTAMPDIFF(SECOND, packed_at, labeled_at) ELSE NULL END AS labeling_duration_seconds,
    CASE WHEN completed_at IS NOT NULL AND started_at IS NOT NULL
         THEN TIMESTAMPDIFF(SECOND, started_at, completed_at) ELSE NULL END AS total_duration_seconds,
    CASE WHEN cdc_operation = 'd' THEN TRUE ELSE FALSE END AS is_deleted,
    CURRENT_TIMESTAMP AS processing_time
FROM (
    SELECT *, ROW_NUMBER() OVER (PARTITION BY task_id ORDER BY cdc_timestamp DESC) as rn
    FROM bronze.pack_tasks_raw
)
WHERE rn = 1;

-- ============================================
-- ROUTING DATA PRODUCT - Silver Layer
-- ============================================

-- Routes Current: Full details with computed metrics
CREATE TABLE IF NOT EXISTS silver.routes_current (
    `route_id` STRING,
    `order_id` STRING,
    `wave_id` STRING,
    `picker_id` STRING,
    `status` STRING,
    `strategy` STRING,
    `stops` ARRAY<ROW<
        stop_number INT,
        location_id STRING,
        aisle STRING,
        rack INT,
        level INT,
        zone STRING,
        sku STRING,
        quantity INT,
        picked_qty INT,
        status STRING,
        tote_id STRING,
        picked_at TIMESTAMP(3)
    >>,
    `zone` STRING,
    `total_items` INT,
    `picked_items` INT,
    `stop_count` INT,
    `zones_visited` ARRAY<STRING>,
    `zone_count` INT,
    `estimated_distance_m` DOUBLE,
    `actual_distance_m` DOUBLE,
    `estimated_time_seconds` BIGINT,
    `actual_time_seconds` BIGINT,
    `is_multi_route` BOOLEAN,
    `parent_order_id` STRING,
    `route_index` INT,
    `total_routes_in_order` INT,
    `source_tote_id` STRING,
    `efficiency_ratio` DOUBLE,
    `distance_accuracy` DOUBLE,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `started_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `duration_seconds` BIGINT,
    `pick_rate` DOUBLE,
    `is_deleted` BOOLEAN,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`route_id`) NOT ENFORCED
) PARTITIONED BY (days(`created_at`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Transform routes from Bronze
INSERT INTO silver.routes_current
SELECT
    route_id,
    order_id,
    wave_id,
    picker_id,
    status,
    strategy,
    CAST(JSON_QUERY(stops, '$[*]' RETURNING ARRAY<ROW<
        stop_number INT, location_id STRING, aisle STRING,
        rack INT, level INT, zone STRING, sku STRING,
        quantity INT, picked_qty INT, status STRING,
        tote_id STRING, picked_at TIMESTAMP(3)
    >>) AS ARRAY<ROW<stop_number INT, location_id STRING, aisle STRING,
        rack INT, level INT, zone STRING, sku STRING,
        quantity INT, picked_qty INT, status STRING,
        tote_id STRING, picked_at TIMESTAMP(3)>>) AS stops,
    zone,
    total_items,
    picked_items,
    COALESCE(JSON_ARRAY_LENGTH(stops), 0) AS stop_count,
    -- Extract unique zones from stops JSON
    ARRAY_DISTINCT(CAST(JSON_QUERY(stops, '$[*].zone' RETURNING ARRAY<STRING>) AS ARRAY<STRING>)) AS zones_visited,
    CARDINALITY(ARRAY_DISTINCT(CAST(JSON_QUERY(stops, '$[*].zone' RETURNING ARRAY<STRING>) AS ARRAY<STRING>))) AS zone_count,
    estimated_distance AS estimated_distance_m,
    actual_distance AS actual_distance_m,
    estimated_time / 1000000000 AS estimated_time_seconds,
    actual_time / 1000000000 AS actual_time_seconds,
    is_multi_route,
    parent_order_id,
    route_index,
    total_routes_in_order,
    source_tote_id,
    CASE WHEN estimated_time > 0
         THEN CAST(actual_time AS DOUBLE) / estimated_time
         ELSE NULL END AS efficiency_ratio,
    CASE WHEN estimated_distance > 0
         THEN actual_distance / estimated_distance
         ELSE NULL END AS distance_accuracy,
    created_at,
    updated_at,
    started_at,
    completed_at,
    CASE WHEN completed_at IS NOT NULL AND started_at IS NOT NULL
         THEN TIMESTAMPDIFF(SECOND, started_at, completed_at)
         ELSE NULL END AS duration_seconds,
    CASE WHEN completed_at IS NOT NULL AND started_at IS NOT NULL AND picked_items > 0
         THEN CAST(picked_items AS DOUBLE) / (TIMESTAMPDIFF(SECOND, started_at, completed_at) / 60.0)
         ELSE NULL END AS pick_rate,
    CASE WHEN cdc_operation = 'd' THEN TRUE ELSE FALSE END AS is_deleted,
    CURRENT_TIMESTAMP AS processing_time
FROM (
    SELECT *, ROW_NUMBER() OVER (PARTITION BY route_id ORDER BY cdc_timestamp DESC) as rn
    FROM bronze.routes_raw
)
WHERE rn = 1;

-- ============================================
-- RECEIVING DATA PRODUCT - Silver Layer
-- ============================================

-- Receipts Current: Full details with computed metrics
CREATE TABLE IF NOT EXISTS silver.receipts_current (
    `receipt_id` STRING,
    `po_number` STRING,
    `vendor_id` STRING,
    `vendor_name` STRING,
    `dock_door` STRING,
    `status` STRING,
    `items` ARRAY<ROW<
        sku STRING,
        product_name STRING,
        expected_qty INT,
        received_qty INT,
        damaged_qty INT,
        status STRING
    >>,
    `total_units` INT,
    `received_units` INT,
    `damaged_units` INT,
    `receiving_accuracy` DOUBLE,
    `expected_at` TIMESTAMP(3),
    `arrived_at` TIMESTAMP(3),
    `unloading_started_at` TIMESTAMP(3),
    `unloading_completed_at` TIMESTAMP(3),
    `inspection_completed_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `arrival_variance_minutes` BIGINT,
    `unloading_duration_minutes` BIGINT,
    `inspection_duration_minutes` BIGINT,
    `dock_to_stock_minutes` BIGINT,
    `worker_id` STRING,
    `notes` STRING,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `is_deleted` BOOLEAN,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`receipt_id`) NOT ENFORCED
) PARTITIONED BY (days(`created_at`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Transform receipts from Bronze
INSERT INTO silver.receipts_current
SELECT
    receipt_id,
    po_number,
    vendor_id,
    vendor_name,
    dock_door,
    status,
    CAST(JSON_QUERY(items, '$[*]' RETURNING ARRAY<ROW<
        sku STRING, product_name STRING, expected_qty INT,
        received_qty INT, damaged_qty INT, status STRING
    >>) AS ARRAY<ROW<sku STRING, product_name STRING, expected_qty INT,
        received_qty INT, damaged_qty INT, status STRING>>) AS items,
    total_units,
    received_units,
    damaged_units,
    CASE WHEN total_units > 0
         THEN CAST(received_units - damaged_units AS DOUBLE) / total_units * 100
         ELSE 100.0 END AS receiving_accuracy,
    expected_at,
    arrived_at,
    unloading_started_at,
    unloading_completed_at,
    inspection_completed_at,
    completed_at,
    CASE WHEN arrived_at IS NOT NULL AND expected_at IS NOT NULL
         THEN TIMESTAMPDIFF(MINUTE, expected_at, arrived_at)
         ELSE NULL END AS arrival_variance_minutes,
    CASE WHEN unloading_completed_at IS NOT NULL AND unloading_started_at IS NOT NULL
         THEN TIMESTAMPDIFF(MINUTE, unloading_started_at, unloading_completed_at)
         ELSE NULL END AS unloading_duration_minutes,
    CASE WHEN inspection_completed_at IS NOT NULL AND unloading_completed_at IS NOT NULL
         THEN TIMESTAMPDIFF(MINUTE, unloading_completed_at, inspection_completed_at)
         ELSE NULL END AS inspection_duration_minutes,
    CASE WHEN completed_at IS NOT NULL AND arrived_at IS NOT NULL
         THEN TIMESTAMPDIFF(MINUTE, arrived_at, completed_at)
         ELSE NULL END AS dock_to_stock_minutes,
    worker_id,
    notes,
    created_at,
    updated_at,
    CASE WHEN cdc_operation = 'd' THEN TRUE ELSE FALSE END AS is_deleted,
    CURRENT_TIMESTAMP AS processing_time
FROM (
    SELECT *, ROW_NUMBER() OVER (PARTITION BY receipt_id ORDER BY cdc_timestamp DESC) as rn
    FROM bronze.receipts_raw
)
WHERE rn = 1;

-- ============================================
-- STOWING DATA PRODUCT - Silver Layer
-- ============================================

-- Stow Tasks Current: Full details with computed metrics
CREATE TABLE IF NOT EXISTS silver.stow_tasks_current (
    `stow_task_id` STRING,
    `receipt_id` STRING,
    `worker_id` STRING,
    `status` STRING,
    `sku` STRING,
    `product_name` STRING,
    `quantity` INT,
    `source_location` STRING,
    `target_location` STRING,
    `suggested_location` STRING,
    `actual_location` STRING,
    `zone` STRING,
    `used_suggested_location` BOOLEAN,
    `priority` INT,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `assigned_at` TIMESTAMP(3),
    `started_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `assignment_to_start_minutes` BIGINT,
    `stow_duration_minutes` BIGINT,
    `total_duration_minutes` BIGINT,
    `stow_rate` DOUBLE,
    `is_deleted` BOOLEAN,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`stow_task_id`) NOT ENFORCED
) PARTITIONED BY (days(`created_at`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Transform stow tasks from Bronze
INSERT INTO silver.stow_tasks_current
SELECT
    stow_task_id,
    receipt_id,
    worker_id,
    status,
    sku,
    product_name,
    quantity,
    source_location,
    target_location,
    suggested_location,
    actual_location,
    zone,
    CASE WHEN actual_location = suggested_location THEN TRUE ELSE FALSE END AS used_suggested_location,
    priority,
    created_at,
    updated_at,
    assigned_at,
    started_at,
    completed_at,
    CASE WHEN started_at IS NOT NULL AND assigned_at IS NOT NULL
         THEN TIMESTAMPDIFF(MINUTE, assigned_at, started_at)
         ELSE NULL END AS assignment_to_start_minutes,
    CASE WHEN completed_at IS NOT NULL AND started_at IS NOT NULL
         THEN TIMESTAMPDIFF(MINUTE, started_at, completed_at)
         ELSE NULL END AS stow_duration_minutes,
    CASE WHEN completed_at IS NOT NULL AND created_at IS NOT NULL
         THEN TIMESTAMPDIFF(MINUTE, created_at, completed_at)
         ELSE NULL END AS total_duration_minutes,
    CASE WHEN completed_at IS NOT NULL AND started_at IS NOT NULL AND quantity > 0
         THEN CAST(quantity AS DOUBLE) / (TIMESTAMPDIFF(SECOND, started_at, completed_at) / 60.0)
         ELSE NULL END AS stow_rate,
    CASE WHEN cdc_operation = 'd' THEN TRUE ELSE FALSE END AS is_deleted,
    CURRENT_TIMESTAMP AS processing_time
FROM (
    SELECT *, ROW_NUMBER() OVER (PARTITION BY stow_task_id ORDER BY cdc_timestamp DESC) as rn
    FROM bronze.stow_tasks_raw
)
WHERE rn = 1;

-- ============================================
-- RETURNS DATA PRODUCT - Silver Layer
-- ============================================

-- Returns Current: Full details with computed metrics
CREATE TABLE IF NOT EXISTS silver.returns_current (
    `return_id` STRING,
    `order_id` STRING,
    `customer_id` STRING,
    `status` STRING,
    `reason` STRING,
    `disposition` STRING,
    `items` ARRAY<ROW<
        sku STRING,
        product_name STRING,
        quantity INT,
        condition STRING,
        disposition STRING,
        restocked BOOLEAN
    >>,
    `total_items` INT,
    `restocked_items` INT,
    `disposed_items` INT,
    `restock_rate` DOUBLE,
    `refund_amount` DOUBLE,
    `tracking_number` STRING,
    `carrier` STRING,
    `worker_id` STRING,
    `notes` STRING,
    `received_at` TIMESTAMP(3),
    `inspected_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `inspection_duration_minutes` BIGINT,
    `processing_duration_minutes` BIGINT,
    `is_deleted` BOOLEAN,
    `processing_time` TIMESTAMP(3),
    PRIMARY KEY (`return_id`) NOT ENFORCED
) PARTITIONED BY (days(`created_at`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Transform returns from Bronze
INSERT INTO silver.returns_current
SELECT
    return_id,
    order_id,
    customer_id,
    status,
    reason,
    disposition,
    CAST(JSON_QUERY(items, '$[*]' RETURNING ARRAY<ROW<
        sku STRING, product_name STRING, quantity INT,
        condition STRING, disposition STRING, restocked BOOLEAN
    >>) AS ARRAY<ROW<sku STRING, product_name STRING, quantity INT,
        condition STRING, disposition STRING, restocked BOOLEAN>>) AS items,
    total_items,
    restocked_items,
    disposed_items,
    CASE WHEN total_items > 0
         THEN CAST(restocked_items AS DOUBLE) / total_items * 100
         ELSE 0.0 END AS restock_rate,
    refund_amount,
    tracking_number,
    carrier,
    worker_id,
    notes,
    received_at,
    inspected_at,
    completed_at,
    created_at,
    updated_at,
    CASE WHEN inspected_at IS NOT NULL AND received_at IS NOT NULL
         THEN TIMESTAMPDIFF(MINUTE, received_at, inspected_at)
         ELSE NULL END AS inspection_duration_minutes,
    CASE WHEN completed_at IS NOT NULL AND received_at IS NOT NULL
         THEN TIMESTAMPDIFF(MINUTE, received_at, completed_at)
         ELSE NULL END AS processing_duration_minutes,
    CASE WHEN cdc_operation = 'd' THEN TRUE ELSE FALSE END AS is_deleted,
    CURRENT_TIMESTAMP AS processing_time
FROM (
    SELECT *, ROW_NUMBER() OVER (PARTITION BY return_id ORDER BY cdc_timestamp DESC) as rn
    FROM bronze.returns_raw
)
WHERE rn = 1;
