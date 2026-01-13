import React from 'react';
import { Package, Search, AlertCircle } from 'lucide-react';
import { Button } from './Button';

export interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  action?: {
    label: string;
    onClick: () => void;
  };
  variant?: 'default' | 'search' | 'error';
}

const defaultIcons = {
  default: <Package className="h-8 w-8" />,
  search: <Search className="h-8 w-8" />,
  error: <AlertCircle className="h-8 w-8" />,
};

const iconContainerStyles = {
  default: 'bg-primary-100 text-primary-500',
  search: 'bg-primary-100 text-primary-500',
  error: 'bg-error-100 text-error-500',
};

export function EmptyState({
  icon,
  title,
  description,
  action,
  variant = 'default',
}: EmptyStateProps) {
  const displayIcon = icon || defaultIcons[variant];

  return (
    <div className="flex flex-col items-center justify-center py-16 px-4 text-center animate-fade-in">
      <div className={`w-20 h-20 rounded-full flex items-center justify-center mb-5 ${iconContainerStyles[variant]}`}>
        {displayIcon}
      </div>
      <h3 className="text-lg font-semibold text-gray-900 mb-2">{title}</h3>
      {description && (
        <p className="text-sm text-gray-500 mb-6 max-w-md leading-relaxed">{description}</p>
      )}
      {action && (
        <Button onClick={action.onClick} size="md">
          {action.label}
        </Button>
      )}
    </div>
  );
}
