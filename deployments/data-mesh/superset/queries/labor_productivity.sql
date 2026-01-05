-- Labor Productivity Query
-- Provides worker productivity metrics for the Labor Performance dashboard

-- Worker productivity by zone and shift
SELECT
    date,
    worker_id,
    worker_name,
    zone,
    shift_type,
    tasks_completed,
    items_processed,
    total_work_hours,
    tasks_per_hour,
    items_per_hour,
    accuracy_rate,
    exceptions_count
FROM iceberg.gold.labor_productivity_daily
WHERE date >= {{ time_range.start }}
  AND date <= {{ time_range.end }}
  {% if zone %}AND zone = '{{ zone }}'{% endif %}
  {% if shift_type %}AND shift_type = '{{ shift_type }}'{% endif %}
  {% if worker_id %}AND worker_id = '{{ worker_id }}'{% endif %}
ORDER BY date DESC, items_per_hour DESC;

-- Top performers query
-- SELECT
--     worker_id,
--     worker_name,
--     zone,
--     AVG(items_per_hour) as avg_items_per_hour,
--     AVG(accuracy_rate) as avg_accuracy,
--     SUM(tasks_completed) as total_tasks,
--     SUM(items_processed) as total_items
-- FROM iceberg.gold.labor_productivity_daily
-- WHERE date >= CURRENT_DATE - INTERVAL '7' DAY
-- GROUP BY worker_id, worker_name, zone
-- ORDER BY avg_items_per_hour DESC
-- LIMIT 10;

-- Labor utilization summary
-- SELECT
--     date,
--     zone,
--     shift_type,
--     total_workers,
--     active_workers,
--     utilization_rate,
--     total_tasks_completed,
--     avg_tasks_per_worker,
--     productivity_rate
-- FROM iceberg.gold.labor_utilization_daily
-- WHERE date >= {{ time_range.start }}
-- ORDER BY date DESC, zone;
