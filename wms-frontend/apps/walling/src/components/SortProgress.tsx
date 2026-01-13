import React, { useState } from 'react';
import { Card, CardHeader, CardContent, Button, Badge } from '@wms/ui';
import { MapPin, Layers, CheckCircle, Box } from 'lucide-react';

interface ItemToSort {
  sku: string;
  productName: string;
  quantity: number;
}

interface SortProgressProps {
  items: ItemToSort[];
  sortedCount: number;
  sortedItems?: Array<{ sku: string; quantity: number }>;
}

export function SortProgress({ items, sortedCount, sortedItems = [] }: SortProgressProps) {
  const totalItems = items.reduce((sum, item) => sum + item.quantity, 0);
  const progress = totalItems > 0 ? (sortedCount / totalItems) * 100 : 0;

  return (
    <Card>
      <CardHeader title="Sort Progress" />
      <CardContent>
        <div className="mb-6">
          <div className="flex justify-between items-center mb-4">
            <div className="flex-1">
              <div className="text-3xl font-bold text-primary-600">{sortedCount}</div>
              <div className="text-sm text-gray-600">Items Sorted</div>
            </div>
            <div className="flex items-center gap-2">
              <MapPin className="h-6 w-6 text-primary-600" />
              <div className="text-3xl font-bold text-gray-900">{totalItems}</div>
              <div className="text-sm text-gray-500">Total Items</div>
            </div>
          </div>

          <div className="w-full bg-gray-200 rounded-full h-3">
            <div
              className="bg-primary-600 h-3 rounded-full transition-all duration-300"
              style={{ width: `${progress}%` }}
            />
          </div>

          <div className="flex items-center justify-between mt-4">
            <span className="text-sm text-gray-600">
              {sortedCount} of {totalItems} items sorted
            </span>
            {progress >= 100 && (
              <div className="flex items-center gap-2 text-green-600">
                <CheckCircle className="h-6 w-6" />
                <span className="font-semibold">All items sorted!</span>
              </div>
            )}
            <span className="text-sm text-gray-600">
              {totalItems - sortedCount} items remaining
            </span>
          </div>
        </div>

        {items.length === 0 ? (
          <div className="text-center py-8">
            <CheckCircle className="h-12 w-12 text-green-500 mx-auto mb-4" />
            <p className="text-lg text-gray-600">All items sorted!</p>
          </div>
        ) : (
          <div className="space-y-3">
            {items.map((item, index) => {
              const sorted = sortedItems.some(s => s.sku === item.sku);
              const quantity = item.quantity;
              const sortedQty = sorted ? item.quantity : 0;

              return (
                <div key={item.sku} className="flex items-center justify-between p-3 border-b border-gray-100">
                  <div className="flex-1">
                    <div className="font-semibold">{item.productName}</div>
                    <Badge variant="neutral">{item.sku}</Badge>
                    <div className="text-sm text-gray-600 mt-1">
                      SKU: {item.sku} • Quantity: {quantity}
                    </div>
                  </div>
                  <div className="flex items-center">
                    {sorted && (
                      <CheckCircle className="h-6 w-6 text-green-600" />
                    )}
                    <div>
                      <div className="font-semibold text-green-600">{sortedQty}</div>
                      <div className="text-sm text-gray-600">/ {quantity - sortedQty}</div>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
