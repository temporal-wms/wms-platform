-- Wave Metrics Query
-- Provides wave performance data for the Wave Performance dashboard

-- Wave performance daily
SELECT
    date,
    wave_type,
    waves_created,
    waves_completed,
    waves_cancelled,
    CAST(waves_completed AS DOUBLE) / NULLIF(waves_created, 0) * 100 as completion_rate,
    avg_orders_per_wave,
    avg_items_per_wave,
    avg_completion_time_hours
FROM iceberg.gold.wave_performance_daily
WHERE date >= {{ time_range.start }}
  AND date <= {{ time_range.end }}
  {% if wave_type %}AND wave_type = '{{ wave_type }}'{% endif %}
ORDER BY date DESC;

-- Wave details from silver layer
-- SELECT
--     wave_id,
--     wave_type,
--     status,
--     CARDINALITY(orders) as order_count,
--     fulfillment_mode,
--     pickers_required,
--     pickers_assigned,
--     created_at,
--     scheduled_at,
--     released_at,
--     completed_at,
--     TIMESTAMPDIFF(HOUR, released_at, completed_at) as cycle_time_hours
-- FROM iceberg.silver.waves_current
-- WHERE created_at >= {{ time_range.start }}
--   AND is_deleted = FALSE
-- ORDER BY created_at DESC;

-- Wave type distribution
-- SELECT
--     wave_type,
--     COUNT(*) as wave_count,
--     AVG(CARDINALITY(orders)) as avg_orders,
--     SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
--     SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END) as cancelled
-- FROM iceberg.silver.waves_current
-- WHERE created_at >= CURRENT_DATE - INTERVAL '7' DAY
-- GROUP BY wave_type
-- ORDER BY wave_count DESC;
