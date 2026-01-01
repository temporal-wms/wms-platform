-- Orders by Customer
-- Usage: Replace 'CUST-XXXXX' with the customer ID
-- Returns: All orders for a specific customer

SELECT
    o.order_id,
    o.status,
    o.priority,
    o.created_at,
    o.updated_at,
    o.promised_delivery_at,
    o.total_items,
    o.total_weight
FROM mongodb.wms.orders o
WHERE o.customer_id = 'CUST-XXXXX'  -- Replace with customer ID
ORDER BY o.created_at DESC
LIMIT 100
