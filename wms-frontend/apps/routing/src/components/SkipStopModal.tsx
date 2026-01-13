import React, { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { routingClient } from '@wms/api-client';
import type { SkipReason } from '@wms/types';
import { Modal, Button, Badge } from '@wms/ui';
import { XCircle, AlertTriangle, Box, Lock, Package, Info } from 'lucide-react';

interface SkipStopModalProps {
  routeId: string;
  stopNumber: number;
  isOpen: boolean;
  onClose: () => void;
}

const skipReasons: Array<{ value: SkipReason; label: string; icon: React.ReactNode; description: string; color: string }> = [
  {
    value: 'out_of_stock',
    label: 'Out of Stock',
    icon: <Package className="h-5 w-5" />,
    description: 'No items available at location',
    color: 'bg-red-100 text-red-800',
  },
  {
    value: 'location_blocked',
    label: 'Location Blocked',
    icon: <Lock className="h-5 w-5" />,
    description: 'Physical access blocked by pallet or other item',
    color: 'bg-orange-100 text-orange-800',
  },
  {
    value: 'item_damaged',
    label: 'Item Damaged',
    icon: <XCircle className="h-5 w-5" />,
    description: 'Items at location are damaged',
    color: 'bg-yellow-100 text-yellow-800',
  },
  {
    value: 'other',
    label: 'Other',
    icon: <Info className="h-5 w-5" />,
    description: 'Other issue - please specify',
    color: 'bg-blue-100 text-blue-800',
  },
];

export function SkipStopModal({ routeId, stopNumber, isOpen, onClose }: SkipStopModalProps) {
  const [selectedReason, setSelectedReason] = useState<SkipReason>('out_of_stock');
  const [notes, setNotes] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const queryClient = useQueryClient();

  const skipMutation = useMutation({
    mutationFn: () => {
      setIsSubmitting(true);
      return routingClient.skipStop(routeId, stopNumber, selectedReason, notes);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['route', routeId] });
      queryClient.invalidateQueries({ queryKey: ['routes'] });
      setIsSubmitting(false);
      setNotes('');
      onClose();
    },
    onError: () => {
      setIsSubmitting(false);
    },
  });

  const handleSubmit = () => {
    if (!selectedReason) {
      alert('Please select a reason for skipping this stop');
      return;
    }
    skipMutation.mutate();
  };

  if (!isOpen) return null;

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`Skip Stop #${stopNumber}`}>
      <div className="space-y-6">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-3">
              Reason for Skipping:
            </label>
            <div className="space-y-3">
              {skipReasons.map((reason) => {
                const isSelected = selectedReason === reason.value;

                return (
                  <button
                    key={reason.value}
                    onClick={() => setSelectedReason(reason.value)}
                    className={`w-full p-4 rounded-lg border-2 text-left transition-all ${
                      isSelected
                        ? `${reason.color} border-2`
                        : 'border-gray-200 hover:border-gray-300'
                    }`}
                  >
                    <div className="flex items-start gap-3">
                      <input
                        type="radio"
                        name="skip-reason"
                        value={reason.value}
                        checked={isSelected}
                        onChange={() => setSelectedReason(reason.value)}
                        className="mt-1"
                      />
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          {reason.icon}
                          <span className="font-semibold">{reason.label}</span>
                          {isSelected && (
                            <Badge variant="success" className="ml-2">Selected</Badge>
                          )}
                        </div>
                        <p className="text-sm text-gray-600">
                          {reason.description}
                        </p>
                      </div>
                    </div>
                  </button>
                );
              })}
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Additional Notes (Optional):
            </label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="Provide any additional context about this skip..."
              rows={3}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 disabled:bg-gray-100 disabled:text-gray-500"
              disabled={isSubmitting}
            />
            <p className="text-xs text-gray-500 mt-1">
              This information will help improve future route planning.
            </p>
          </div>

          <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
            <div className="flex items-start gap-2">
              <AlertTriangle className="h-5 w-5 text-yellow-600 flex-shrink-0 mt-0.5" />
              <div>
                <p className="font-medium text-yellow-800 mb-1">
                  Skipping a stop will create an exception record
                </p>
                <p className="text-sm text-yellow-700">
                  The route will be marked with a skip for this stop. The skipped item will be returned to the picking queue for reassignment.
                </p>
              </div>
            </div>
          </div>
        </div>

        <div className="flex gap-3 justify-end mt-6 pt-4 border-t">
          <Button variant="outline" onClick={onClose} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!selectedReason || isSubmitting}
          >
            {isSubmitting ? 'Skipping...' : 'Skip Stop'}
          </Button>
        </div>
    </Modal>
  );
}
