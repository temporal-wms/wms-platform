import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { EmptyState } from './EmptyState';
import { Star } from 'lucide-react';

describe('EmptyState', () => {
  describe('Rendering', () => {
    it('renders with required props', () => {
      render(<EmptyState title="No items found" />);

      expect(screen.getByText('No items found')).toBeInTheDocument();
    });

    it('renders with title and description', () => {
      render(
        <EmptyState
          title="No orders"
          description="You haven't created any orders yet"
        />
      );

      expect(screen.getByText('No orders')).toBeInTheDocument();
      expect(screen.getByText("You haven't created any orders yet")).toBeInTheDocument();
    });

    it('does not render description when not provided', () => {
      const { container } = render(<EmptyState title="No items" />);
      const descriptions = container.querySelectorAll('p');

      expect(descriptions.length).toBe(0);
    });
  });

  describe('Variants', () => {
    it('renders default variant with Package icon', () => {
      const { container } = render(<EmptyState title="No items" variant="default" />);

      expect(container.querySelector('.bg-primary-100')).toBeInTheDocument();
      expect(container.querySelector('.text-primary-500')).toBeInTheDocument();
    });

    it('renders search variant with Search icon', () => {
      const { container } = render(<EmptyState title="No results" variant="search" />);

      expect(container.querySelector('.bg-primary-100')).toBeInTheDocument();
      expect(container.querySelector('.text-primary-500')).toBeInTheDocument();
    });

    it('renders error variant with AlertCircle icon', () => {
      const { container } = render(<EmptyState title="Error occurred" variant="error" />);

      expect(container.querySelector('.bg-error-100')).toBeInTheDocument();
      expect(container.querySelector('.text-error-500')).toBeInTheDocument();
    });

    it('uses default variant when not specified', () => {
      const { container } = render(<EmptyState title="No items" />);

      expect(container.querySelector('.bg-primary-100')).toBeInTheDocument();
    });
  });

  describe('Custom icon', () => {
    it('renders custom icon instead of default', () => {
      render(
        <EmptyState
          title="No favorites"
          icon={<Star data-testid="custom-icon" />}
        />
      );

      expect(screen.getByTestId('custom-icon')).toBeInTheDocument();
    });

    it('uses custom icon regardless of variant', () => {
      render(
        <EmptyState
          title="No items"
          variant="error"
          icon={<Star data-testid="custom-icon" />}
        />
      );

      expect(screen.getByTestId('custom-icon')).toBeInTheDocument();
    });
  });

  describe('Action button', () => {
    it('renders action button when provided', () => {
      const onClick = vi.fn();
      render(
        <EmptyState
          title="No items"
          action={{ label: 'Create item', onClick }}
        />
      );

      expect(screen.getByRole('button', { name: 'Create item' })).toBeInTheDocument();
    });

    it('does not render button when action not provided', () => {
      render(<EmptyState title="No items" />);

      expect(screen.queryByRole('button')).not.toBeInTheDocument();
    });

    it('calls onClick when button is clicked', async () => {
      const user = userEvent.setup();
      const onClick = vi.fn();
      render(
        <EmptyState
          title="No items"
          action={{ label: 'Add new', onClick }}
        />
      );

      const button = screen.getByRole('button', { name: 'Add new' });
      await user.click(button);

      expect(onClick).toHaveBeenCalledOnce();
    });

    it('renders action with custom label', () => {
      render(
        <EmptyState
          title="No items"
          action={{ label: 'Custom Action', onClick: vi.fn() }}
        />
      );

      expect(screen.getByRole('button', { name: 'Custom Action' })).toBeInTheDocument();
    });
  });

  describe('Styling', () => {
    it('has correct container classes', () => {
      const { container } = render(<EmptyState title="Test" />);
      const wrapper = container.firstChild as HTMLElement;

      expect(wrapper).toHaveClass('flex', 'flex-col', 'items-center', 'justify-center');
      expect(wrapper).toHaveClass('py-16', 'px-4', 'text-center');
    });

    it('has fade-in animation', () => {
      const { container } = render(<EmptyState title="Test" />);
      const wrapper = container.firstChild as HTMLElement;

      expect(wrapper).toHaveClass('animate-fade-in');
    });

    it('has correct icon container size', () => {
      const { container } = render(<EmptyState title="Test" />);
      const iconContainer = container.querySelector('.w-20');

      expect(iconContainer).toHaveClass('h-20', 'rounded-full');
    });

    it('has correct title styling', () => {
      render(<EmptyState title="Test Title" />);
      const title = screen.getByText('Test Title');

      expect(title.tagName).toBe('H3');
      expect(title).toHaveClass('text-lg', 'font-semibold', 'text-gray-900', 'mb-2');
    });

    it('has correct description styling', () => {
      render(<EmptyState title="Test" description="Test description" />);
      const description = screen.getByText('Test description');

      expect(description.tagName).toBe('P');
      expect(description).toHaveClass('text-sm', 'text-gray-500', 'mb-6', 'max-w-md', 'leading-relaxed');
    });
  });

  describe('Complete examples', () => {
    it('renders complete state with all props', () => {
      const onClick = vi.fn();
      render(
        <EmptyState
          title="No orders found"
          description="Start by creating your first order"
          variant="search"
          action={{ label: 'Create Order', onClick }}
        />
      );

      expect(screen.getByText('No orders found')).toBeInTheDocument();
      expect(screen.getByText('Start by creating your first order')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Create Order' })).toBeInTheDocument();
    });

    it('renders minimal state with only title', () => {
      render(<EmptyState title="Empty" />);

      expect(screen.getByText('Empty')).toBeInTheDocument();
      expect(screen.queryByRole('button')).not.toBeInTheDocument();
    });

    it('renders error state with action', async () => {
      const user = userEvent.setup();
      const onRetry = vi.fn();
      render(
        <EmptyState
          title="Failed to load"
          description="An error occurred while fetching data"
          variant="error"
          action={{ label: 'Retry', onClick: onRetry }}
        />
      );

      const retryButton = screen.getByRole('button', { name: 'Retry' });
      await user.click(retryButton);

      expect(onRetry).toHaveBeenCalledOnce();
    });
  });
});
