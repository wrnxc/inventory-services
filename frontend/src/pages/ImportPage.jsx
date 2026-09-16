import React, {
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react';

import { useNavigate } from 'react-router-dom';

import {
  ArrowUpTrayIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline';

import {
  getImportHistory,
  importEquipment,
  previewImportEquipment,
} from '../api/importClient';

import ConfirmModal from '../components/modal/ConfirmModal';
import ResponseModal from '../components/modal/ResponseModal';

import ImportPreviewTable from '../components/table/ImportPreviewTable';
import ImportHistoryTable from '../components/table/ImportHistoryTable';

function ImportPage({
  user,
  initialState = 'success',
}) {
  const navigate = useNavigate();

  const fileInputRef = useRef(null);

  const canImport =
    user?.role === 'admin' ||
    user?.role === 'system_admin';

  const [file, setFile] =
    useState(null);

  const [status, setStatus] =
    useState(initialState);

  const [result, setResult] =
    useState(null);

  const [preview, setPreview] =
    useState(null);

  const [previewLoading, setPreviewLoading] =
    useState(false);

  const [history, setHistory] =
    useState([]);

  const [
    historyLoading,
    setHistoryLoading,
  ] = useState(true);

  const [
    confirmImport,
    setConfirmImport,
  ] = useState(false);

  const [
    responseModal,
    setResponseModal,
  ] = useState(null);

  const loadHistory = useCallback(
    async () => {
      if (!canImport) {
        setHistoryLoading(false);
        return;
      }

      try {
        setHistoryLoading(true);

        const data =
          await getImportHistory();

        setHistory(
          Array.isArray(data)
            ? data
            : []
        );
      } catch (error) {
        console.error(
          'Unable to load import history:',
          error
        );
      } finally {
        setHistoryLoading(false);
      }
    },
    [canImport]
  );

  useEffect(() => {
    loadHistory();
  }, [loadHistory]);

  function handleFileChange(event) {
    const selectedFile = event.target.files?.[0] || null;
    setFile(selectedFile);
    setPreview(null);
    setResult(null);
    setStatus('success');
    setResponseModal(null);
  }

  function clearSelectedFile() {
    setFile(null);
    setPreview(null);
    setResult(null);
    setStatus('success');
    setConfirmImport(false);
    setResponseModal(null);

    if (fileInputRef.current) {
      fileInputRef.current.value =
        '';
    }
  }

  async function handleSubmit(event) {
    event.preventDefault();
    if (!file || previewLoading) return;
    try {
      setPreviewLoading(true); setPreview(null); setResult(null); setResponseModal(null);
      const data = await previewImportEquipment(file);
      setPreview(data);
    } catch (error) {
      console.error('Unable to preview import:', error);
      setPreview(null);
      setResponseModal({ type: 'error', title: 'ไม่สามารถอ่านไฟล์ได้', description: error.message || 'ไม่สามารถอ่านตัวอย่างข้อมูลจากไฟล์ได้' });
    } finally { setPreviewLoading(false); }
  }

  async function handleConfirmImport() {
    if (!file) {
      setConfirmImport(false);
      return;
    }

    try {
      setConfirmImport(false);

      setStatus('loading');
      setResult(null);
      setResponseModal(null);

      const data =
        await importEquipment(file);

      setResult(data);
      setStatus('success');

      // Import สำเร็จแล้ว ซ่อน Preview และปุ่มยืนยัน
      setPreview(null);

      await loadHistory();
    } catch (error) {
      console.error(
        'Unable to import equipment:',
        error
      );

      if (
        error.code ===
        'DUPLICATE_IMPORT'
      ) {
        setResponseModal({
          type: 'warning',
          title:
            'ไม่สามารถนำเข้าข้อมูลได้',
          description:
            'ไม่สามารถนำเข้ารายการซ้ำได้ เนื่องจากข้อมูลทั้งหมดในไฟล์มีอยู่ในระบบแล้ว',
        });
      } else {
        setResponseModal({
          type: 'error',
          title:
            'ไม่สามารถนำเข้าข้อมูลได้',
          description:
            error.message ||
            'เกิดข้อผิดพลาดในการนำเข้าข้อมูล',
        });
      }

      setStatus('error');

      await loadHistory();
    }
  }

  function resetImport() {
    clearSelectedFile();
  }

  if (!canImport) {
    return (
      <div className="min-h-screen bg-zinc-100 p-6">
        <div className="rounded-xl border border-zinc-200 bg-white p-8 text-sm text-zinc-500">
          หน้านี้สำหรับ Admin และ
          System Admin
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-zinc-100 text-zinc-800">
      <main className="p-6">

        {/* ================= Import ================= */}

        <section className="rounded-xl border border-zinc-200 bg-white p-5">

          <div className="mb-4">
            <h2 className="text-sm font-semibold">
              นำเข้าข้อมูลอุปกรณ์
            </h2>

            <p className="mt-1 text-xs text-zinc-400">
              รองรับไฟล์ .csv และ .xlsx
            </p>
          </div>

          {!result && (
            <form onSubmit={handleSubmit}>
            <div className="flex flex-wrap items-end gap-3">
              <label className="flex min-w-[240px] flex-1 flex-col gap-1 text-xs text-zinc-500">
                ไฟล์ CSV / XLSX
                <div className="relative">
                  <input ref={fileInputRef} type="file" accept=".csv,.xlsx,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" onChange={handleFileChange} disabled={status === 'loading'} className="h-9 w-full rounded-lg border border-zinc-200 bg-white px-3 pr-10 text-sm disabled:cursor-not-allowed disabled:bg-zinc-50 disabled:opacity-70" />
                  {file && (
                    <button type="button" title="ยกเลิกไฟล์" aria-label="ยกเลิกไฟล์ที่เลือก" onClick={clearSelectedFile} disabled={status === 'loading'} className="absolute right-2 top-1/2 flex h-6 w-6 -translate-y-1/2 items-center justify-center rounded-md bg-white text-zinc-400 transition-colors hover:bg-red-50 hover:text-red-600 disabled:opacity-50">
                      <XMarkIcon className="h-4 w-4" />
                    </button>
                  )}
                </div>
              </label>
              <button type="submit" disabled={!file || previewLoading || Boolean(preview) || status === 'loading'} className="inline-flex h-9 items-center gap-2 rounded-lg bg-red-600 px-4 text-sm font-medium text-white transition-colors hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50">
                <ArrowUpTrayIcon className="h-4 w-4" />
                {previewLoading ? 'กำลังอ่านไฟล์...' : 'นำเข้าไฟล์'}
              </button>
            </div>

            {previewLoading && (
              <div className="mt-5 rounded-lg border border-zinc-100 bg-zinc-50 px-4 py-6 text-center"><p className="text-sm text-zinc-400">กำลังอ่านตัวอย่างข้อมูล...</p></div>
            )}

            {!previewLoading && preview && (
              <>
                <ImportPreviewTable items={preview.preview || []} total={preview.total || 0} />
                <div className="mt-4 flex justify-end gap-2">
                  <button type="button" onClick={clearSelectedFile} className="h-9 rounded-lg border border-zinc-200 bg-white px-4 text-sm text-zinc-700 transition-colors hover:bg-zinc-50">ยกเลิก</button>
                  <button type="button" onClick={() => setConfirmImport(true)} className="h-9 rounded-lg bg-red-600 px-4 text-sm font-medium text-white transition-colors hover:bg-red-700">ยืนยันการนำเข้า</button>
                </div>
              </>
            )}
            </form>
          )}

          {/* ================= Import Result ================= */}

        {result &&
          status === 'success' && (
            <div className="mt-5 rounded-xl border border-emerald-200 bg-emerald-50 p-5">

              <p className="text-sm font-semibold text-emerald-700">
                ✓ นำเข้าข้อมูลเสร็จสิ้น
              </p>

              <div className="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-4">

                <div className="rounded-lg bg-white/70 px-4 py-3">
                  <p className="text-xs text-zinc-400">
                    ทั้งหมด
                  </p>

                  <p className="mt-1 text-lg font-semibold text-zinc-700">
                    {result.total ?? 0}
                  </p>
                </div>

                <div className="rounded-lg bg-white/70 px-4 py-3">
                  <p className="text-xs text-zinc-400">
                    สำเร็จ
                  </p>

                  <p className="mt-1 text-lg font-semibold text-emerald-600">
                    {result.success ?? 0}
                  </p>
                </div>

                <div className="rounded-lg bg-white/70 px-4 py-3">
                  <p className="text-xs text-zinc-400">
                    ข้อมูลซ้ำ
                  </p>

                  <p className="mt-1 text-lg font-semibold text-amber-600">
                    {result.duplicate ?? 0}
                  </p>
                </div>

                <div className="rounded-lg bg-white/70 px-4 py-3">
                  <p className="text-xs text-zinc-400">
                    ไม่สำเร็จ
                  </p>

                  <p className="mt-1 text-lg font-semibold text-red-600">
                    {result.failed ?? 0}
                  </p>
                </div>
              </div>

              <div className="mt-4 flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={
                    resetImport
                  }
                  className="h-9 rounded-lg border border-zinc-200 bg-white px-4 text-sm text-zinc-700 transition-colors hover:bg-zinc-50"
                >
                  นำเข้าไฟล์อื่น
                </button>

                <button
                  type="button"
                  onClick={() =>
                    navigate(
                      '/equipment'
                    )
                  }
                  className="h-9 rounded-lg bg-red-600 px-4 text-sm font-medium text-white transition-colors hover:bg-red-700"
                >
                  ไปที่รายการอุปกรณ์
                </button>
              </div>
            </div>
          )}
      </section>

      {/* ================= History ================= */}

      <section className="mt-6 rounded-xl border border-zinc-200 bg-white p-5">

        <div className="mb-4">
          <h2 className="text-sm font-semibold">
            ประวัติการนำเข้าข้อมูล
          </h2>

          <p className="mt-1 text-xs text-zinc-400">
            รายการไฟล์ที่เคยนำเข้าสู่ระบบ
          </p>
        </div>

        <ImportHistoryTable
          items={history}
          loading={
            historyLoading
          }
        />
      </section>
    </main>

      {/* ================= Confirm Import ================= */ }

  {
    confirmImport && (
      <ConfirmModal
        title="ยืนยันการนำเข้าข้อมูล"
        description={
          <div>
            <p>
              คุณต้องการนำเข้าข้อมูลจากไฟล์
            </p>

            <p className="mt-1 font-medium text-zinc-700">
              {file?.name}
            </p>

            {preview && (
              <p className="mt-2">
                จำนวนทั้งหมด{' '}
                <span className="font-medium text-zinc-700">
                  {preview.total?.toLocaleString()}
                </span>{' '}
                รายการ
              </p>
            )}
          </div>
        }
        confirmLabel="ยืนยัน"
        cancelLabel="ยกเลิก"
        loading={false}
        onConfirm={
          handleConfirmImport
        }
        onCancel={() =>
          setConfirmImport(false)
        }
      />
    )
  }

  {
    responseModal && (
      <ResponseModal
        type={
          responseModal.type
        }
        title={
          responseModal.title
        }
        description={
          responseModal.description
        }
        confirmLabel="ตกลง"
        onClose={() =>
          setResponseModal(null)
        }
      />
    )
  }
    </div>
  );
}

export default ImportPage;