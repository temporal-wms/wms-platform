-- Route Performance by Zone Query for Apache Superset
-- For Routing Dashboard: Analyze picking efficiency across warehouse zones

-- ============================================
-- Query 1: Zone Performance Summary (from Gold)
-- Primary query for dashboard
-- ============================================
SELECT
    `date`,
    zone,
    total_routes,
    completed_routes,
    completion_rate,
    total_items_picked,
    avg_items_per_route,
    avg_stops_per_route,
    avg_distance_per_route_m,
    avg_duration_minutes,
    p50_duration_minutes,
    p95_duration_minutes,
    avg_pick_rate,
    unique_pickers,
    routes_per_picker,
    avg_efficiency_ratio,
    multi_route_count
FROM iceberg.gold.route_performance_by_zone_daily
WHERE `date` >= CURRENT_DATE - INTERVAL '{{ time_range | default(7) }}' DAY
  {% if zone_filter %}
  AND zone = '{{ zone_filter }}'
  {% endif %}
ORDER BY `date` DESC, total_routes DESC;

-- ============================================
-- Query 2: Zone Comparison (aggregated)
-- ============================================
/*
SELECT
    zone,
    SUM(total_routes) AS total_routes,
    SUM(completed_routes) AS completed_routes,
    SUM(total_items_picked) AS total_items_picked,
    AVG(avg_duration_minutes) AS avg_duration_minutes,
    AVG(avg_pick_rate) AS avg_pick_rate,
    AVG(completion_rate) AS avg_completion_rate,
    SUM(unique_pickers) AS unique_pickers
FROM iceberg.gold.route_performance_by_zone_daily
WHERE `date` >= CURRENT_DATE - INTERVAL '{{ time_range | default(7) }}' DAY
GROUP BY zone
ORDER BY total_routes DESC;
*/

-- ============================================
-- Query 3: Time Series by Zone
-- For trend analysis
-- ============================================
/*
SELECT
    `date`,
    zone,
    total_routes,
    avg_duration_minutes,
    avg_pick_rate,
    completion_rate
FROM iceberg.gold.route_performance_by_zone_daily
WHERE `date` >= CURRENT_DATE - INTERVAL '{{ time_range | default(30) }}' DAY
ORDER BY `date` ASC, zone;
*/

-- ============================================
-- Query 4: Picker Performance by Zone
-- ============================================
/*
SELECT
    `date`,
    picker_id,
    zone,
    routes_completed,
    total_items_picked,
    items_per_hour,
    avg_efficiency_ratio
FROM iceberg.gold.picker_route_performance_daily
WHERE `date` >= CURRENT_DATE - INTERVAL '{{ time_range | default(7) }}' DAY
  {% if zone_filter %}
  AND zone = '{{ zone_filter }}'
  {% endif %}
ORDER BY items_per_hour DESC;
*/
