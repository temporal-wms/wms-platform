import { useMemo } from 'react';
import type { Order } from '../types';

interface OrderTableProps {
  orders: Order[];
  isLoading: boolean;
  error: Error | null;
  onRefetch: () => void;
  selectedOrders: string[];
  onSelectionChange: (orders: string[]) => void;
  priorityFilter: string[];
  stateFilter: string[];
}

export function OrderTable({
  orders,
  isLoading,
  error,
  onRefetch,
  selectedOrders,
  onSelectionChange,
  priorityFilter,
  stateFilter,
}: OrderTableProps) {
  const filteredOrders = useMemo(() => {
    return orders.filter(order => {
      const matchesPriority = priorityFilter.length === 0 ||
        priorityFilter.includes(order.priority);
      const matchesState = stateFilter.length === 0 ||
        stateFilter.includes(order.shipToState);
      return matchesPriority && matchesState;
    });
  }, [orders, priorityFilter, stateFilter]);

  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      onSelectionChange(filteredOrders.map(o => o.orderId));
    } else {
      onSelectionChange([]);
    }
  };

  const handleSelectOrder = (orderId: string, checked: boolean) => {
    if (checked) {
      onSelectionChange([...selectedOrders, orderId]);
    } else {
      onSelectionChange(selectedOrders.filter(id => id !== orderId));
    }
  };

  if (isLoading) return <div className="loading">Loading orders...</div>;
  if (error) return (
    <div className="error">
      <p>Error loading orders. Make sure the order-service is running.</p>
      <button onClick={() => onRefetch()} className="retry-btn">Retry</button>
    </div>
  );

  const allSelected = filteredOrders.length > 0 &&
    filteredOrders.every(o => selectedOrders.includes(o.orderId));

  if (orders.length === 0) {
    return (
      <div className="empty-state">
        <p>No validated orders available for waving.</p>
        <button onClick={() => onRefetch()} className="retry-btn">Refresh</button>
      </div>
    );
  }

  if (filteredOrders.length === 0) {
    return (
      <div className="empty-state">
        <p>No orders match the current filters.</p>
      </div>
    );
  }

  return (
    <div className="table-container">
      <table className="order-table">
        <thead>
          <tr>
            <th>
              <input
                type="checkbox"
                checked={allSelected}
                onChange={(e) => handleSelectAll(e.target.checked)}
              />
            </th>
            <th>Order ID</th>
            <th>Customer</th>
            <th>Priority</th>
            <th>Items</th>
            <th>Weight</th>
            <th>Ship To</th>
            <th>Promised Delivery</th>
          </tr>
        </thead>
        <tbody>
          {filteredOrders.map((order: Order) => (
            <tr key={order.orderId} className={selectedOrders.includes(order.orderId) ? 'selected' : ''}>
              <td>
                <input
                  type="checkbox"
                  checked={selectedOrders.includes(order.orderId)}
                  onChange={(e) => handleSelectOrder(order.orderId, e.target.checked)}
                />
              </td>
              <td className="order-id">{order.orderId}</td>
              <td>{order.customerId}</td>
              <td>
                <span className={`priority-badge priority-${order.priority}`}>
                  {order.priority.replace('_', ' ')}
                </span>
              </td>
              <td>{order.totalItems}</td>
              <td>{order.totalWeight?.toFixed(2) || '0.00'} kg</td>
              <td>{order.shipToCity}, {order.shipToState}</td>
              <td>{new Date(order.promisedDeliveryAt).toLocaleDateString()}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
