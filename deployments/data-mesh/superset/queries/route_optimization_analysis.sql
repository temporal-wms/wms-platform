-- Route Optimization Analysis Query for Apache Superset
-- For Routing Dashboard: Identify routes needing optimization

-- ============================================
-- Query 1: Optimization Summary by Complexity
-- ============================================
SELECT
    `date`,
    route_complexity,
    total_routes,
    avg_stops,
    avg_zones_visited,
    avg_distance_m,
    avg_estimated_time_min,
    avg_actual_time_min,
    avg_efficiency_ratio,
    routes_exceeding_time_estimate,
    routes_with_multi_zone
FROM iceberg.gold.route_optimization_daily
WHERE `date` >= CURRENT_DATE - INTERVAL '{{ time_range | default(7) }}' DAY
ORDER BY `date` DESC, route_complexity;

-- ============================================
-- Query 2: Routes Needing Optimization
-- High optimization score routes
-- ============================================
/*
SELECT
    route_id,
    order_id,
    `date`,
    zone,
    strategy,
    stop_count,
    zone_count,
    estimated_distance_m,
    actual_distance_m,
    estimated_time_min,
    actual_time_min,
    efficiency_ratio,
    items_per_stop,
    optimization_score,
    has_high_zone_count,
    is_inefficient,
    is_very_inefficient,
    is_long_distance,
    has_many_stops,
    distance_exceeded
FROM iceberg.gold.route_optimization_candidates
WHERE `date` >= CURRENT_DATE - INTERVAL '{{ time_range | default(7) }}' DAY
  AND optimization_score >= {{ min_score | default(3) }}
  {% if zone_filter %}
  AND zone = '{{ zone_filter }}'
  {% endif %}
ORDER BY optimization_score DESC, `date` DESC
LIMIT 100;
*/

-- ============================================
-- Query 3: Optimization Flags Distribution
-- ============================================
/*
SELECT
    'high_zone_count' AS flag,
    COUNT(CASE WHEN has_high_zone_count THEN 1 END) AS route_count
FROM iceberg.gold.route_optimization_candidates
WHERE `date` >= CURRENT_DATE - INTERVAL '{{ time_range | default(7) }}' DAY

UNION ALL

SELECT
    'inefficient' AS flag,
    COUNT(CASE WHEN is_inefficient THEN 1 END) AS route_count
FROM iceberg.gold.route_optimization_candidates
WHERE `date` >= CURRENT_DATE - INTERVAL '{{ time_range | default(7) }}' DAY

UNION ALL

SELECT
    'very_inefficient' AS flag,
    COUNT(CASE WHEN is_very_inefficient THEN 1 END) AS route_count
FROM iceberg.gold.route_optimization_candidates
WHERE `date` >= CURRENT_DATE - INTERVAL '{{ time_range | default(7) }}' DAY

UNION ALL

SELECT
    'long_distance' AS flag,
    COUNT(CASE WHEN is_long_distance THEN 1 END) AS route_count
FROM iceberg.gold.route_optimization_candidates
WHERE `date` >= CURRENT_DATE - INTERVAL '{{ time_range | default(7) }}' DAY

UNION ALL

SELECT
    'many_stops' AS flag,
    COUNT(CASE WHEN has_many_stops THEN 1 END) AS route_count
FROM iceberg.gold.route_optimization_candidates
WHERE `date` >= CURRENT_DATE - INTERVAL '{{ time_range | default(7) }}' DAY

UNION ALL

SELECT
    'distance_exceeded' AS flag,
    COUNT(CASE WHEN distance_exceeded THEN 1 END) AS route_count
FROM iceberg.gold.route_optimization_candidates
WHERE `date` >= CURRENT_DATE - INTERVAL '{{ time_range | default(7) }}' DAY

ORDER BY route_count DESC;
*/

-- ============================================
-- Query 4: Strategy Efficiency Comparison
-- ============================================
/*
SELECT
    r.strategy,
    COUNT(*) AS total_routes,
    AVG(r.stop_count) AS avg_stops,
    AVG(r.zone_count) AS avg_zones,
    AVG(r.efficiency_ratio) AS avg_efficiency,
    AVG(r.actual_time_min) AS avg_duration_min,
    COUNT(CASE WHEN r.optimization_score >= 3 THEN 1 END) AS problematic_routes
FROM iceberg.gold.route_optimization_candidates r
WHERE `date` >= CURRENT_DATE - INTERVAL '{{ time_range | default(7) }}' DAY
GROUP BY r.strategy
ORDER BY avg_efficiency ASC;
*/
