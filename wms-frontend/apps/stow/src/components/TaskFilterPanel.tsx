import React from 'react';
import type { StorageStrategy } from '@wms/types';
import { Card, CardHeader, CardContent, Button } from '@wms/ui';
import { Filter, X, CheckCircle, ChevronDown } from 'lucide-react';

interface TaskFilterPanelProps {
  onFilterChange: (key: string, value: string | undefined) => void;
  filters: Record<string, any>;
  isOpen?: boolean;
  onToggle?: () => void;
}

export function TaskFilterPanel({ onFilterChange, filters, isOpen, onToggle }: TaskFilterPanelProps) {
  const filterOptions = {
    status: ['All', 'Pending', 'Assigned', 'In Progress', 'Completed', 'Failed', 'Cancelled'],
    strategy: ['All', 'Chaotic', 'Directed', 'Velocity', 'Zone Based'],
    worker: ['All Workers', 'PICKER-001', 'PICKER-002', 'PICKER-003', 'PICKER-004'],
  };

  return (
    <Card className="w-full">
      {isOpen && (
        <>
          <CardHeader
            title="Filters"
            subtitle="Click to apply"
          />
          <CardContent>
            <div className="space-y-4">
              <div className="space-y-3">
                <h4 className="font-semibold text-gray-900 mb-3">Status</h4>
                <div className="flex flex-wrap gap-2">
                  {filterOptions.status.map((option) => (
                    <Button
                      key={option}
                      variant={filters.status === option ? 'primary' : 'outline'}
                      size="sm"
                      onClick={() => onFilterChange('status', option === 'All' ? undefined : option)}
                    >
                      {option}
                    </Button>
                  ))}
                </div>
              </div>

              <div className="space-y-3">
                <h4 className="font-semibold text-gray-900 mb-3">Storage Strategy</h4>
                <StrategyBadge 
                  strategy={filters.strategy} 
                  onClick={() => onFilterChange('strategy', filters.strategy === 'All' ? undefined : filters.strategy)}
                />
              </div>

              <div className="space-y-3">
                <h4 className="font-semibold text-gray-900 mb-3">Assigned Worker</h4>
                <Button
                  variant={filters.workerId ? 'primary' : 'outline'}
                  size="sm"
                  onClick={() => onFilterChange('workerId', filters.workerId === 'All Workers' ? undefined : filters.workerId)}
                >
                  {filters.workerId || 'All Workers'}
                </Button>
              </div>

              <div className="pt-4 border-t border-gray-200">
                <Button variant="outline" onClick={onToggle} className="w-full">
                  <X className="h-4 w-4 mr-2" />
                  Close Filters
                </Button>
              </div>
            </div>
          </CardContent>
        </>
      )}
    </Card>
  );
}

interface StrategyBadgeProps {
  strategy: string;
  onClick?: () => void;
}

export function StrategyBadge({ strategy, onClick }: StrategyBadgeProps) {
  const strategies: Record<string, { label: string; color: string; icon: React.ReactNode }> = {
    chaotic: { label: 'Chaotic', color: 'bg-orange-100 text-orange-800', icon: <CheckCircle /> },
    directed: { label: 'Directed', color: 'bg-blue-100 text-blue-800', icon: <CheckCircle /> },
    velocity: { label: 'Velocity', color: 'bg-green-100 text-green-800', icon: <CheckCircle /> },
    zone_based: { label: 'Zone Based', color: 'bg-purple-100 text-purple-800', icon: <CheckCircle /> },
  };

  const isAll = strategy === 'All';
  const strategyConfig = strategies[strategy as keyof typeof strategies] || strategies.chaotic;

  return (
    <div
      onClick={() => onClick && onClick()}
      className={`inline-flex items-center gap-2 px-3 py-2 rounded-lg border-2 transition-all cursor-pointer ${
        !isAll && strategyConfig ? `${strategyConfig.color} border-${strategyConfig.color.replace('-100', '-300')}` : 'bg-white border-gray-200 hover:border-gray-300'
      }`}
    >
      {strategyConfig.icon}
      <span className={`font-medium ${!isAll ? strategyConfig.color.replace('-100', '-700') : 'text-gray-600'}`}>
        {strategyConfig.label}
      </span>
      <ChevronDown className="h-4 w-4 text-gray-400" />
    </div>
  );
}
