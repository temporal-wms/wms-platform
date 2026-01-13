import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Badge, StatusBadge } from './Badge';

describe('Badge', () => {
  describe('Rendering', () => {
    it('renders with children', () => {
      render(<Badge>Active</Badge>);

      expect(screen.getByText('Active')).toBeInTheDocument();
    });

    it('renders with custom className', () => {
      render(<Badge className="custom-class">Test</Badge>);
      const badge = screen.getByText('Test');

      expect(badge).toHaveClass('custom-class');
    });
  });

  describe('Variants', () => {
    it('renders default variant', () => {
      render(<Badge variant="default">Default</Badge>);
      const badge = screen.getByText('Default');

      expect(badge).toHaveClass('bg-primary-100', 'text-primary-800');
    });

    it('renders success variant', () => {
      render(<Badge variant="success">Success</Badge>);
      const badge = screen.getByText('Success');

      expect(badge).toHaveClass('bg-success-100', 'text-success-700');
    });

    it('renders warning variant', () => {
      render(<Badge variant="warning">Warning</Badge>);
      const badge = screen.getByText('Warning');

      expect(badge).toHaveClass('bg-warning-100', 'text-warning-700');
    });

    it('renders error variant', () => {
      render(<Badge variant="error">Error</Badge>);
      const badge = screen.getByText('Error');

      expect(badge).toHaveClass('bg-error-100', 'text-error-700');
    });

    it('renders info variant', () => {
      render(<Badge variant="info">Info</Badge>);
      const badge = screen.getByText('Info');

      expect(badge).toHaveClass('bg-info-100', 'text-info-700');
    });

    it('renders neutral variant', () => {
      render(<Badge variant="neutral">Neutral</Badge>);
      const badge = screen.getByText('Neutral');

      expect(badge).toHaveClass('bg-gray-100', 'text-gray-700');
    });

    it('uses default variant when not specified', () => {
      render(<Badge>Default</Badge>);
      const badge = screen.getByText('Default');

      expect(badge).toHaveClass('bg-primary-100', 'text-primary-800');
    });
  });

  describe('Sizes', () => {
    it('renders small size', () => {
      render(<Badge size="sm">Small</Badge>);
      const badge = screen.getByText('Small');

      expect(badge).toHaveClass('px-2', 'py-0.5', 'text-xs');
    });

    it('renders medium size', () => {
      render(<Badge size="md">Medium</Badge>);
      const badge = screen.getByText('Medium');

      expect(badge).toHaveClass('px-2.5', 'py-1', 'text-sm');
    });

    it('uses medium size by default', () => {
      render(<Badge>Default Size</Badge>);
      const badge = screen.getByText('Default Size');

      expect(badge).toHaveClass('px-2.5', 'py-1', 'text-sm');
    });
  });

  describe('Styling', () => {
    it('has correct base classes', () => {
      render(<Badge>Test</Badge>);
      const badge = screen.getByText('Test');

      expect(badge).toHaveClass('inline-flex', 'items-center', 'font-medium', 'rounded-full');
    });

    it('combines variant, size, and custom classes', () => {
      render(
        <Badge variant="success" size="sm" className="extra-class">
          Test
        </Badge>
      );
      const badge = screen.getByText('Test');

      expect(badge).toHaveClass('bg-success-100', 'text-success-700'); // variant
      expect(badge).toHaveClass('px-2', 'py-0.5', 'text-xs'); // size
      expect(badge).toHaveClass('extra-class'); // custom
    });

    it('renders as span element', () => {
      render(<Badge>Test</Badge>);
      const badge = screen.getByText('Test');

      expect(badge.tagName).toBe('SPAN');
    });
  });

  describe('Content types', () => {
    it('renders text content', () => {
      render(<Badge>Text Content</Badge>);

      expect(screen.getByText('Text Content')).toBeInTheDocument();
    });

    it('renders numeric content', () => {
      render(<Badge>{42}</Badge>);

      expect(screen.getByText('42')).toBeInTheDocument();
    });

    it('renders mixed content', () => {
      render(<Badge>Count: {10}</Badge>);

      expect(screen.getByText('Count: 10')).toBeInTheDocument();
    });
  });
});

describe('StatusBadge', () => {
  describe('Order statuses', () => {
    it('renders PENDING status', () => {
      render(<StatusBadge status="PENDING" />);
      const badge = screen.getByText('PENDING');

      expect(badge).toBeInTheDocument();
      expect(badge).toHaveClass('bg-warning-100', 'text-warning-700');
    });

    it('renders VALIDATED status', () => {
      render(<StatusBadge status="VALIDATED" />);
      const badge = screen.getByText('VALIDATED');

      expect(badge).toHaveClass('bg-info-100', 'text-info-700');
    });

    it('renders COMPLETED status', () => {
      render(<StatusBadge status="COMPLETED" />);
      const badge = screen.getByText('COMPLETED');

      expect(badge).toHaveClass('bg-success-100', 'text-success-700');
    });

    it('renders FAILED status', () => {
      render(<StatusBadge status="FAILED" />);
      const badge = screen.getByText('FAILED');

      expect(badge).toHaveClass('bg-error-100', 'text-error-700');
    });

    it('renders DLQ status', () => {
      render(<StatusBadge status="DLQ" />);
      const badge = screen.getByText('DLQ');

      expect(badge).toHaveClass('bg-error-100', 'text-error-700');
    });

    it('renders PICKED status', () => {
      render(<StatusBadge status="PICKED" />);
      const badge = screen.getByText('PICKED');

      expect(badge).toHaveClass('bg-primary-100', 'text-primary-800');
    });

    it('renders SHIPPED status', () => {
      render(<StatusBadge status="SHIPPED" />);
      const badge = screen.getByText('SHIPPED');

      expect(badge).toHaveClass('bg-success-100', 'text-success-700');
    });
  });

  describe('Wave statuses', () => {
    it('renders PLANNING status', () => {
      render(<StatusBadge status="PLANNING" />);
      const badge = screen.getByText('PLANNING');

      expect(badge).toHaveClass('bg-gray-100', 'text-gray-700');
    });

    it('renders READY status', () => {
      render(<StatusBadge status="READY" />);
      const badge = screen.getByText('READY');

      expect(badge).toHaveClass('bg-info-100', 'text-info-700');
    });

    it('renders RELEASED status', () => {
      render(<StatusBadge status="RELEASED" />);
      const badge = screen.getByText('RELEASED');

      expect(badge).toHaveClass('bg-primary-100', 'text-primary-800');
    });

    it('renders IN_PROGRESS status with formatted text', () => {
      render(<StatusBadge status="IN_PROGRESS" />);
      const badge = screen.getByText('IN PROGRESS');

      expect(badge).toBeInTheDocument();
      expect(badge).toHaveClass('bg-warning-100', 'text-warning-700');
    });

    it('renders CANCELLED status', () => {
      render(<StatusBadge status="CANCELLED" />);
      const badge = screen.getByText('CANCELLED');

      expect(badge).toHaveClass('bg-gray-100', 'text-gray-700');
    });
  });

  describe('Worker statuses', () => {
    it('renders AVAILABLE status', () => {
      render(<StatusBadge status="AVAILABLE" />);
      const badge = screen.getByText('AVAILABLE');

      expect(badge).toHaveClass('bg-success-100', 'text-success-700');
    });

    it('renders BUSY status', () => {
      render(<StatusBadge status="BUSY" />);
      const badge = screen.getByText('BUSY');

      expect(badge).toHaveClass('bg-warning-100', 'text-warning-700');
    });

    it('renders BREAK status', () => {
      render(<StatusBadge status="BREAK" />);
      const badge = screen.getByText('BREAK');

      expect(badge).toHaveClass('bg-gray-100', 'text-gray-700');
    });

    it('renders OFFLINE status', () => {
      render(<StatusBadge status="OFFLINE" />);
      const badge = screen.getByText('OFFLINE');

      expect(badge).toHaveClass('bg-gray-100', 'text-gray-700');
    });
  });

  describe('Text formatting', () => {
    it('replaces underscores with spaces', () => {
      render(<StatusBadge status="IN_PROGRESS" />);

      expect(screen.getByText('IN PROGRESS')).toBeInTheDocument();
      expect(screen.queryByText('IN_PROGRESS')).not.toBeInTheDocument();
    });

    it('handles status without underscores', () => {
      render(<StatusBadge status="PENDING" />);

      expect(screen.getByText('PENDING')).toBeInTheDocument();
    });

    it('handles multiple underscores', () => {
      render(<StatusBadge status="SOME_LONG_STATUS" />);

      expect(screen.getByText('SOME LONG STATUS')).toBeInTheDocument();
    });
  });

  describe('Unknown status', () => {
    it('renders unknown status with neutral variant', () => {
      render(<StatusBadge status="UNKNOWN_STATUS" />);
      const badge = screen.getByText('UNKNOWN STATUS');

      expect(badge).toBeInTheDocument();
      expect(badge).toHaveClass('bg-gray-100', 'text-gray-700');
    });

    it('handles empty string status', () => {
      const { container } = render(<StatusBadge status="" />);
      const badge = container.querySelector('span');

      expect(badge).toHaveClass('bg-gray-100', 'text-gray-700');
    });
  });

  describe('Custom className', () => {
    it('applies custom className', () => {
      render(<StatusBadge status="PENDING" className="custom-badge" />);
      const badge = screen.getByText('PENDING');

      expect(badge).toHaveClass('custom-badge');
    });

    it('combines status variant with custom className', () => {
      render(<StatusBadge status="COMPLETED" className="ml-2" />);
      const badge = screen.getByText('COMPLETED');

      expect(badge).toHaveClass('bg-success-100', 'text-success-700', 'ml-2');
    });
  });
});
