import React from 'react';

function ImportPreviewTable({
  items = [],
  total = 0,
}) {
  if (items.length === 0) {
    return null;
  }

  return (
    <div className="mt-5">
      <div className="mb-3 flex flex-wrap items-end justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold text-zinc-700">
            ตัวอย่างข้อมูล
          </h3>

          <p className="mt-1 text-xs text-zinc-400">
            ตรวจสอบข้อมูลก่อนนำเข้าสู่ระบบ
          </p>
        </div>

        <p className="text-xs text-zinc-400">
          แสดง {items.length} จาก{' '}
          {total.toLocaleString()} รายการ
        </p>
      </div>

      <div className="w-full overflow-x-auto rounded-lg border border-zinc-100">
        <table className="w-full min-w-[800px] table-fixed border-collapse text-sm">
          <thead>
            <tr className="border-b border-zinc-200 bg-zinc-50/60 text-center text-[11px] font-semibold tracking-wide text-zinc-500">
              <th className="w-[8%] px-4 py-3">
                ลำดับ
              </th>

              <th className="w-[22%] px-4 py-3">
                Asset Name
              </th>

              <th className="w-[24%] px-4 py-3">
                Product Name
              </th>

              <th className="w-[24%] px-4 py-3">
                Serial Number
              </th>

              <th className="w-[22%] px-4 py-3">
                ประเภท
              </th>
            </tr>
          </thead>

          <tbody className="divide-y divide-zinc-100">
            {items.map((item, index) => (
              <tr
                key={`${item.row_number}-${index}`}
                className="text-center align-middle transition-colors hover:bg-zinc-50/70"
              >
                <td className="px-4 py-3.5 text-xs font-medium text-zinc-400">
                  {index + 1}
                </td>

                <td className="px-4 py-3.5 text-zinc-600">
                  <span
                    className="block truncate"
                    title={item.asset_name || ''}
                  >
                    {item.asset_name || '-'}
                  </span>
                </td>

                <td className="px-4 py-3.5 text-zinc-500">
                  <span
                    className="block truncate"
                    title={item.product_name || ''}
                  >
                    {item.product_name || '-'}
                  </span>
                </td>

                <td className="px-4 py-3.5 text-zinc-500">
                  <span
                    className="block truncate"
                    title={item.asset_serial_no || ''}
                  >
                    {item.asset_serial_no || '-'}
                  </span>
                </td>

                <td className="px-4 py-3.5 text-zinc-500">
                  <span
                    className="block truncate"
                    title={item.product_type || ''}
                  >
                    {item.product_type || '-'}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export default ImportPreviewTable;