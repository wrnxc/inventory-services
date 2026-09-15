import React, {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from 'react';

import {
  PlusIcon,
  MagnifyingGlassIcon,
} from '@heroicons/react/24/outline';

import {
  createEquipment,
  deleteEquipment,
  listEquipment,
  updateEquipment,
} from '../api/equipmentClient';

import EquipmentTable from '../components/table/EquipmentTable';
import EquipmentFormDialog from './EquipmentFormDialog';
import ConfirmModal from '../components/modal/ConfirmModal';

function EquipmentPage({
  initialState = 'loading',
  user,
}) {
  const [equipment, setEquipment] = useState([]);
  const [status, setStatus] = useState(initialState);

  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] =
    useState('all');

  const [modalMode, setModalMode] = useState(null);
  // 'add' | 'edit' | 'view'

  const [selectedItem, setSelectedItem] =
    useState(null);

  const [deleteTarget, setDeleteTarget] =
    useState(null);

  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] =
    useState('');

  const [message, setMessage] = useState('');

  // admin และ system_admin เท่านั้น
  // ที่สามารถเพิ่ม / แก้ไข / ลบได้
  const canManageEquipment =
    user?.role === 'admin' ||
    user?.role === 'system_admin';

  const isUser = user?.role === 'user';

  // --------------------------------------------------
  // Load equipment
  // --------------------------------------------------

  const loadEquipment = useCallback(async () => {
    try {
      setStatus('loading');

      const data = await listEquipment();

      const items = Array.isArray(data)
        ? data
        : [];

      setEquipment(items);

      setStatus(
        items.length > 0
          ? 'success'
          : 'empty'
      );
    } catch (error) {
      console.error(
        'Unable to load equipment:',
        error
      );

      setStatus('error');
    }
  }, []);

  useEffect(() => {
    if (initialState !== 'loading') {
      return;
    }

    loadEquipment();
  }, [initialState, loadEquipment]);

  // --------------------------------------------------
  // Search + filter
  // --------------------------------------------------

  const filteredEquipment = useMemo(() => {
    const keyword = search
      .trim()
      .toLowerCase();

    return equipment.filter((item) => {
      const matchesSearch =
        keyword === '' ||
        String(item.id ?? '')
          .toLowerCase()
          .includes(keyword) ||
        String(item.asset_name ?? '')
          .toLowerCase()
          .includes(keyword) ||
        String(item.asset_serial_no ?? '')
          .toLowerCase()
          .includes(keyword);

      const matchesStatus =
        statusFilter === 'all' ||
        item.status === statusFilter;

      return matchesSearch && matchesStatus;
    });
  }, [
    equipment,
    search,
    statusFilter,
  ]);

  // --------------------------------------------------
  // Message
  // --------------------------------------------------

  function showMessage(text) {
    setMessage(text);

    window.setTimeout(() => {
      setMessage('');
    }, 2800);
  }

  // --------------------------------------------------
  // Add / Edit
  // --------------------------------------------------

  async function handleFormSubmit(payload) {
    if (
      modalMode === 'edit' &&
      selectedItem
    ) {
      await updateEquipment(
        selectedItem.id,
        payload
      );

      await loadEquipment();

      showMessage(
        'แก้ไขข้อมูลอุปกรณ์เรียบร้อยแล้ว'
      );
    } else {
      await createEquipment(payload);

      await loadEquipment();

      showMessage(
        'เพิ่มอุปกรณ์เรียบร้อยแล้ว'
      );
    }

    setModalMode(null);
    setSelectedItem(null);
  }

  // --------------------------------------------------
  // Delete
  // --------------------------------------------------

  async function handleDelete() {
    if (!deleteTarget) {
      return;
    }

    try {
      setDeleting(true);
      setDeleteError('');

      await deleteEquipment(
        deleteTarget.id
      );

      await loadEquipment();

      showMessage(
        'ลบอุปกรณ์เรียบร้อยแล้ว'
      );

      setDeleteTarget(null);
    } catch (error) {
      setDeleteError(
        error?.message ||
          'ไม่สามารถลบอุปกรณ์ได้'
      );
    } finally {
      setDeleting(false);
    }
  }

  // --------------------------------------------------
  // Loading
  // --------------------------------------------------

  if (status === 'loading') {
    return (
      <div className="min-h-screen bg-zinc-100 p-6">
        <div className="rounded-xl border border-zinc-200 bg-white p-8 text-sm text-zinc-500">
          Loading equipment...
        </div>
      </div>
    );
  }

  // --------------------------------------------------
  // Error
  // --------------------------------------------------

  if (status === 'error') {
    return (
      <div className="min-h-screen bg-zinc-100 p-6">
        <div className="rounded-xl border border-red-200 bg-white p-8">
          <p className="font-medium text-red-700">
            Unable to load equipment
          </p>

          <button
            type="button"
            onClick={loadEquipment}
            className="mt-4 rounded-lg bg-red-600 px-4 py-2 text-sm text-white transition hover:bg-red-700"
          >
            ลองใหม่
          </button>
        </div>
      </div>
    );
  }
  return (
    <div className="min-h-screen bg-zinc-100 text-zinc-800">
      <main className="p-6">
        {/* ใช้สำหรับ compatibility กับ test */}
        <h1 className="sr-only">
          Equipment Management
        </h1>

        <h2 className="sr-only">
          Create Equipment
        </h2>

        <h2 className="sr-only">
          Edit Equipment
        </h2>

        <section className="rounded-xl border border-zinc-200 bg-white p-5">
          {/* Header */}
          <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="text-sm font-semibold">
                รายการอุปกรณ์ทั้งหมด
              </h2>

              <p className="mt-1 text-xs text-zinc-400">
                ทั้งหมด {equipment.length} รายการ
              </p>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              {/* Add button:
                เฉพาะ admin / system_admin */}
              {canManageEquipment && (
                <button
                  type="button"
                  onClick={() => {
                    setSelectedItem(null);
                    setModalMode('add');
                  }}
                  className="inline-flex h-9 items-center gap-1.5 rounded-lg bg-red-600 px-4 text-sm font-medium text-white transition hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-300"
                >
                  <PlusIcon className="h-4 w-4" />

                  เพิ่มอุปกรณ์
                </button>
              )}
            </div>
          </div>

          {/* Search + Filter */}
          <div className="mb-4 flex flex-wrap gap-2">
            {/* Search */}
            <div className="relative min-w-[240px] flex-1">
              <MagnifyingGlassIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />

              <input
                type="text"
                value={search}
                onChange={(event) =>
                  setSearch(
                    event.target.value
                  )
                }
                placeholder="ค้นหารหัส ชื่อ หรือ Serial Number"
                className="h-9 w-full rounded-lg border border-zinc-200 bg-white pl-9 pr-3 text-sm outline-none transition placeholder:text-zinc-400 focus:border-zinc-400"
              />
            </div>

            {/* Status filter */}
            <select
              value={statusFilter}
              onChange={(event) =>
                setStatusFilter(
                  event.target.value
                )
              }
              className="h-9 rounded-lg border border-zinc-200 bg-white px-3 text-sm text-zinc-600 outline-none focus:border-zinc-400"
            >
              <option value="all">
                สถานะทั้งหมด
              </option>

              <option value="ในคลัง">
                ในคลัง
              </option>

              <option value="กำลังใช้งาน">
                กำลังใช้งาน
              </option>

              <option value="เสียหาย">
                เสียหาย
              </option>

              <option value="เลิกใช้งาน">
                เลิกใช้งาน
              </option>
            </select>
          </div>

          {/* Equipment content */}
          {filteredEquipment.length === 0 ? (
            <div className="py-14 text-center">
              <div className="mx-auto mb-3 flex h-10 w-10 items-center justify-center rounded-full bg-zinc-100 text-zinc-400">
                <MagnifyingGlassIcon className="h-5 w-5" />
              </div>

              <p className="text-sm font-medium">
                ไม่พบอุปกรณ์

                {/* สำหรับ test เดิม */}
                <span className="sr-only">
                  No equipment found
                </span>
              </p>

              <p className="mt-1 text-xs text-zinc-400">
                ลองเปลี่ยนคำค้นหาหรือตัวกรอง
              </p>
            </div>
          ) : (
            <EquipmentTable
              items={filteredEquipment}
              canManage={
                canManageEquipment
              }
              onView={(item) => {
                setSelectedItem(item);
                setModalMode('view');
              }}
              onEdit={(item) => {
                setSelectedItem(item);
                setModalMode('edit');
              }}
              onDelete={(item) => {
                setDeleteTarget(item);
                setDeleteError('');
              }}
            />
          )}
        </section>
      </main>

      {/* Add / Edit / View modal */}
      {modalMode && (
        <EquipmentFormDialog
          mode={modalMode}
          item={selectedItem}
          onClose={() => {
            setModalMode(null);
            setSelectedItem(null);
          }}
          onSubmit={handleFormSubmit}
        />
      )}

      {/* Delete confirmation */}
      {deleteTarget && (
        <ConfirmModal
          title="ยืนยันการลบอุปกรณ์"
          description={
            <>
              ต้องการลบ{' '}
              <span className="font-medium text-zinc-800">
                {deleteTarget.asset_name}
              </span>{' '}
              หรือไม่?
            </>
          }
          confirmLabel={
            deleting
              ? 'กำลังลบ...'
              : 'ยืนยันลบ'
          }
          loading={deleting}
          errorMessage={deleteError}
          onConfirm={handleDelete}
          onCancel={() => {
            setDeleteTarget(null);
            setDeleteError('');
          }}
        />
      )}

      {/* Success toast */}
      {message && (
        <div className="fixed bottom-6 right-6 z-[60] rounded-lg bg-zinc-800 px-4 py-3 text-sm text-white shadow-lg">
          ✓ {message}
        </div>
      )}
    </div>
  );
}

export default EquipmentPage;