import React, { useState } from 'react';
import { ArrowUpTrayIcon } from '@heroicons/react/24/outline';
import { importEquipment } from '../api/importClient';

function ImportPage({
  user,
  initialState = 'success',
}) {
  const canImport =
    user?.role === 'admin' ||
    user?.role === 'system_admin';

  const [file, setFile] = useState(null);
  const [status, setStatus] =
    useState(initialState);
  const [result, setResult] =
    useState(null);

  // --------------------------------------------------
  // Access guard
  // --------------------------------------------------

  if (!canImport) {
    return (
      <div className="min-h-screen bg-zinc-100 p-6">
        <div className="rounded-xl border border-zinc-200 bg-white p-8 text-sm text-zinc-500">
          หน้านี้สำหรับ Admin และ System Admin
        </div>
      </div>
    );
  }

  // --------------------------------------------------
  // Import
  // --------------------------------------------------

  async function handleSubmit(event) {
    event.preventDefault();

    if (!file) {
      return;
    }

    try {
      setStatus('loading');

      const data = await importEquipment(
        file
      );

      setResult(data);

      setStatus('success');
    } catch (error) {
      console.error(
        'Unable to import equipment:',
        error
      );

      setStatus('error');
    }
  }

  return (
    <div className="min-h-screen bg-zinc-100 text-zinc-800">
      <main className="p-6">
        <section className="rounded-xl border border-zinc-200 bg-white p-5">
          {/* Header */}
          <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="text-sm font-semibold">
                นำเข้าข้อมูลอุปกรณ์
              </h2>

              <p className="mt-1 text-xs text-zinc-400">
                รองรับไฟล์ .csv เท่านั้น
              </p>
            </div>
          </div>

          {/* Form */}
          <form
            onSubmit={handleSubmit}
            className="flex flex-wrap items-end gap-3"
          >
            <label className="flex min-w-[240px] flex-1 flex-col gap-1 text-xs text-zinc-500">
              ไฟล์ CSV
              <input
                id="csv-file"
                type="file"
                accept=".csv,text/csv"
                onChange={(event) =>
                  setFile(
                    event.target
                      .files?.[0] || null
                  )
                }
                className="h-9 w-full rounded-lg border border-zinc-200 bg-white px-3 text-sm text-zinc-600 outline-none file:mr-3 file:h-full file:rounded-md file:border-0 file:bg-zinc-100 file:px-3 file:text-xs file:font-medium file:text-zinc-600 focus:border-zinc-400"
              />
            </label>

            <button
              type="submit"
              disabled={
                !file ||
                status === 'loading'
              }
              className="inline-flex h-9 items-center gap-1.5 rounded-lg bg-red-600 px-4 text-sm font-medium text-white transition hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-300 disabled:opacity-50"
            >
              {status === 'loading'
                ? 'กำลังนำเข้า...'
                : 'นำเข้า'}
            </button>
          </form>

          {/* Import result */}
          {status === 'error' && (
            <div className="mt-4 rounded-lg border border-red-200 bg-red-50 p-4">
              <p className="text-sm font-medium text-red-700">
                ไม่สามารถนำเข้าข้อมูลได้
              </p>
            </div>
          )}

          {result && (
            <pre className="mt-4 overflow-auto rounded-lg border border-zinc-200 bg-zinc-50 p-4 text-xs text-zinc-700">
              {JSON.stringify(
                result,
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

export default ImportPage;
