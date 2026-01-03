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
