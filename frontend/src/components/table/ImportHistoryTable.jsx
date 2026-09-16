import React from 'react';

import {
  ArrowUpTrayIcon,
} from '@heroicons/react/24/outline';

const STATUS_STYLE = {
  success: {
    label: 'สำเร็จ',
    badge: 'bg-emerald-50 text-emerald-700',
    dot: 'bg-emerald-500',
  },

  partial: {
    label: 'สำเร็จบางส่วน',
    badge: 'bg-amber-50 text-amber-700',
    dot: 'bg-amber-500',
  },

  duplicate: {
    label: 'ข้อมูลซ้ำ',
    badge: 'bg-amber-50 text-amber-700',
    dot: 'bg-amber-500',
  },

  failed: {
    label: 'ไม่สำเร็จ',
    badge: 'bg-red-50 text-red-700',
    dot: 'bg-red-500',
  },
};

function getImportStatus(item) {
  const total = Number(item.total) || 0;
  const success = Number(item.success) || 0;
  const duplicate =
    Number(item.duplicate) || 0;
  const failed = Number(item.failed) || 0;

  // ซ้ำทั้งหมด
  if (
    total > 0 &&
    success === 0 &&
    duplicate === total &&
    failed === 0
  ) {
    return STATUS_STYLE.duplicate;
  }

  // สำเร็จทั้งหมด
  if (
    total > 0 &&
    success === total &&
    duplicate === 0 &&
    failed === 0
  ) {
    return STATUS_STYLE.success;
  }

  // สำเร็จบางส่วน
  if (
    success > 0 &&
    (duplicate > 0 || failed > 0)
  ) {
    return STATUS_STYLE.partial;
  }

  // ไม่สำเร็จ
  return STATUS_STYLE.failed;
}

function formatImportDate(value) {
  if (!value) return '-';

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return '-';
  }

  return date.toLocaleString('th-TH');
}

function ImportHistoryTable({
  items = [],
  loading = false,
}) {
  if (loading) {
    return (
      <div className="flex items-center justify-center py-16">
        <p className="text-sm text-zinc-400">
          กำลังโหลดข้อมูล...
        </p>
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <div className="mb-3 flex h-11 w-11 items-center justify-center rounded-full bg-zinc-100 text-zinc-400">
          <ArrowUpTrayIcon className="h-5 w-5" />
        </div>

        <p className="text-sm font-medium text-zinc-700">
          ยังไม่มีประวัติการนำเข้า
        </p>

        <p className="mt-1 text-xs text-zinc-400">
          ประวัติจะปรากฏหลังจากนำเข้าไฟล์
        </p>
      </div>
    );
  }

  return (
    <div className="w-full overflow-x-auto rounded-lg border border-zinc-100">
      <table className="w-full min-w-[1000px] table-fixed border-collapse text-sm">
        <thead>
          <tr className="border-b border-zinc-200 bg-zinc-50/60 text-center text-[11px] font-semibold tracking-wide text-zinc-500">
            <th className="w-[6%] px-4 py-3">
              ลำดับ
            </th>

            <th className="w-[18%] px-4 py-3">
              วันที่/เวลา
            </th>

            <th className="w-[22%] px-4 py-3">
              ชื่อไฟล์
            </th>

            <th className="w-[12%] px-4 py-3">
              ผู้นำเข้า
            </th>

            <th className="w-[8%] px-4 py-3">
              ทั้งหมด
            </th>

            <th className="w-[8%] px-4 py-3">
              สำเร็จ
            </th>

            <th className="w-[8%] px-4 py-3">
              ข้อมูลซ้ำ
            </th>

            <th className="w-[8%] px-4 py-3">
              ไม่สำเร็จ
            </th>

            <th className="w-[14%] px-4 py-3">
              สถานะ
            </th>
          </tr>
        </thead>

        <tbody className="divide-y divide-zinc-100">
          {items.map((item, index) => {
            const statusStyle =
              getImportStatus(item);

            return (
              <tr
                key={item.id}
                className="group text-center align-middle transition-colors hover:bg-zinc-50/70"
              >
                {/* ลำดับ */}
                <td className="px-4 py-3.5 text-xs font-medium text-zinc-400">
                  {index + 1}
                </td>

                {/* วันที่/เวลา */}
                <td className="px-4 py-3.5 text-zinc-500">
                  <span className="whitespace-nowrap">
                    {formatImportDate(
                      item.imported_at
                    )}
                  </span>
                </td>

                {/* ชื่อไฟล์ */}
                <td className="px-4 py-3.5 text-zinc-500">
                  <span
                    className="block truncate"
                    title={item.file_name || ''}
                  >
                    {item.file_name || '-'}
                  </span>
                </td>

                {/* ผู้นำเข้า */}
                <td className="px-4 py-3.5 text-zinc-500">
                  <span
                    className="block truncate"
                    title={item.username || ''}
                  >
                    {item.username || '-'}
                  </span>
                </td>

                {/* ทั้งหมด */}
                <td className="px-4 py-3.5 font-medium text-zinc-600">
                  {item.total ?? 0}
                </td>

                {/* สำเร็จ */}
                <td className="px-4 py-3.5 font-medium text-emerald-600">
                  {item.success ?? 0}
                </td>

                {/* ข้อมูลซ้ำ */}
                <td className="px-4 py-3.5 font-medium text-amber-600">
                  {item.duplicate ?? 0}
                </td>

                {/* ไม่สำเร็จ */}
                <td className="px-4 py-3.5 font-medium text-red-600">
                  {item.failed ?? 0}
                </td>

                {/* สถานะ */}
                <td className="px-4 py-3.5">
                  <span
                    className={`inline-flex items-center justify-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-medium ${statusStyle.badge}`}
                  >
                    <span
                      className={`h-1.5 w-1.5 rounded-full ${statusStyle.dot}`}
                    />

                    {statusStyle.label}
                  </span>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

export default ImportHistoryTable;