import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Modal, ConfirmDialog } from './Modal';

describe('Modal', () => {
  beforeEach(() => {
    // Reset body overflow before each test
    document.body.style.overflow = '';
  });

  afterEach(() => {
    // Clean up after each test
    document.body.style.overflow = '';
  });

  describe('Visibility', () => {
    it('renders when isOpen is true', () => {
      render(
        <Modal isOpen onClose={vi.fn()}>
          <div>Modal Content</div>
        </Modal>
      );

      expect(screen.getByText('Modal Content')).toBeInTheDocument();
    });

    it('does not render when isOpen is false', () => {
      render(
        <Modal isOpen={false} onClose={vi.fn()}>
          <div>Modal Content</div>
        </Modal>
      );

      expect(screen.queryByText('Modal Content')).not.toBeInTheDocument();
    });

    it('returns null when closed', () => {
      const { container } = render(
        <Modal isOpen={false} onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      expect(container.firstChild).toBeNull();
    });
  });

  describe('Title', () => {
    it('renders title when provided', () => {
      render(
        <Modal isOpen title="Modal Title" onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      expect(screen.getByText('Modal Title')).toBeInTheDocument();
    });

    it('does not render title when not provided', () => {
      render(
        <Modal isOpen onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      expect(screen.queryByRole('heading')).not.toBeInTheDocument();
    });

    it('renders title with correct heading level', () => {
      render(
        <Modal isOpen title="My Modal" onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      const title = screen.getByText('My Modal');
      expect(title.tagName).toBe('H2');
    });

    it('applies id to title for aria-labelledby', () => {
      render(
        <Modal isOpen title="My Modal" onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      const title = screen.getByText('My Modal');
      expect(title).toHaveAttribute('id', 'modal-title');
    });
  });

  describe('Close button', () => {
    it('renders close button by default', () => {
      render(
        <Modal isOpen onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      expect(screen.getByLabelText('Close modal')).toBeInTheDocument();
    });

    it('does not render close button when showCloseButton is false', () => {
      render(
        <Modal isOpen showCloseButton={false} onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      expect(screen.queryByLabelText('Close modal')).not.toBeInTheDocument();
    });

    it('calls onClose when close button clicked', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();

      render(
        <Modal isOpen onClose={onClose}>
          <div>Content</div>
        </Modal>
      );

      await user.click(screen.getByLabelText('Close modal'));

      expect(onClose).toHaveBeenCalledOnce();
    });
  });

  describe('Escape key handling', () => {
    it('closes on Escape key by default', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();

      render(
        <Modal isOpen onClose={onClose}>
          <div>Content</div>
        </Modal>
      );

      await user.keyboard('{Escape}');

      expect(onClose).toHaveBeenCalledOnce();
    });

    it('does not close on Escape when closeOnEscape is false', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();

      render(
        <Modal isOpen closeOnEscape={false} onClose={onClose}>
          <div>Content</div>
        </Modal>
      );

      await user.keyboard('{Escape}');

      expect(onClose).not.toHaveBeenCalled();
    });

    it('does not close on other keys', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();

      render(
        <Modal isOpen onClose={onClose}>
          <div>Content</div>
        </Modal>
      );

      await user.keyboard('{Enter}');
      await user.keyboard('{Space}');
      await user.keyboard('{Tab}');

      expect(onClose).not.toHaveBeenCalled();
    });
  });

  describe('Overlay click handling', () => {
    it('closes on overlay click by default', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();

      render(
        <Modal isOpen onClose={onClose}>
          <div>Content</div>
        </Modal>
      );

      const overlay = screen.getByRole('dialog').parentElement!;
      await user.click(overlay);

      expect(onClose).toHaveBeenCalledOnce();
    });

    it('does not close on overlay click when closeOnOverlayClick is false', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();

      render(
        <Modal isOpen closeOnOverlayClick={false} onClose={onClose}>
          <div>Content</div>
        </Modal>
      );

      const overlay = screen.getByRole('dialog').parentElement!;
      await user.click(overlay);

      expect(onClose).not.toHaveBeenCalled();
    });

    it('does not close when clicking modal content', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();

      render(
        <Modal isOpen onClose={onClose}>
          <div>Modal Content</div>
        </Modal>
      );

      const content = screen.getByText('Modal Content');
      await user.click(content);

      expect(onClose).not.toHaveBeenCalled();
    });

    it('does not close when clicking modal dialog', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();

      render(
        <Modal isOpen onClose={onClose}>
          <div>Content</div>
        </Modal>
      );

      const dialog = screen.getByRole('dialog');
      await user.click(dialog);

      expect(onClose).not.toHaveBeenCalled();
    });
  });

  describe('Body scroll lock', () => {
    it('locks body scroll when modal opens', () => {
      render(
        <Modal isOpen onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      expect(document.body.style.overflow).toBe('hidden');
    });

    it('restores body scroll when modal closes', () => {
      const { rerender } = render(
        <Modal isOpen onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      expect(document.body.style.overflow).toBe('hidden');

      rerender(
        <Modal isOpen={false} onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      expect(document.body.style.overflow).toBe('');
    });

    it('does not lock body scroll when modal is not open', () => {
      render(
        <Modal isOpen={false} onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      expect(document.body.style.overflow).toBe('');
    });
  });

  describe('Footer', () => {
    it('renders footer when provided', () => {
      render(
        <Modal isOpen onClose={vi.fn()} footer={<button>Save</button>}>
          <div>Content</div>
        </Modal>
      );

      expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument();
    });

    it('does not render footer when not provided', () => {
      const { container } = render(
        <Modal isOpen onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      const footer = container.querySelector('.bg-gray-50\\/80');
      expect(footer).not.toBeInTheDocument();
    });

    it('renders multiple footer elements', () => {
      render(
        <Modal
          isOpen
          onClose={vi.fn()}
          footer={
            <>
              <button>Cancel</button>
              <button>Save</button>
            </>
          }
        >
          <div>Content</div>
        </Modal>
      );

      expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument();
    });
  });

  describe('Size variants', () => {
    it('renders small size', () => {
      render(
        <Modal isOpen size="sm" onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveClass('max-w-sm');
    });

    it('renders medium size by default', () => {
      render(
        <Modal isOpen onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveClass('max-w-md');
    });

    it('renders large size', () => {
      render(
        <Modal isOpen size="lg" onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveClass('max-w-lg');
    });

    it('renders extra large size', () => {
      render(
        <Modal isOpen size="xl" onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveClass('max-w-xl');
    });

    it('renders full size', () => {
      render(
        <Modal isOpen size="full" onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveClass('max-w-4xl');
    });
  });

  describe('Accessibility', () => {
    it('has dialog role', () => {
      render(
        <Modal isOpen onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      expect(screen.getByRole('dialog')).toBeInTheDocument();
    });

    it('has aria-modal attribute', () => {
      render(
        <Modal isOpen onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveAttribute('aria-modal', 'true');
    });

    it('has aria-labelledby when title is provided', () => {
      render(
        <Modal isOpen title="My Modal" onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveAttribute('aria-labelledby', 'modal-title');
    });

    it('does not have aria-labelledby when no title', () => {
      render(
        <Modal isOpen onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      const dialog = screen.getByRole('dialog');
      expect(dialog).not.toHaveAttribute('aria-labelledby');
    });

    it('close button has accessible label', () => {
      render(
        <Modal isOpen onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      expect(screen.getByLabelText('Close modal')).toBeInTheDocument();
    });
  });

  describe('Styling', () => {
    it('has overlay with correct classes', () => {
      const { container } = render(
        <Modal isOpen onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      const overlay = container.querySelector('.fixed.inset-0');
      expect(overlay).toHaveClass('z-50', 'flex', 'items-center', 'justify-center', 'bg-black/40', 'backdrop-blur-sm');
    });

    it('has fade-in animation on overlay', () => {
      const { container } = render(
        <Modal isOpen onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      const overlay = container.querySelector('.fixed.inset-0');
      expect(overlay).toHaveClass('animate-fade-in');
    });

    it('has scale-in animation on dialog', () => {
      render(
        <Modal isOpen onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveClass('animate-scale-in');
    });

    it('has correct dialog styling', () => {
      render(
        <Modal isOpen onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveClass('bg-white', 'rounded-2xl', 'shadow-elevated', 'w-full');
    });

    it('has scrollable content area', () => {
      const { container } = render(
        <Modal isOpen onClose={vi.fn()}>
          <div>Content</div>
        </Modal>
      );

      const content = container.querySelector('.overflow-y-auto');
      expect(content).toBeInTheDocument();
      expect(content).toHaveClass('max-h-[calc(100vh-200px)]');
    });
  });

  describe('Complete examples', () => {
    it('renders complete modal with all props', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();
      const onSave = vi.fn();

      render(
        <Modal
          isOpen
          title="Edit Profile"
          size="lg"
          onClose={onClose}
          footer={
            <>
              <button onClick={onClose}>Cancel</button>
              <button onClick={onSave}>Save Changes</button>
            </>
          }
        >
          <p>Edit your profile information here</p>
        </Modal>
      );

      expect(screen.getByText('Edit Profile')).toBeInTheDocument();
      expect(screen.getByText('Edit your profile information here')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Save Changes' })).toBeInTheDocument();

      await user.click(screen.getByLabelText('Close modal'));
      expect(onClose).toHaveBeenCalledOnce();
    });

    it('renders minimal modal', () => {
      render(
        <Modal isOpen onClose={vi.fn()}>
          <div>Simple content</div>
        </Modal>
      );

      expect(screen.getByText('Simple content')).toBeInTheDocument();
      expect(screen.getByLabelText('Close modal')).toBeInTheDocument();
    });
  });
});

describe('ConfirmDialog', () => {
  describe('Rendering', () => {
    it('renders with required props', () => {
      render(
        <ConfirmDialog
          isOpen
          onClose={vi.fn()}
          onConfirm={vi.fn()}
          title="Confirm Action"
          message="Are you sure?"
        />
      );

      expect(screen.getByText('Confirm Action')).toBeInTheDocument();
      expect(screen.getByText('Are you sure?')).toBeInTheDocument();
    });

    it('renders default button labels', () => {
      render(
        <ConfirmDialog
          isOpen
          onClose={vi.fn()}
          onConfirm={vi.fn()}
          title="Confirm"
          message="Are you sure?"
        />
      );

      expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Confirm' })).toBeInTheDocument();
    });

    it('renders custom button labels', () => {
      render(
        <ConfirmDialog
          isOpen
          onClose={vi.fn()}
          onConfirm={vi.fn()}
          title="Delete Item"
          message="This action cannot be undone"
          confirmLabel="Delete"
          cancelLabel="Go Back"
        />
      );

      expect(screen.getByRole('button', { name: 'Go Back' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument();
    });

    it('does not render when isOpen is false', () => {
      render(
        <ConfirmDialog
          isOpen={false}
          onClose={vi.fn()}
          onConfirm={vi.fn()}
          title="Confirm"
          message="Are you sure?"
        />
      );

      expect(screen.queryByText('Confirm')).not.toBeInTheDocument();
    });
  });

  describe('Button interactions', () => {
    it('calls onClose when cancel button clicked', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();

      render(
        <ConfirmDialog
          isOpen
          onClose={onClose}
          onConfirm={vi.fn()}
          title="Confirm"
          message="Are you sure?"
        />
      );

      await user.click(screen.getByRole('button', { name: 'Cancel' }));

      expect(onClose).toHaveBeenCalledOnce();
    });

    it('calls onConfirm when confirm button clicked', async () => {
      const user = userEvent.setup();
      const onConfirm = vi.fn();

      render(
        <ConfirmDialog
          isOpen
          onClose={vi.fn()}
          onConfirm={onConfirm}
          title="Confirm"
          message="Are you sure?"
        />
      );

      await user.click(screen.getByRole('button', { name: 'Confirm' }));

      expect(onConfirm).toHaveBeenCalledOnce();
    });

    it('calls onClose when close button in header clicked', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();

      render(
        <ConfirmDialog
          isOpen
          onClose={onClose}
          onConfirm={vi.fn()}
          title="Confirm"
          message="Are you sure?"
        />
      );

      await user.click(screen.getByLabelText('Close modal'));

      expect(onClose).toHaveBeenCalledOnce();
    });
  });

  describe('Variant styling', () => {
    it('renders default variant with primary confirm button', () => {
      render(
        <ConfirmDialog
          isOpen
          onClose={vi.fn()}
          onConfirm={vi.fn()}
          title="Confirm"
          message="Are you sure?"
          variant="default"
        />
      );

      const confirmButton = screen.getByRole('button', { name: 'Confirm' });
      expect(confirmButton).toHaveClass('bg-primary-600', 'text-white');
    });

    it('renders danger variant with danger confirm button', () => {
      render(
        <ConfirmDialog
          isOpen
          onClose={vi.fn()}
          onConfirm={vi.fn()}
          title="Delete Item"
          message="This cannot be undone"
          variant="danger"
        />
      );

      const confirmButton = screen.getByRole('button', { name: 'Confirm' });
      expect(confirmButton).toHaveClass('bg-error-600', 'text-white');
    });

    it('renders warning variant with primary confirm button', () => {
      render(
        <ConfirmDialog
          isOpen
          onClose={vi.fn()}
          onConfirm={vi.fn()}
          title="Warning"
          message="Proceed with caution"
          variant="warning"
        />
      );

      const confirmButton = screen.getByRole('button', { name: 'Confirm' });
      expect(confirmButton).toHaveClass('bg-primary-600', 'text-white');
    });

    it('renders cancel button with secondary variant', () => {
      render(
        <ConfirmDialog
          isOpen
          onClose={vi.fn()}
          onConfirm={vi.fn()}
          title="Confirm"
          message="Are you sure?"
        />
      );

      const cancelButton = screen.getByRole('button', { name: 'Cancel' });
      expect(cancelButton).toHaveClass('bg-white', 'text-gray-700');
    });
  });

  describe('Loading state', () => {
    it('shows loading on confirm button when loading', () => {
      render(
        <ConfirmDialog
          isOpen
          onClose={vi.fn()}
          onConfirm={vi.fn()}
          title="Confirm"
          message="Are you sure?"
          loading
        />
      );

      const confirmButton = screen.getByRole('button', { name: 'Confirm' });
      expect(confirmButton).toBeDisabled();
    });

    it('disables cancel button when loading', () => {
      render(
        <ConfirmDialog
          isOpen
          onClose={vi.fn()}
          onConfirm={vi.fn()}
          title="Confirm"
          message="Are you sure?"
          loading
        />
      );

      const cancelButton = screen.getByRole('button', { name: 'Cancel' });
      expect(cancelButton).toBeDisabled();
    });

    it('does not disable buttons when not loading', () => {
      render(
        <ConfirmDialog
          isOpen
          onClose={vi.fn()}
          onConfirm={vi.fn()}
          title="Confirm"
          message="Are you sure?"
          loading={false}
        />
      );

      const cancelButton = screen.getByRole('button', { name: 'Cancel' });
      const confirmButton = screen.getByRole('button', { name: 'Confirm' });

      expect(cancelButton).not.toBeDisabled();
      expect(confirmButton).not.toBeDisabled();
    });
  });

  describe('Modal integration', () => {
    it('uses small modal size', () => {
      render(
        <ConfirmDialog
          isOpen
          onClose={vi.fn()}
          onConfirm={vi.fn()}
          title="Confirm"
          message="Are you sure?"
        />
      );

      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveClass('max-w-sm');
    });

    it('inherits modal close behavior', async () => {
      const user = userEvent.setup();
      const onClose = vi.fn();

      render(
        <ConfirmDialog
          isOpen
          onClose={onClose}
          onConfirm={vi.fn()}
          title="Confirm"
          message="Are you sure?"
        />
      );

      await user.keyboard('{Escape}');

      expect(onClose).toHaveBeenCalledOnce();
    });
  });

  describe('Complete examples', () => {
    it('renders delete confirmation dialog', async () => {
      const user = userEvent.setup();
      const onConfirm = vi.fn();
      const onClose = vi.fn();

      render(
        <ConfirmDialog
          isOpen
          onClose={onClose}
          onConfirm={onConfirm}
          title="Delete Order"
          message="Are you sure you want to delete this order? This action cannot be undone."
          confirmLabel="Delete"
          cancelLabel="Cancel"
          variant="danger"
        />
      );

      expect(screen.getByText('Delete Order')).toBeInTheDocument();
      expect(screen.getByText('Are you sure you want to delete this order? This action cannot be undone.')).toBeInTheDocument();

      await user.click(screen.getByRole('button', { name: 'Delete' }));
      expect(onConfirm).toHaveBeenCalledOnce();
    });

    it('renders loading confirmation dialog', () => {
      render(
        <ConfirmDialog
          isOpen
          onClose={vi.fn()}
          onConfirm={vi.fn()}
          title="Saving Changes"
          message="Please wait..."
          loading
        />
      );

      const confirmButton = screen.getByRole('button', { name: 'Confirm' });
      const cancelButton = screen.getByRole('button', { name: 'Cancel' });

      expect(confirmButton).toBeDisabled();
      expect(cancelButton).toBeDisabled();
    });
  });
});
