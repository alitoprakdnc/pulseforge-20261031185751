import React, { useState, useContext, Fragment } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/router';
import Image from 'next/image';
import { AuthContext } from '../context/AuthContext';

/* Icons – simple inline SVGs to avoid extra dependencies */
const DashboardIcon = (
  <svg
    className="h-5 w-5"
    xmlns="http://www.w3.org/2000/svg"
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
    aria-hidden="true"
  >
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 12h18M3 6h18M3 18h18" />
  </svg>
);

const PipelinesIcon = (
  <svg
    className="h-5 w-5"
    xmlns="http://www.w3.org/2000/svg"
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
    aria-hidden="true"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M9 12h6m-3-3v6m5-9h2a2 2 0 012 2v12a2 2 0 01-2 2h-2M5 6h2a2 2 0 012 2v12a2 2 0 01-2 2H5a2 2 0 01-2-2V8a2 2 0 012-2z"
    />
  </svg>
);

const SettingsIcon = (
  <svg
    className="h-5 w-5"
    xmlns="http://www.w3.org/2000/svg"
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
    aria-hidden="true"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l.7 2.147a1.5 1.5 0 001.415 1.045h2.262c.969 0 1.371 1.24.588 1.81l-1.83 1.33a1.5 1.5 0 00-.543 1.68l.7 2.147c.3.921-.755 1.688-1.54 1.045l-1.83-1.33a1.5 1.5 0 00-1.754 0l-1.83 1.33c-.785.643-1.84-.124-1.54-1.045l.7-2.147a1.5 1.5 0 00-.543-1.68l-1.83-1.33c-.783-.57-.38-1.81.588-1.81h2.262a1.5 1.5 0 001.415-1.045l.7-2.147z"
    />
  </svg>
);

/* Navigation definition */
type NavItem = {
  name: string;
  href: string;
  icon: React.ReactNode;
};

const NAV_ITEMS: NavItem[] = [
  { name: 'Dashboard', href: '/dashboard', icon: DashboardIcon },
  { name: 'Pipelines', href: '/pipelines', icon: PipelinesIcon },
  { name: 'Settings', href: '/settings', icon: SettingsIcon },
];

type LayoutProps = {
  children: React.ReactNode;
};

export const Layout: React.FC<LayoutProps> = ({ children }) => {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const router = useRouter();
  const { user, logout } = useContext(AuthContext);

  const isActive = (href: string) => router.pathname.startsWith(href);

  const toggleSidebar = () => setSidebarOpen((prev) => !prev);
  const closeSidebar = () => setSidebarOpen(false);

  return (
    <div className="flex min-h-screen bg-background">
      {/* Mobile overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-30 bg-black bg-opacity-30 lg:hidden"
          aria-hidden="true"
          onClick={closeSidebar}
        />
      )}

      {/* Sidebar */}
      <aside
        className={`
          fixed inset-y-0 left-0 z-40 w-64 transform overflow-y-auto bg-card transition-transform duration-200 ease-in-out
          lg:translate-x-0 lg:static lg:inset-auto
          ${sidebarOpen ? 'translate-x-0' : '-translate-x-full'}
        `}
        aria-label="Primary"
      >
        <div className="flex h-16 items-center border-b border-border px-4">
          <Image src="/favicon.ico" alt="PulseForge" width={32} height={32} className="mr-2" />
          <span className="text-xl font-semibold text-primary">PulseForge</span>
        </div>
        <nav className="mt-4 space-y-1 px-2">
          {NAV_ITEMS.map((item) => (
            <Link key={item.name} href={item.href}>
              <a
                className={`
                  group flex items-center rounded-md px-3 py-2 text-sm font-medium
                  ${isActive(item.href)
                    ? 'bg-primary-100 text-primary'
                    : 'text-secondary hover:bg-gray-100 hover:text-primary'}
                `}
                aria-current={isActive(item.href) ? 'page' : undefined}
              >
                <span className="mr-3 text-current">{item.icon}</span>
                {item.name}
              </a>
            </Link>
          ))}
        </nav>
      </aside>

      {/* Main content area */}
      <div className="flex flex-1 flex-col lg:ml-64">
        {/* Header */}
        <header className="flex h-16 items-center justify-between border-b border-border bg-card px-4 shadow-sm">
          {/* Mobile menu button */}
          <button
            type="button"
            className="inline-flex items-center justify-center rounded-md p-2 text-secondary hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-primary lg:hidden"
            aria-controls="mobile-menu"
            aria-expanded={sidebarOpen}
            onClick={toggleSidebar}
          >
            <span className="sr-only">Open main menu</span>
            {/* Hamburger icon */}
            <svg
              className="h-6 w-6"
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              aria-hidden="true"
            >
              {sidebarOpen ? (
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              ) : (
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
              )}
            </svg>
          </button>

          {/* Right side: user menu */}
          <div className="flex items-center space-x-4">
            <span className="text-sm font-medium text-secondary">{user?.name ?? 'Guest'}</span>
            {user?.avatarUrl ? (
              <Image
                src={user.avatarUrl}
                alt={`${user.name}'s avatar`}
                width={32}
                height={32}
                className="rounded-full"
              />
            ) : (
              <div className="h-8 w-8 rounded-full bg-gray-300" />
            )}
            <button
              type="button"
              onClick={logout}
              className="rounded-md bg-danger px-3 py-1.5 text-sm font-medium text-white hover:bg-danger/80 focus:outline-none focus:ring-2 focus:ring-danger"
            >
              Sign out
            </button>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-y-auto p-6">{children}</main>
      </div>
    </div>
  );
};

export default Layout;