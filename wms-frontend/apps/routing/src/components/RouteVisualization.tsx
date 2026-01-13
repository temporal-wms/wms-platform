import React from 'react';
import type { RouteStop } from '@wms/types';
import { MapPin, Layers, RefreshCw } from 'lucide-react';

interface RouteVisualizationProps {
  stops: RouteStop[];
  currentStop?: number;
}

export function RouteVisualization({ stops, currentStop }: RouteVisualizationProps) {
  const stopsByZone = stops.reduce((acc, stop) => {
    if (!acc[stop.zone]) acc[stop.zone] = [];
    acc[stop.zone].push(stop);
    return acc;
  }, {} as Record<string, RouteStop[]>);

  const zones = Object.keys(stopsByZone);
  const aisles = Array.from(new Set(stops.map(s => s.aisle))).sort();

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-gray-900">Warehouse Floor Plan</h3>
        <Button variant="outline" size="sm">
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh Map
        </Button>
      </div>

      <div className="bg-gray-100 rounded-lg p-6 border-2 border-gray-200">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          {zones.map((zone, zoneIndex) => {
            const zoneStops = stopsByZone[zone] || [];
            const zoneAisles = aisles.filter(a => zoneStops.some(s => s.aisle === a));

            return (
              <div key={zone} className="space-y-2">
                <div className="text-sm font-semibold text-gray-700 mb-3">
                  Zone {zone}
                </div>
                <div className="space-y-2">
                  {zoneAisles.map((aisle, aisleIndex) => {
                    const aisleStops = zoneStops.filter(s => s.aisle === aisle);
                    const bays = Array.from(new Set(aisleStops.map(s => s.bay))).sort();

                    return (
                      <div key={`${zone}-${aisle}`} className="space-y-1">
                        <div className="text-xs font-medium text-gray-500">
                          Aisle {aisle}
                        </div>
                        <div className="grid grid-cols-4 gap-1">
                          {bays.map((bay, bayIndex) => {
                            const bayStops = aisleStops.filter(s => s.bay === bay);
                            const levels = Array.from(new Set(bayStops.map(s => s.level))).sort();

                            return (
                              <div key={`${zone}-${aisle}-${bay}`} className="space-y-1">
                                <div className="text-xs font-medium text-gray-500">
                                  Bay {bay}
                                </div>
                                <div className="grid grid-cols-2 gap-1">
                                  {levels.map((level, levelIndex) => {
                                    const levelStop = bayStops.find(s => s.level === level);

                                    if (!levelStop) {
                                      return (
                                        <div 
                                          key={`${zone}-${aisle}-${bay}-${level}`}
                                          className="h-12 bg-gray-200 rounded border border-gray-300"
                                        />
                                      );
                                    }

                                    const isCurrent = levelStop.sequence === currentStop;
                                    const isCompleted = levelStop.status === 'completed';
                                    const isSkipped = levelStop.status === 'skipped';

                                    const statusColor = isCompleted 
                                      ? 'bg-green-100 border-green-600 text-green-800'
                                      : isSkipped
                                      ? 'bg-red-100 border-red-600 text-red-800'
                                      : isCurrent
                                      ? 'bg-blue-100 border-blue-600 text-blue-800'
                                      : 'bg-gray-100 border-gray-300 text-gray-600';

                                    return (
                                      <div
                                        key={`${zone}-${aisle}-${bay}-${level}`}
                                        className={`h-12 rounded border-2 flex items-center justify-center transition-all ${statusColor}`}
                                      >
                                        <div className="flex flex-col items-center">
                                          <span className="text-xs font-bold">
                                            #{levelStop.sequence}
                                          </span>
                                          <MapPin className="h-3 w-3" />
                                        </div>
                                      </div>
                                    );
                                  })}
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            );
          })}
        </div>
      </div>

      <div className="flex items-center justify-around mt-6 pt-4 border-t border-gray-300">
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-green-100 border-2 border-green-600" />
          <span className="text-sm text-gray-600">Completed</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-blue-100 border-2 border-blue-600" />
          <span className="text-sm text-gray-600">Current</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-gray-100 border-2 border-gray-300" />
          <span className="text-sm text-gray-600">Pending</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-red-100 border-2 border-red-600" />
          <span className="text-sm text-gray-600">Skipped</span>
        </div>
      </div>
    </div>
  );
}

function Button({ variant, size, children, ...props }: any) {
  return (
    <button
      className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
        variant === 'outline'
          ? 'bg-white border border-gray-300 hover:bg-gray-50'
          : 'bg-primary-600 hover:bg-primary-700 text-white'
      } ${size === 'sm' ? 'px-3 py-1.5 text-xs' : ''}`}
      {...props}
    >
      {children}
    </button>
  );
}
