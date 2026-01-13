import React from 'react';
import { Loader2 } from 'lucide-react';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost' | 'outline';
  size?: 'sm' | 'md' | 'lg';
  loading?: boolean;
  icon?: React.ReactNode;
  iconPosition?: 'left' | 'right';
}

// Modern button styles with micro-interactions
const variantStyles = {
  primary: `
    bg-primary-600 text-white
    hover:bg-primary-700
    active:scale-[0.98]
    shadow-sm hover:shadow-md
    focus:ring-primary-500
  `,
  secondary: `
    bg-white text-gray-700
    border border-gray-200
    hover:bg-gray-50 hover:border-gray-300
    active:scale-[0.98]
    shadow-sm
    focus:ring-gray-400
  `,
  danger: `
    bg-error-600 text-white
    hover:bg-error-700
    active:scale-[0.98]
    shadow-sm hover:shadow-md
    focus:ring-error-500
  `,
  ghost: `
    bg-transparent text-primary-600
    hover:bg-primary-50
    active:bg-primary-100
    focus:ring-primary-500
  `,
  outline: `
    border-2 border-primary-600 bg-transparent text-primary-600
    hover:bg-primary-50
    active:scale-[0.98]
    focus:ring-primary-500
  `,
};

const sizeStyles = {
  sm: 'px-3 py-1.5 text-sm gap-1.5',
  md: 'px-4 py-2 text-sm gap-2',
  lg: 'px-6 py-2.5 text-base gap-2',
};

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      className = '',
      variant = 'primary',
      size = 'md',
      loading = false,
      disabled,
      icon,
      iconPosition = 'left',
      children,
      ...props
    },
    ref
  ) => {
    const isDisabled = disabled || loading;

    return (
      <button
        ref={ref}
        className={`
          inline-flex items-center justify-center rounded-lg font-medium
          transition-all duration-150 ease-out
          focus:outline-none focus:ring-2 focus:ring-offset-2
          disabled:opacity-50 disabled:cursor-not-allowed disabled:scale-100 disabled:shadow-none
          ${variantStyles[variant]}
          ${sizeStyles[size]}
          ${className}
        `}
        disabled={isDisabled}
        {...props}
      >
        {loading && <Loader2 className="h-4 w-4 animate-spin" />}
        {!loading && icon && iconPosition === 'left' && icon}
        {children}
        {!loading && icon && iconPosition === 'right' && icon}
      </button>
    );
  }
);

Button.displayName = 'Button';
