import React from 'react';
import { TrendingUp, TrendingDown, Minus } from 'lucide-react';

export interface MetricCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  trend?: {
    value: number;
    direction: 'up' | 'down' | 'neutral';
    label?: string;
  };
  icon?: React.ReactNode;
  variant?: 'default' | 'success' | 'warning' | 'error';
  className?: string;
}

const variantStyles = {
  default: 'border-gray-200',
  success: 'border-l-4 border-l-success-500 border-y-gray-200 border-r-gray-200',
  warning: 'border-l-4 border-l-warning-500 border-y-gray-200 border-r-gray-200',
  error: 'border-l-4 border-l-error-500 border-y-gray-200 border-r-gray-200',
};

const trendColors = {
  up: 'text-success-600',
  down: 'text-error-600',
  neutral: 'text-gray-500',
};

const TrendIcon = ({ direction }: { direction: 'up' | 'down' | 'neutral' }) => {
  const iconClass = 'h-4 w-4';
  switch (direction) {
    case 'up':
      return <TrendingUp className={iconClass} />;
    case 'down':
      return <TrendingDown className={iconClass} />;
    default:
      return <Minus className={iconClass} />;
  }
};

export function MetricCard({
  title,
  value,
  subtitle,
  trend,
  icon,
  variant = 'default',
  className = '',
}: MetricCardProps) {
  return (
    <div
      className={`
        bg-white rounded-lg border shadow-sm p-4
        ${variantStyles[variant]}
        ${className}
      `}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <p className="text-sm font-medium text-gray-500">{title}</p>
          <p className="mt-1 text-2xl font-semibold text-gray-900">{value}</p>
          {subtitle && <p className="mt-0.5 text-sm text-gray-500">{subtitle}</p>}
          {trend && (
            <div className={`mt-2 flex items-center gap-1 text-sm ${trendColors[trend.direction]}`}>
              <TrendIcon direction={trend.direction} />
              <span>{trend.value}%</span>
              {trend.label && <span className="text-gray-500">{trend.label}</span>}
            </div>
          )}
        </div>
        {icon && (
          <div className="p-2 bg-gray-50 rounded-lg text-gray-500">
            {icon}
          </div>
        )}
      </div>
    </div>
  );
}

// Grid layout for metrics
export interface MetricGridProps {
  children: React.ReactNode;
  columns?: 2 | 3 | 4 | 5;
  className?: string;
}

const columnStyles = {
  2: 'grid-cols-1 md:grid-cols-2',
  3: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3',
  4: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-4',
  5: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5',
};

export function MetricGrid({ children, columns = 4, className = '' }: MetricGridProps) {
  return (
    <div className={`grid gap-4 ${columnStyles[columns]} ${className}`}>
      {children}
    </div>
  );
}
