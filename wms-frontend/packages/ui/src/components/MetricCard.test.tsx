import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MetricCard, MetricGrid } from './MetricCard';
import { Package } from 'lucide-react';

describe('MetricCard', () => {
  describe('Rendering', () => {
    it('renders with required props', () => {
      render(<MetricCard title="Total Orders" value={1250} />);

      expect(screen.getByText('Total Orders')).toBeInTheDocument();
      expect(screen.getByText('1250')).toBeInTheDocument();
    });

    it('renders with string value', () => {
      render(<MetricCard title="Status" value="Active" />);

      expect(screen.getByText('Status')).toBeInTheDocument();
      expect(screen.getByText('Active')).toBeInTheDocument();
    });

    it('renders with number value', () => {
      render(<MetricCard title="Count" value={42} />);

      expect(screen.getByText('42')).toBeInTheDocument();
    });

    it('renders with subtitle', () => {
      render(
        <MetricCard
          title="Total Orders"
          value={1250}
          subtitle="Last 30 days"
        />
      );

      expect(screen.getByText('Total Orders')).toBeInTheDocument();
      expect(screen.getByText('1250')).toBeInTheDocument();
      expect(screen.getByText('Last 30 days')).toBeInTheDocument();
    });

    it('does not render subtitle when not provided', () => {
      const { container } = render(<MetricCard title="Total" value={100} />);
      const subtitles = Array.from(container.querySelectorAll('p')).filter(
        p => p.textContent === 'Last 30 days'
      );

      expect(subtitles.length).toBe(0);
    });

    it('renders with custom className', () => {
      const { container } = render(
        <MetricCard title="Test" value={100} className="custom-class" />
      );
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('custom-class');
    });
  });

  describe('Variants', () => {
    it('renders default variant', () => {
      const { container } = render(
        <MetricCard title="Test" value={100} variant="default" />
      );
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('bg-white', 'border', 'border-gray-100');
    });

    it('renders success variant with left border', () => {
      const { container } = render(
        <MetricCard title="Test" value={100} variant="success" />
      );
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('bg-white', 'border-l-4', 'border-l-success-500');
    });

    it('renders warning variant with left border', () => {
      const { container } = render(
        <MetricCard title="Test" value={100} variant="warning" />
      );
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('bg-white', 'border-l-4', 'border-l-warning-500');
    });

    it('renders error variant with left border', () => {
      const { container } = render(
        <MetricCard title="Test" value={100} variant="error" />
      );
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('bg-white', 'border-l-4', 'border-l-error-500');
    });

    it('renders hero variant with gradient background', () => {
      const { container } = render(
        <MetricCard title="Test" value={100} variant="hero" />
      );
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('bg-gradient-to-br', 'from-primary-500', 'to-primary-600', 'text-white');
    });

    it('uses default variant when not specified', () => {
      const { container } = render(<MetricCard title="Test" value={100} />);
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('bg-white', 'border', 'border-gray-100');
    });
  });

  describe('Icon rendering', () => {
    it('renders icon when provided', () => {
      render(
        <MetricCard
          title="Orders"
          value={100}
          icon={<Package data-testid="metric-icon" />}
        />
      );

      expect(screen.getByTestId('metric-icon')).toBeInTheDocument();
    });

    it('does not render icon container when icon not provided', () => {
      const { container } = render(<MetricCard title="Test" value={100} />);
      const iconContainer = container.querySelector('.w-12.h-12');

      expect(iconContainer).not.toBeInTheDocument();
    });

    it('renders icon with default variant styling', () => {
      const { container } = render(
        <MetricCard
          title="Test"
          value={100}
          variant="default"
          icon={<Package data-testid="icon" />}
        />
      );
      const iconContainer = container.querySelector('.w-12');

      expect(iconContainer).toHaveClass('bg-primary-100', 'text-primary-600');
    });

    it('renders icon with success variant styling', () => {
      const { container } = render(
        <MetricCard
          title="Test"
          value={100}
          variant="success"
          icon={<Package data-testid="icon" />}
        />
      );
      const iconContainer = container.querySelector('.w-12');

      expect(iconContainer).toHaveClass('bg-success-100', 'text-success-600');
    });

    it('renders icon with warning variant styling', () => {
      const { container } = render(
        <MetricCard
          title="Test"
          value={100}
          variant="warning"
          icon={<Package data-testid="icon" />}
        />
      );
      const iconContainer = container.querySelector('.w-12');

      expect(iconContainer).toHaveClass('bg-warning-100', 'text-warning-600');
    });

    it('renders icon with error variant styling', () => {
      const { container } = render(
        <MetricCard
          title="Test"
          value={100}
          variant="error"
          icon={<Package data-testid="icon" />}
        />
      );
      const iconContainer = container.querySelector('.w-12');

      expect(iconContainer).toHaveClass('bg-error-100', 'text-error-600');
    });

    it('renders icon with hero variant styling', () => {
      const { container } = render(
        <MetricCard
          title="Test"
          value={100}
          variant="hero"
          icon={<Package data-testid="icon" />}
        />
      );
      const iconContainer = container.querySelector('.w-12');

      expect(iconContainer).toHaveClass('bg-white/20', 'text-white');
    });
  });

  describe('Trend indicator', () => {
    it('renders upward trend', () => {
      render(
        <MetricCard
          title="Sales"
          value={1000}
          trend={{ value: 12.5, direction: 'up' }}
        />
      );

      expect(screen.getByText('12.5%')).toBeInTheDocument();
    });

    it('renders downward trend', () => {
      render(
        <MetricCard
          title="Sales"
          value={1000}
          trend={{ value: 5.2, direction: 'down' }}
        />
      );

      expect(screen.getByText('5.2%')).toBeInTheDocument();
    });

    it('renders neutral trend', () => {
      render(
        <MetricCard
          title="Sales"
          value={1000}
          trend={{ value: 0, direction: 'neutral' }}
        />
      );

      expect(screen.getByText('0%')).toBeInTheDocument();
    });

    it('renders trend with label', () => {
      render(
        <MetricCard
          title="Sales"
          value={1000}
          trend={{ value: 12.5, direction: 'up', label: 'vs last month' }}
        />
      );

      expect(screen.getByText('12.5%')).toBeInTheDocument();
      expect(screen.getByText('vs last month')).toBeInTheDocument();
    });

    it('does not render trend label when not provided', () => {
      render(
        <MetricCard
          title="Sales"
          value={1000}
          trend={{ value: 12.5, direction: 'up' }}
        />
      );

      expect(screen.queryByText('vs last month')).not.toBeInTheDocument();
    });

    it('does not render trend when not provided', () => {
      render(<MetricCard title="Sales" value={1000} />);

      expect(screen.queryByText('%')).not.toBeInTheDocument();
    });

    it('applies success color to upward trend', () => {
      const { container } = render(
        <MetricCard
          title="Test"
          value={100}
          trend={{ value: 10, direction: 'up' }}
        />
      );
      const trendContainer = container.querySelector('.text-success-600');

      expect(trendContainer).toBeInTheDocument();
    });

    it('applies error color to downward trend', () => {
      const { container } = render(
        <MetricCard
          title="Test"
          value={100}
          trend={{ value: 10, direction: 'down' }}
        />
      );
      const trendContainer = container.querySelector('.text-error-600');

      expect(trendContainer).toBeInTheDocument();
    });

    it('applies neutral color to neutral trend', () => {
      const { container } = render(
        <MetricCard
          title="Test"
          value={100}
          trend={{ value: 0, direction: 'neutral' }}
        />
      );
      const trendContainer = container.querySelector('.text-gray-500');

      expect(trendContainer).toBeInTheDocument();
    });
  });

  describe('Hero variant text colors', () => {
    it('applies hero text colors to title', () => {
      render(
        <MetricCard title="Hero Title" value={1000} variant="hero" />
      );
      const title = screen.getByText('Hero Title');

      expect(title).toHaveClass('text-primary-100');
    });

    it('applies hero text colors to value', () => {
      render(
        <MetricCard title="Test" value={1000} variant="hero" />
      );
      const value = screen.getByText('1000');

      expect(value).toHaveClass('text-white');
    });

    it('applies hero text colors to subtitle', () => {
      render(
        <MetricCard
          title="Test"
          value={1000}
          subtitle="Subtitle"
          variant="hero"
        />
      );
      const subtitle = screen.getByText('Subtitle');

      expect(subtitle).toHaveClass('text-primary-100');
    });

    it('applies hero-specific trend colors for upward trend', () => {
      const { container } = render(
        <MetricCard
          title="Test"
          value={100}
          variant="hero"
          trend={{ value: 10, direction: 'up' }}
        />
      );
      const trendContainer = container.querySelector('.text-success-200');

      expect(trendContainer).toBeInTheDocument();
    });

    it('applies hero-specific trend colors for downward trend', () => {
      const { container } = render(
        <MetricCard
          title="Test"
          value={100}
          variant="hero"
          trend={{ value: 10, direction: 'down' }}
        />
      );
      const trendContainer = container.querySelector('.text-error-200');

      expect(trendContainer).toBeInTheDocument();
    });

    it('applies hero-specific trend colors for neutral trend', () => {
      const { container } = render(
        <MetricCard
          title="Test"
          value={100}
          variant="hero"
          trend={{ value: 0, direction: 'neutral' }}
        />
      );
      const trendContainer = container.querySelector('.text-white\\/70');

      expect(trendContainer).toBeInTheDocument();
    });

    it('applies hero text color to trend label', () => {
      render(
        <MetricCard
          title="Test"
          value={100}
          variant="hero"
          trend={{ value: 10, direction: 'up', label: 'vs last month' }}
        />
      );
      const label = screen.getByText('vs last month');

      expect(label).toHaveClass('text-primary-200');
    });
  });

  describe('Non-hero variant text colors', () => {
    it('applies default text colors to title', () => {
      render(<MetricCard title="Default Title" value={1000} />);
      const title = screen.getByText('Default Title');

      expect(title).toHaveClass('text-gray-500');
    });

    it('applies default text colors to value', () => {
      render(<MetricCard title="Test" value={1000} />);
      const value = screen.getByText('1000');

      expect(value).toHaveClass('text-gray-900');
    });

    it('applies default text colors to subtitle', () => {
      render(
        <MetricCard title="Test" value={1000} subtitle="Subtitle" />
      );
      const subtitle = screen.getByText('Subtitle');

      expect(subtitle).toHaveClass('text-gray-500');
    });

    it('applies default text color to trend label', () => {
      render(
        <MetricCard
          title="Test"
          value={100}
          trend={{ value: 10, direction: 'up', label: 'vs last month' }}
        />
      );
      const label = screen.getByText('vs last month');

      expect(label).toHaveClass('text-gray-400');
    });
  });

  describe('Styling', () => {
    it('has correct base classes', () => {
      const { container } = render(<MetricCard title="Test" value={100} />);
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('rounded-xl', 'shadow-card', 'p-5');
    });

    it('has hover effect', () => {
      const { container } = render(<MetricCard title="Test" value={100} />);
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('hover:shadow-card-hover');
    });

    it('has transition classes', () => {
      const { container } = render(<MetricCard title="Test" value={100} />);
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('transition-all', 'duration-200');
    });

    it('has correct title text size', () => {
      render(<MetricCard title="Test" value={100} />);
      const title = screen.getByText('Test');

      expect(title).toHaveClass('text-sm', 'font-medium');
    });

    it('has correct value text size', () => {
      render(<MetricCard title="Test" value={100} />);
      const value = screen.getByText('100');

      expect(value).toHaveClass('text-3xl', 'font-bold');
    });

    it('has correct subtitle text size', () => {
      render(<MetricCard title="Test" value={100} subtitle="Sub" />);
      const subtitle = screen.getByText('Sub');

      expect(subtitle).toHaveClass('text-sm');
    });
  });

  describe('Complete examples', () => {
    it('renders complete metric card with all props', () => {
      render(
        <MetricCard
          title="Total Revenue"
          value="$125,430"
          subtitle="Last 30 days"
          trend={{ value: 12.5, direction: 'up', label: 'vs last month' }}
          icon={<Package data-testid="icon" />}
          variant="success"
          className="extra-class"
        />
      );

      expect(screen.getByText('Total Revenue')).toBeInTheDocument();
      expect(screen.getByText('$125,430')).toBeInTheDocument();
      expect(screen.getByText('Last 30 days')).toBeInTheDocument();
      expect(screen.getByText('12.5%')).toBeInTheDocument();
      expect(screen.getByText('vs last month')).toBeInTheDocument();
      expect(screen.getByTestId('icon')).toBeInTheDocument();
    });

    it('renders minimal metric card', () => {
      render(<MetricCard title="Count" value={42} />);

      expect(screen.getByText('Count')).toBeInTheDocument();
      expect(screen.getByText('42')).toBeInTheDocument();
      expect(screen.queryByText('%')).not.toBeInTheDocument();
    });

    it('renders hero metric card with trend', () => {
      const { container } = render(
        <MetricCard
          title="Total Orders"
          value={5432}
          variant="hero"
          trend={{ value: 8.2, direction: 'up', label: 'from yesterday' }}
          icon={<Package data-testid="icon" />}
        />
      );
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('bg-gradient-to-br', 'from-primary-500');
      expect(screen.getByText('Total Orders')).toHaveClass('text-primary-100');
      expect(screen.getByText('5432')).toHaveClass('text-white');
      expect(screen.getByText('8.2%')).toBeInTheDocument();
    });
  });
});

describe('MetricGrid', () => {
  describe('Rendering', () => {
    it('renders children', () => {
      render(
        <MetricGrid>
          <MetricCard title="Test 1" value={100} />
          <MetricCard title="Test 2" value={200} />
        </MetricGrid>
      );

      expect(screen.getByText('Test 1')).toBeInTheDocument();
      expect(screen.getByText('Test 2')).toBeInTheDocument();
    });

    it('renders with custom className', () => {
      const { container } = render(
        <MetricGrid className="custom-grid">
          <MetricCard title="Test" value={100} />
        </MetricGrid>
      );
      const grid = container.firstChild as HTMLElement;

      expect(grid).toHaveClass('custom-grid');
    });
  });

  describe('Column variants', () => {
    it('renders 2 column grid', () => {
      const { container } = render(
        <MetricGrid columns={2}>
          <MetricCard title="Test" value={100} />
        </MetricGrid>
      );
      const grid = container.firstChild as HTMLElement;

      expect(grid).toHaveClass('grid-cols-1', 'md:grid-cols-2');
    });

    it('renders 3 column grid', () => {
      const { container } = render(
        <MetricGrid columns={3}>
          <MetricCard title="Test" value={100} />
        </MetricGrid>
      );
      const grid = container.firstChild as HTMLElement;

      expect(grid).toHaveClass('grid-cols-1', 'md:grid-cols-2', 'lg:grid-cols-3');
    });

    it('renders 4 column grid by default', () => {
      const { container } = render(
        <MetricGrid>
          <MetricCard title="Test" value={100} />
        </MetricGrid>
      );
      const grid = container.firstChild as HTMLElement;

      expect(grid).toHaveClass('grid-cols-1', 'md:grid-cols-2', 'lg:grid-cols-4');
    });

    it('renders 5 column grid', () => {
      const { container } = render(
        <MetricGrid columns={5}>
          <MetricCard title="Test" value={100} />
        </MetricGrid>
      );
      const grid = container.firstChild as HTMLElement;

      expect(grid).toHaveClass('grid-cols-1', 'md:grid-cols-2', 'lg:grid-cols-3', 'xl:grid-cols-5');
    });
  });

  describe('Styling', () => {
    it('has grid layout class', () => {
      const { container } = render(
        <MetricGrid>
          <MetricCard title="Test" value={100} />
        </MetricGrid>
      );
      const grid = container.firstChild as HTMLElement;

      expect(grid).toHaveClass('grid');
    });

    it('has gap between items', () => {
      const { container } = render(
        <MetricGrid>
          <MetricCard title="Test" value={100} />
        </MetricGrid>
      );
      const grid = container.firstChild as HTMLElement;

      expect(grid).toHaveClass('gap-4');
    });
  });

  describe('Complete grid examples', () => {
    it('renders multiple metrics in grid', () => {
      render(
        <MetricGrid columns={3}>
          <MetricCard title="Orders" value={1250} variant="default" />
          <MetricCard title="Revenue" value="$45,230" variant="success" />
          <MetricCard title="Returns" value={18} variant="warning" />
        </MetricGrid>
      );

      expect(screen.getByText('Orders')).toBeInTheDocument();
      expect(screen.getByText('1250')).toBeInTheDocument();
      expect(screen.getByText('Revenue')).toBeInTheDocument();
      expect(screen.getByText('$45,230')).toBeInTheDocument();
      expect(screen.getByText('Returns')).toBeInTheDocument();
      expect(screen.getByText('18')).toBeInTheDocument();
    });

    it('renders grid with all metric variants', () => {
      const { container } = render(
        <MetricGrid columns={5}>
          <MetricCard title="Default" value={100} variant="default" />
          <MetricCard title="Success" value={200} variant="success" />
          <MetricCard title="Warning" value={300} variant="warning" />
          <MetricCard title="Error" value={400} variant="error" />
          <MetricCard title="Hero" value={500} variant="hero" />
        </MetricGrid>
      );
      const grid = container.firstChild as HTMLElement;

      expect(grid).toHaveClass('xl:grid-cols-5');
      expect(screen.getByText('Default')).toBeInTheDocument();
      expect(screen.getByText('Success')).toBeInTheDocument();
      expect(screen.getByText('Warning')).toBeInTheDocument();
      expect(screen.getByText('Error')).toBeInTheDocument();
      expect(screen.getByText('Hero')).toBeInTheDocument();
    });
  });
});
