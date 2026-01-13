import React from 'react';
import { Card, CardHeader, CardContent, Badge } from '@wms/ui';
import type { Discrepancy, DiscrepancyType } from '@wms/types';
import { AlertTriangle, Package, XCircle, MinusCircle, Info, CheckCircle } from 'lucide-react';

interface DiscrepancyPanelProps {
  discrepancies: Discrepancy[];
  shipmentId: string;
}

const discrepancyIcons: Record<DiscrepancyType, React.ReactNode> = {
  shortage: <MinusCircle className="h-5 w-5 text-red-600" />,
  overage: <Info className="h-5 w-5 text-yellow-600" />,
  damage: <XCircle className="h-5 w-5 text-orange-600" />,
  wrong_item: <Package className="h-5 w-5 text-blue-600" />,
};

const discrepancyBadges: Record<DiscrepancyType, string> = {
  shortage: 'bg-red-100 text-red-800',
  overage: 'bg-yellow-100 text-yellow-800',
  damage: 'bg-orange-100 text-orange-800',
  wrong_item: 'bg-blue-100 text-blue-800',
};

export function DiscrepancyPanel({ discrepancies, shipmentId }: DiscrepancyPanelProps) {
  const discrepanciesByType = discrepancies.reduce((acc, disc) => {
    const type = disc.type || 'unknown';
    if (!acc[type]) acc[type] = [];
    acc[type].push(disc);
    return acc;
  }, {} as Record<string, Discrepancy[]>);

  const totalDiscrepancies = discrepancies.length;
  const severity = totalDiscrepancies > 5 ? 'high' : totalDiscrepancies > 2 ? 'medium' : 'low';

  const severityConfig = {
    high: { bg: 'bg-red-50 border-red-200', text: 'text-red-800', icon: 'bg-red-100' },
    medium: { bg: 'bg-yellow-50 border-yellow-200', text: 'text-yellow-800', icon: 'bg-yellow-100' },
    low: { bg: 'bg-green-50 border-green-200', text: 'text-green-800', icon: 'bg-green-100' },
  };

  const config = severityConfig[severity];

  return (
    <Card className={`${config.bg} border-2`}>
      <div className="flex items-center gap-3">
        <div className={`p-2 rounded-full ${config.icon}`}>
          <AlertTriangle className={`h-5 w-5 ${severity === 'high' ? 'text-red-600' : severity === 'medium' ? 'text-yellow-600' : 'text-green-600'}`} />
        </div>
        <CardHeader
          title={`Discrepancies (${totalDiscrepancies})`}
          subtitle={severity === 'high' ? 'High attention required' : severity === 'medium' ? 'Review needed' : 'Minor issues'}
        />
      </div>
      <CardContent>
        {totalDiscrepancies === 0 ? (
          <div className="text-center py-8">
            <CheckCircle className="h-12 w-12 text-green-500 mx-auto mb-4" />
            <p className="text-gray-600">No discrepancies detected for this shipment.</p>
          </div>
        ) : (
          <>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
              {Object.entries(discrepancyIcons).map(([type, icon]) => {
                const count = discrepanciesByType[type]?.length || 0;
                const badgeClass = discrepancyBadges[type as DiscrepancyType] || 'bg-gray-100 text-gray-800';
                return (
                  <div
                    key={type}
                    className={`${config.icon} rounded-lg p-4 text-center cursor-pointer hover:opacity-80 transition-opacity`}
                  >
                    {icon}
                    <div className="mt-2">
                      <div className="text-2xl font-bold">{count}</div>
                      <div className="text-xs text-gray-600 capitalize">{type.replace('_', ' ')}</div>
                    </div>
                  </div>
                );
              })}
            </div>

            <div className="space-y-3">
              <h3 className="font-semibold text-gray-900 mb-4">Discrepancy Details</h3>
              {discrepancies.map((discrepancy, index) => {
                const icon = discrepancyIcons[discrepancy.type || 'unknown'];
                const badgeClass = discrepancyBadges[discrepancy.type || 'unknown'];
                const difference = discrepancy.expectedQty - discrepancy.actualQty;

                return (
                  <div key={index} className="border-l-4 border-red-500 bg-white rounded-r-lg p-4 shadow-sm">
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-2">
                          {icon}
                          <span className={`px-2 py-1 rounded text-xs font-semibold uppercase ${badgeClass}`}>
                            {discrepancy.type.replace('_', ' ')}
                          </span>
                          <span className="font-mono text-sm font-medium">{discrepancy.sku}</span>
                        </div>
                        <div className="text-sm text-gray-600">
                          <div>{discrepancy.description}</div>
                        </div>
                      </div>
                      <div className="text-right ml-4">
                        <div className="flex items-center gap-4">
                          <div>
                            <div className="text-xs text-gray-500">Expected</div>
                            <div className="font-semibold">{discrepancy.expectedQty}</div>
                          </div>
                          <div>
                            <div className="text-xs text-gray-500">Actual</div>
                            <div className={`font-semibold ${difference > 0 ? 'text-red-600' : 'text-yellow-600'}`}>
                              {discrepancy.actualQty}
                            </div>
                          </div>
                          <div>
                            <div className="text-xs text-gray-500">Difference</div>
                            <div className={`font-bold text-lg ${difference > 0 ? 'text-red-600' : difference < 0 ? 'text-yellow-600' : 'text-gray-600'}`}>
                              {difference > 0 ? `-${difference}` : difference < 0 ? `+${Math.abs(difference)}` : '0'}
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                    <div className="text-xs text-gray-500 mt-2 flex items-center">
                      <AlertTriangle className="h-3 w-3 mr-1" />
                      Detected: {new Date(discrepancy.detectedAt).toLocaleString()}
                    </div>
                  </div>
                );
              })}
            </div>

            <div className="mt-6 pt-4 border-t border-gray-200">
              <div className={`flex items-center gap-2 ${config.text}`}>
                <AlertTriangle className="h-5 w-5" />
                <span className="text-sm font-medium">
                  {severity === 'high' && 'Immediate attention required - Contact supplier immediately'}
                  {severity === 'medium' && 'Discrepancies require review - Proceed with caution'}
                  {severity === 'low' && 'Minor discrepancies - Continue processing'}
                </span>
              </div>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}
