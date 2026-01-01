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
