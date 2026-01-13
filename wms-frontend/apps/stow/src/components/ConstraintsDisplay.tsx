import React from 'react';
import type { ItemConstraints } from '@wms/types';
import { Card, CardHeader, CardContent, Badge } from '@wms/ui';
import { AlertTriangle, Flame, Snowflake, Package, Zap, Star, CheckCircle } from 'lucide-react';

interface ConstraintsDisplayProps {
  constraints: ItemConstraints;
}

export function ConstraintsDisplay({ constraints }: ConstraintsDisplayProps) {
  const constraintBadges = [
    { key: 'hazmat', label: 'Hazmat', icon: <Flame className="h-5 w-5" />, color: 'bg-red-100 text-red-800', has: constraints.hazmat },
    { key: 'coldChain', label: 'Cold Chain', icon: <Snowflake className="h-5 w-5" />, color: 'bg-cyan-100 text-cyan-800', has: constraints.coldChain },
    { key: 'oversized', label: 'Oversized', icon: <Package className="h-5 w-5" />, color: 'bg-yellow-100 text-yellow-800', has: constraints.oversized },
    { key: 'fragile', label: 'Fragile', icon: <Package className="h-5 w-5" />, color: 'bg-pink-100 text-pink-800', has: constraints.fragile },
    { key: 'highValue', label: 'High Value', icon: <Star className="h-5 w-5" />, color: 'bg-purple-100 text-purple-800', has: constraints.highValue },
  ];

  const activeConstraints = constraintBadges.filter(c => c.has);

  return (
    <Card>
      <CardHeader
        title="Storage Requirements"
        subtitle={`${activeConstraints.length} of ${constraintBadges.length} requirements`}
      />
      <CardContent>
        {activeConstraints.length === 0 ? (
          <div className="text-center py-6">
            <CheckCircle className="h-12 w-12 text-green-500 mx-auto mb-4" />
            <p className="text-gray-600">No special storage requirements</p>
          </div>
        ) : (
          <div className="space-y-3">
            <h4 className="font-semibold text-gray-900 mb-3">
              This item requires:
            </h4>

            <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
              {constraintBadges.map((constraint) => {
                return (
                  <Card
                    key={constraint.key}
                    className={`${constraint.has ? constraint.color : 'bg-gray-100 text-gray-400'} p-4 rounded-lg border-2`}
                  >
                    <CardContent>
                      <div className="flex items-center gap-3 mb-2">
                        {constraint.icon}
                        <span className="font-medium text-gray-900">{constraint.label}</span>
                      </div>
                      {constraint.has && (
                        <Badge className={constraint.color.replace('-100', '-800')}>Required</Badge>
                      )}
                    </CardContent>
                  </Card>
                );
              })}
            </div>

            {activeConstraints.length >= 2 && (
              <div className="mt-6 pt-4 border-t border-orange-200 bg-orange-50 p-4">
                <div className="flex items-start gap-3">
                  <AlertTriangle className="h-5 w-5 text-orange-600 flex-shrink-0 mt-0.5" />
                  <div>
                    <p className="font-medium text-orange-800 mb-1">Multiple Storage Requirements</p>
                    <p className="text-sm text-orange-700 mt-1">
                      This item has multiple special storage requirements. Ensure the target location supports all requirements:
                    </p>
                    <ul className="list-disc list-inside mt-2 space-y-1">
                      {activeConstraints.map(c => (
                        <li key={c.key}>
                          <strong>{c.label}</strong> storage zone
                        </li>
                      ))}
                    </ul>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
