-- Orders by Status
-- Usage: Change the status value to filter by different workflow stages
-- Valid statuses: created, validated, waved, picking, picked, consolidating,
--                 consolidated, packing, packed, shipping, shipped, delivered, failed

SELECT
    o.order_id,
    o.customer_id,
    o.status,
    o.priority,
    o.created_at,
    o.updated_at,
    o.promised_delivery_at,
    o.total_items
FROM mongodb.wms.orders o
WHERE o.status = 'validated'  -- Change status here
ORDER BY o.created_at DESC
LIMIT 100
