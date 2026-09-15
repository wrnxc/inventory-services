import React from 'react';
import { Outlet, useLocation } from 'react-router-dom';

import Navbar from '../components/containers/Navbar';
import Sidebar from '../components/containers/Sidebar';

function AppLayout({ user, onLogout }) {
  const location = useLocation();

  const pageTitle =
    location.pathname === '/equipment'
      ? 'อุปกรณ์'
      : location.pathname === '/repair-equipment'
        ? 'การแจ้งซ่อม'
        : location.pathname === '/borrow'
        ? 'เบิก-คืนอุปกรณ์'
          : location.pathname === '/dashboard'
            ? 'แดชบอร์ด'
            : location.pathname === '/report'
              ? 'รายงาน'
              : location.pathname === '/import-data'
                ? 'นำเข้าข้อมูล'
                : location.pathname === '/activity-logs'
                  ? 'ประวัติการใช้งาน'
                  : 'EasyBuy Inventory';

  return (
    <div className="flex min-h-screen bg-[#F4F4F5]">
      <Sidebar user={user} />

      <div className="flex min-w-0 flex-1 flex-col">
        <Navbar
          pageTitle={pageTitle}
          user={user}
          onLogout={onLogout}
        />

        <main className="flex-1 overflow-y-auto">
          <Outlet />
        </main>
      </div>
    </div>
  );
}

export default AppLayout;