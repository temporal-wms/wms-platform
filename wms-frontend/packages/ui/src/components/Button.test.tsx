import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from './Button';
import { Star } from 'lucide-react';

describe('Button', () => {
  describe('Rendering', () => {
    it('renders with children', () => {
      render(<Button>Click me</Button>);

      expect(screen.getByRole('button', { name: 'Click me' })).toBeInTheDocument();
    });

    it('renders with custom className', () => {
      render(<Button className="custom-class">Test</Button>);
      const button = screen.getByRole('button');

      expect(button).toHaveClass('custom-class');
    });

    it('forwards ref to button element', () => {
      const ref = vi.fn();
      render(<Button ref={ref}>Test</Button>);

      expect(ref).toHaveBeenCalledWith(expect.any(HTMLButtonElement));
    });
  });

  describe('Variants', () => {
    it('renders primary variant by default', () => {
      render(<Button>Primary</Button>);
      const button = screen.getByRole('button');

      expect(button).toHaveClass('bg-primary-600', 'text-white');
    });

    it('renders secondary variant', () => {
      render(<Button variant="secondary">Secondary</Button>);
      const button = screen.getByRole('button');

      expect(button).toHaveClass('bg-white', 'text-gray-700', 'border');
    });

    it('renders danger variant', () => {
      render(<Button variant="danger">Danger</Button>);
      const button = screen.getByRole('button');

      expect(button).toHaveClass('bg-error-600', 'text-white');
    });

    it('renders ghost variant', () => {
      render(<Button variant="ghost">Ghost</Button>);
      const button = screen.getByRole('button');

      expect(button).toHaveClass('bg-transparent', 'text-primary-600');
    });

    it('renders outline variant', () => {
      render(<Button variant="outline">Outline</Button>);
      const button = screen.getByRole('button');

      expect(button).toHaveClass('border-2', 'border-primary-600', 'bg-transparent');
    });
  });

  describe('Sizes', () => {
    it('renders small size', () => {
      render(<Button size="sm">Small</Button>);
      const button = screen.getByRole('button');

      expect(button).toHaveClass('px-3', 'py-1.5', 'text-sm');
    });

    it('renders medium size by default', () => {
      render(<Button>Medium</Button>);
      const button = screen.getByRole('button');

      expect(button).toHaveClass('px-4', 'py-2', 'text-sm');
    });

    it('renders large size', () => {
      render(<Button size="lg">Large</Button>);
      const button = screen.getByRole('button');

      expect(button).toHaveClass('px-6', 'py-2.5', 'text-base');
    });
  });

  describe('Loading state', () => {
    it('shows loading spinner when loading', () => {
      render(<Button loading>Submit</Button>);

      expect(screen.getByRole('button')).toBeInTheDocument();
      const spinner = screen.getByRole('button').querySelector('.animate-spin');
      expect(spinner).toBeInTheDocument();
    });

    it('disables button when loading', () => {
      render(<Button loading>Submit</Button>);

      expect(screen.getByRole('button')).toBeDisabled();
    });

    it('does not show icon when loading', () => {
      render(
        <Button loading icon={<Star data-testid="icon" />}>
          Submit
        </Button>
      );

      expect(screen.queryByTestId('icon')).not.toBeInTheDocument();
    });

    it('still shows children text when loading', () => {
      render(<Button loading>Processing...</Button>);

      expect(screen.getByText('Processing...')).toBeInTheDocument();
    });

    it('is not disabled when not loading', () => {
      render(<Button loading={false}>Submit</Button>);

      expect(screen.getByRole('button')).not.toBeDisabled();
    });
  });

  describe('Disabled state', () => {
    it('disables button when disabled prop is true', () => {
      render(<Button disabled>Disabled</Button>);

      expect(screen.getByRole('button')).toBeDisabled();
    });

    it('applies disabled styles', () => {
      render(<Button disabled>Disabled</Button>);
      const button = screen.getByRole('button');

      expect(button).toHaveClass('disabled:opacity-50', 'disabled:cursor-not-allowed');
    });

    it('does not call onClick when disabled', async () => {
      const user = userEvent.setup();
      const onClick = vi.fn();
      render(
        <Button disabled onClick={onClick}>
          Disabled
        </Button>
      );

      await user.click(screen.getByRole('button'));

      expect(onClick).not.toHaveBeenCalled();
    });
  });

  describe('Icons', () => {
    it('renders icon on left by default', () => {
      render(
        <Button icon={<Star data-testid="icon" />}>
          With Icon
        </Button>
      );

      const button = screen.getByRole('button');
      const icon = screen.getByTestId('icon');

      expect(icon).toBeInTheDocument();
      expect(button.firstChild).toContainElement(icon);
    });

    it('renders icon on right when specified', () => {
      render(
        <Button icon={<Star data-testid="icon" />} iconPosition="right">
          With Icon
        </Button>
      );

      const button = screen.getByRole('button');
      const icon = screen.getByTestId('icon');

      expect(icon).toBeInTheDocument();
      expect(button.lastChild).toContainElement(icon);
    });

    it('renders button without icon when not provided', () => {
      render(<Button>No Icon</Button>);

      expect(screen.getByRole('button')).toBeInTheDocument();
    });
  });

  describe('User interactions', () => {
    it('calls onClick when clicked', async () => {
      const user = userEvent.setup();
      const onClick = vi.fn();
      render(<Button onClick={onClick}>Click me</Button>);

      await user.click(screen.getByRole('button'));

      expect(onClick).toHaveBeenCalledOnce();
    });

    it('calls onClick multiple times', async () => {
      const user = userEvent.setup();
      const onClick = vi.fn();
      render(<Button onClick={onClick}>Click me</Button>);

      const button = screen.getByRole('button');
      await user.click(button);
      await user.click(button);
      await user.click(button);

      expect(onClick).toHaveBeenCalledTimes(3);
    });

    it('does not call onClick when loading', async () => {
      const user = userEvent.setup();
      const onClick = vi.fn();
      render(
        <Button loading onClick={onClick}>
          Loading
        </Button>
      );

      await user.click(screen.getByRole('button'));

      expect(onClick).not.toHaveBeenCalled();
    });
  });

  describe('HTML button attributes', () => {
    it('passes through type attribute', () => {
      render(<Button type="submit">Submit</Button>);

      expect(screen.getByRole('button')).toHaveAttribute('type', 'submit');
    });

    it('passes through name attribute', () => {
      render(<Button name="submit-button">Submit</Button>);

      expect(screen.getByRole('button')).toHaveAttribute('name', 'submit-button');
    });

    it('passes through form attribute', () => {
      render(<Button form="my-form">Submit</Button>);

      expect(screen.getByRole('button')).toHaveAttribute('form', 'my-form');
    });

    it('passes through aria attributes', () => {
      render(<Button aria-label="Submit form">Submit</Button>);

      expect(screen.getByRole('button')).toHaveAttribute('aria-label', 'Submit form');
    });

    it('passes through data attributes', () => {
      render(<Button data-testid="custom-button">Submit</Button>);

      expect(screen.getByTestId('custom-button')).toBeInTheDocument();
    });
  });

  describe('Styling', () => {
    it('has correct base classes', () => {
      render(<Button>Test</Button>);
      const button = screen.getByRole('button');

      expect(button).toHaveClass(
        'inline-flex',
        'items-center',
        'justify-center',
        'rounded-lg',
        'font-medium'
      );
    });

    it('has transition classes', () => {
      render(<Button>Test</Button>);
      const button = screen.getByRole('button');

      expect(button).toHaveClass('transition-all', 'duration-150', 'ease-out');
    });

    it('has focus ring classes', () => {
      render(<Button>Test</Button>);
      const button = screen.getByRole('button');

      expect(button).toHaveClass('focus:outline-none', 'focus:ring-2', 'focus:ring-offset-2');
    });

    it('combines variant, size, and custom classes correctly', () => {
      render(
        <Button variant="danger" size="lg" className="extra-class">
          Test
        </Button>
      );
      const button = screen.getByRole('button');

      expect(button).toHaveClass('bg-error-600'); // variant
      expect(button).toHaveClass('px-6', 'py-2.5'); // size
      expect(button).toHaveClass('extra-class'); // custom
    });
  });

  describe('Complete examples', () => {
    it('renders primary button with icon and loading', () => {
      const { rerender } = render(
        <Button variant="primary" icon={<Star data-testid="icon" />}>
          Save
        </Button>
      );

      expect(screen.getByRole('button')).toBeInTheDocument();
      expect(screen.getByTestId('icon')).toBeInTheDocument();

      rerender(
        <Button variant="primary" loading icon={<Star data-testid="icon" />}>
          Saving...
        </Button>
      );

      expect(screen.getByRole('button')).toBeDisabled();
      expect(screen.queryByTestId('icon')).not.toBeInTheDocument();
    });

    it('renders full-featured button', async () => {
      const user = userEvent.setup();
      const onClick = vi.fn();
      render(
        <Button
          variant="secondary"
          size="lg"
          icon={<Star data-testid="icon" />}
          iconPosition="right"
          onClick={onClick}
          className="w-full"
          type="submit"
        >
          Submit Form
        </Button>
      );

      const button = screen.getByRole('button');

      expect(button).toHaveClass('bg-white', 'text-gray-700'); // variant
      expect(button).toHaveClass('px-6', 'py-2.5'); // size
      expect(button).toHaveClass('w-full'); // custom class
      expect(button).toHaveAttribute('type', 'submit');
      expect(screen.getByTestId('icon')).toBeInTheDocument();

      await user.click(button);
      expect(onClick).toHaveBeenCalledOnce();
    });
  });

  describe('Accessibility', () => {
    it('is keyboard accessible', async () => {
      const user = userEvent.setup();
      const onClick = vi.fn();
      render(<Button onClick={onClick}>Submit</Button>);

      const button = screen.getByRole('button');
      button.focus();

      await user.keyboard('{Enter}');

      expect(onClick).toHaveBeenCalledOnce();
    });

    it('has correct button role', () => {
      render(<Button>Test</Button>);

      expect(screen.getByRole('button')).toBeInTheDocument();
    });
  });
});
