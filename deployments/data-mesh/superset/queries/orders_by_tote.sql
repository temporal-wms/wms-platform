-- Orders by Tote Lookup Query - Table Query
-- For use in Apache Superset Consolidation Dashboard
-- Queries MongoDB directly for real-time tote lookup

-- ============================================
-- Query 1: Orders by Tote (Real-time from MongoDB)
-- Primary query for tote lookup
-- Replace {{ tote_id }} with Jinja template parameter in Superset
-- ============================================
SELECT
    c.order_id,
    c.consolidation_id,
    c.status AS consolidation_status,
    c.station,
    c.worker_id,
    c.total_expected,
    c.total_consolidated,
    c.ready_for_packing,
    c.created_at,
    c.updated_at
FROM mongodb.wms.consolidations c
WHERE ARRAY_CONTAINS(c.source_totes, '{{ tote_id }}')
  {% if from_date %}
  AND c.created_at >= TIMESTAMP '{{ from_date }}'
  {% endif %}
  {% if to_date %}
  AND c.created_at <= TIMESTAMP '{{ to_date }}'
  {% endif %}
ORDER BY c.created_at DESC;

-- ============================================
-- Query 2: Items in Tote Detail
-- Detailed view of items from a specific tote
-- ============================================
/*
SELECT
    c.order_id,
    c.consolidation_id,
    c.station,
    item.sku,
    item.product_name,
    item.quantity,
    item.received,
    item.status AS item_status,
    item.source_tote_id
FROM mongodb.wms.consolidations c
CROSS JOIN UNNEST(c.expected_items) AS t(item)
WHERE item.source_tote_id = '{{ tote_id }}'
  {% if from_date %}
  AND c.created_at >= TIMESTAMP '{{ from_date }}'
  {% endif %}
ORDER BY c.order_id, item.sku;
*/

-- ============================================
-- Query 3: From Silver Layer (alternative)
-- Use when fresher data is not critical
-- ============================================
/*
SELECT
    c.order_id,
    c.consolidation_id,
    c.status AS consolidation_status,
    c.station,
    c.worker_id,
    c.total_expected,
    c.total_consolidated,
    c.consolidation_rate,
    c.ready_for_packing,
    c.created_at
FROM iceberg.silver.consolidations_current c
WHERE ARRAY_CONTAINS(c.source_totes, '{{ tote_id }}')
  AND c.is_deleted = FALSE
ORDER BY c.created_at DESC;
*/

-- ============================================
-- Query 4: Aggregated metrics for tote
-- ============================================
/*
SELECT
    COUNT(DISTINCT order_id) AS order_count,
    SUM(total_expected) AS total_items,
    SUM(total_consolidated) AS consolidated_items,
    SUM(total_consolidated) * 100.0 / NULLIF(SUM(total_expected), 0) AS consolidation_progress,
    COUNT(CASE WHEN ready_for_packing = TRUE THEN 1 END) AS ready_for_packing_count
FROM mongodb.wms.consolidations
WHERE ARRAY_CONTAINS(source_totes, '{{ tote_id }}');
*/
