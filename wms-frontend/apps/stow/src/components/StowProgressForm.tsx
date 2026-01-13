import React, { useState } from 'react';
import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query';
import { stowClient } from '@wms/api-client';
import type { PutawayTask } from '@wms/types';
import { Card, CardHeader, CardContent, Button, PageLoading, EmptyState, Input, Badge } from '@wms/ui';
import { MapPin, User, Scan, CheckCircle } from 'lucide-react';

interface StowProgressFormProps {
  taskId: string;
  task: PutawayTask;
  targetLocationId?: string;
}

export function StowProgressForm({ taskId, task, targetLocationId }: StowProgressFormProps) {
  const queryClient = useQueryClient();
  const [quantity, setQuantity] = useState<number>(1);
  const [selectedLocation, setSelectedLocation] = useState(targetLocationId || '');
  const [notes, setNotes] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const stowMutation = useMutation({
    mutationFn: (data: { locationId: string; qty: number }) => 
      stowClient.stowItem(taskId, data.locationId, data.qty),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['stow-task', taskId] });
      queryClient.invalidateQueries({ queryKey: ['stow-tasks'] });
      setQuantity(1);
      setSelectedLocation('');
      setNotes('');
      setIsSubmitting(false);
    },
    onError: () => {
      setIsSubmitting(false);
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedLocation) {
      alert('Please select a target location');
      return;
    }
    setIsSubmitting(true);
    stowMutation.mutate({ locationId: selectedLocation, qty: quantity });
  };

  const remainingQty = task.quantity - ((task.stowedQuantity || 0) + quantity);

  return (
    <Card>
      <CardHeader title="Record Stow Progress" />
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">
              Item to Stow:
            </label>
            <div className="flex items-center gap-2 p-3 bg-blue-50 rounded-lg">
              <div className="flex-1">
                <div className="font-mono font-semibold text-gray-900">{task.sku}</div>
                <div className="text-sm text-gray-600">- {task.productName}</div>
              </div>
              <Badge variant="neutral">Qty: {task.quantity}</Badge>
            </div>

            <div className="flex items-center gap-2 mt-3">
              <div className="text-sm text-gray-600">
                <div>Remaining to stow:</div>
                <div className="text-lg font-semibold text-gray-900">{remainingQty}</div>
              </div>
            </div>
          </div>

          <div className="space-y-3">
            <label className="block text-sm font-medium text-gray-700">
              Stow Quantity:
            </label>
            <div className="flex items-center gap-3">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setQuantity(Math.max(1, quantity - 1))}
                disabled={isSubmitting || quantity <= 1}
              >
                −
              </Button>
              <Input
                type="number"
                min="1"
                max={remainingQty}
                value={quantity}
                onChange={(e) => setQuantity(parseInt(e.target.value) || 1)}
                className="w-20 text-center font-semibold"
                disabled={isSubmitting}
                required
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setQuantity(quantity + 1)}
                disabled={isSubmitting || quantity >= remainingQty}
              >
                +
              </Button>
            </div>
            <p className="text-xs text-gray-500 mt-1">
              Max: {remainingQty}
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Target Location:
            </label>
            <Input
              placeholder="e.g., A-01-02-03"
              value={selectedLocation}
              onChange={(e) => setSelectedLocation(e.target.value)}
              disabled={isSubmitting}
              required
              className="mb-2"
            />
            <div className="flex items-center gap-2">
              <Scan className="h-4 w-4 text-gray-400" />
              <span className="text-sm text-gray-600">
                Scan location or enter manually
              </span>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Stow Notes (Optional):
            </label>
            <textarea
              placeholder="Any notes about this stow..."
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              disabled={isSubmitting}
              rows={2}
              className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent disabled:bg-gray-100 disabled:text-gray-500"
            />
          </div>

          <div className="flex gap-3 pt-4">
            <Button variant="outline" type="button" disabled={isSubmitting}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? 'Stowing...' : 'Stow Items'}
              <CheckCircle className="h-4 w-4 mr-2" />
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
