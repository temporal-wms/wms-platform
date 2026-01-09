-- Operations KPI Query
-- Provides executive-level metrics for the Operations KPI dashboard

-- Daily operations summary
SELECT
    date,
    -- Orders
    orders_received,
    orders_completed,
    orders_cancelled,
    order_completion_rate * 100 as order_completion_rate_pct,
    avg_order_fulfillment_hours,
    -- Picking
    pick_tasks_completed,
    items_picked,
    avg_pick_rate,
    picking_accuracy * 100 as picking_accuracy_pct,
    -- Packing
    packages_packed,
    avg_pack_time_minutes,
    -- Shipping
    shipments_sent,
    on_time_shipments,
    shipping_on_time_rate * 100 as shipping_on_time_rate_pct,
    -- Receiving
    receipts_completed,
    units_received,
    avg_dock_to_stock_minutes,
    -- Returns
    returns_processed,
    return_restock_rate * 100 as return_restock_rate_pct,
    -- Labor
    active_workers,
    avg_productivity,
    -- Inventory
    low_stock_alerts
FROM iceberg.gold.operations_summary_daily
WHERE date >= {{ time_range.start }}
  AND date <= {{ time_range.end }}
ORDER BY date DESC;

-- Hourly throughput for today
-- SELECT
--     date,
--     hour,
--     orders_received,
--     orders_completed,
--     items_picked,
--     packages_packed,
--     shipments_sent,
--     units_received,
--     active_workers
-- FROM iceberg.gold.hourly_throughput
-- WHERE date = CURRENT_DATE
-- ORDER BY hour;

-- Real-time operations snapshot
-- SELECT
--     snapshot_time,
--     active_orders,
--     active_waves,
--     active_pick_tasks,
--     active_pack_tasks,
--     pending_shipments,
--     active_workers,
--     orders_last_hour,
--     shipments_last_hour
-- FROM iceberg.gold.active_operations_snapshot
-- ORDER BY snapshot_time DESC
-- LIMIT 1;

-- Stage bottleneck analysis
-- SELECT
--     'Validation' as stage, COUNT(*) as count FROM iceberg.silver.orders_current WHERE status = 'received' AND is_deleted = FALSE
-- UNION ALL
-- SELECT
--     'Wave Assignment' as stage, COUNT(*) FROM iceberg.silver.orders_current WHERE status = 'validated' AND is_deleted = FALSE
-- UNION ALL
-- SELECT
--     'Picking' as stage, COUNT(*) FROM iceberg.silver.orders_current WHERE status = 'wave_assigned' AND is_deleted = FALSE
-- UNION ALL
-- SELECT
--     'Consolidation' as stage, COUNT(*) FROM iceberg.silver.orders_current WHERE status = 'picking' AND is_deleted = FALSE
-- UNION ALL
-- SELECT
--     'Packing' as stage, COUNT(*) FROM iceberg.silver.orders_current WHERE status = 'consolidating' AND is_deleted = FALSE
-- UNION ALL
-- SELECT
--     'Shipping' as stage, COUNT(*) FROM iceberg.silver.orders_current WHERE status = 'packing' AND is_deleted = FALSE;
