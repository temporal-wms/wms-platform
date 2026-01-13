import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Card, CardHeader, CardContent, CardFooter } from './Card';

describe('Card', () => {
  describe('Rendering', () => {
    it('renders with children', () => {
      const { container } = render(<Card>Card content</Card>);

      expect(screen.getByText('Card content')).toBeInTheDocument();
      expect(container.firstChild).toBeInTheDocument();
    });

    it('renders with custom className', () => {
      const { container } = render(<Card className="custom-class">Content</Card>);
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('custom-class');
    });
  });

  describe('Padding variants', () => {
    it('renders with no padding', () => {
      const { container } = render(<Card padding="none">Content</Card>);
      const card = container.firstChild as HTMLElement;

      expect(card).not.toHaveClass('p-3', 'p-4', 'p-6');
    });

    it('renders with small padding', () => {
      const { container } = render(<Card padding="sm">Content</Card>);
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('p-3');
    });

    it('renders with medium padding by default', () => {
      const { container } = render(<Card>Content</Card>);
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('p-4');
    });

    it('renders with large padding', () => {
      const { container } = render(<Card padding="lg">Content</Card>);
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('p-6');
    });
  });

  describe('Shadow variants', () => {
    it('renders with no shadow', () => {
      const { container } = render(<Card shadow="none">Content</Card>);
      const card = container.firstChild as HTMLElement;

      expect(card).not.toHaveClass('shadow-card', 'shadow-soft', 'shadow-lg', 'shadow-elevated');
    });

    it('renders with small shadow by default', () => {
      const { container } = render(<Card>Content</Card>);
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('shadow-card');
    });

    it('renders with medium shadow', () => {
      const { container } = render(<Card shadow="md">Content</Card>);
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('shadow-soft');
    });

    it('renders with large shadow', () => {
      const { container } = render(<Card shadow="lg">Content</Card>);
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('shadow-lg');
    });

    it('renders with elevated shadow', () => {
      const { container } = render(<Card shadow="elevated">Content</Card>);
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('shadow-elevated');
    });
  });

  describe('Hover effect', () => {
    it('applies hover effect when hover prop is true', () => {
      const { container } = render(<Card hover>Content</Card>);
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('hover:shadow-card-hover', 'hover:-translate-y-0.5');
    });

    it('does not apply hover effect by default', () => {
      const { container } = render(<Card>Content</Card>);
      const card = container.firstChild as HTMLElement;

      expect(card).not.toHaveClass('hover:shadow-card-hover');
    });

    it('applies hover effect when onClick is provided', () => {
      const { container } = render(<Card onClick={vi.fn()}>Content</Card>);
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('hover:shadow-card-hover', 'hover:-translate-y-0.5');
    });
  });

  describe('Clickable behavior', () => {
    it('is not clickable by default', () => {
      const { container } = render(<Card>Content</Card>);
      const card = container.firstChild as HTMLElement;

      expect(card).not.toHaveClass('cursor-pointer');
      expect(card).not.toHaveAttribute('role', 'button');
      expect(card).not.toHaveAttribute('tabIndex');
    });

    it('becomes clickable when onClick is provided', () => {
      const { container } = render(<Card onClick={vi.fn()}>Content</Card>);
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('cursor-pointer');
      expect(card).toHaveAttribute('role', 'button');
      expect(card).toHaveAttribute('tabIndex', '0');
    });

    it('calls onClick when clicked', async () => {
      const user = userEvent.setup();
      const onClick = vi.fn();
      render(<Card onClick={onClick}>Content</Card>);

      await user.click(screen.getByRole('button'));

      expect(onClick).toHaveBeenCalledOnce();
    });

    it('calls onClick when Enter key is pressed', async () => {
      const user = userEvent.setup();
      const onClick = vi.fn();
      render(<Card onClick={onClick}>Content</Card>);

      const card = screen.getByRole('button');
      card.focus();
      await user.keyboard('{Enter}');

      expect(onClick).toHaveBeenCalledOnce();
    });

    it('does not call onClick for other keys', async () => {
      const user = userEvent.setup();
      const onClick = vi.fn();
      render(<Card onClick={onClick}>Content</Card>);

      const card = screen.getByRole('button');
      card.focus();
      await user.keyboard('{Space}');
      await user.keyboard('{Escape}');

      expect(onClick).not.toHaveBeenCalled();
    });
  });

  describe('Styling', () => {
    it('has correct base classes', () => {
      const { container } = render(<Card>Content</Card>);
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('bg-white', 'rounded-xl', 'border', 'border-gray-100');
    });

    it('has transition classes', () => {
      const { container } = render(<Card>Content</Card>);
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('transition-all', 'duration-200', 'ease-out');
    });

    it('combines padding, shadow, and custom classes', () => {
      const { container } = render(
        <Card padding="lg" shadow="elevated" className="extra-class">
          Content
        </Card>
      );
      const card = container.firstChild as HTMLElement;

      expect(card).toHaveClass('p-6'); // padding
      expect(card).toHaveClass('shadow-elevated'); // shadow
      expect(card).toHaveClass('extra-class'); // custom
    });
  });

  describe('Accessibility', () => {
    it('has button role when clickable', () => {
      render(<Card onClick={vi.fn()}>Content</Card>);
      const card = screen.getByRole('button');

      expect(card).toBeInTheDocument();
    });

    it('is keyboard accessible when clickable', () => {
      render(<Card onClick={vi.fn()}>Content</Card>);
      const card = screen.getByRole('button');

      expect(card).toHaveAttribute('tabIndex', '0');
    });

    it('does not have button role when not clickable', () => {
      render(<Card>Content</Card>);

      expect(screen.queryByRole('button')).not.toBeInTheDocument();
    });
  });
});

describe('CardHeader', () => {
  describe('Rendering', () => {
    it('renders title', () => {
      render(<CardHeader title="Card Title" />);

      expect(screen.getByText('Card Title')).toBeInTheDocument();
    });

    it('renders title and subtitle', () => {
      render(<CardHeader title="Title" subtitle="Subtitle text" />);

      expect(screen.getByText('Title')).toBeInTheDocument();
      expect(screen.getByText('Subtitle text')).toBeInTheDocument();
    });

    it('does not render subtitle when not provided', () => {
      const { container } = render(<CardHeader title="Title" />);
      const subtitles = container.querySelectorAll('p');

      expect(subtitles.length).toBe(0);
    });

    it('renders with action', () => {
      render(<CardHeader title="Title" action={<button>Action</button>} />);

      expect(screen.getByText('Title')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Action' })).toBeInTheDocument();
    });

    it('renders with custom className', () => {
      const { container } = render(<CardHeader title="Title" className="custom-class" />);
      const header = container.firstChild;

      expect(header).toHaveClass('custom-class');
    });
  });

  describe('Styling', () => {
    it('has correct title styles', () => {
      render(<CardHeader title="Title" />);
      const title = screen.getByText('Title');

      expect(title.tagName).toBe('H3');
      expect(title).toHaveClass('text-lg', 'font-semibold', 'text-gray-900');
    });

    it('has correct subtitle styles', () => {
      render(<CardHeader title="Title" subtitle="Subtitle" />);
      const subtitle = screen.getByText('Subtitle');

      expect(subtitle.tagName).toBe('P');
      expect(subtitle).toHaveClass('text-sm', 'text-gray-500', 'mt-0.5');
    });

    it('has flexbox layout', () => {
      const { container } = render(<CardHeader title="Title" />);
      const header = container.firstChild;

      expect(header).toHaveClass('flex', 'items-start', 'justify-between');
    });
  });
});

describe('CardContent', () => {
  describe('Rendering', () => {
    it('renders children', () => {
      render(<CardContent>Content here</CardContent>);

      expect(screen.getByText('Content here')).toBeInTheDocument();
    });

    it('renders with custom className', () => {
      const { container } = render(<CardContent className="custom-class">Content</CardContent>);
      const content = container.firstChild;

      expect(content).toHaveClass('custom-class');
    });
  });

  describe('Styling', () => {
    it('has margin top', () => {
      const { container } = render(<CardContent>Content</CardContent>);
      const content = container.firstChild;

      expect(content).toHaveClass('mt-4');
    });
  });
});

describe('CardFooter', () => {
  describe('Rendering', () => {
    it('renders children', () => {
      render(<CardFooter>Footer content</CardFooter>);

      expect(screen.getByText('Footer content')).toBeInTheDocument();
    });

    it('renders with custom className', () => {
      const { container } = render(<CardFooter className="custom-class">Footer</CardFooter>);
      const footer = container.firstChild;

      expect(footer).toHaveClass('custom-class');
    });
  });

  describe('Styling', () => {
    it('has correct spacing and border', () => {
      const { container } = render(<CardFooter>Footer</CardFooter>);
      const footer = container.firstChild;

      expect(footer).toHaveClass('mt-4', 'pt-4', 'border-t', 'border-gray-100');
    });
  });
});

describe('Card composition', () => {
  it('renders complete card with all subcomponents', async () => {
    const user = userEvent.setup();
    const onEdit = vi.fn();
    const onSave = vi.fn();
    const onCancel = vi.fn();

    render(
      <Card padding="lg" shadow="elevated">
        <CardHeader
          title="User Profile"
          subtitle="Edit your profile information"
          action={<button onClick={onEdit}>Edit</button>}
        />
        <CardContent>
          <p>Profile content goes here</p>
        </CardContent>
        <CardFooter>
          <button onClick={onCancel}>Cancel</button>
          <button onClick={onSave}>Save</button>
        </CardFooter>
      </Card>
    );

    expect(screen.getByText('User Profile')).toBeInTheDocument();
    expect(screen.getByText('Edit your profile information')).toBeInTheDocument();
    expect(screen.getByText('Profile content goes here')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Edit' }));
    expect(onEdit).toHaveBeenCalledOnce();

    await user.click(screen.getByRole('button', { name: 'Save' }));
    expect(onSave).toHaveBeenCalledOnce();

    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it('renders minimal card with only content', () => {
    render(
      <Card>
        <p>Simple content</p>
      </Card>
    );

    expect(screen.getByText('Simple content')).toBeInTheDocument();
  });

  it('renders clickable card with header and content', async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();

    render(
      <Card onClick={onClick} hover>
        <CardHeader title="Clickable Card" subtitle="Click to view details" />
        <CardContent>Card content</CardContent>
      </Card>
    );

    const card = screen.getByRole('button');
    await user.click(card);

    expect(onClick).toHaveBeenCalledOnce();
  });
});
