import React from 'react';
import type { ReceivingShipmentStatus } from '@wms/types';
import { CheckCircle, Clock, Package, ArrowRight, XCircle } from 'lucide-react';

interface ProgressStepperProps {
  status: ReceivingShipmentStatus;
  createdAt: string;
  arrivedAt?: string;
  completedAt?: string;
}

interface Step {
  label: string;
  status: 'completed' | 'active' | 'pending' | 'failed';
  timestamp?: string;
  icon: React.ReactNode;
}

export function ProgressStepper({ status, createdAt, arrivedAt, completedAt }: ProgressStepperProps) {
  const steps: Step[] = [
    {
      label: 'Expected',
      status: ['expected', 'arrived', 'receiving', 'completed'].includes(status) ? 'completed' : status === 'cancelled' ? 'failed' : 'pending',
      timestamp: createdAt,
      icon: <Package className="h-5 w-5" />,
    },
    {
      label: 'Arrived',
      status: ['arrived', 'receiving', 'completed'].includes(status) ? 'completed' : status === 'expected' ? 'active' : status === 'cancelled' ? 'failed' : 'pending',
      timestamp: arrivedAt,
      icon: <Clock className="h-5 w-5" />,
    },
    {
      label: 'Receiving',
      status: ['receiving', 'completed'].includes(status) ? 'completed' : status === 'arrived' ? 'active' : ['expected', 'cancelled'].includes(status) ? 'failed' : 'pending',
      timestamp: status === 'receiving' || status === 'completed' ? new Date().toISOString() : undefined,
      icon: <Package className="h-5 w-5" />,
    },
    {
      label: 'Completed',
      status: status === 'completed' ? 'completed' : ['expected', 'arrived', 'receiving'].includes(status) ? 'pending' : 'failed',
      timestamp: completedAt,
      icon: <CheckCircle className="h-5 w-5" />,
    },
  ];

  const stepStyles = {
    completed: 'bg-green-100 border-green-500 text-green-700',
    active: 'bg-blue-100 border-blue-500 text-blue-700',
    pending: 'bg-gray-100 border-gray-300 text-gray-400',
    failed: 'bg-red-100 border-red-500 text-red-700',
  };

  const iconStyles = {
    completed: 'text-green-600',
    active: 'text-blue-600',
    pending: 'text-gray-400',
    failed: 'text-red-600',
  };

  return (
    <div className="w-full">
      <div className="flex items-center justify-between px-2">
        {steps.map((step, index) => {
          const isFirst = index === 0;
          const isLast = index === steps.length - 1;

          return (
            <React.Fragment key={step.label}>
              <div className={`flex-1 flex flex-col items-center ${stepStyles[step.status]}`}>
                <div className={`w-10 h-10 rounded-full border-2 flex items-center justify-center ${stepStyles[step.status]} mb-2`}>
                  <span className={iconStyles[step.status]}>{step.icon}</span>
                </div>
                <div className="text-sm font-medium">{step.label}</div>
                {step.timestamp && (
                  <div className="text-xs text-gray-500 mt-1">
                    {new Date(step.timestamp).toLocaleString([], { 
                      month: 'short', 
                      day: 'numeric', 
                      hour: '2-digit', 
                      minute: '2-digit' 
                    })}
                  </div>
                )}
              </div>

              {!isLast && (
                <div className="px-3">
                  <ArrowRight className={`h-6 w-6 ${steps[index + 1]?.status === 'failed' || step.status === 'failed' ? 'text-red-400' : steps[index + 1]?.status === 'completed' || step.status === 'completed' ? 'text-green-500' : 'text-gray-300'}`} />
                </div>
              )}
            </React.Fragment>
          );
        })}
      </div>

      <div className="mt-6 flex items-center gap-6 text-sm">
        {status === 'cancelled' && (
          <div className="flex items-center gap-2 text-red-600">
            <XCircle className="h-4 w-4" />
            <span className="font-medium">This shipment has been cancelled</span>
          </div>
        )}
        {status === 'receiving' && (
          <div className="flex items-center gap-2 text-blue-600">
            <Clock className="h-4 w-4" />
            <span className="font-medium">Currently receiving items</span>
          </div>
        )}
        {status === 'completed' && completedAt && (
          <div className="flex items-center gap-2 text-green-600">
            <CheckCircle className="h-4 w-4" />
            <span className="font-medium">
              Completed {new Date(completedAt).toLocaleString()}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
