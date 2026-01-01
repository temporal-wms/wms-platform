import { useMutation, useQueryClient } from '@tanstack/react-query';
import { createWaveFromOrders } from '../api/wavingService';

interface CreateWaveButtonProps {
  selectedOrders: string[];
  onSuccess: () => void;
}

export function CreateWaveButton({ selectedOrders, onSuccess }: CreateWaveButtonProps) {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: createWaveFromOrders,
    onSuccess: (data) => {
      const message = data.failedOrders.length > 0
        ? `Wave ${data.wave.waveId} created with ${data.wave.orderCount} orders. ${data.failedOrders.length} orders failed.`
        : `Wave ${data.wave.waveId} created successfully with ${data.wave.orderCount} orders!`;
      alert(message);
      queryClient.invalidateQueries({ queryKey: ['validatedOrders'] });
      onSuccess();
    },
    onError: (error: Error) => {
      alert(`Failed to create wave: ${error.message}`);
    },
  });

  const handleClick = () => {
    if (selectedOrders.length === 0) {
      alert('Please select at least one order');
      return;
    }

    mutation.mutate({
      orderIds: selectedOrders,
      waveType: 'digital',
      fulfillmentMode: 'wave',
      zone: 'default',
    });
  };

  return (
    <button
      className="create-wave-btn"
      onClick={handleClick}
      disabled={selectedOrders.length === 0 || mutation.isPending}
    >
      {mutation.isPending
        ? 'Creating Wave...'
        : `Create Wave (${selectedOrders.length} orders)`}
    </button>
  );
}
