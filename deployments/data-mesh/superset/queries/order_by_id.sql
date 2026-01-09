-- Order Lookup by ID
-- Usage: Replace 'ORD-XXXXX' with the order ID you want to search
-- Returns: Full order details with wave and shipping information

SELECT
    o._id as mongo_id,
    o.order_id,
    o.customer_id,
    o.status,
    o.priority,
    o.created_at,
    o.updated_at,
    o.promised_delivery_at,
    o.items,
    o.total_items,
    o.total_weight
FROM mongodb.wms.orders o
WHERE o.order_id = 'ORD-XXXXX'  -- Replace with your order ID
