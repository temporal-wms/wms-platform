import React from 'react';

export interface BadgeProps {
  variant?: 'default' | 'success' | 'warning' | 'error' | 'info' | 'neutral';
  size?: 'sm' | 'md';
  children: React.ReactNode;
  className?: string;
}

const variantStyles = {
  default: 'bg-primary-100 text-primary-800',
  success: 'bg-success-100 text-success-700',
  warning: 'bg-warning-100 text-warning-700',
  error: 'bg-error-100 text-error-700',
  info: 'bg-info-100 text-info-700',
  neutral: 'bg-gray-100 text-gray-700',
};

const sizeStyles = {
  sm: 'px-2 py-0.5 text-xs',
  md: 'px-2.5 py-1 text-sm',
};

export function Badge({
  variant = 'default',
  size = 'md',
  children,
  className = '',
}: BadgeProps) {
  return (
    <span
      className={`
        inline-flex items-center font-medium rounded-full
        ${variantStyles[variant]}
        ${sizeStyles[size]}
        ${className}
      `}
    >
      {children}
    </span>
  );
}

// Status-specific badge for WMS statuses
export interface StatusBadgeProps {
  status: string;
  className?: string;
}

const statusVariantMap: Record<string, BadgeProps['variant']> = {
  // Order statuses
  PENDING: 'warning',
  VALIDATED: 'info',
  WAVED: 'info',
  PICKING: 'default',
  PICKED: 'default',
  PACKING: 'default',
  PACKED: 'success',
  SHIPPING: 'info',
  SHIPPED: 'success',
  COMPLETED: 'success',
  FAILED: 'error',
  DLQ: 'error',
  // Wave statuses
  PLANNING: 'neutral',
  READY: 'info',
  RELEASED: 'default',
  IN_PROGRESS: 'warning',
  CANCELLED: 'neutral',
  // Worker statuses
  AVAILABLE: 'success',
  BUSY: 'warning',
  BREAK: 'neutral',
  OFFLINE: 'neutral',
};

export function StatusBadge({ status, className = '' }: StatusBadgeProps) {
  const variant = statusVariantMap[status] || 'neutral';
  const displayText = status.replace(/_/g, ' ');

  return (
    <Badge variant={variant} className={className}>
      {displayText}
    </Badge>
  );
}
