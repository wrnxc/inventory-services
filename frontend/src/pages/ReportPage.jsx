import React, { useCallback, useEffect, useState } from 'react';
import { ClipboardDocumentListIcon } from '@heroicons/react/24/outline';
import { getReport } from '../api/reportClient';

const REPORT_TYPES = [
  { value: 'stock', label: 'Stock' },
  { value: 'borrow', label: 'Borrow' },
  { value: 'return', label: 'Return' },
  { value: 'repair', label: 'Repair' },
];

function ReportPage({ user, initialState = 'loading' }) {
  const canViewReports =
    user?.role === 'admin' ||
    user?.role === 'system_admin';

  const [status, setStatus] = useState(initialState);

  const [type, setType] = useState('stock');
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');

  const [report, setReport] = useState({ items: [] });

  // --------------------------------------------------
  // Load report
  // --------------------------------------------------

  const loadReport = useCallback(async () => {
    try {
      setStatus('loading');

      const data = await getReport({
        type,
        from,
        to,
      });

      setReport(data);

      setStatus('success');
    } catch (error) {
      console.error(
        'Unable to load report:',
        error
      );

      setStatus('error');
    }
  }, [from, to, type]);

  useEffect(() => {
    if (
      canViewReports &&
      initialState !== 'error'
    ) {
      loadReport();
    }
  }, [
    canViewReports,
    initialState,
    loadReport,
  ]);

  // --------------------------------------------------
  // Access guard
  // --------------------------------------------------

  if (!canViewReports) {
    return (
      <div className="min-h-screen bg-zinc-100 p-6">
        <div className="rounded-xl border border-zinc-200 bg-white p-8 text-sm text-zinc-500">
          หน้านี้สำหรับ Admin และ System Admin
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-zinc-100 text-zinc-800">
      <main className="p-6">
        <section className="rounded-xl border border-zinc-200 bg-white p-5">
          {/* Header */}
          <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="text-sm font-semibold">
                รายงาน
                {' '}
                {
                  REPORT_TYPES.find(
                    (item) =>
                      item.value === type
                  )?.label
                }
              </h2>

              <p className="mt-1 text-xs text-zinc-400">
                {status === 'success'
                  ? `ทั้งหมด ${
                      report.items?.length ??
                      0
                    } รายการ`
                  : 'เลือกช่วงวันที่และประเภทเพื่อดูรายงาน'}
              </p>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <button
                type="button"
                onClick={loadReport}
                className="inline-flex h-9 items-center gap-1.5 rounded-lg bg-red-600 px-4 text-sm font-medium text-white transition hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-300"
              >
                โหลดรายงาน
              </button>
            </div>
          </div>

          {/* Filters */}
          <div className="mb-4 flex flex-wrap items-end gap-2">
            <label className="flex flex-col gap-1 text-xs text-zinc-500">
              ประเภท
              <select
                value={type}
                onChange={(event) =>
                  setType(
                    event.target.value
                  )
                }
                className="h-9 rounded-lg border border-zinc-200 bg-white px-3 text-sm text-zinc-600 outline-none focus:border-zinc-400"
              >
                {REPORT_TYPES.map(
                  (item) => (
                    <option
                      key={item.value}
                      value={item.value}
                    >
                      {item.label}
                    </option>
                  )
                )}
              </select>
            </label>

            <label className="flex flex-col gap-1 text-xs text-zinc-500">
              จากวันที่
              <input
                type="date"
                value={from}
                onChange={(event) =>
                  setFrom(
                    event.target.value
                  )
                }
                className="h-9 rounded-lg border border-zinc-200 bg-white px-3 text-sm outline-none focus:border-zinc-400"
              />
            </label>

            <label className="flex flex-col gap-1 text-xs text-zinc-500">
              ถึงวันที่
              <input
                type="date"
                value={to}
                onChange={(event) =>
                  setTo(
                    event.target.value
                  )
                }
                className="h-9 rounded-lg border border-zinc-200 bg-white px-3 text-sm outline-none focus:border-zinc-400"
              />
            </label>
          </div>

          {/* Report content */}
          {status === 'loading' && (
            <div className="py-14 text-center text-sm text-zinc-500">
              กำลังโหลดรายงาน...
            </div>
          )}

          {status === 'error' && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-4">
              <p className="text-sm font-medium text-red-700">
                ไม่สามารถโหลดรายงานได้
              </p>

              <button
                type="button"
                onClick={loadReport}
                className="mt-3 rounded-lg bg-red-600 px-4 py-2 text-sm text-white transition hover:bg-red-700"
              >
                ลองใหม่
              </button>
            </div>
          )}

          {status === 'success' && (
            <pre className="overflow-auto rounded-lg border border-zinc-200 bg-zinc-50 p-4 text-xs text-zinc-700">
              {JSON.stringify(
                report,
                null,
                2
              )}
            </pre>
          )}
        </section>
      </main>
    </div>
  );
}

export default ReportPage;
