import React from 'react';
import { Routes, Route, useNavigate } from 'react-router-dom';
import { OrderList } from './components/OrderList';
import { OrderDetails } from './components/OrderDetails';
import { CreateOrder } from './components/CreateOrder';
import { DLQOrders } from './components/DLQOrders';

function App() {
  return (
    <div className="orders-app">
      <Routes>
        <Route path="/" element={<OrderList />} />
        <Route path="/new" element={<CreateOrder />} />
        <Route path="/dlq" element={<DLQOrders />} />
        <Route path="/:orderId" element={<OrderDetails />} />
      </Routes>
    </div>
  );
}

export default App;
