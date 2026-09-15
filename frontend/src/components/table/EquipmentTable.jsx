import React from 'react';

import { MagnifyingGlassIcon } from '@heroicons/react/24/outline';

import ActionMenu from '../action/ActionMenu';

const STATUS_STYLE = {
  'ในคลัง': { badge: 'bg-zinc-100 text-zinc-600', dot: 'bg-zinc-400' },
  'กำลังใช้งาน': { badge: 'bg-emerald-50 text-emerald-700', dot: 'bg-emerald-500' },
  'เสียหาย': { badge: 'bg-red-50 text-red-700', dot: 'bg-red-500' },
  'เลิกใช้งาน': { badge: 'bg-zinc-200 text-zinc-500', dot: 'bg-zinc-400' },
};

const DEFAULT_STATUS_STYLE = { badge: 'bg-zinc-100 text-zinc-600', dot: 'bg-zinc-400' };

function EquipmentTable({ items, canManage, onView, onEdit, onDelete }) {
  if (items.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <div className="mb-3 flex h-11 w-11 items-center justify-center rounded-full bg-zinc-100 text-zinc-400">
          <MagnifyingGlassIcon className="h-5 w-5" />
        </div>
        <p className="text-sm font-medium text-zinc-700">ไม่พบอุปกรณ์</p>
        <p className="mt-1 text-xs text-zinc-400">ลองเปลี่ยนคำค้นหาหรือตัวกรอง</p>
      </div>
    );
  }

  return (
    <div className="w-full overflow-x-auto rounded-lg border border-zinc-100">
      <table className="w-full min-w-[720px] table-fixed border-collapse text-sm">
        <thead>
          <tr className="border-b border-zinc-200 bg-zinc-50/60 text-center text-[11px] font-semibold text-sm tracking-wide text-zinc-500">
            <th className="w-[8%] px-4 py-3">ลำดับ</th>
            <th className="w-[24%] px-4 py-3">Asset Name</th>
            <th className="w-[22%] px-4 py-3">Product Name</th>
            <th className="w-[20%] px-4 py-3">Serial Number</th>
            <th className="w-[14%] px-4 py-3">ประเภท</th>
            <th className="w-[20%] px-4 py-3">สถานะ</th>
            <th className="w-[8%] px-4 py-3">จัดการ</th>
          </tr>
        </thead>

        <tbody className="divide-y divide-zinc-100">
          {items.map((item, index) => {
            const statusStyle = STATUS_STYLE[item.status] || DEFAULT_STATUS_STYLE;

            return (
              <tr key={item.id} className="group text-center align-middle transition-colors hover:bg-zinc-50/70">
                <td className="px-4 py-3.5 align-middle text-xs font-medium text-zinc-400">
                  {String(index + 1)}
                </td>

                <td className="px-4 py-3.5 align-middle text-zinc-500">
                  <span className="block truncate" title={item.asset_name || ''}>
                    {item.asset_name || '-'}
                  </span>
                </td>

                <td className="px-4 py-3.5 align-middle text-zinc-500">
                  <span className="block truncate" title={item.product_name || ''}>
                    {item.product_name || '-'}
                  </span>
                </td>

                <td className="px-4 py-3.5 align-middle text-zinc-500">
                  <span className="block truncate" title={item.asset_serial_no || ''}>
                    {item.asset_serial_no || '-'}
                  </span>
                </td>

                <td className="px-4 py-3.5 align-middle text-zinc-500">
                  <span className="block truncate" title={item.type_name || ''}>
                    {item.type_name || '-'}
                  </span>
                </td>

                <td className="px-4 py-3.5 align-middle">
                  <span
                    className={`inline-flex items-center justify-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-medium ${statusStyle.badge}`}
                  >
                    <span className={`h-1.5 w-1.5 rounded-full ${statusStyle.dot}`} />
                    {item.status || '-'}
                  </span>
                </td>

                <td className="px-4 py-3.5 align-middle">
                  <div className="flex justify-center opacity-70 transition-opacity group-hover:opacity-100">
                    <ActionMenu
                      label={item.asset_name}
                      canManage={canManage}
                      onView={() => onView(item)}
                      onEdit={() => onEdit(item)}
                      onDelete={() => onDelete(item)}
                    />
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

export default EquipmentTable;