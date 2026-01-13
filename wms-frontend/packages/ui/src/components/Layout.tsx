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
  ArrowDownCircle,
  MoveDown,
  Route,
  Network,
  Settings,
  Merge,
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
  { label: 'Receiving', path: '/receiving', icon: <ArrowDownCircle className="h-5 w-5" /> },
  { label: 'Stow', path: '/stow', icon: <MoveDown className="h-5 w-5" /> },
  { label: 'Routing', path: '/routing', icon: <Route className="h-5 w-5" /> },
  { label: 'Walling', path: '/walling', icon: <Layers className="h-5 w-5" /> },
  { label: 'Consolidation', path: '/consolidation', icon: <Merge className="h-5 w-5" /> },
  { label: 'Sortation', path: '/sortation', icon: <Network className="h-5 w-5" /> },
  { label: 'Facility', path: '/facility', icon: <Settings className="h-5 w-5" /> },
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
        fixed left-0 top-0 h-full bg-white border-r border-gray-200 z-40
        transition-all duration-300 ease-in-out
        ${collapsed ? 'w-16' : 'w-64'}
      `}
    >
      {/* Logo */}
      <div className="h-16 flex items-center justify-between px-4 border-b border-gray-100">
        {!collapsed && (
          <div className="flex items-center gap-2">
            <div className="w-9 h-9 bg-primary-600 rounded-lg flex items-center justify-center">
              <Warehouse className="h-5 w-5 text-white" />
            </div>
            <span className="font-bold text-lg text-gray-900">WMS Platform</span>
          </div>
        )}
        {collapsed && (
          <div className="w-9 h-9 bg-primary-600 rounded-lg flex items-center justify-center mx-auto">
            <Warehouse className="h-5 w-5 text-white" />
          </div>
        )}
        {!collapsed && (
          <button
            onClick={() => onCollapse?.(!collapsed)}
            className="p-1.5 rounded-lg hover:bg-gray-100 text-gray-500 transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        )}
      </div>

      {/* Collapse button when collapsed */}
      {collapsed && (
        <div className="px-2 py-2 border-b border-gray-100">
          <button
            onClick={() => onCollapse?.(!collapsed)}
            className="w-full p-2 rounded-lg hover:bg-gray-100 text-gray-500 transition-colors flex items-center justify-center"
          >
            <Menu className="h-5 w-5" />
          </button>
        </div>
      )}

      {/* Navigation */}
      <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
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
                  transition-all duration-150 relative
                  ${active
                    ? 'bg-primary-50 text-primary-700 font-medium before:absolute before:left-0 before:top-1/2 before:-translate-y-1/2 before:w-1 before:h-6 before:bg-primary-600 before:rounded-r-full'
                    : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                  }
                `}
              >
                <span className={active ? 'text-primary-600' : 'text-gray-400'}>{item.icon}</span>
                {!collapsed && (
                  <>
                    <span className="flex-1">{item.label}</span>
                    {item.badge && (
                      <span className="px-2 py-0.5 text-xs bg-error-500 text-white rounded-full">
                        {item.badge}
                      </span>
                    )}
                    {item.children && (
                      <ChevronDown
                        className={`h-4 w-4 text-gray-400 transition-transform ${expanded ? 'rotate-180' : ''}`}
                      />
                    )}
                  </>
                )}
              </Link>
              {!collapsed && item.children && expanded && (
                <div className="ml-9 mt-1 space-y-1 border-l-2 border-gray-100 pl-3">
                  {item.children.map((child) => (
                    <Link
                      key={child.path}
                      to={child.path}
                      className={`
                        block px-3 py-2 text-sm rounded-lg transition-colors
                        ${location.pathname === child.path
                          ? 'bg-primary-50 text-primary-700 font-medium'
                          : 'text-gray-500 hover:text-gray-900 hover:bg-gray-50'
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
        fixed top-0 right-0 h-16 bg-white/95 backdrop-blur-sm border-b border-gray-100 z-30
        shadow-sm transition-all duration-300 ease-in-out
        ${sidebarCollapsed ? 'left-16' : 'left-64'}
      `}
    >
      <div className="h-full px-6 flex items-center justify-between">
        {/* Search */}
        <div className="flex-1 max-w-lg">
          <div className="relative">
            <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
            <input
              type="search"
              placeholder="Search orders, waves, inventory..."
              className="
                w-full pl-11 pr-4 py-2.5
                bg-gray-100 border-0 rounded-full
                text-sm text-gray-900 placeholder-gray-500
                focus:bg-white focus:ring-2 focus:ring-primary-500 focus:shadow-sm
                transition-all duration-200
              "
            />
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-3">
          <button className="relative p-2.5 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-full transition-colors">
            <Bell className="h-5 w-5" />
            {notifications > 0 && (
              <span className="absolute top-0.5 right-0.5 w-5 h-5 flex items-center justify-center text-xs font-medium bg-error-500 text-white rounded-full animate-pulse-slow">
                {notifications > 9 ? '9+' : notifications}
              </span>
            )}
          </button>
          <div className="h-9 w-9 rounded-full bg-gradient-to-br from-primary-500 to-primary-600 text-white flex items-center justify-center font-medium shadow-sm">
            A
          </div>
        </div>
      </div>
    </header>
  );
}

export interface MainLayoutProps {
  children: React.ReactNode;
  navItems?: NavItem[];
}

export function MainLayout({ children, navItems }: MainLayoutProps) {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  return (
    <div className="min-h-screen bg-gray-50">
      <Sidebar
        navItems={navItems}
        collapsed={sidebarCollapsed}
        onCollapse={setSidebarCollapsed}
      />
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
