import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { routingClient } from '@wms/api-client';
import type { RouteItem } from '@wms/types';
import { Card, CardHeader, CardContent, Button, Badge } from '@wms/ui';
import { Route, CheckCircle, Sparkles } from 'lucide-react';

const strategies = [
  { value: 'shortest_path', label: 'Shortest Path', description: 'Optimizes for minimum travel distance' },
  { value: 'zone_based', label: 'Zone Based', description: 'Groups picks by warehouse zone' },
  { value: 'priority_first', label: 'Priority First', description: 'Prioritizes high-priority items' },
  { value: 'batch_pick', label: 'Batch Pick', description: 'Optimizes for multiple orders' },
];

interface StrategySelectorProps {
  items: RouteItem[];
  routeId?: string;
  onStrategyChange?: (strategy: string) => void;
  currentStrategy?: string;
}

export function StrategySelector({ items, routeId, onStrategyChange, currentStrategy }: StrategySelectorProps) {
  const [isLoading, setIsLoading] = useState(false);
  const [suggestion, setSuggestion] = useState<{
    recommendedStrategy: string;
    confidence: number;
    reasoning: string;
  } | null>(null);

  const { data: analysisData } = useQuery({
    queryKey: ['route-suggestion', items.map(i => i.sku).join(',')],
    queryFn: () => routingClient.suggestStrategy(items),
    enabled: items.length > 0,
  });

  const getRecommendation = () => {
    setIsLoading(true);
    setTimeout(() => {
      setIsLoading(false);
    }, 500);
  };

  if (isLoading || !analysisData) {
    return (
      <Card>
        <CardHeader title="Strategy Selection" />
        <CardContent>
          <div className="space-y-4">
            {strategies.map((strategy) => (
              <button
                key={strategy.value}
                onClick={() => onStrategyChange?.(strategy.value)}
                className={`w-full p-4 rounded-lg border-2 text-left transition-all ${
                  currentStrategy === strategy.value
                    ? 'border-primary-600 bg-primary-50 text-primary-900'
                    : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50'
                }`}
              >
                <div className="flex items-start gap-3">
                  <input
                    type="radio"
                    name="strategy"
                    value={strategy.value}
                    checked={currentStrategy === strategy.value}
                    onChange={() => onStrategyChange?.(strategy.value)}
                    className="mt-1"
                  />
                  <div className="flex-1">
                    <div className="font-semibold text-gray-900 mb-1">
                      {strategy.label}
                    </div>
                    <div className="text-sm text-gray-600">
                      {strategy.description}
                    </div>
                  </div>
                </div>
              </button>
            ))}
          </div>

          {items.length > 0 && analysisData && (
            <div className="mt-6 pt-4 border-t border-gray-200">
              <div className="flex items-center gap-2 mb-2">
                <Sparkles className="h-5 w-5 text-yellow-600" />
                <span className="font-semibold text-gray-900">AI Recommendation</span>
                <Button size="sm" variant="outline" onClick={getRecommendation}>
                  <Route className="h-4 w-4 mr-2" />
                  Refresh
                </Button>
              </div>
              {analysisData.recommendedStrategy && (
                <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
                  <div className="flex items-start gap-2 mb-2">
                    <CheckCircle className="h-5 w-5 text-green-600 flex-shrink-0 mt-0.5" />
                    <div>
                      <div className="font-semibold text-gray-900">
                        Recommended: {strategies.find(s => s.value === analysisData.recommendedStrategy)?.label}
                      </div>
                      <div className="text-sm text-gray-600">
                        Confidence: {(analysisData.confidence * 100).toFixed(0)}%
                      </div>
                    </div>
                  </div>
                  <div className="text-sm text-gray-600 mt-3">
                    {analysisData.reasoning}
                  </div>
                </div>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader title="Strategy Selection" />
      <CardContent>
        <div className="space-y-4">
          <p className="text-sm text-gray-600 mb-4">
            Select a routing strategy for your pick route:
          </p>
          
          {strategies.map((strategy) => (
            <button
              key={strategy.value}
              onClick={() => onStrategyChange?.(strategy.value)}
              className={`w-full p-4 rounded-lg border-2 text-left transition-all ${
                currentStrategy === strategy.value
                  ? 'border-primary-600 bg-primary-50 text-primary-900'
                  : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50'
              }`}
            >
              <div className="flex items-start gap-3">
                <input
                  type="radio"
                  name="strategy"
                  value={strategy.value}
                  checked={currentStrategy === strategy.value}
                  onChange={() => onStrategyChange?.(strategy.value)}
                  className="mt-1"
                />
                <div className="flex-1">
                  <div className="font-semibold text-gray-900 mb-1">
                    {strategy.label}
                  </div>
                  <div className="text-sm text-gray-600">
                    {strategy.description}
                  </div>
                </div>
              </div>
            </button>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
