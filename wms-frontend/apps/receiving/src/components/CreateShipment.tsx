import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { receivingClient } from '@wms/api-client';
import type { CreateShipmentRequest, ExpectedItem, ItemCondition } from '@wms/types';
import { Card, CardHeader, CardContent, Input, Button, PageLoading, Select } from '@wms/ui';
import { Package, Plus, Trash2 } from 'lucide-react';

export function CreateShipment() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [formData, setFormData] = useState<CreateShipmentRequest>({
    purchaseOrderId: '',
    asn: {
      asnId: '',
      shippingCarrier: '',
      trackingNumber: '',
      estimatedArrival: '',
    },
    supplier: {
      supplierId: '',
      name: '',
      code: '',
      contactEmail: '',
    },
    expectedItems: [],
  });

  const createMutation = useMutation({
    mutationFn: (request: CreateShipmentRequest) => receivingClient.createShipment(request),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['shipments'] });
      navigate(`/receiving/${data.shipmentId}`);
    },
  });

  const handleAddItem = () => {
    setFormData({
      ...formData,
      expectedItems: [...formData.expectedItems, {
        sku: '',
        productName: '',
        expectedQuantity: 1,
        unitCost: 0,
        weight: 0,
        isHazmat: false,
        requiresColdChain: false,
      }],
    });
  };

  const handleRemoveItem = (index: number) => {
    setFormData({
      ...formData,
      expectedItems: formData.expectedItems.filter((_, i) => i !== index),
    });
  };

  const handleItemChange = (index: number, field: keyof ExpectedItem, value: any) => {
    const updatedItems = [...formData.expectedItems];
    updatedItems[index] = { ...updatedItems[index], [field]: value };
    setFormData({ ...formData, expectedItems: updatedItems });
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.purchaseOrderId || !formData.asn.asnId || !formData.supplier.name || formData.expectedItems.length === 0) {
      alert('Please fill in all required fields');
      return;
    }
    createMutation.mutate(formData);
  };

  if (createMutation.isPending) return <PageLoading message="Creating shipment..." />;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Create New Shipment</h1>
          <p className="text-gray-500">Create an inbound shipment with ASN and expected items</p>
        </div>
        <Button variant="outline" onClick={() => navigate('/receiving')}>
          Cancel
        </Button>
      </div>

      <Card>
        <CardHeader title="Shipment Information" />
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Purchase Order ID *
                </label>
                <Input
                  required
                  value={formData.purchaseOrderId}
                  onChange={(e) => setFormData({ ...formData, purchaseOrderId: e.target.value })}
                  placeholder="PO-2024-001234"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Supplier ID *
                </label>
                <Input
                  required
                  value={formData.supplier.supplierId}
                  onChange={(e) => setFormData({
                    ...formData,
                    supplier: { ...formData.supplier, supplierId: e.target.value },
                  })}
                  placeholder="SUP-001"
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Supplier Name *
              </label>
              <Input
                required
                value={formData.supplier.name}
                onChange={(e) => setFormData({
                  ...formData,
                  supplier: { ...formData.supplier, name: e.target.value },
                })}
                placeholder="Acme Supplies"
              />
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Supplier Code
                </label>
                <Input
                  value={formData.supplier.code}
                  onChange={(e) => setFormData({
                    ...formData,
                    supplier: { ...formData.supplier, code: e.target.value },
                  })}
                  placeholder="ACME"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Contact Email
                </label>
                <Input
                  type="email"
                  value={formData.supplier.contactEmail}
                  onChange={(e) => setFormData({
                    ...formData,
                    supplier: { ...formData.supplier, contactEmail: e.target.value },
                  })}
                  placeholder="supplier@acme.com"
                />
              </div>
            </div>

            <CardHeader title="ASN Information" />
            <div className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    ASN ID *
                  </label>
                  <Input
                    required
                    value={formData.asn.asnId}
                    onChange={(e) => setFormData({
                      ...formData,
                      asn: { ...formData.asn, asnId: e.target.value },
                    })}
                    placeholder="ASN-001234"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Shipping Carrier *
                  </label>
                  <Select
                    required
                    value={formData.asn.shippingCarrier}
                    onChange={(e) => setFormData({
                      ...formData,
                      asn: { ...formData.asn, shippingCarrier: e.target.value },
                    })}
                    options={[
                      { value: '', label: 'Select Carrier' },
                      { value: 'FedEx', label: 'FedEx' },
                      { value: 'UPS', label: 'UPS' },
                      { value: 'USPS', label: 'USPS' },
                      { value: 'DHL', label: 'DHL' },
                    ]}
                  />
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Tracking Number *
                  </label>
                  <Input
                    required
                    value={formData.asn.trackingNumber}
                    onChange={(e) => setFormData({
                      ...formData,
                      asn: { ...formData.asn, trackingNumber: e.target.value },
                    })}
                    placeholder="794644790132"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Estimated Arrival *
                  </label>
                  <Input
                    type="datetime-local"
                    required
                    value={formData.asn.estimatedArrival}
                    onChange={(e) => setFormData({
                      ...formData,
                      asn: { ...formData.asn, estimatedArrival: e.target.value },
                    })}
                  />
                </div>
              </div>
            </div>

            <CardHeader
              title="Expected Items"
              subtitle={`${formData.expectedItems.length} items`}
            />
            <div className="space-y-4">
              {formData.expectedItems.map((item, index) => (
                <Card key={index} className="border border-gray-200">
                  <CardContent className="pt-4">
                    <div className="flex justify-between items-start mb-4">
                      <h3 className="font-semibold">Item {index + 1}</h3>
                      {formData.expectedItems.length > 1 && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleRemoveItem(index)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      )}
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                          SKU *
                        </label>
                        <Input
                          required
                          value={item.sku}
                          onChange={(e) => handleItemChange(index, 'sku', e.target.value)}
                          placeholder="SKU-12345"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                          Product Name *
                        </label>
                        <Input
                          required
                          value={item.productName}
                          onChange={(e) => handleItemChange(index, 'productName', e.target.value)}
                          placeholder="Widget A"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                          Expected Quantity *
                        </label>
                        <Input
                          type="number"
                          required
                          min="1"
                          value={item.expectedQuantity}
                          onChange={(e) => handleItemChange(index, 'expectedQuantity', parseInt(e.target.value) || 0)}
                          placeholder="100"
                        />
                      </div>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-4">
                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                          Unit Cost
                        </label>
                        <Input
                          type="number"
                          min="0"
                          step="0.01"
                          value={item.unitCost}
                          onChange={(e) => handleItemChange(index, 'unitCost', parseFloat(e.target.value) || 0)}
                          placeholder="9.99"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                          Weight (kg)
                        </label>
                        <Input
                          type="number"
                          min="0"
                          step="0.1"
                          value={item.weight}
                          onChange={(e) => handleItemChange(index, 'weight', parseFloat(e.target.value) || 0)}
                          placeholder="0.5"
                        />
                      </div>

                      <div className="flex items-center">
                        <input
                          type="checkbox"
                          id={`hazmat-${index}`}
                          checked={item.isHazmat}
                          onChange={(e) => handleItemChange(index, 'isHazmat', e.target.checked)}
                          className="mr-2 h-4 w-4"
                        />
                        <label htmlFor={`hazmat-${index}`} className="text-sm font-medium text-gray-700">
                          Hazmat
                        </label>
                      </div>

                      <div className="flex items-center">
                        <input
                          type="checkbox"
                          id={`coldchain-${index}`}
                          checked={item.requiresColdChain}
                          onChange={(e) => handleItemChange(index, 'requiresColdChain', e.target.checked)}
                          className="mr-2 h-4 w-4"
                        />
                        <label htmlFor={`coldchain-${index}`} className="text-sm font-medium text-gray-700">
                          Cold Chain
                        </label>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}

              <Button
                variant="outline"
                type="button"
                onClick={handleAddItem}
                className="w-full"
              >
                <Plus className="h-4 w-4 mr-2" />
                Add Expected Item
              </Button>
            </div>

            <div className="flex gap-4 pt-4">
              <Button variant="outline" type="button" onClick={() => navigate('/receiving')}>
                Cancel
              </Button>
              <Button type="submit" className="flex-1">
                <Package className="h-4 w-4 mr-2" />
                Create Shipment
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
