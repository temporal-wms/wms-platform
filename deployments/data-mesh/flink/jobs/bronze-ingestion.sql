-- Bronze Layer Ingestion: Raw CDC Events to Iceberg Tables
-- This job consumes CDC events from Debezium and writes to Bronze layer

-- Configure Kafka source for Orders CDC
CREATE TABLE orders_cdc_source (
    `_id` STRING,
    `order_id` STRING,
    `customer_id` STRING,
    `status` STRING,
    `priority` STRING,
    `items` STRING,  -- JSON array
    `shipping_address` STRING,  -- JSON object
    `wave_id` STRING,
    `tracking_number` STRING,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.orders',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-orders',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Iceberg catalog
CREATE CATALOG iceberg_catalog WITH (
    'type' = 'iceberg',
    'catalog-type' = 'hive',
    'uri' = 'thrift://hive-metastore.data-mesh.svc.cluster.local:9083',
    'warehouse' = 's3://iceberg-warehouse',
    's3.endpoint' = 'http://minio.data-mesh.svc.cluster.local:9000',
    's3.path-style-access' = 'true'
);

-- Create Bronze database
CREATE DATABASE IF NOT EXISTS iceberg_catalog.bronze;

-- Create Bronze Orders table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.orders_raw (
    `_id` STRING,
    `order_id` STRING,
    `customer_id` STRING,
    `status` STRING,
    `priority` STRING,
    `items` STRING,
    `shipping_address` STRING,
    `wave_id` STRING,
    `tracking_number` STRING,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer
INSERT INTO iceberg_catalog.bronze.orders_raw
SELECT
    `_id`,
    `order_id`,
    `customer_id`,
    `status`,
    `priority`,
    `items`,
    `shipping_address`,
    `wave_id`,
    `tracking_number`,
    `created_at`,
    `updated_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM orders_cdc_source;

-- Similar patterns for other collections:
-- inventory_cdc_source -> bronze.inventory_raw
-- waves_cdc_source -> bronze.waves_raw
-- shipments_cdc_source -> bronze.shipments_raw
-- workers_cdc_source -> bronze.workers_raw

-- ============================================
-- PICKING DATA PRODUCT - Bronze Layer
-- ============================================

-- Configure Kafka source for Pick Tasks CDC
CREATE TABLE pick_tasks_cdc_source (
    `_id` STRING,
    `task_id` STRING,
    `order_id` STRING,
    `wave_id` STRING,
    `route_id` STRING,
    `picker_id` STRING,
    `status` STRING,
    `method` STRING,
    `items` STRING,  -- JSON array
    `tote_id` STRING,
    `zone` STRING,
    `priority` INT,
    `total_items` INT,
    `picked_items` INT,
    `exceptions` STRING,  -- JSON array
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `assigned_at` TIMESTAMP(3),
    `started_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.pick_tasks',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-pick-tasks',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Bronze Pick Tasks table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.pick_tasks_raw (
    `_id` STRING,
    `task_id` STRING,
    `order_id` STRING,
    `wave_id` STRING,
    `route_id` STRING,
    `picker_id` STRING,
    `status` STRING,
    `method` STRING,
    `items` STRING,
    `tote_id` STRING,
    `zone` STRING,
    `priority` INT,
    `total_items` INT,
    `picked_items` INT,
    `exceptions` STRING,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `assigned_at` TIMESTAMP(3),
    `started_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer
INSERT INTO iceberg_catalog.bronze.pick_tasks_raw
SELECT
    `_id`,
    `task_id`,
    `order_id`,
    `wave_id`,
    `route_id`,
    `picker_id`,
    `status`,
    `method`,
    `items`,
    `tote_id`,
    `zone`,
    `priority`,
    `total_items`,
    `picked_items`,
    `exceptions`,
    `created_at`,
    `updated_at`,
    `assigned_at`,
    `started_at`,
    `completed_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM pick_tasks_cdc_source;

-- ============================================
-- CONSOLIDATION DATA PRODUCT - Bronze Layer
-- ============================================

-- Configure Kafka source for Consolidation CDC
CREATE TABLE consolidations_cdc_source (
    `_id` STRING,
    `consolidation_id` STRING,
    `order_id` STRING,
    `wave_id` STRING,
    `status` STRING,
    `strategy` STRING,
    `expected_items` STRING,  -- JSON array
    `consolidated_items` STRING,  -- JSON array
    `source_totes` STRING,  -- JSON array
    `destination_bin` STRING,
    `station` STRING,
    `worker_id` STRING,
    `total_expected` INT,
    `total_consolidated` INT,
    `ready_for_packing` BOOLEAN,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `started_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.consolidations',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-consolidations',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Bronze Consolidation table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.consolidations_raw (
    `_id` STRING,
    `consolidation_id` STRING,
    `order_id` STRING,
    `wave_id` STRING,
    `status` STRING,
    `strategy` STRING,
    `expected_items` STRING,
    `consolidated_items` STRING,
    `source_totes` STRING,
    `destination_bin` STRING,
    `station` STRING,
    `worker_id` STRING,
    `total_expected` INT,
    `total_consolidated` INT,
    `ready_for_packing` BOOLEAN,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `started_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer
INSERT INTO iceberg_catalog.bronze.consolidations_raw
SELECT
    `_id`,
    `consolidation_id`,
    `order_id`,
    `wave_id`,
    `status`,
    `strategy`,
    `expected_items`,
    `consolidated_items`,
    `source_totes`,
    `destination_bin`,
    `station`,
    `worker_id`,
    `total_expected`,
    `total_consolidated`,
    `ready_for_packing`,
    `created_at`,
    `updated_at`,
    `started_at`,
    `completed_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM consolidations_cdc_source;

-- ============================================
-- PACKING DATA PRODUCT - Bronze Layer
-- ============================================

-- Configure Kafka source for Pack Tasks CDC
CREATE TABLE pack_tasks_cdc_source (
    `_id` STRING,
    `task_id` STRING,
    `order_id` STRING,
    `consolidation_id` STRING,
    `wave_id` STRING,
    `packer_id` STRING,
    `status` STRING,
    `items` STRING,  -- JSON array
    `package` STRING,  -- JSON object (type, dimensions, weight, sealed, materials)
    `shipping_label` STRING,  -- JSON object (tracking_number, carrier, service_type)
    `station` STRING,
    `priority` INT,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `started_at` TIMESTAMP(3),
    `packed_at` TIMESTAMP(3),
    `labeled_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.pack_tasks',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-pack-tasks',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Bronze Pack Tasks table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.pack_tasks_raw (
    `_id` STRING,
    `task_id` STRING,
    `order_id` STRING,
    `consolidation_id` STRING,
    `wave_id` STRING,
    `packer_id` STRING,
    `status` STRING,
    `items` STRING,
    `package` STRING,
    `shipping_label` STRING,
    `station` STRING,
    `priority` INT,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `started_at` TIMESTAMP(3),
    `packed_at` TIMESTAMP(3),
    `labeled_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer
INSERT INTO iceberg_catalog.bronze.pack_tasks_raw
SELECT
    `_id`,
    `task_id`,
    `order_id`,
    `consolidation_id`,
    `wave_id`,
    `packer_id`,
    `status`,
    `items`,
    `package`,
    `shipping_label`,
    `station`,
    `priority`,
    `created_at`,
    `updated_at`,
    `started_at`,
    `packed_at`,
    `labeled_at`,
    `completed_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM pack_tasks_cdc_source;

-- ============================================
-- ROUTING DATA PRODUCT - Bronze Layer
-- ============================================

-- Configure Kafka source for Routes CDC
CREATE TABLE routes_cdc_source (
    `_id` STRING,
    `route_id` STRING,
    `order_id` STRING,
    `wave_id` STRING,
    `picker_id` STRING,
    `status` STRING,
    `strategy` STRING,
    `stops` STRING,  -- JSON array of route stops
    `zone` STRING,
    `total_items` INT,
    `picked_items` INT,
    `estimated_distance` DOUBLE,
    `actual_distance` DOUBLE,
    `estimated_time` BIGINT,  -- Duration in nanoseconds
    `actual_time` BIGINT,
    `start_location` STRING,  -- JSON object
    `end_location` STRING,  -- JSON object
    `is_multi_route` BOOLEAN,
    `parent_order_id` STRING,
    `route_index` INT,
    `total_routes_in_order` INT,
    `source_tote_id` STRING,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `started_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.routes',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-routes',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Bronze Routes table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.routes_raw (
    `_id` STRING,
    `route_id` STRING,
    `order_id` STRING,
    `wave_id` STRING,
    `picker_id` STRING,
    `status` STRING,
    `strategy` STRING,
    `stops` STRING,
    `zone` STRING,
    `total_items` INT,
    `picked_items` INT,
    `estimated_distance` DOUBLE,
    `actual_distance` DOUBLE,
    `estimated_time` BIGINT,
    `actual_time` BIGINT,
    `start_location` STRING,
    `end_location` STRING,
    `is_multi_route` BOOLEAN,
    `parent_order_id` STRING,
    `route_index` INT,
    `total_routes_in_order` INT,
    `source_tote_id` STRING,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `started_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer
INSERT INTO iceberg_catalog.bronze.routes_raw
SELECT
    `_id`,
    `route_id`,
    `order_id`,
    `wave_id`,
    `picker_id`,
    `status`,
    `strategy`,
    `stops`,
    `zone`,
    `total_items`,
    `picked_items`,
    `estimated_distance`,
    `actual_distance`,
    `estimated_time`,
    `actual_time`,
    `start_location`,
    `end_location`,
    `is_multi_route`,
    `parent_order_id`,
    `route_index`,
    `total_routes_in_order`,
    `source_tote_id`,
    `created_at`,
    `updated_at`,
    `started_at`,
    `completed_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM routes_cdc_source;

-- ============================================
-- RECEIVING DATA PRODUCT - Bronze Layer
-- ============================================

-- Configure Kafka source for Receipts CDC
CREATE TABLE receipts_cdc_source (
    `_id` STRING,
    `receipt_id` STRING,
    `po_number` STRING,
    `vendor_id` STRING,
    `vendor_name` STRING,
    `dock_door` STRING,
    `status` STRING,
    `items` STRING,  -- JSON array
    `total_units` INT,
    `received_units` INT,
    `damaged_units` INT,
    `expected_at` TIMESTAMP(3),
    `arrived_at` TIMESTAMP(3),
    `unloading_started_at` TIMESTAMP(3),
    `unloading_completed_at` TIMESTAMP(3),
    `inspection_completed_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `worker_id` STRING,
    `notes` STRING,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.receipts',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-receipts',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Bronze Receipts table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.receipts_raw (
    `_id` STRING,
    `receipt_id` STRING,
    `po_number` STRING,
    `vendor_id` STRING,
    `vendor_name` STRING,
    `dock_door` STRING,
    `status` STRING,
    `items` STRING,
    `total_units` INT,
    `received_units` INT,
    `damaged_units` INT,
    `expected_at` TIMESTAMP(3),
    `arrived_at` TIMESTAMP(3),
    `unloading_started_at` TIMESTAMP(3),
    `unloading_completed_at` TIMESTAMP(3),
    `inspection_completed_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `worker_id` STRING,
    `notes` STRING,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer
INSERT INTO iceberg_catalog.bronze.receipts_raw
SELECT
    `_id`,
    `receipt_id`,
    `po_number`,
    `vendor_id`,
    `vendor_name`,
    `dock_door`,
    `status`,
    `items`,
    `total_units`,
    `received_units`,
    `damaged_units`,
    `expected_at`,
    `arrived_at`,
    `unloading_started_at`,
    `unloading_completed_at`,
    `inspection_completed_at`,
    `completed_at`,
    `worker_id`,
    `notes`,
    `created_at`,
    `updated_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM receipts_cdc_source;

-- ============================================
-- STOWING DATA PRODUCT - Bronze Layer
-- ============================================

-- Configure Kafka source for Stow Tasks CDC
CREATE TABLE stow_tasks_cdc_source (
    `_id` STRING,
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
    `priority` INT,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `assigned_at` TIMESTAMP(3),
    `started_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.stow_tasks',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-stow-tasks',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Bronze Stow Tasks table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.stow_tasks_raw (
    `_id` STRING,
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
    `priority` INT,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `assigned_at` TIMESTAMP(3),
    `started_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer
INSERT INTO iceberg_catalog.bronze.stow_tasks_raw
SELECT
    `_id`,
    `stow_task_id`,
    `receipt_id`,
    `worker_id`,
    `status`,
    `sku`,
    `product_name`,
    `quantity`,
    `source_location`,
    `target_location`,
    `suggested_location`,
    `actual_location`,
    `zone`,
    `priority`,
    `created_at`,
    `updated_at`,
    `assigned_at`,
    `started_at`,
    `completed_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM stow_tasks_cdc_source;

-- ============================================
-- RETURNS DATA PRODUCT - Bronze Layer
-- ============================================

-- Configure Kafka source for Returns CDC
CREATE TABLE returns_cdc_source (
    `_id` STRING,
    `return_id` STRING,
    `order_id` STRING,
    `customer_id` STRING,
    `status` STRING,
    `reason` STRING,
    `disposition` STRING,
    `items` STRING,  -- JSON array
    `total_items` INT,
    `restocked_items` INT,
    `disposed_items` INT,
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
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.returns',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-returns',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Bronze Returns table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.returns_raw (
    `_id` STRING,
    `return_id` STRING,
    `order_id` STRING,
    `customer_id` STRING,
    `status` STRING,
    `reason` STRING,
    `disposition` STRING,
    `items` STRING,
    `total_items` INT,
    `restocked_items` INT,
    `disposed_items` INT,
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
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer
INSERT INTO iceberg_catalog.bronze.returns_raw
SELECT
    `_id`,
    `return_id`,
    `order_id`,
    `customer_id`,
    `status`,
    `reason`,
    `disposition`,
    `items`,
    `total_items`,
    `restocked_items`,
    `disposed_items`,
    `refund_amount`,
    `tracking_number`,
    `carrier`,
    `worker_id`,
    `notes`,
    `received_at`,
    `inspected_at`,
    `completed_at`,
    `created_at`,
    `updated_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM returns_cdc_source;

-- ============================================
-- BILLING DATA PRODUCT - Bronze Layer
-- ============================================

-- Configure Kafka source for Billing Activities CDC
CREATE TABLE billing_activities_cdc_source (
    `_id` STRING,
    `activity_id` STRING,
    `tenant_id` STRING,
    `seller_id` STRING,
    `facility_id` STRING,
    `type` STRING,
    `description` STRING,
    `quantity` DOUBLE,
    `unit_price` DOUBLE,
    `amount` DOUBLE,
    `currency` STRING,
    `reference_type` STRING,
    `reference_id` STRING,
    `activity_date` TIMESTAMP(3),
    `billing_date` TIMESTAMP(3),
    `invoice_id` STRING,
    `invoiced` BOOLEAN,
    `metadata` STRING,  -- JSON object
    `created_at` TIMESTAMP(3),
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.billing_activities',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-billing-activities',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Bronze Billing Activities table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.billing_activities_raw (
    `_id` STRING,
    `activity_id` STRING,
    `tenant_id` STRING,
    `seller_id` STRING,
    `facility_id` STRING,
    `type` STRING,
    `description` STRING,
    `quantity` DOUBLE,
    `unit_price` DOUBLE,
    `amount` DOUBLE,
    `currency` STRING,
    `reference_type` STRING,
    `reference_id` STRING,
    `activity_date` TIMESTAMP(3),
    `billing_date` TIMESTAMP(3),
    `invoice_id` STRING,
    `invoiced` BOOLEAN,
    `metadata` STRING,
    `created_at` TIMESTAMP(3),
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer
INSERT INTO iceberg_catalog.bronze.billing_activities_raw
SELECT
    `_id`,
    `activity_id`,
    `tenant_id`,
    `seller_id`,
    `facility_id`,
    `type`,
    `description`,
    `quantity`,
    `unit_price`,
    `amount`,
    `currency`,
    `reference_type`,
    `reference_id`,
    `activity_date`,
    `billing_date`,
    `invoice_id`,
    `invoiced`,
    `metadata`,
    `created_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM billing_activities_cdc_source;

-- Configure Kafka source for Invoices CDC
CREATE TABLE invoices_cdc_source (
    `_id` STRING,
    `invoice_id` STRING,
    `tenant_id` STRING,
    `seller_id` STRING,
    `facility_id` STRING,
    `status` STRING,
    `invoice_number` STRING,
    `period_start` TIMESTAMP(3),
    `period_end` TIMESTAMP(3),
    `line_items` STRING,  -- JSON array
    `subtotal` DOUBLE,
    `tax_rate` DOUBLE,
    `tax_amount` DOUBLE,
    `discount` DOUBLE,
    `total` DOUBLE,
    `currency` STRING,
    `due_date` TIMESTAMP(3),
    `paid_at` TIMESTAMP(3),
    `payment_method` STRING,
    `payment_ref` STRING,
    `seller_name` STRING,
    `seller_email` STRING,
    `notes` STRING,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `finalized_at` TIMESTAMP(3),
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.invoices',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-invoices',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Bronze Invoices table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.invoices_raw (
    `_id` STRING,
    `invoice_id` STRING,
    `tenant_id` STRING,
    `seller_id` STRING,
    `facility_id` STRING,
    `status` STRING,
    `invoice_number` STRING,
    `period_start` TIMESTAMP(3),
    `period_end` TIMESTAMP(3),
    `line_items` STRING,
    `subtotal` DOUBLE,
    `tax_rate` DOUBLE,
    `tax_amount` DOUBLE,
    `discount` DOUBLE,
    `total` DOUBLE,
    `currency` STRING,
    `due_date` TIMESTAMP(3),
    `paid_at` TIMESTAMP(3),
    `payment_method` STRING,
    `payment_ref` STRING,
    `seller_name` STRING,
    `seller_email` STRING,
    `notes` STRING,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `finalized_at` TIMESTAMP(3),
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer
INSERT INTO iceberg_catalog.bronze.invoices_raw
SELECT
    `_id`,
    `invoice_id`,
    `tenant_id`,
    `seller_id`,
    `facility_id`,
    `status`,
    `invoice_number`,
    `period_start`,
    `period_end`,
    `line_items`,
    `subtotal`,
    `tax_rate`,
    `tax_amount`,
    `discount`,
    `total`,
    `currency`,
    `due_date`,
    `paid_at`,
    `payment_method`,
    `payment_ref`,
    `seller_name`,
    `seller_email`,
    `notes`,
    `created_at`,
    `updated_at`,
    `finalized_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM invoices_cdc_source;

-- ============================================
-- SELLER DATA PRODUCT - Bronze Layer
-- ============================================

-- Configure Kafka source for Sellers CDC
CREATE TABLE sellers_cdc_source (
    `_id` STRING,
    `seller_id` STRING,
    `tenant_id` STRING,
    `name` STRING,
    `email` STRING,
    `phone` STRING,
    `status` STRING,
    `tier` STRING,
    `contract_start_date` TIMESTAMP(3),
    `contract_end_date` TIMESTAMP(3),
    `fee_schedule` STRING,  -- JSON object
    `facilities` STRING,  -- JSON array
    `channels` STRING,  -- JSON array
    `address` STRING,  -- JSON object
    `settings` STRING,  -- JSON object
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.sellers',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-sellers',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Bronze Sellers table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.sellers_raw (
    `_id` STRING,
    `seller_id` STRING,
    `tenant_id` STRING,
    `name` STRING,
    `email` STRING,
    `phone` STRING,
    `status` STRING,
    `tier` STRING,
    `contract_start_date` TIMESTAMP(3),
    `contract_end_date` TIMESTAMP(3),
    `fee_schedule` STRING,
    `facilities` STRING,
    `channels` STRING,
    `address` STRING,
    `settings` STRING,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer
INSERT INTO iceberg_catalog.bronze.sellers_raw
SELECT
    `_id`,
    `seller_id`,
    `tenant_id`,
    `name`,
    `email`,
    `phone`,
    `status`,
    `tier`,
    `contract_start_date`,
    `contract_end_date`,
    `fee_schedule`,
    `facilities`,
    `channels`,
    `address`,
    `settings`,
    `created_at`,
    `updated_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM sellers_cdc_source;

-- ============================================
-- CHANNEL DATA PRODUCT - Bronze Layer
-- ============================================

-- Configure Kafka source for Channels CDC
CREATE TABLE channels_cdc_source (
    `_id` STRING,
    `channel_id` STRING,
    `seller_id` STRING,
    `tenant_id` STRING,
    `platform` STRING,
    `store_name` STRING,
    `store_url` STRING,
    `status` STRING,
    `credentials` STRING,  -- JSON object (encrypted)
    `settings` STRING,  -- JSON object
    `sync_config` STRING,  -- JSON object
    `last_sync_at` TIMESTAMP(3),
    `last_sync_status` STRING,
    `orders_imported` BIGINT,
    `products_synced` BIGINT,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.channels',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-channels',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Bronze Channels table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.channels_raw (
    `_id` STRING,
    `channel_id` STRING,
    `seller_id` STRING,
    `tenant_id` STRING,
    `platform` STRING,
    `store_name` STRING,
    `store_url` STRING,
    `status` STRING,
    `settings` STRING,
    `sync_config` STRING,
    `last_sync_at` TIMESTAMP(3),
    `last_sync_status` STRING,
    `orders_imported` BIGINT,
    `products_synced` BIGINT,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer (excluding credentials for security)
INSERT INTO iceberg_catalog.bronze.channels_raw
SELECT
    `_id`,
    `channel_id`,
    `seller_id`,
    `tenant_id`,
    `platform`,
    `store_name`,
    `store_url`,
    `status`,
    `settings`,
    `sync_config`,
    `last_sync_at`,
    `last_sync_status`,
    `orders_imported`,
    `products_synced`,
    `created_at`,
    `updated_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM channels_cdc_source;

-- ============================================
-- FACILITY DATA PRODUCT - Bronze Layer
-- ============================================

-- Configure Kafka source for Stations CDC
CREATE TABLE stations_cdc_source (
    `_id` STRING,
    `station_id` STRING,
    `tenant_id` STRING,
    `facility_id` STRING,
    `warehouse_id` STRING,
    `name` STRING,
    `zone` STRING,
    `station_type` STRING,
    `status` STRING,
    `capabilities` STRING,  -- JSON array
    `max_concurrent_tasks` INT,
    `current_tasks` INT,
    `assigned_worker_id` STRING,
    `equipment` STRING,  -- JSON array
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.stations',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-stations',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Bronze Stations table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.stations_raw (
    `_id` STRING,
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
    `equipment` STRING,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer
INSERT INTO iceberg_catalog.bronze.stations_raw
SELECT
    `_id`,
    `station_id`,
    `tenant_id`,
    `facility_id`,
    `warehouse_id`,
    `name`,
    `zone`,
    `station_type`,
    `status`,
    `capabilities`,
    `max_concurrent_tasks`,
    `current_tasks`,
    `assigned_worker_id`,
    `equipment`,
    `created_at`,
    `updated_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM stations_cdc_source;

-- ============================================
-- SORTATION DATA PRODUCT - Bronze Layer
-- ============================================

-- Configure Kafka source for Sortation Batches CDC
CREATE TABLE sortation_batches_cdc_source (
    `_id` STRING,
    `batch_id` STRING,
    `sortation_center` STRING,
    `destination_group` STRING,
    `carrier_id` STRING,
    `packages` STRING,  -- JSON array
    `assigned_chute` STRING,
    `status` STRING,
    `total_packages` INT,
    `sorted_count` INT,
    `total_weight` DOUBLE,
    `trailer_id` STRING,
    `dispatch_dock` STRING,
    `scheduled_dispatch` TIMESTAMP(3),
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `dispatched_at` TIMESTAMP(3),
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.sortation_batches',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-sortation-batches',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Bronze Sortation Batches table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.sortation_batches_raw (
    `_id` STRING,
    `batch_id` STRING,
    `sortation_center` STRING,
    `destination_group` STRING,
    `carrier_id` STRING,
    `packages` STRING,
    `assigned_chute` STRING,
    `status` STRING,
    `total_packages` INT,
    `sorted_count` INT,
    `total_weight` DOUBLE,
    `trailer_id` STRING,
    `dispatch_dock` STRING,
    `scheduled_dispatch` TIMESTAMP(3),
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `dispatched_at` TIMESTAMP(3),
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer
INSERT INTO iceberg_catalog.bronze.sortation_batches_raw
SELECT
    `_id`,
    `batch_id`,
    `sortation_center`,
    `destination_group`,
    `carrier_id`,
    `packages`,
    `assigned_chute`,
    `status`,
    `total_packages`,
    `sorted_count`,
    `total_weight`,
    `trailer_id`,
    `dispatch_dock`,
    `scheduled_dispatch`,
    `created_at`,
    `updated_at`,
    `dispatched_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM sortation_batches_cdc_source;

-- ============================================
-- WALLING DATA PRODUCT - Bronze Layer
-- ============================================

-- Configure Kafka source for Wall Assignments CDC
CREATE TABLE wall_assignments_cdc_source (
    `_id` STRING,
    `assignment_id` STRING,
    `wall_id` STRING,
    `order_id` STRING,
    `wave_id` STRING,
    `bin_id` STRING,
    `worker_id` STRING,
    `status` STRING,
    `expected_items` INT,
    `placed_items` INT,
    `items` STRING,  -- JSON array
    `zone` STRING,
    `priority` INT,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `started_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.wall_assignments',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-wall-assignments',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Bronze Wall Assignments table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.wall_assignments_raw (
    `_id` STRING,
    `assignment_id` STRING,
    `wall_id` STRING,
    `order_id` STRING,
    `wave_id` STRING,
    `bin_id` STRING,
    `worker_id` STRING,
    `status` STRING,
    `expected_items` INT,
    `placed_items` INT,
    `items` STRING,
    `zone` STRING,
    `priority` INT,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `started_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer
INSERT INTO iceberg_catalog.bronze.wall_assignments_raw
SELECT
    `_id`,
    `assignment_id`,
    `wall_id`,
    `order_id`,
    `wave_id`,
    `bin_id`,
    `worker_id`,
    `status`,
    `expected_items`,
    `placed_items`,
    `items`,
    `zone`,
    `priority`,
    `created_at`,
    `updated_at`,
    `started_at`,
    `completed_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM wall_assignments_cdc_source;

-- ============================================
-- UNIT DATA PRODUCT - Bronze Layer
-- ============================================

-- Configure Kafka source for Units CDC
CREATE TABLE units_cdc_source (
    `_id` STRING,
    `unit_id` STRING,
    `sku` STRING,
    `seller_id` STRING,
    `tenant_id` STRING,
    `location` STRING,
    `status` STRING,
    `receipt_id` STRING,
    `order_id` STRING,
    `tote_id` STRING,
    `lot_number` STRING,
    `serial_number` STRING,
    `expiration_date` TIMESTAMP(3),
    `condition` STRING,
    `weight` DOUBLE,
    `dimensions` STRING,  -- JSON object
    `last_movement_at` TIMESTAMP(3),
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.units',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-units',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Bronze Units table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.units_raw (
    `_id` STRING,
    `unit_id` STRING,
    `sku` STRING,
    `seller_id` STRING,
    `tenant_id` STRING,
    `location` STRING,
    `status` STRING,
    `receipt_id` STRING,
    `order_id` STRING,
    `tote_id` STRING,
    `lot_number` STRING,
    `serial_number` STRING,
    `expiration_date` TIMESTAMP(3),
    `condition` STRING,
    `weight` DOUBLE,
    `dimensions` STRING,
    `last_movement_at` TIMESTAMP(3),
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer
INSERT INTO iceberg_catalog.bronze.units_raw
SELECT
    `_id`,
    `unit_id`,
    `sku`,
    `seller_id`,
    `tenant_id`,
    `location`,
    `status`,
    `receipt_id`,
    `order_id`,
    `tote_id`,
    `lot_number`,
    `serial_number`,
    `expiration_date`,
    `condition`,
    `weight`,
    `dimensions`,
    `last_movement_at`,
    `created_at`,
    `updated_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM units_cdc_source;

-- ============================================
-- PROCESS PATH DATA PRODUCT - Bronze Layer
-- ============================================

-- Configure Kafka source for Process Paths CDC
CREATE TABLE process_paths_cdc_source (
    `_id` STRING,
    `path_id` STRING,
    `order_id` STRING,
    `tenant_id` STRING,
    `path_type` STRING,
    `status` STRING,
    `handling_requirements` STRING,  -- JSON array
    `stations` STRING,  -- JSON array
    `estimated_duration` BIGINT,
    `actual_duration` BIGINT,
    `determined_at` TIMESTAMP(3),
    `override_reason` STRING,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.process_paths',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-process-paths',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Bronze Process Paths table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.process_paths_raw (
    `_id` STRING,
    `path_id` STRING,
    `order_id` STRING,
    `tenant_id` STRING,
    `path_type` STRING,
    `status` STRING,
    `handling_requirements` STRING,
    `stations` STRING,
    `estimated_duration` BIGINT,
    `actual_duration` BIGINT,
    `determined_at` TIMESTAMP(3),
    `override_reason` STRING,
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer
INSERT INTO iceberg_catalog.bronze.process_paths_raw
SELECT
    `_id`,
    `path_id`,
    `order_id`,
    `tenant_id`,
    `path_type`,
    `status`,
    `handling_requirements`,
    `stations`,
    `estimated_duration`,
    `actual_duration`,
    `determined_at`,
    `override_reason`,
    `created_at`,
    `updated_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM process_paths_cdc_source;

-- ============================================
-- WES DATA PRODUCT - Bronze Layer
-- ============================================

-- Configure Kafka source for WES Stages CDC
CREATE TABLE wes_stages_cdc_source (
    `_id` STRING,
    `stage_id` STRING,
    `order_id` STRING,
    `tenant_id` STRING,
    `stage_type` STRING,
    `status` STRING,
    `sequence_number` INT,
    `task_id` STRING,
    `worker_id` STRING,
    `station_id` STRING,
    `input_data` STRING,  -- JSON object
    `output_data` STRING,  -- JSON object
    `error_message` STRING,
    `retry_count` INT,
    `started_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `__op` STRING,
    `__source_ts_ms` BIGINT,
    `proctime` AS PROCTIME()
) WITH (
    'connector' = 'kafka',
    'topic' = 'cdc.wms.wes_stages',
    'properties.bootstrap.servers' = 'wms-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
    'properties.group.id' = 'flink-bronze-wes-stages',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'json',
    'json.ignore-parse-errors' = 'true'
);

-- Create Bronze WES Stages table
CREATE TABLE IF NOT EXISTS iceberg_catalog.bronze.wes_stages_raw (
    `_id` STRING,
    `stage_id` STRING,
    `order_id` STRING,
    `tenant_id` STRING,
    `stage_type` STRING,
    `status` STRING,
    `sequence_number` INT,
    `task_id` STRING,
    `worker_id` STRING,
    `station_id` STRING,
    `input_data` STRING,
    `output_data` STRING,
    `error_message` STRING,
    `retry_count` INT,
    `started_at` TIMESTAMP(3),
    `completed_at` TIMESTAMP(3),
    `created_at` TIMESTAMP(3),
    `updated_at` TIMESTAMP(3),
    `cdc_operation` STRING,
    `cdc_timestamp` BIGINT,
    `ingestion_time` TIMESTAMP(3),
    PRIMARY KEY (`_id`) NOT ENFORCED
) PARTITIONED BY (days(`ingestion_time`))
WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Insert CDC events into Bronze layer
INSERT INTO iceberg_catalog.bronze.wes_stages_raw
SELECT
    `_id`,
    `stage_id`,
    `order_id`,
    `tenant_id`,
    `stage_type`,
    `status`,
    `sequence_number`,
    `task_id`,
    `worker_id`,
    `station_id`,
    `input_data`,
    `output_data`,
    `error_message`,
    `retry_count`,
    `started_at`,
    `completed_at`,
    `created_at`,
    `updated_at`,
    `__op` AS cdc_operation,
    `__source_ts_ms` AS cdc_timestamp,
    CURRENT_TIMESTAMP AS ingestion_time
FROM wes_stages_cdc_source;
