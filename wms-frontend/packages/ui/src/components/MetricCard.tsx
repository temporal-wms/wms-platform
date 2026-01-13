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
  variant?: 'default' | 'success' | 'warning' | 'error' | 'hero';
  className?: string;
}

const variantStyles = {
  default: 'bg-white border border-gray-100',
  success: 'bg-white border-l-4 border-l-success-500 border-y border-r border-y-gray-100 border-r-gray-100',
  warning: 'bg-white border-l-4 border-l-warning-500 border-y border-r border-y-gray-100 border-r-gray-100',
  error: 'bg-white border-l-4 border-l-error-500 border-y border-r border-y-gray-100 border-r-gray-100',
  hero: 'bg-gradient-to-br from-primary-500 to-primary-600 text-white border-0',
};

const iconContainerStyles = {
  default: 'bg-primary-100 text-primary-600',
  success: 'bg-success-100 text-success-600',
  warning: 'bg-warning-100 text-warning-600',
  error: 'bg-error-100 text-error-600',
  hero: 'bg-white/20 text-white',
};

const trendColors = {
  up: 'text-success-600',
  down: 'text-error-600',
  neutral: 'text-gray-500',
};

const trendColorsHero = {
  up: 'text-success-200',
  down: 'text-error-200',
  neutral: 'text-white/70',
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
  const isHero = variant === 'hero';
  const trendStyle = isHero ? trendColorsHero : trendColors;

  return (
    <div
      className={`
        rounded-xl shadow-card p-5
        transition-all duration-200 hover:shadow-card-hover
        ${variantStyles[variant]}
        ${className}
      `}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <p className={`text-sm font-medium ${isHero ? 'text-primary-100' : 'text-gray-500'}`}>
            {title}
          </p>
          <p className={`mt-2 text-3xl font-bold ${isHero ? 'text-white' : 'text-gray-900'}`}>
            {value}
          </p>
          {subtitle && (
            <p className={`mt-1 text-sm ${isHero ? 'text-primary-100' : 'text-gray-500'}`}>
              {subtitle}
            </p>
          )}
          {trend && (
            <div className={`mt-3 flex items-center gap-1.5 text-sm font-medium ${trendStyle[trend.direction]}`}>
              <TrendIcon direction={trend.direction} />
              <span>{trend.value}%</span>
              {trend.label && (
                <span className={isHero ? 'text-primary-200' : 'text-gray-400'}>{trend.label}</span>
              )}
            </div>
          )}
        </div>
        {icon && (
          <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${iconContainerStyles[variant]}`}>
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
