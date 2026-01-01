import React, { useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import {
  Package,
  Layers,
  MapPin,
  Box,
  Truck,
  Warehouse,
  Users,
  LayoutDashboard,
  Menu,
  X,
  ChevronDown,
  Bell,
  Search,
} from 'lucide-react';

export interface NavItem {
  label: string;
  path: string;
  icon: React.ReactNode;
  badge?: number;
  children?: Array<{ label: string; path: string }>;
}

const defaultNavItems: NavItem[] = [
  { label: 'Dashboard', path: '/', icon: <LayoutDashboard className="h-5 w-5" /> },
  { label: 'Orders', path: '/orders', icon: <Package className="h-5 w-5" /> },
  { label: 'Waves', path: '/waves', icon: <Layers className="h-5 w-5" /> },
  { label: 'Picking', path: '/picking', icon: <MapPin className="h-5 w-5" /> },
  { label: 'Packing', path: '/packing', icon: <Box className="h-5 w-5" /> },
  { label: 'Shipping', path: '/shipping', icon: <Truck className="h-5 w-5" /> },
  { label: 'Inventory', path: '/inventory', icon: <Warehouse className="h-5 w-5" /> },
  { label: 'Labor', path: '/labor', icon: <Users className="h-5 w-5" /> },
];

export interface SidebarProps {
  navItems?: NavItem[];
  collapsed?: boolean;
  onCollapse?: (collapsed: boolean) => void;
}

export function Sidebar({ navItems = defaultNavItems, collapsed = false, onCollapse }: SidebarProps) {
  const location = useLocation();
  const [expandedItems, setExpandedItems] = useState<string[]>([]);

  const toggleExpanded = (label: string) => {
    setExpandedItems((prev) =>
      prev.includes(label) ? prev.filter((l) => l !== label) : [...prev, label]
    );
  };

  const isActive = (path: string) => {
    if (path === '/') return location.pathname === '/';
    return location.pathname.startsWith(path);
  };

  return (
    <aside
      className={`
        fixed left-0 top-0 h-full bg-gray-900 text-white z-40
        transition-all duration-300 ease-in-out
        ${collapsed ? 'w-16' : 'w-64'}
      `}
    >
      {/* Logo */}
      <div className="h-16 flex items-center justify-between px-4 border-b border-gray-800">
        {!collapsed && (
          <div className="flex items-center gap-2">
            <Warehouse className="h-8 w-8 text-primary-400" />
            <span className="font-bold text-lg">WMS Platform</span>
          </div>
        )}
        <button
          onClick={() => onCollapse?.(!collapsed)}
          className="p-1.5 rounded-lg hover:bg-gray-800 transition-colors"
        >
          {collapsed ? <Menu className="h-5 w-5" /> : <X className="h-5 w-5" />}
        </button>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-2 py-4 space-y-1 overflow-y-auto">
        {navItems.map((item) => {
          const active = isActive(item.path);
          const expanded = expandedItems.includes(item.label);

          return (
            <div key={item.label}>
              <Link
                to={item.children ? '#' : item.path}
                onClick={(e) => {
                  if (item.children) {
                    e.preventDefault();
                    toggleExpanded(item.label);
                  }
                }}
                className={`
                  flex items-center gap-3 px-3 py-2.5 rounded-lg
                  transition-colors duration-150
                  ${active
                    ? 'bg-primary-600 text-white'
                    : 'text-gray-300 hover:bg-gray-800 hover:text-white'
                  }
                `}
              >
                {item.icon}
                {!collapsed && (
                  <>
                    <span className="flex-1 font-medium">{item.label}</span>
                    {item.badge && (
                      <span className="px-2 py-0.5 text-xs bg-error-500 text-white rounded-full">
                        {item.badge}
                      </span>
                    )}
                    {item.children && (
                      <ChevronDown
                        className={`h-4 w-4 transition-transform ${expanded ? 'rotate-180' : ''}`}
                      />
                    )}
                  </>
                )}
              </Link>
              {!collapsed && item.children && expanded && (
                <div className="ml-8 mt-1 space-y-1">
                  {item.children.map((child) => (
                    <Link
                      key={child.path}
                      to={child.path}
                      className={`
                        block px-3 py-2 text-sm rounded-lg transition-colors
                        ${location.pathname === child.path
                          ? 'bg-gray-800 text-white'
                          : 'text-gray-400 hover:text-white hover:bg-gray-800'
                        }
                      `}
                    >
                      {child.label}
                    </Link>
                  ))}
                </div>
              )}
            </div>
          );
        })}
      </nav>
    </aside>
  );
}

export interface HeaderProps {
  sidebarCollapsed?: boolean;
  notifications?: number;
}

export function Header({ sidebarCollapsed = false, notifications = 0 }: HeaderProps) {
  return (
    <header
      className={`
        fixed top-0 right-0 h-16 bg-white border-b border-gray-200 z-30
        transition-all duration-300 ease-in-out
        ${sidebarCollapsed ? 'left-16' : 'left-64'}
      `}
    >
      <div className="h-full px-6 flex items-center justify-between">
        {/* Search */}
        <div className="flex-1 max-w-lg">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-gray-400" />
            <input
              type="search"
              placeholder="Search orders, waves, inventory..."
              className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-4">
          <button className="relative p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg">
            <Bell className="h-5 w-5" />
            {notifications > 0 && (
              <span className="absolute -top-1 -right-1 w-5 h-5 flex items-center justify-center text-xs font-medium bg-error-500 text-white rounded-full">
                {notifications > 9 ? '9+' : notifications}
              </span>
            )}
          </button>
          <div className="h-8 w-8 rounded-full bg-primary-100 text-primary-700 flex items-center justify-center font-medium">
            A
          </div>
        </div>
      </div>
    </header>
  );
}

export interface MainLayoutProps {
  children: React.ReactNode;
}

export function MainLayout({ children }: MainLayoutProps) {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  return (
    <div className="min-h-screen bg-gray-50">
      <Sidebar collapsed={sidebarCollapsed} onCollapse={setSidebarCollapsed} />
      <Header sidebarCollapsed={sidebarCollapsed} />
      <main
        className={`
          pt-16 min-h-screen transition-all duration-300 ease-in-out
          ${sidebarCollapsed ? 'ml-16' : 'ml-64'}
        `}
      >
        <div className="p-6">{children}</div>
      </main>
    </div>
  );
}
