import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { LoadingSpinner, LoadingOverlay, PageLoading } from './LoadingSpinner';

describe('LoadingSpinner', () => {
  describe('Rendering', () => {
    it('renders with default props', () => {
      render(<LoadingSpinner />);
      const spinner = screen.getByRole('status');

      expect(spinner).toBeInTheDocument();
      expect(spinner).toHaveAttribute('aria-label', 'Loading');
    });

    it('renders with custom className', () => {
      render(<LoadingSpinner className="custom-class" />);
      const spinner = screen.getByRole('status');

      expect(spinner).toHaveClass('custom-class');
    });
  });

  describe('Size variants', () => {
    it('renders small size', () => {
      render(<LoadingSpinner size="sm" />);
      const spinner = screen.getByRole('status');

      expect(spinner).toHaveClass('h-4', 'w-4', 'border-2');
    });

    it('renders medium size by default', () => {
      render(<LoadingSpinner />);
      const spinner = screen.getByRole('status');

      expect(spinner).toHaveClass('h-8', 'w-8');
    });

    it('renders large size', () => {
      render(<LoadingSpinner size="lg" />);
      const spinner = screen.getByRole('status');

      expect(spinner).toHaveClass('h-12', 'w-12');
    });
  });

  describe('Styling', () => {
    it('has correct color classes', () => {
      render(<LoadingSpinner />);
      const spinner = screen.getByRole('status');

      expect(spinner).toHaveClass('border-primary-100', 'border-t-primary-600');
    });

    it('has animation class', () => {
      render(<LoadingSpinner />);
      const spinner = screen.getByRole('status');

      expect(spinner).toHaveClass('animate-spin');
    });

    it('has rounded full class', () => {
      render(<LoadingSpinner />);
      const spinner = screen.getByRole('status');

      expect(spinner).toHaveClass('rounded-full');
    });
  });

  describe('Accessibility', () => {
    it('has status role', () => {
      render(<LoadingSpinner />);
      const spinner = screen.getByRole('status');

      expect(spinner).toBeInTheDocument();
    });

    it('has aria-label', () => {
      render(<LoadingSpinner />);
      const spinner = screen.getByRole('status');

      expect(spinner).toHaveAttribute('aria-label', 'Loading');
    });
  });
});

describe('LoadingOverlay', () => {
  describe('Rendering', () => {
    it('renders with default message', () => {
      render(<LoadingOverlay />);

      expect(screen.getByText('Loading...')).toBeInTheDocument();
      expect(screen.getByRole('status')).toBeInTheDocument();
    });

    it('renders with custom message', () => {
      render(<LoadingOverlay message="Processing your request..." />);

      expect(screen.getByText('Processing your request...')).toBeInTheDocument();
    });

    it('includes large spinner', () => {
      render(<LoadingOverlay />);
      const spinner = screen.getByRole('status');

      expect(spinner).toHaveClass('h-12', 'w-12');
    });
  });

  describe('Styling', () => {
    it('has overlay styles', () => {
      const { container } = render(<LoadingOverlay />);
      const overlay = container.firstChild as HTMLElement;

      expect(overlay).toHaveClass('absolute', 'inset-0', 'bg-white/90');
    });

    it('has backdrop blur', () => {
      const { container } = render(<LoadingOverlay />);
      const overlay = container.firstChild as HTMLElement;

      expect(overlay).toHaveClass('backdrop-blur-sm');
    });

    it('has fade-in animation', () => {
      const { container } = render(<LoadingOverlay />);
      const overlay = container.firstChild as HTMLElement;

      expect(overlay).toHaveClass('animate-fade-in');
    });

    it('has z-index for stacking', () => {
      const { container } = render(<LoadingOverlay />);
      const overlay = container.firstChild as HTMLElement;

      expect(overlay).toHaveClass('z-10');
    });

    it('centers content', () => {
      const { container } = render(<LoadingOverlay />);
      const overlay = container.firstChild as HTMLElement;

      expect(overlay).toHaveClass('flex', 'items-center', 'justify-center');
    });
  });

  describe('Message styling', () => {
    it('applies correct text styles', () => {
      render(<LoadingOverlay message="Custom message" />);
      const message = screen.getByText('Custom message');

      expect(message).toHaveClass('text-sm', 'text-gray-600', 'font-medium');
    });
  });
});

describe('PageLoading', () => {
  describe('Rendering', () => {
    it('renders with default message', () => {
      render(<PageLoading />);

      expect(screen.getByText('Loading...')).toBeInTheDocument();
      expect(screen.getByRole('status')).toBeInTheDocument();
    });

    it('renders with custom message', () => {
      render(<PageLoading message="Fetching data..." />);

      expect(screen.getByText('Fetching data...')).toBeInTheDocument();
    });

    it('includes large spinner', () => {
      render(<PageLoading />);
      const spinner = screen.getByRole('status');

      expect(spinner).toHaveClass('h-12', 'w-12');
    });
  });

  describe('Styling', () => {
    it('has minimum height', () => {
      const { container } = render(<PageLoading />);
      const wrapper = container.firstChild as HTMLElement;

      expect(wrapper).toHaveClass('min-h-[400px]');
    });

    it('centers content', () => {
      const { container } = render(<PageLoading />);
      const wrapper = container.firstChild as HTMLElement;

      expect(wrapper).toHaveClass('flex', 'items-center', 'justify-center');
    });

    it('has fade-in animation', () => {
      const { container } = render(<PageLoading />);
      const wrapper = container.firstChild as HTMLElement;

      expect(wrapper).toHaveClass('animate-fade-in');
    });
  });

  describe('Icon container', () => {
    it('has background circle', () => {
      const { container } = render(<PageLoading />);
      const iconContainer = container.querySelector('.bg-primary-50');

      expect(iconContainer).toBeInTheDocument();
      expect(iconContainer).toHaveClass('w-16', 'h-16', 'rounded-full');
    });

    it('centers spinner within', () => {
      const { container } = render(<PageLoading />);
      const iconContainer = container.querySelector('.bg-primary-50');

      expect(iconContainer).toHaveClass('flex', 'items-center', 'justify-center');
    });
  });

  describe('Message styling', () => {
    it('applies correct text styles', () => {
      render(<PageLoading message="Custom message" />);
      const message = screen.getByText('Custom message');

      expect(message).toHaveClass('text-sm', 'text-gray-500', 'font-medium');
    });
  });
});
