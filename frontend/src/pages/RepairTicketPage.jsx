import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  ArchiveBoxIcon,
  ArrowPathIcon,
  ChevronUpDownIcon,
  ClipboardDocumentListIcon,
  ExclamationTriangleIcon,
  MagnifyingGlassIcon,
  PlusIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline';
import { CheckCircleIcon } from '@heroicons/react/24/solid';

import {
  createRepairTicket,
  listRepairTickets,
  updateRepairTicketStatus,
} from '../api/repairClient';

// listEquipment() has no server-side lookup-by-id — it returns the full
// equipment table, which we index by id below so each ticket row can show
// the equipment's product name / asset tag instead of a bare equipment_id.
// Same pattern already used on the borrow page.
import { listEquipment } from '../api/equipmentClient';

const DEFAULT_FORM = {
  equipment_id: '',
  issue_description: '',
  urgency: 'normal',
};

const MAX_VISIBLE_RESULTS = 20;

// Maps each backend status to a Tailwind badge style + Thai label.
const STATUS_META = {
  'ส่งซ่อม': { label: 'รอตรวจสอบ', badge: 'bg-red-50 text-red-700' },
  'กำลังซ่อม': { label: 'กำลังซ่อม', badge: 'bg-amber-50 text-amber-700' },
  'ซ่อมเสร็จ': { label: 'ซ่อมเสร็จ', badge: 'bg-zinc-100 text-zinc-600' },
  'ซ่อมไม่ได้': { label: 'ซ่อมไม่ได้', badge: 'bg-red-50 text-red-700' },
};

const NEXT_STATUS = {
  'ส่งซ่อม': { next: 'กำลังซ่อม', actionLabel: 'เริ่มตรวจสอบ' },
  'กำลังซ่อม': { next: 'ซ่อมเสร็จ', actionLabel: 'ปิดงาน' },
};

function statusMeta(status) {
  return STATUS_META[status] || { label: status, badge: 'bg-zinc-100 text-zinc-600' };
}

// Buddhist-era date, same helper as the borrow page uses.
function formatThaiDate(dateStr) {
  if (!dateStr) return null;
  const d = new Date(dateStr);
  if (Number.isNaN(d.getTime())) return null;
  const day = String(d.getDate()).padStart(2, '0');
  const month = String(d.getMonth() + 1).padStart(2, '0');
  return `${day}/${month}/${d.getFullYear() + 543}`;
}

const inputClass = (hasError) =>
  `h-10 w-full rounded-lg border px-3 text-sm outline-none transition focus:ring-2 ${
    hasError
      ? 'border-red-300 focus:border-red-500 focus:ring-red-100'
      : 'border-zinc-200 focus:border-red-500 focus:ring-red-100'
  }`;

// Searchable equipment picker — same pattern as the one on the borrow
// page. Repair reports can target equipment in any status (something
// already "กำลังใช้งาน" can still break), so unlike the borrow picker this
// one does NOT filter the list down to "ในคลัง" only.
function EquipmentPicker({ value, onChange, disabled, equipmentOptions, loadingOptions, optionsError, hasError }) {
  const [query, setQuery] = useState('');
  const [selected, setSelected] = useState(null);
  const [open, setOpen] = useState(false);
  const containerRef = useRef(null);
  const inputRef = useRef(null);

  useEffect(() => {
    function handleClickOutside(event) {
      if (containerRef.current && !containerRef.current.contains(event.target)) {
        setOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // If the parent clears `value` (e.g. modal reset), clear the local
  // selection label too so stale text can't linger in the picker.
  useEffect(() => {
    if (!value) setSelected(null);
  }, [value]);

  const matches = useMemo(() => {
    if (!query.trim()) return equipmentOptions;
    const q = query.trim().toLowerCase();
    return equipmentOptions.filter(
      (item) =>
        (item.product_name || '').toLowerCase().includes(q) ||
        (item.asset_name || '').toLowerCase().includes(q) ||
        (item.asset_tag || '').toLowerCase().includes(q),
    );
  }, [equipmentOptions, query]);

  const visibleMatches = matches.slice(0, MAX_VISIBLE_RESULTS);
  const hiddenCount = matches.length - visibleMatches.length;

  function handleSelect(item) {
    onChange(item.id);
    setSelected(item);
    setQuery('');
    setOpen(false);
  }

  function handleClear() {
    onChange('');
    setSelected(null);
    setQuery('');
    requestAnimationFrame(() => inputRef.current?.focus());
  }

  if (value && selected) {
    return (
      <div className="flex items-center justify-between gap-2 rounded-lg border border-zinc-200 bg-zinc-50 px-3 py-2">
        <div className="flex min-w-0 items-center gap-2.5">
          <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-md bg-white text-emerald-600 ring-1 ring-emerald-100">
            <CheckCircleIcon className="h-5 w-5" />
          </div>
          <div className="min-w-0">
            <p className="truncate text-sm font-medium text-zinc-800">{selected.product_name}</p>
            <p className="truncate text-xs text-zinc-500">
              {selected.asset_name}
              {selected.asset_tag ? ` · ${selected.asset_tag}` : ''}
            </p>
          </div>
        </div>
        <button
          type="button"
          onClick={handleClear}
          disabled={disabled}
          className="flex-shrink-0 rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-200/70 hover:text-zinc-600 disabled:opacity-50"
          aria-label="เปลี่ยนอุปกรณ์ที่เลือก"
        >
          <XMarkIcon className="h-4 w-4" />
        </button>
      </div>
    );
  }

  return (
    <div ref={containerRef} className="relative">
      <div className="relative">
        <MagnifyingGlassIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
        <input
          ref={inputRef}
          type="text"
          value={query}
          onFocus={() => setOpen(true)}
          onChange={(event) => {
            setQuery(event.target.value);
            setOpen(true);
          }}
          placeholder="พิมพ์ชื่อรุ่นหรือรหัสอุปกรณ์"
          disabled={disabled || loadingOptions}
          className={`${inputClass(hasError)} pl-9 pr-9`}
        />
        {query ? (
          <button
            type="button"
            onClick={() => setQuery('')}
            className="absolute right-2.5 top-1/2 -translate-y-1/2 rounded p-0.5 text-zinc-400 hover:text-zinc-600"
            aria-label="ล้างคำค้นหา"
          >
            <XMarkIcon className="h-4 w-4" />
          </button>
        ) : (
          <ChevronUpDownIcon className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
        )}

        {open ? (
          <div className="absolute z-10 mt-1 max-h-64 w-full overflow-auto rounded-lg border border-zinc-200 bg-white py-1 shadow-lg">
            {loadingOptions ? (
              <div className="flex items-center gap-2 px-3 py-2.5 text-xs text-zinc-400">
                <ArrowPathIcon className="h-3.5 w-3.5 animate-spin" />
                กำลังโหลดรายการอุปกรณ์...
              </div>
            ) : optionsError ? (
              <p className="px-3 py-2.5 text-xs text-red-600">{optionsError}</p>
            ) : visibleMatches.length === 0 ? (
              <div className="px-3 py-5 text-center">
                <ArchiveBoxIcon className="mx-auto mb-1.5 h-6 w-6 text-zinc-300" />
                <p className="text-xs text-zinc-400">
                  {query ? 'ไม่พบอุปกรณ์ที่ตรงกับคำค้นหา' : 'ไม่มีอุปกรณ์ในระบบ'}
                </p>
              </div>
            ) : (
              <>
                {visibleMatches.map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => handleSelect(item)}
                    className="flex w-full items-center gap-2.5 px-3 py-2 text-left text-sm hover:bg-zinc-50"
                  >
                    <ArchiveBoxIcon className="h-4 w-4 flex-shrink-0 text-zinc-400" />
                    <span className="min-w-0">
                      <span className="block truncate font-medium text-zinc-800">{item.product_name}</span>
                      <span className="block truncate text-xs text-zinc-500">
                        {item.asset_name}
                        {item.asset_tag ? ` · ${item.asset_tag}` : ''}
                      </span>
                    </span>
                  </button>
                ))}
                {hiddenCount > 0 ? (
                  <p className="px-3 py-1.5 text-xs text-zinc-400">
                    พิมพ์เพื่อค้นหาให้ตรงมากขึ้น (พบอีก {hiddenCount} รายการ)
                  </p>
                ) : null}
              </>
            )}
          </div>
        ) : null}
      </div>
    </div>
  );
}

function RepairTicketPage({ initialState = 'loading', user }) {
  const canManageRepair = user?.role === 'admin' || user?.role === 'system_admin';
  const isUser = user?.role === 'user';

  const [tickets, setTickets] = useState([]);
  const [status, setStatus] = useState(initialState);
  const [form, setForm] = useState(DEFAULT_FORM);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [formErrors, setFormErrors] = useState({});

  const [equipmentOptions, setEquipmentOptions] = useState([]);
  const [loadingEquipment, setLoadingEquipment] = useState(true);
  const [equipmentError, setEquipmentError] = useState('');

  const loadTickets = async () => {
    try {
      setLoading(true);
      setError('');
      const data = await listRepairTickets();
      const items = Array.isArray(data) ? data : [];
      setTickets(items);
      setStatus(items.length > 0 ? 'success' : 'empty');
    } catch (loadError) {
      setError(loadError?.message || 'Unable to load repair tickets');
      setStatus('error');
    } finally {
      setLoading(false);
    }
  };

  const loadEquipmentOptions = async () => {
    try {
      setLoadingEquipment(true);
      setEquipmentError('');
      const data = await listEquipment();
      setEquipmentOptions(Array.isArray(data) ? data : []);
    } catch (loadError) {
      // Non-fatal for the ticket list — rows just fall back to showing the
      // bare equipment_id — but the picker inside the create modal does
      // need to know a load failed, so it's still surfaced there.
      setEquipmentError(loadError?.message || 'ไม่สามารถโหลดรายการอุปกรณ์ได้');
    } finally {
      setLoadingEquipment(false);
    }
  };

  useEffect(() => {
    if (initialState !== 'loading') {
      return;
    }

    loadTickets();
    loadEquipmentOptions();
  }, [initialState]);

  const equipmentById = useMemo(() => {
    const map = new Map();
    equipmentOptions.forEach((eq) => map.set(eq.id, eq));
    return map;
  }, [equipmentOptions]);

  const visibleTickets = useMemo(() => tickets, [tickets]);

  function openCreateModal() {
    setForm(DEFAULT_FORM);
    setFormErrors({});
    setShowCreateModal(true);
  }

  function closeCreateModal() {
    if (submitting) return;
    setShowCreateModal(false);
    setForm(DEFAULT_FORM);
    setFormErrors({});
  }

  async function handleCreate(event) {
    event.preventDefault();

    if (!isUser) {
      return;
    }

    const nextErrors = {};
    if (!form.equipment_id) {
      nextErrors.equipment_id = 'กรุณาเลือกอุปกรณ์ที่ต้องการแจ้งซ่อม';
    }
    if (!form.issue_description.trim()) {
      nextErrors.issue_description = 'กรุณาอธิบายอาการที่พบ';
    }
    if (Object.keys(nextErrors).length > 0) {
      setFormErrors(nextErrors);
      return;
    }
    setFormErrors({});

    try {
      setSubmitting(true);
      setError('');
      await createRepairTicket({
        equipment_id: Number(form.equipment_id),
        issue_description: form.issue_description,
        urgency: form.urgency,
      });
      closeCreateModal();
      await loadTickets();
    } catch (submitError) {
      setError(submitError?.message || 'Unable to create repair ticket');
    } finally {
      setSubmitting(false);
    }
  }

  async function handleStatusUpdate(id, nextStatus) {
    if (!canManageRepair) {
      return;
    }

    try {
      setError('');
      await updateRepairTicketStatus(id, nextStatus);
      await loadTickets();
    } catch (updateError) {
      setError(updateError?.message || 'Unable to update repair ticket status');
    }
  }

  if (status === 'loading') {
    return (
      <div className="min-h-screen bg-zinc-100 p-6">
        <div className="rounded-xl border border-zinc-200 bg-white p-8 text-sm text-zinc-500">
          กำลังโหลดข้อมูล...
        </div>
      </div>
    );
  }

  if (status === 'error') {
    return (
      <div className="min-h-screen bg-zinc-100 p-6">
        <div className="rounded-xl border border-zinc-200 bg-white p-8">
          <p className="font-medium text-red-700">ไม่สามารถโหลดรายการแจ้งซ่อมได้</p>
          <button
            type="button"
            onClick={loadTickets}
            className="mt-4 h-9 rounded-lg bg-red-600 px-4 text-sm font-medium text-white hover:bg-red-700"
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
        <section className="rounded-xl border border-zinc-200 bg-white p-5">
          <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="text-sm font-semibold text-zinc-800">รายการแจ้งซ่อม</h2>
              <p className="mt-1 text-xs text-zinc-400">ทั้งหมด {visibleTickets.length} รายการ</p>
            </div>

            {isUser ? (
              <button
                type="button"
                onClick={openCreateModal}
                className="inline-flex h-9 items-center gap-1.5 rounded-lg bg-red-600 px-4 text-sm font-medium text-white transition hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-300"
              >
                <PlusIcon className="h-4 w-4" />
                แจ้งซ่อม
              </button>
            ) : null}
          </div>

          {error ? (
            <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
              {error}
            </div>
          ) : null}

          {loading ? (
            <p className="py-6 text-sm text-zinc-500">กำลังโหลดข้อมูล...</p>
          ) : visibleTickets.length === 0 ? (
            <div className="py-9 text-center">
              <ClipboardDocumentListIcon className="mx-auto mb-2.5 h-8 w-8 text-zinc-300" />
              <p className="text-[13.5px] font-medium text-zinc-700">ไม่มีใบแจ้งซ่อม</p>
              <span className="text-xs text-zinc-400">รายการแจ้งซ่อมของคุณจะแสดงที่นี่</span>
            </div>
          ) : (
            <div className="space-y-2">
              {visibleTickets.map((ticket) => {
                const meta = statusMeta(ticket.status);
                const transition = NEXT_STATUS[ticket.status];
                const isPending = ticket.status === 'ส่งซ่อม';
                const isDone = ticket.status === 'ซ่อมเสร็จ' || ticket.status === 'ซ่อมไม่ได้';

                const equipment = equipmentById.get(ticket.equipment_id);
                const title = equipment
                  ? `#${ticket.id} · ${equipment.product_name}${equipment.asset_tag ? ` · ${equipment.asset_tag}` : ''}`
                  : `#${ticket.id} · อุปกรณ์รหัส ${ticket.equipment_id}`;

                // NOTE: reporter name / reported date field names aren't
                // confirmed against the real ticket schema yet — trying the
                // common candidates and falling back gracefully if absent.
                // Tell me the real field names and I'll lock these down.
                const reporter = ticket.reported_by || ticket.reporter_name || ticket.created_by_name;
                const reportedOn = formatThaiDate(ticket.created_at || ticket.reported_at);
                const metaParts = [
                  reporter ? `แจ้งโดย: ${reporter}` : null,
                  reportedOn,
                  ticket.issue_description || ticket.issue,
                ].filter(Boolean);

                return (
                  <div
                    key={ticket.id}
                    className={[
                      'flex flex-wrap items-center justify-between gap-3 rounded-lg border px-3.5 py-3 transition',
                      isPending ? 'border-red-300 bg-red-50' : 'border-zinc-200 hover:border-zinc-300',
                      isDone ? 'opacity-60' : '',
                    ].join(' ')}
                  >
                    <div>
                      <p className={['text-[13px] font-medium', isPending ? 'text-red-700' : 'text-zinc-800'].join(' ')}>
                        {title}
                      </p>
                      <p className={['mt-0.5 text-xs', isPending ? 'text-red-600' : 'text-zinc-500'].join(' ')}>
                        {metaParts.length > 0 ? metaParts.join(' · ') : '—'}
                      </p>
                    </div>

                    <div className="flex flex-wrap items-center gap-2">
                      <span className={`inline-flex rounded-full px-2.5 py-1 text-xs font-medium ${meta.badge}`}>
                        {meta.label}
                      </span>

                      {canManageRepair && transition ? (
                        <button
                          type="button"
                          onClick={() => handleStatusUpdate(ticket.id, transition.next)}
                          className="h-[30px] rounded-lg border border-zinc-200 bg-white px-3 text-xs font-medium text-zinc-700 transition hover:bg-zinc-50"
                        >
                          {transition.actionLabel}
                        </button>
                      ) : null}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </section>
      </main>

      {showCreateModal && isUser ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-5">
          <div className="w-[420px] max-w-full rounded-xl bg-white p-6">
            <h3 className="text-base font-semibold text-zinc-800">แจ้งซ่อมอุปกรณ์</h3>
            <p className="mb-4 mt-1 text-xs text-zinc-400">
              helpdesk/service จะเข้ามาตรวจสอบและอัปเดตสถานะให้
            </p>

            <form onSubmit={handleCreate}>
              <div className="mb-3">
                <label className="mb-1.5 flex items-center gap-1.5 text-xs font-medium text-zinc-600">
                  อุปกรณ์ที่ต้องการแจ้งซ่อม
                  <span className="text-red-600">*</span>
                </label>
                <EquipmentPicker
                  value={form.equipment_id}
                  onChange={(id) => {
                    setForm({ ...form, equipment_id: id });
                    if (formErrors.equipment_id) setFormErrors({ ...formErrors, equipment_id: undefined });
                  }}
                  disabled={submitting}
                  equipmentOptions={equipmentOptions}
                  loadingOptions={loadingEquipment}
                  optionsError={equipmentError}
                  hasError={Boolean(formErrors.equipment_id)}
                />
                {formErrors.equipment_id ? (
                  <p className="mt-1 flex items-center gap-1 text-[11.5px] text-red-600">
                    <ExclamationTriangleIcon className="h-3 w-3 flex-shrink-0" />
                    {formErrors.equipment_id}
                  </p>
                ) : null}
              </div>

              <div className="mb-3">
                <label className="mb-1 block text-xs text-zinc-500">
                  อาการที่พบ <span className="text-red-600">*</span>
                </label>
                <textarea
                  value={form.issue_description}
                  onChange={(event) => {
                    setForm({ ...form, issue_description: event.target.value });
                    if (formErrors.issue_description) setFormErrors({ ...formErrors, issue_description: undefined });
                  }}
                  placeholder="อธิบายอาการ เช่น เปิดไม่ติด, จอไม่แสดงผล"
                  rows={4}
                  className={[
                    'w-full rounded-lg border px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-red-100',
                    formErrors.issue_description ? 'border-red-400' : 'border-zinc-200 focus:border-red-500',
                  ].join(' ')}
                  required
                />
                {formErrors.issue_description ? (
                  <span className="mt-1 block text-xs text-red-600">{formErrors.issue_description}</span>
                ) : null}
              </div>

              <div className="mb-1">
                <label className="mb-1 block text-xs text-zinc-500">ความเร่งด่วน</label>
                <select
                  value={form.urgency}
                  onChange={(event) => setForm({ ...form, urgency: event.target.value })}
                  className="h-10 w-full rounded-lg border border-zinc-200 px-3 text-sm outline-none focus:border-red-500 focus:ring-2 focus:ring-red-100"
                >
                  <option value="normal">ปกติ</option>
                  <option value="high">เร่งด่วน</option>
                  <option value="critical">วิกฤต</option>
                </select>
              </div>

              <div className="mt-5 flex justify-end gap-2">
                <button
                  type="button"
                  onClick={closeCreateModal}
                  disabled={submitting}
                  className="h-9 rounded-lg border border-zinc-200 bg-white px-4 text-sm hover:bg-zinc-50 disabled:opacity-60"
                >
                  ยกเลิก
                </button>
                <button
                  type="submit"
                  disabled={submitting}
                  className="inline-flex h-9 items-center gap-1.5 rounded-lg bg-red-600 px-4 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-60"
                >
                  {submitting ? (
                    <>
                      <span className="inline-block h-3 w-3 animate-spin rounded-full border-2 border-white/50 border-t-white" />
                      กำลังบันทึก...
                    </>
                  ) : (
                    'ส่งเรื่องแจ้งซ่อม'
                  )}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </div>
  );
}

export default RepairTicketPage;
