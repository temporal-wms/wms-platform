-- Order Statistics Summary
-- Provides overview of order counts by status and priority
-- Good for dashboard widgets

SELECT
    o.status,
    o.priority,
    count(*) as order_count,
    min(o.created_at) as oldest_order,
    max(o.created_at) as newest_order
FROM mongodb.wms.orders o
WHERE o.created_at >= current_date - interval '24' hour
GROUP BY o.status, o.priority
ORDER BY order_count DESC
