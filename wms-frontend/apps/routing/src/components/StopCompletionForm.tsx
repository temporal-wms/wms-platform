import React, { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { routingClient } from '@wms/api-client';
import { Modal, Input, Button, Badge } from '@wms/ui';
import { Package, MapPin, CheckCircle2, AlertTriangle, Minus, Plus } from 'lucide-react';

interface StopCompletionFormProps {
  routeId: string;
  stopNumber: number;
  isOpen: boolean;
  onClose: () => void;
  stopData: {
    sequence: number;
    locationId: string;
    sku: string;
    quantity: number;
    status: string;
  };
}

export function StopCompletionForm({ routeId, stopNumber, isOpen, onClose, stopData }: StopCompletionFormProps) {
  const [actualQuantity, setActualQuantity] = useState<number>(stopData.quantity);
  const [notes, setNotes] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showLocationInfo, setShowLocationInfo] = useState(true);
  const [showNotesInfo, setShowNotesInfo] = useState(true);
  const [showAdjustment, setShowAdjustment] = useState(false);
  const queryClient = useQueryClient();

  const completeMutation = useMutation({
    mutationFn: (data: { actualQuantity: number; notes: string }) =>
      routingClient.completeStop(routeId, stopNumber, data.actualQuantity, data.notes),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['route', routeId] });
      queryClient.invalidateQueries({ queryKey: ['routes'] });
      setActualQuantity(stopData.quantity);
      setNotes('');
      setShowLocationInfo(true);
      setShowNotesInfo(true);
      setShowAdjustment(false);
      setIsSubmitting(false);
      onClose();
    },
    onError: (error) => {
      setIsSubmitting(false);
      alert(`Error completing stop: ${error.message}`);
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (actualQuantity < 0) {
      alert('Quantity must be greater than or equal to 0');
      return;
    }

    if (actualQuantity > stopData.quantity) {
      if (!showAdjustment) {
        setShowAdjustment(true);
        return;
      }
    }

    setIsSubmitting(true);
    completeMutation.mutate({ actualQuantity, notes });
  };

  const handleQuickComplete = () => {
    setActualQuantity(stopData.quantity);
    setNotes(`Auto-completed at ${new Date().toLocaleTimeString()}`);
    completeMutation.mutate({ actualQuantity: stopData.quantity, notes: `Auto-completed at ${new Date().toLocaleTimeString()}` });
  };

  const handleDecrease = () => {
    setActualQuantity(prev => Math.max(0, prev - 1));
  };

  const handleIncrease = () => {
    if (actualQuantity >= stopData.quantity) {
      alert(`Cannot exceed expected quantity of ${stopData.quantity}`);
      return;
    }
    setActualQuantity(prev => prev + 1);
  };

  if (!isOpen) return null;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`Complete Stop #${stopNumber}`}
      size="lg"
      footer={
        <form onSubmit={handleSubmit} className="w-full">
          <div className="flex gap-3 justify-between w-full">
            <div className="flex gap-3">
              <Button variant="outline" onClick={onClose} disabled={isSubmitting} type="button">
                Cancel
              </Button>
              <Button
                variant="outline"
                type="button"
                onClick={handleQuickComplete}
                disabled={isSubmitting || actualQuantity !== stopData.quantity}
                className="relative"
              >
                <Package className="h-4 w-4 mr-2" />
                Complete Full Qty
              </Button>
            </div>
            <Button
              type="submit"
              disabled={isSubmitting || actualQuantity === 0}
            >
              {isSubmitting ? 'Completing...' : 'Complete Stop'}
            </Button>
          </div>
        </form>
      }
    >
      <div className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Location ID
              </label>
              <div className="flex items-center gap-2 p-3 bg-gray-100 rounded-lg">
                <MapPin className="h-5 w-5 text-primary-600" />
                <span className="font-mono font-semibold text-gray-900">{stopData.locationId}</span>
                <button
                  type="button"
                  onClick={() => setShowLocationInfo(!showLocationInfo)}
                  className="ml-auto text-gray-500 hover:text-gray-700"
                >
                  {showLocationInfo ? '−' : '+'}
                </button>
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                SKU
              </label>
              <div className="flex items-center gap-2 p-3 bg-blue-50 rounded-lg">
                <Package className="h-5 w-5 text-blue-600" />
                <span className="font-mono font-semibold text-gray-900">{stopData.sku}</span>
              </div>
            </div>
          </div>

          <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
            <div className="flex items-start gap-3 mb-3">
              <AlertTriangle className="h-5 w-5 text-yellow-600 flex-shrink-0 mt-0.5" />
              <div>
                <p className="font-medium text-yellow-800 mb-1">
                  Expected Quantity: <strong className="text-yellow-900">{stopData.quantity}</strong>
                </p>
                <p className="text-sm text-yellow-700">
                  Enter the actual quantity you picked at this location
                </p>
              </div>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Actual Quantity Picked
            </label>
            <div className="flex items-center gap-3">
              <div className="flex-1">
                <Input
                  type="number"
                  min="0"
                  max={stopData.quantity}
                  value={actualQuantity}
                  onChange={(e) => setActualQuantity(parseInt(e.target.value) || 0)}
                  className="text-center text-lg font-semibold"
                  placeholder="0"
                  disabled={isSubmitting}
                  required
                />
                <p className="text-xs text-gray-500 mt-1">
                  Maximum: {stopData.quantity}
                </p>
              </div>
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={handleDecrease}
                  disabled={isSubmitting || actualQuantity <= 0}
                  className="p-2 border border-gray-300 rounded hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <Minus className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  onClick={handleIncrease}
                  disabled={isSubmitting || actualQuantity >= stopData.quantity}
                  className="p-2 border border-gray-300 rounded hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <Plus className="h-4 w-4" />
                </button>
              </div>
            </div>
          </div>

          {showLocationInfo && (
            <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
              <div className="flex items-start gap-2">
                <MapPin className="h-5 w-5 text-gray-600 flex-shrink-0 mt-0.5" />
                <div>
                  <p className="font-medium text-gray-900 mb-1">Location: {stopData.locationId}</p>
                  <p className="text-sm text-gray-600">
                    Verify this location ID matches the actual pick location in the warehouse
                  </p>
                  <p className="text-sm text-gray-600">
                    Zone: {stopData.locationId.split('-')[0]} | Aisle: {stopData.locationId.split('-')[1]} | Bay: {stopData.locationId.split('-')[2]} | Level: {stopData.locationId.split('-')[3]}
                  </p>
                </div>
              </div>
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Completion Notes
            </label>
            <div className="relative">
              <textarea
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                placeholder="Enter any notes about this pick (e.g., item condition, location issues...)"
                rows={3}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 resize-none"
                disabled={isSubmitting}
              />
              <button
                type="button"
                onClick={() => setShowNotesInfo(!showNotesInfo)}
                className="absolute right-3 top-3 text-gray-500 hover:text-gray-700"
              >
                {showNotesInfo ? '−' : '+'}
              </button>
            </div>
            {showNotesInfo && (
              <div className="mt-2 bg-blue-50 border border-blue-200 rounded-lg p-3">
                <p className="text-sm text-blue-800">
                  <strong>Tips for helpful notes:</strong>
                </p>
                <ul className="text-sm text-blue-700 mt-2 space-y-1">
                  <li>• Note any item damage or defects</li>
                  <li>• Report location access issues</li>
                  <li>• Note if bin/level was difficult to reach</li>
                  <li>• Document any equipment issues</li>
                </ul>
              </div>
            )}
          </div>

          {actualQuantity < stopData.quantity && showAdjustment && (
            <div className="bg-orange-50 border border-orange-200 rounded-lg p-4 mb-4">
              <div className="flex items-start gap-2">
                <AlertTriangle className="h-5 w-5 text-orange-600 flex-shrink-0 mt-0.5" />
                <div>
                  <p className="font-medium text-orange-800 mb-1">
                    Quantity Adjustment Required
                  </p>
                  <p className="text-sm text-orange-700">
                    You're picking less than the expected quantity ({actualQuantity} vs {stopData.quantity}). Please confirm:
                  </p>
                  <ul className="text-sm text-orange-700 mt-2 space-y-1">
                    <li>• Was the correct location used?</li>
                    <li>• Is there a discrepancy in the expected quantity?</li>
                    <li>• Should this be reported as a skip with a reason?</li>
                  </ul>
                </div>
              </div>
            </div>
          )}

          {actualQuantity === stopData.quantity && (
            <div className="bg-green-50 border border-green-200 rounded-lg p-4 mb-4">
              <div className="flex items-center gap-2">
                <CheckCircle2 className="h-5 w-5 text-green-600 flex-shrink-0" />
                <p className="font-medium text-green-800">
                  Full quantity confirmed - ready to complete
                </p>
              </div>
            </div>
          )}
      </div>
    </Modal>
  );
}
