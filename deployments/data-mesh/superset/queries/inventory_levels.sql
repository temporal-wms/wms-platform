-- Inventory Levels Query
-- Provides inventory analytics for the Inventory Analytics dashboard

-- Current inventory by zone
SELECT
    sku,
    product_name,
    total_quantity,
    reserved_quantity,
    available_quantity,
    reorder_point,
    CASE
        WHEN available_quantity <= reorder_point THEN 'Low Stock'
        WHEN available_quantity <= reorder_point * 1.5 THEN 'Medium'
        ELSE 'Healthy'
    END as stock_status,
    CASE
        WHEN available_quantity = 0 THEN 'Out of Stock'
        WHEN available_quantity <= reorder_point THEN 'Reorder Required'
        ELSE 'OK'
    END as alert_level,
    updated_at
FROM iceberg.silver.inventory_current
{% if sku %}WHERE sku LIKE '%{{ sku }}%'{% endif %}
ORDER BY
    CASE WHEN available_quantity <= reorder_point THEN 0 ELSE 1 END,
    available_quantity ASC;

-- Inventory metrics daily
-- SELECT
--     date,
--     sku,
--     product_name,
--     starting_quantity,
--     ending_quantity,
--     quantity_received,
--     quantity_picked,
--     quantity_adjusted,
--     turnover_rate,
--     days_of_supply,
--     is_low_stock
-- FROM iceberg.gold.inventory_metrics_daily
-- WHERE date >= {{ time_range.start }}
--   AND date <= {{ time_range.end }}
-- ORDER BY date DESC, turnover_rate DESC;

-- Velocity class distribution
-- SELECT
--     velocity_class,
--     COUNT(*) as sku_count,
--     SUM(total_quantity) as total_units,
--     AVG(turnover_rate) as avg_turnover
-- FROM iceberg.silver.inventory_current
-- GROUP BY velocity_class
-- ORDER BY velocity_class;
