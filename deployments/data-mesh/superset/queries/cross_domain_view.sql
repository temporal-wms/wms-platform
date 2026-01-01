-- Cross-Domain Order View
-- Joins orders with waves and shipments for complete order lifecycle view
-- Note: This query may be slower due to cross-collection joins

SELECT
    o.order_id,
    o.customer_id,
    o.status as order_status,
    o.priority,
    o.created_at as order_created,
    o.updated_at as order_updated,
    o.promised_delivery_at,
    o.total_items,
    w.wave_id,
    w.status as wave_status,
    w.zone,
    s.shipment_id,
    s.tracking_number,
    s.carrier_code,
    s.status as shipping_status,
    s.shipped_at
FROM mongodb.wms.orders o
LEFT JOIN mongodb.wms.waves w ON w.order_id = o.order_id
LEFT JOIN mongodb.wms.shipments s ON s.order_id = o.order_id
WHERE o.created_at >= current_date - interval '7' day
ORDER BY o.created_at DESC
LIMIT 500
