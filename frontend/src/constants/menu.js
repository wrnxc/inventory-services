import React from 'react';
import {
  ChartBarIcon,
  ArrowUpTrayIcon,
  ArrowsRightLeftIcon,
  WrenchScrewdriverIcon,
  ArchiveBoxIcon,
  ClipboardDocumentListIcon,
} from '@heroicons/react/24/outline';

const commonMenu = [
  {
    label: 'แดชบอร์ด',
    to: '/dashboard',
    icon: <ChartBarIcon className="h-5 w-5" />,
  },
  {
    label: 'อุปกรณ์',
    to: '/equipment',
    icon: <ArchiveBoxIcon className="h-5 w-5" />,
  },
  {
    label: 'เบิก-คืนอุปกรณ์',
    to: '/borrow',
    icon: <ArrowsRightLeftIcon className="h-5 w-5" />,
  },
  {
    label: 'การแจ้งซ่อม',
    to: '/repair-equipment',
    icon: <WrenchScrewdriverIcon className="h-5 w-5" />,
  },
];

const adminOnlyMenu = [
  {
    label: 'รายงาน',
    to: '/report',
    icon: <ClipboardDocumentListIcon className="h-5 w-5" />,
  },
  {
    label: 'นำเข้าข้อมูล',
    to: '/import-data',
    icon: <ArrowUpTrayIcon className="h-5 w-5" />,
  },
];

const menu = {
  system_admin: [
    ...commonMenu,
    ...adminOnlyMenu,
  ],

  admin: [
    ...commonMenu,
    ...adminOnlyMenu,
  ],

  user: [
    ...commonMenu,
  ],
};

export default menu;