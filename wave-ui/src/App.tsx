import { QueryClient, QueryClientProvider, useQuery } from '@tanstack/react-query';
import { useState, useMemo } from 'react';
import { Header } from './components/Header';
import { OrderTable } from './components/OrderTable';
import { CreateWaveButton } from './components/CreateWaveButton';
import { FilterBar } from './components/FilterBar';
import { fetchValidatedOrders } from './api/orderService';
import './App.css';

const queryClient = new QueryClient();

function AppContent() {
  const [selectedOrders, setSelectedOrders] = useState<string[]>([]);
  const [priorityFilter, setPriorityFilter] = useState<string[]>([]);
  const [stateFilter, setStateFilter] = useState<string[]>([]);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['validatedOrders'],
    queryFn: fetchValidatedOrders,
    refetchInterval: 30000,
  });

  const orders = data?.data || [];

  const availableStates = useMemo(() => {
    const states = new Set(orders.map(o => o.shipToState).filter(Boolean));
    return Array.from(states).sort();
  }, [orders]);

  return (
    <div className="app">
      <Header />
      <main className="main-content">
        <div className="actions-bar">
          <h2>Validated Orders</h2>
          <CreateWaveButton
            selectedOrders={selectedOrders}
            onSuccess={() => setSelectedOrders([])}
          />
        </div>
        <FilterBar
          priorityFilter={priorityFilter}
          onPriorityChange={setPriorityFilter}
          stateFilter={stateFilter}
          onStateChange={setStateFilter}
          availableStates={availableStates}
        />
        <OrderTable
          orders={orders}
          isLoading={isLoading}
          error={error}
          onRefetch={refetch}
          selectedOrders={selectedOrders}
          onSelectionChange={setSelectedOrders}
          priorityFilter={priorityFilter}
          stateFilter={stateFilter}
        />
      </main>
    </div>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AppContent />
    </QueryClientProvider>
  );
}

export default App;
