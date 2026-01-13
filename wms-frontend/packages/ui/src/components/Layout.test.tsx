import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { Sidebar, Header, MainLayout, NavItem } from './Layout';

const mockNavItems: NavItem[] = [
  {
    label: 'Dashboard',
    path: '/',
    icon: <div data-testid="dashboard-icon">D</div>,
  },
  {
    label: 'Orders',
    path: '/orders',
    icon: <div data-testid="orders-icon">O</div>,
    badge: 5,
  },
  {
    label: 'Settings',
    path: '/settings',
    icon: <div data-testid="settings-icon">S</div>,
    children: [
      { label: 'Profile', path: '/settings/profile' },
      { label: 'Security', path: '/settings/security' },
    ],
  },
];

describe('Sidebar', () => {
  describe('Rendering', () => {
    it('renders sidebar', () => {
      render(
        <MemoryRouter>
          <Sidebar />
        </MemoryRouter>
      );

      expect(screen.getByText('WMS Platform')).toBeInTheDocument();
    });

    it('renders with custom nav items', () => {
      render(
        <MemoryRouter>
          <Sidebar navItems={mockNavItems} />
        </MemoryRouter>
      );

      expect(screen.getByText('Dashboard')).toBeInTheDocument();
      expect(screen.getByText('Orders')).toBeInTheDocument();
      expect(screen.getByText('Settings')).toBeInTheDocument();
    });

    it('renders default nav items when not provided', () => {
      render(
        <MemoryRouter>
          <Sidebar />
        </MemoryRouter>
      );

      // Check for some default nav items
      expect(screen.getByText('Dashboard')).toBeInTheDocument();
      expect(screen.getByText('Orders')).toBeInTheDocument();
      expect(screen.getByText('Picking')).toBeInTheDocument();
    });

    it('renders nav item icons', () => {
      render(
        <MemoryRouter>
          <Sidebar navItems={mockNavItems} />
        </MemoryRouter>
      );

      expect(screen.getByTestId('dashboard-icon')).toBeInTheDocument();
      expect(screen.getByTestId('orders-icon')).toBeInTheDocument();
    });

    it('renders nav item badges', () => {
      render(
        <MemoryRouter>
          <Sidebar navItems={mockNavItems} />
        </MemoryRouter>
      );

      expect(screen.getByText('5')).toBeInTheDocument();
    });

    it('renders logo when not collapsed', () => {
      render(
        <MemoryRouter>
          <Sidebar collapsed={false} />
        </MemoryRouter>
      );

      expect(screen.getByText('WMS Platform')).toBeInTheDocument();
    });

    it('does not render logo text when collapsed', () => {
      render(
        <MemoryRouter>
          <Sidebar collapsed />
        </MemoryRouter>
      );

      expect(screen.queryByText('WMS Platform')).not.toBeInTheDocument();
    });
  });

  describe('Collapsed state', () => {
    it('applies collapsed width class', () => {
      const { container } = render(
        <MemoryRouter>
          <Sidebar collapsed />
        </MemoryRouter>
      );

      const sidebar = container.querySelector('aside');
      expect(sidebar).toHaveClass('w-16');
    });

    it('applies expanded width class', () => {
      const { container } = render(
        <MemoryRouter>
          <Sidebar collapsed={false} />
        </MemoryRouter>
      );

      const sidebar = container.querySelector('aside');
      expect(sidebar).toHaveClass('w-64');
    });

    it('hides nav item labels when collapsed', () => {
      render(
        <MemoryRouter>
          <Sidebar navItems={mockNavItems} collapsed />
        </MemoryRouter>
      );

      // Labels should not be visible when collapsed
      expect(screen.queryByText('Dashboard')).not.toBeInTheDocument();
      expect(screen.queryByText('Orders')).not.toBeInTheDocument();
    });

    it('shows collapse button when collapsed', () => {
      const { container } = render(
        <MemoryRouter>
          <Sidebar collapsed />
        </MemoryRouter>
      );

      // Menu icon should be visible
      const menuIcon = container.querySelector('svg');
      expect(menuIcon).toBeInTheDocument();
    });
  });

  describe('Collapse functionality', () => {
    it('calls onCollapse when collapse button clicked (expanded)', async () => {
      const user = userEvent.setup();
      const onCollapse = vi.fn();

      const { container } = render(
        <MemoryRouter>
          <Sidebar collapsed={false} onCollapse={onCollapse} />
        </MemoryRouter>
      );

      // Find the X button
      const collapseButton = container.querySelector('button');
      await user.click(collapseButton!);

      expect(onCollapse).toHaveBeenCalledWith(true);
    });

    it('calls onCollapse when menu button clicked (collapsed)', async () => {
      const user = userEvent.setup();
      const onCollapse = vi.fn();

      const { container } = render(
        <MemoryRouter>
          <Sidebar collapsed onCollapse={onCollapse} />
        </MemoryRouter>
      );

      // Find the menu button
      const buttons = container.querySelectorAll('button');
      const menuButton = buttons[0];
      await user.click(menuButton);

      expect(onCollapse).toHaveBeenCalledWith(false);
    });
  });

  describe('Active route detection', () => {
    it('highlights active route', () => {
      const { container } = render(
        <MemoryRouter initialEntries={['/orders']}>
          <Sidebar navItems={mockNavItems} />
        </MemoryRouter>
      );

      const ordersLink = screen.getByText('Orders').closest('a');
      expect(ordersLink).toHaveClass('bg-primary-50', 'text-primary-700');
    });

    it('highlights root path only when exactly at root', () => {
      const { container } = render(
        <MemoryRouter initialEntries={['/']}>
          <Sidebar navItems={mockNavItems} />
        </MemoryRouter>
      );

      const dashboardLink = screen.getByText('Dashboard').closest('a');
      expect(dashboardLink).toHaveClass('bg-primary-50', 'text-primary-700');
    });

    it('does not highlight inactive routes', () => {
      render(
        <MemoryRouter initialEntries={['/orders']}>
          <Sidebar navItems={mockNavItems} />
        </MemoryRouter>
      );

      const settingsLink = screen.getByText('Settings').closest('a');
      expect(settingsLink).toHaveClass('text-gray-600');
      expect(settingsLink).not.toHaveClass('bg-primary-50');
    });
  });

  describe('Children/Submenu', () => {
    it('does not show children initially', () => {
      render(
        <MemoryRouter>
          <Sidebar navItems={mockNavItems} />
        </MemoryRouter>
      );

      expect(screen.queryByText('Profile')).not.toBeInTheDocument();
      expect(screen.queryByText('Security')).not.toBeInTheDocument();
    });

    it('expands children when parent clicked', async () => {
      const user = userEvent.setup();

      render(
        <MemoryRouter>
          <Sidebar navItems={mockNavItems} />
        </MemoryRouter>
      );

      await user.click(screen.getByText('Settings'));

      expect(screen.getByText('Profile')).toBeInTheDocument();
      expect(screen.getByText('Security')).toBeInTheDocument();
    });

    it('collapses children when parent clicked again', async () => {
      const user = userEvent.setup();

      render(
        <MemoryRouter>
          <Sidebar navItems={mockNavItems} />
        </MemoryRouter>
      );

      // Expand
      await user.click(screen.getByText('Settings'));
      expect(screen.getByText('Profile')).toBeInTheDocument();

      // Collapse
      await user.click(screen.getByText('Settings'));
      expect(screen.queryByText('Profile')).not.toBeInTheDocument();
    });

    it('renders chevron icon for items with children', () => {
      const { container } = render(
        <MemoryRouter>
          <Sidebar navItems={mockNavItems} />
        </MemoryRouter>
      );

      const settingsLink = screen.getByText('Settings').closest('a');
      const chevron = settingsLink?.querySelector('svg');

      expect(chevron).toBeInTheDocument();
    });

    it('rotates chevron when expanded', async () => {
      const user = userEvent.setup();

      const { container } = render(
        <MemoryRouter>
          <Sidebar navItems={mockNavItems} />
        </MemoryRouter>
      );

      const settingsLink = screen.getByText('Settings').closest('a');
      const chevron = settingsLink?.querySelector('svg');

      expect(chevron).not.toHaveClass('rotate-180');

      await user.click(screen.getByText('Settings'));

      expect(chevron).toHaveClass('rotate-180');
    });

    it('does not show children when collapsed', async () => {
      const user = userEvent.setup();

      render(
        <MemoryRouter>
          <Sidebar navItems={mockNavItems} collapsed />
        </MemoryRouter>
      );

      // Try to expand (should not work when sidebar is collapsed)
      const links = screen.queryAllByRole('link');
      if (links.length > 0) {
        await user.click(links[0]);
      }

      expect(screen.queryByText('Profile')).not.toBeInTheDocument();
    });
  });

  describe('Styling', () => {
    it('has correct base classes', () => {
      const { container } = render(
        <MemoryRouter>
          <Sidebar />
        </MemoryRouter>
      );

      const sidebar = container.querySelector('aside');
      expect(sidebar).toHaveClass('fixed', 'left-0', 'top-0', 'h-full', 'bg-white', 'border-r');
    });

    it('has transition classes', () => {
      const { container } = render(
        <MemoryRouter>
          <Sidebar />
        </MemoryRouter>
      );

      const sidebar = container.querySelector('aside');
      expect(sidebar).toHaveClass('transition-all', 'duration-300', 'ease-in-out');
    });
  });
});

describe('Header', () => {
  describe('Rendering', () => {
    it('renders header', () => {
      render(<Header />);

      expect(screen.getByPlaceholderText('Search orders, waves, inventory...')).toBeInTheDocument();
    });

    it('renders search input', () => {
      render(<Header />);

      const searchInput = screen.getByPlaceholderText('Search orders, waves, inventory...');
      expect(searchInput).toBeInTheDocument();
      expect(searchInput).toHaveAttribute('type', 'search');
    });

    it('renders notification bell', () => {
      const { container } = render(<Header />);

      // Bell icon should be present
      const bell = container.querySelector('svg');
      expect(bell).toBeInTheDocument();
    });

    it('renders user avatar', () => {
      render(<Header />);

      // Avatar with letter 'A'
      expect(screen.getByText('A')).toBeInTheDocument();
    });
  });

  describe('Notification badge', () => {
    it('does not show badge when notifications is 0', () => {
      render(<Header notifications={0} />);

      // Badge should not be visible
      expect(screen.queryByText('1')).not.toBeInTheDocument();
    });

    it('shows notification count when notifications > 0', () => {
      render(<Header notifications={3} />);

      expect(screen.getByText('3')).toBeInTheDocument();
    });

    it('shows "9+" when notifications > 9', () => {
      render(<Header notifications={15} />);

      expect(screen.getByText('9+')).toBeInTheDocument();
    });

    it('shows exact count when notifications <= 9', () => {
      render(<Header notifications={7} />);

      expect(screen.getByText('7')).toBeInTheDocument();
    });

    it('applies pulse animation to badge', () => {
      const { container } = render(<Header notifications={5} />);

      const badge = screen.getByText('5');
      expect(badge).toHaveClass('animate-pulse-slow');
    });
  });

  describe('Sidebar collapsed state', () => {
    it('adjusts left position when sidebar is expanded', () => {
      const { container } = render(<Header sidebarCollapsed={false} />);

      const header = container.querySelector('header');
      expect(header).toHaveClass('left-64');
    });

    it('adjusts left position when sidebar is collapsed', () => {
      const { container } = render(<Header sidebarCollapsed />);

      const header = container.querySelector('header');
      expect(header).toHaveClass('left-16');
    });
  });

  describe('Styling', () => {
    it('has correct base classes', () => {
      const { container } = render(<Header />);

      const header = container.querySelector('header');
      expect(header).toHaveClass('fixed', 'top-0', 'right-0', 'h-16', 'bg-white/95', 'backdrop-blur-sm');
    });

    it('has transition classes', () => {
      const { container } = render(<Header />);

      const header = container.querySelector('header');
      expect(header).toHaveClass('transition-all', 'duration-300', 'ease-in-out');
    });
  });
});

describe('MainLayout', () => {
  describe('Rendering', () => {
    it('renders children', () => {
      render(
        <MemoryRouter>
          <MainLayout>
            <div>Page Content</div>
          </MainLayout>
        </MemoryRouter>
      );

      expect(screen.getByText('Page Content')).toBeInTheDocument();
    });

    it('renders Sidebar', () => {
      render(
        <MemoryRouter>
          <MainLayout>
            <div>Content</div>
          </MainLayout>
        </MemoryRouter>
      );

      expect(screen.getByText('WMS Platform')).toBeInTheDocument();
    });

    it('renders Header', () => {
      render(
        <MemoryRouter>
          <MainLayout>
            <div>Content</div>
          </MainLayout>
        </MemoryRouter>
      );

      expect(screen.getByPlaceholderText('Search orders, waves, inventory...')).toBeInTheDocument();
    });

    it('renders with custom nav items', () => {
      render(
        <MemoryRouter>
          <MainLayout navItems={mockNavItems}>
            <div>Content</div>
          </MainLayout>
        </MemoryRouter>
      );

      expect(screen.getByText('Dashboard')).toBeInTheDocument();
      expect(screen.getByText('Orders')).toBeInTheDocument();
    });
  });

  describe('Collapse state management', () => {
    it('starts with sidebar expanded', () => {
      const { container } = render(
        <MemoryRouter>
          <MainLayout>
            <div>Content</div>
          </MainLayout>
        </MemoryRouter>
      );

      const sidebar = container.querySelector('aside');
      expect(sidebar).toHaveClass('w-64');
    });

    it('toggles sidebar collapse state', async () => {
      const user = userEvent.setup();

      const { container } = render(
        <MemoryRouter>
          <MainLayout>
            <div>Content</div>
          </MainLayout>
        </MemoryRouter>
      );

      const sidebar = container.querySelector('aside');
      expect(sidebar).toHaveClass('w-64');

      // Click collapse button
      const collapseButton = container.querySelector('button');
      if (collapseButton) {
        await user.click(collapseButton);
      }

      expect(sidebar).toHaveClass('w-16');
    });

    it('adjusts main content margin when sidebar collapses', async () => {
      const user = userEvent.setup();

      const { container } = render(
        <MemoryRouter>
          <MainLayout>
            <div>Content</div>
          </MainLayout>
        </MemoryRouter>
      );

      const main = container.querySelector('main');
      expect(main).toHaveClass('ml-64');

      // Collapse sidebar
      const collapseButton = container.querySelector('button');
      if (collapseButton) {
        await user.click(collapseButton);
      }

      expect(main).toHaveClass('ml-16');
    });

    it('adjusts header position when sidebar collapses', async () => {
      const user = userEvent.setup();

      const { container } = render(
        <MemoryRouter>
          <MainLayout>
            <div>Content</div>
          </MainLayout>
        </MemoryRouter>
      );

      const header = container.querySelector('header');
      expect(header).toHaveClass('left-64');

      // Collapse sidebar
      const collapseButton = container.querySelector('button');
      if (collapseButton) {
        await user.click(collapseButton);
      }

      expect(header).toHaveClass('left-16');
    });
  });

  describe('Layout structure', () => {
    it('has correct background color', () => {
      const { container } = render(
        <MemoryRouter>
          <MainLayout>
            <div>Content</div>
          </MainLayout>
        </MemoryRouter>
      );

      const wrapper = container.firstChild as HTMLElement;
      expect(wrapper).toHaveClass('min-h-screen', 'bg-gray-50');
    });

    it('applies padding to main content', () => {
      const { container } = render(
        <MemoryRouter>
          <MainLayout>
            <div>Content</div>
          </MainLayout>
        </MemoryRouter>
      );

      const main = container.querySelector('main');
      expect(main).toHaveClass('pt-16');
    });

    it('has transition classes on main', () => {
      const { container } = render(
        <MemoryRouter>
          <MainLayout>
            <div>Content</div>
          </MainLayout>
        </MemoryRouter>
      );

      const main = container.querySelector('main');
      expect(main).toHaveClass('transition-all', 'duration-300', 'ease-in-out');
    });
  });

  describe('Complete examples', () => {
    it('renders complete layout with navigation and content', async () => {
      const user = userEvent.setup();

      render(
        <MemoryRouter initialEntries={['/orders']}>
          <MainLayout navItems={mockNavItems}>
            <div>
              <h1>Orders Page</h1>
              <p>Order list content</p>
            </div>
          </MainLayout>
        </MemoryRouter>
      );

      // Check sidebar navigation
      expect(screen.getByText('Dashboard')).toBeInTheDocument();
      expect(screen.getByText('Orders')).toBeInTheDocument();

      // Check active state
      const ordersLink = screen.getByText('Orders').closest('a');
      expect(ordersLink).toHaveClass('bg-primary-50');

      // Check header
      expect(screen.getByPlaceholderText('Search orders, waves, inventory...')).toBeInTheDocument();

      // Check content
      expect(screen.getByText('Orders Page')).toBeInTheDocument();
      expect(screen.getByText('Order list content')).toBeInTheDocument();

      // Test collapse
      const { container } = render(
        <MemoryRouter>
          <MainLayout>
            <div>Content</div>
          </MainLayout>
        </MemoryRouter>
      );

      const collapseButton = container.querySelector('button');
      if (collapseButton) {
        await user.click(collapseButton);
        const sidebar = container.querySelector('aside');
        expect(sidebar).toHaveClass('w-16');
      }
    });
  });
});
