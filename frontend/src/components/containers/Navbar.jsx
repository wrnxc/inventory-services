import React from 'react';
import { BellIcon, ArrowRightStartOnRectangleIcon } from '@heroicons/react/24/outline';

function Navbar({
  pageTitle,
  user,
  onLogout,
  hasNotifications = false,
}) {
  const initials =
    user?.username
      ?.slice(0, 2)
      .toUpperCase() || 'US';

  const roleLabel = {
    system_admin: 'System Admin',
    admin: 'Admin',
    user: 'User',
  }[user?.role] || 'User';

  return (
    <nav className="sticky top-0 z-40 border-b border-zinc-200 bg-white/80 backdrop-blur-sm">
      <div className="flex h-16 items-center justify-between px-6">
        <h1 className="text-[17px] font-semibold tracking-tight text-zinc-800">
          {pageTitle}
        </h1>

        <div className="flex items-center gap-2">
          <button
            type="button"
            aria-label="การแจ้งเตือน"
            className="relative rounded-lg p-2 transition-colors hover:bg-zinc-100"
          >
            <BellIcon className="h-5 w-5 text-zinc-500" />
            {hasNotifications && (
              <span className="absolute right-1.5 top-1.5 h-2 w-2 rounded-full bg-red-600 ring-2 ring-white" />
            )}
          </button>

          <div className="mx-2 hidden h-8 w-px bg-zinc-200 sm:block" />

          <div className="hidden text-right sm:block">
            <div className="text-[13px] font-medium leading-tight text-zinc-700">
              {user?.username}
            </div>
            <div className="text-[11px] leading-tight text-zinc-400">
              {roleLabel}
            </div>
          </div>

          <div className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full bg-red-50 text-[12px] font-semibold text-red-600 ring-1 ring-red-100">
            {initials}
          </div>

          <button
            type="button"
            onClick={onLogout}
            aria-label="Logout"
            className="rounded-lg p-2 text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-red-600"
          >
            <ArrowRightStartOnRectangleIcon className="h-5 w-5" />
          </button>
        </div>
      </div>
    </nav>
  );
}

export default Navbar;