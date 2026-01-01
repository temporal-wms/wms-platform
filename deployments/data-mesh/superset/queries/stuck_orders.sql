-- Stuck Orders
-- Finds orders that haven't progressed in over 1 hour
-- Useful for identifying workflow bottlenecks

SELECT
    o.order_id,
    o.customer_id,
    o.status,
    o.priority,
    o.created_at,
    o.updated_at,
    o.promised_delivery_at,
    date_diff('minute', o.updated_at, current_timestamp) as minutes_stuck
FROM mongodb.wms.orders o
WHERE o.status NOT IN ('shipped', 'delivered', 'cancelled')
  AND o.updated_at < current_timestamp - interval '1' hour
ORDER BY o.updated_at ASC
LIMIT 100
