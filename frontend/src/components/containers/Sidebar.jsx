import React from 'react';
import { NavLink } from 'react-router-dom';
import menu from '../../constants/menu';

const Sidebar = ({ user }) => {
  const menuItems = menu[user?.role] || [];

  return (
    <aside className="sticky top-0 flex h-screen w-56 flex-shrink-0 flex-col bg-zinc-900 text-white">
      {/* Brand */}
      <div className="flex h-16 items-center gap-3 border-b border-white/10 px-5">
        <img
          src="/images/easybuy-logo.png"
          alt="EasyBuy"
          className="h-9 w-9 flex-shrink-0 object-contain"
        />
        <div className="flex flex-col leading-tight">
          <span className="text-[15px] font-semibold text-white">
            EasyBuy
          </span>
          <span className="text-[11px] font-medium tracking-wide text-zinc-500">
            Service Inventory
          </span>
        </div>
      </div>

      {/* Menu */}
      <nav className="flex-1 overflow-y-auto p-3 scrollbar-thin scrollbar-thumb-white/10 scrollbar-track-transparent">
        <div className="space-y-1">
          {menuItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                [
                  'group flex items-center gap-3 rounded-lg',
                  'border-l-[3px] px-4 py-2.5',
                  'text-[13.5px] font-medium',
                  'transition-all duration-150',

                  isActive
                    ? 'border-[#D6272C] bg-red-600/15 text-white shadow-sm'
                    : 'border-transparent text-zinc-400 hover:translate-x-0.5 hover:bg-white/5 hover:text-white',
                ].join(' ')
              }
            >
              <span
                className={[
                  'flex h-5 w-5 flex-shrink-0 items-center justify-center',
                  'transition-colors duration-150',
                ].join(' ')}
              >
                {item.icon}
              </span>

              <span>{item.label}</span>
            </NavLink>
          ))}
        </div>
      </nav>
    </aside>
  );
};

export default Sidebar;