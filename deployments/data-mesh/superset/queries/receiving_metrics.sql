-- Receiving Metrics Query
-- Provides inbound operations data for the Receiving dashboard

-- Receiving metrics by dock door
SELECT
    date,
    dock_door,
    total_receipts,
    completed_receipts,
    cancelled_receipts,
    completion_rate * 100 as completion_rate_pct,
    total_units_expected,
    total_units_received,
    total_units_damaged,
    receiving_accuracy,
    avg_dock_to_stock_minutes,
    p50_dock_to_stock_minutes,
    p95_dock_to_stock_minutes,
    avg_unloading_minutes,
    avg_inspection_minutes,
    on_time_arrivals,
    late_arrivals,
    on_time_rate * 100 as on_time_rate_pct,
    unique_vendors
FROM iceberg.gold.receiving_metrics_daily
WHERE date >= {{ time_range.start }}
  AND date <= {{ time_range.end }}
  {% if dock_door %}AND dock_door = '{{ dock_door }}'{% endif %}
ORDER BY date DESC, dock_door;

-- Vendor performance
-- SELECT
--     date,
--     vendor_id,
--     vendor_name,
--     total_receipts,
--     total_units,
--     units_received,
--     units_damaged,
--     damage_rate,
--     on_time_receipts,
--     late_receipts,
--     on_time_rate * 100 as on_time_rate_pct,
--     avg_arrival_variance_minutes,
--     avg_dock_to_stock_minutes
-- FROM iceberg.gold.vendor_performance_daily
-- WHERE date >= {{ time_range.start }}
--   AND date <= {{ time_range.end }}
--   {% if vendor_id %}AND vendor_id = '{{ vendor_id }}'{% endif %}
-- ORDER BY date DESC, total_units DESC;

-- Active receipts from silver
-- SELECT
--     receipt_id,
--     po_number,
--     vendor_name,
--     dock_door,
--     status,
--     total_units,
--     received_units,
--     damaged_units,
--     receiving_accuracy,
--     expected_at,
--     arrived_at,
--     dock_to_stock_minutes,
--     worker_id
-- FROM iceberg.silver.receipts_current
-- WHERE status NOT IN ('completed', 'cancelled')
--   AND is_deleted = FALSE
-- ORDER BY expected_at ASC;
