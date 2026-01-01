-- Orders by Special Requirements - Bar Chart Query
-- For use in Apache Superset Orders Dashboard
-- Queries the Gold layer via Trino

-- Historical aggregation from Gold layer (recommended for dashboards)
SELECT
    requirement,
    SUM(order_count) as total_orders,
    AVG(percentage_of_total) as avg_percentage
FROM iceberg.gold.orders_by_requirements_daily
WHERE `date` >= CURRENT_DATE - INTERVAL '7' DAY
GROUP BY requirement
ORDER BY total_orders DESC;

-- ============================================
-- Alternative: Real-time query from MongoDB
-- Use this for current-day real-time data
-- ============================================
/*
WITH order_requirements AS (
    SELECT
        o._id as order_id,
        o.gift_wrap,
        o.process_requirements
    FROM mongodb.wms.orders o
    WHERE o.created_at >= CURRENT_DATE
)
SELECT
    requirement,
    COUNT(DISTINCT order_id) as order_count
FROM (
    -- Gift Wrap
    SELECT order_id, 'gift_wrap' as requirement
    FROM order_requirements WHERE gift_wrap = true

    UNION ALL

    -- Multi-Item (based on items array length)
    SELECT order_id, 'multi_item' as requirement
    FROM order_requirements
    WHERE JSON_ARRAY_LENGTH(JSON_QUERY(process_requirements, '$.requirements')) > 1

    UNION ALL

    -- Single-Item
    SELECT order_id, 'single_item' as requirement
    FROM order_requirements
    WHERE JSON_ARRAY_LENGTH(JSON_QUERY(process_requirements, '$.requirements')) = 1
) exploded
GROUP BY requirement
ORDER BY order_count DESC;
*/

-- ============================================
-- Time-series view for trend analysis
-- ============================================
/*
SELECT
    `date`,
    requirement,
    order_count,
    percentage_of_total
FROM iceberg.gold.orders_by_requirements_daily
WHERE `date` >= CURRENT_DATE - INTERVAL '30' DAY
ORDER BY `date` DESC, order_count DESC;
*/
