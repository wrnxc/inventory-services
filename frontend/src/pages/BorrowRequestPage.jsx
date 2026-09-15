import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
    ArchiveBoxIcon,
    ArrowPathIcon,
    BuildingOfficeIcon,
    CalendarDaysIcon,
    ChatBubbleLeftEllipsisIcon,
    ChevronUpDownIcon,
    ClipboardDocumentListIcon,
    ExclamationTriangleIcon,
    HashtagIcon,
    IdentificationIcon,
    MagnifyingGlassIcon,
    PlusIcon,
    TagIcon,
    UserIcon,
    XMarkIcon,
} from '@heroicons/react/24/outline';
import { CheckCircleIcon } from '@heroicons/react/24/solid';

import {
    approveBorrowRequest,
    confirmReturnBorrowRequest,
    createBorrowRequest,
    listBorrowRequests,
    rejectBorrowRequest,
    returnBorrowRequest,
} from '../api/borrowClient';

// listEquipment() has no search/filter/limit params — it always returns
// the full equipment table. That's fine for now (filtered client-side
// below), but it's the ceiling on how large the catalog can get before
// this needs to move to server-side search. See the comment on
// EquipmentPicker for what that upgrade looks like.
import { listEquipment } from '../api/equipmentClient';

function todayIsoDate() {
    return new Date().toISOString().slice(0, 10);
}

// Mirrors the paper form's "Part#1: User" fields. `request_date` defaults
// to today (see openModal) but stays editable, matching the paper log
// where the date is filled in by hand and can be backdated.
const DEFAULT_FORM = {
    request_date: '',
    it_request_no: '',
    borrower_name: '',
    emp_id: '',
    department: '',
    equipment_id: '',
    borrow_type: 'เบิกชั่วคราว',
    return_date: '',
    reason: '',
};

// Friendly badge label + color per raw DB status. `อนุมัติ` and
// `รอตรวจรับคืน` both read as "กำลังเบิกอยู่" to the user (the item is out
// with them either way) — `เกินกำหนดคืน` stays visually distinct since it
// needs attention. Adjust the grouping here if the mockup's wording should
// map 1:1 to statuses instead.
function getStatusDisplay(status) {
    switch (status) {
        case 'รออนุมัติ':
            return { label: 'รออนุมัติ', className: 'bg-amber-50 text-amber-700' };
        case 'อนุมัติ':
            return { label: 'กำลังเบิกอยู่', className: 'bg-zinc-100 text-zinc-600' };
        case 'รอตรวจรับคืน':
            return { label: 'รอรับคืน', className: 'bg-amber-50 text-amber-700' };
        case 'เกินกำหนดคืน':
            return { label: 'เกินกำหนดคืน', className: 'bg-red-50 text-red-700' };
        case 'คืนแล้ว':
            return { label: 'คืนแล้ว', className: 'bg-zinc-100 text-zinc-500' };
        case 'ไม่อนุมัติ':
            return { label: 'ไม่อนุมัติ', className: 'bg-zinc-100 text-zinc-500' };
        default:
            return { label: status || '-', className: 'bg-zinc-100 text-zinc-600' };
    }
}

const TABS = [
    {
        key: 'in_process',
        label: 'กำลังดำเนินการ',
        statuses: ['รออนุมัติ', 'อนุมัติ', 'เกินกำหนดคืน', 'รอตรวจรับคืน'],
    },
    { key: 'pending', label: 'รออนุมัติ', statuses: ['รออนุมัติ'] },
    { key: 'active', label: 'กำลังเบิกอยู่', statuses: ['อนุมัติ', 'เกินกำหนดคืน'] },
    { key: 'awaiting_return', label: 'รอรับคืน', statuses: ['รอตรวจรับคืน'] },
    { key: 'history', label: 'ประวัติ', statuses: ['คืนแล้ว', 'ไม่อนุมัติ'] },
];

const MAX_VISIBLE_RESULTS = 20;

// Formats an ISO date string as a Buddhist-era date. `withYear=false` gives
// a short "D/MM" form (no leading zero on the day) used for the start of a
// borrowed date range, matching the mockup's "8/07 - 15/07/2569" style.
function formatThaiDate(dateStr, withYear = true) {
    if (!dateStr) return null;
    const d = new Date(dateStr);
    if (Number.isNaN(d.getTime())) return null;
    const month = String(d.getMonth() + 1).padStart(2, '0');
    if (!withYear) return `${d.getDate()}/${month}`;
    const day = String(d.getDate()).padStart(2, '0');
    return `${day}/${month}/${d.getFullYear() + 543}`;
}

function buildMetaLine(item) {
    const parts = [`ผู้ขอ: ${item.borrower_name}`, item.borrow_type];

    if (item.status === 'คืนแล้ว') {
        const returnedOn = formatThaiDate(item.actual_return_date || item.updated_at || item.return_date);
        parts.push(returnedOn ? `คืนแล้ว ${returnedOn}` : 'คืนแล้ว');
    } else if (item.borrow_type === 'เบิกชั่วคราว' && item.return_date) {
        const start = formatThaiDate(item.borrow_date || item.created_at, false);
        const end = formatThaiDate(item.return_date);
        parts.push(start && end ? `${start} - ${end}` : end);
    } else {
        const requestedOn = formatThaiDate(item.borrow_date || item.created_at);
        if (requestedOn) parts.push(requestedOn);
    }

    return parts.filter(Boolean).join(' · ');
}

function validateForm(form) {
    const errors = {};
    if (!form.request_date) errors.request_date = 'กรุณาระบุวันที่';
    if (!form.borrower_name.trim()) errors.borrower_name = 'กรุณากรอกชื่อผู้ขอเบิก';
    if (!form.emp_id.trim()) errors.emp_id = 'กรุณากรอกรหัสพนักงาน';
    if (!form.department.trim()) errors.department = 'กรุณากรอกแผนก';
    if (!form.equipment_id) errors.equipment_id = 'กรุณาเลือกอุปกรณ์ที่ต้องการเบิก';
    if (form.borrow_type === 'เบิกชั่วคราว' && !form.return_date) {
        errors.return_date = 'กรุณาระบุวันที่ต้องการคืน';
    }
    if (!form.reason.trim()) errors.reason = 'กรุณาระบุรายละเอียด/เหตุผลการเบิก';
    return errors;
}

function FieldLabel({ icon: Icon, children, required }) {
    return (
        <label className="mb-1.5 flex items-center gap-1.5 text-xs font-medium text-zinc-600">
            <Icon className="h-3.5 w-3.5 text-zinc-400" />
            {children}
            {required ? <span className="text-red-600">*</span> : null}
        </label>
    );
}

function FieldError({ message }) {
    if (!message) return null;
    return (
        <p className="mt-1 flex items-center gap-1 text-[11.5px] text-red-600">
            <ExclamationTriangleIcon className="h-3 w-3 flex-shrink-0" />
            {message}
        </p>
    );
}

const inputClass = (hasError) =>
    `h-10 w-full rounded-lg border px-3 text-sm outline-none transition focus:ring-2 ${
        hasError
            ? 'border-red-300 focus:border-red-500 focus:ring-red-100'
            : 'border-zinc-200 focus:border-red-500 focus:ring-red-100'
    }`;

// Small "label: value" row used inside the selected-equipment summary to
// mirror the paper form's "Part#2: Equipment Information" columns (Group
// Hardware, Brand, Model, Asset no, Label name). These read from the
// equipment record returned by listEquipment() — if your equipment API
// doesn't yet expose group_hardware/brand/model/label_name, add them there
// so this can pick them up; until then each falls back to "-".
function EquipmentDetailRow({ label, value }) {
    return (
        <div className="flex items-center justify-between gap-3 py-1 text-xs">
            <span className="text-zinc-400">{label}</span>
            <span className="max-w-[60%] truncate text-right font-medium text-zinc-700">{value || '-'}</span>
        </div>
    );
}

// Searchable equipment picker. `listEquipment()` has no server-side
// search/filter, so this fetches the full list once (shared by the parent,
// see `equipmentOptions` prop) and filters it in the browser as the user
// types. Fine for catalogs up to roughly a few thousand items — beyond
// that, the fetch itself gets slow and this should switch to a paginated
// search endpoint (see the comment above the `listEquipment` import).
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

    const availableEquipment = useMemo(
        () => equipmentOptions.filter((item) => item.status === 'ในคลัง'),
        [equipmentOptions],
    );

    const matches = useMemo(() => {
        if (!query.trim()) return availableEquipment;
        const q = query.trim().toLowerCase();
        return availableEquipment.filter(
            (item) =>
                (item.product_name || '').toLowerCase().includes(q) ||
                (item.asset_name || '').toLowerCase().includes(q) ||
                (item.asset_tag || '').toLowerCase().includes(q),
        );
    }, [availableEquipment, query]);

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
            <div className="rounded-lg border border-zinc-200 bg-zinc-50 px-3 py-2.5">
                <div className="flex items-center justify-between gap-2">
                    <div className="flex min-w-0 items-center gap-2.5">
                        <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-md bg-white text-emerald-600 ring-1 ring-emerald-100">
                            <CheckCircleIcon className="h-5 w-5" />
                        </div>
                        <div className="min-w-0">
                            <p className="truncate text-sm font-medium text-zinc-800">{selected.product_name}</p>
                            <p className="truncate text-xs text-zinc-500">{selected.asset_name}</p>
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

                {/* Part#2: Equipment Information — pulled from the selected
                    equipment record rather than typed in by hand. */}
                <div className="mt-2.5 divide-y divide-zinc-200/70 border-t border-zinc-200/70 pt-1.5">
                    <EquipmentDetailRow label="Group Hardware" value={selected.group_hardware} />
                    <EquipmentDetailRow label="Brand" value={selected.brand} />
                    <EquipmentDetailRow label="Model" value={selected.model} />
                    <EquipmentDetailRow label="Asset no" value={selected.asset_tag} />
                    <EquipmentDetailRow label="Label name" value={selected.label_name} />
                </div>
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
                                    {query ? 'ไม่พบอุปกรณ์ที่ตรงกับคำค้นหา' : 'ไม่มีอุปกรณ์ที่พร้อมให้เบิกในขณะนี้'}
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
                                            <span className="block truncate text-xs text-zinc-500">{item.asset_name}</span>
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

function ListSkeleton() {
    return (
        <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="flex animate-pulse items-center gap-3 rounded-lg border border-zinc-200 px-3.5 py-3">
                    <div className="h-9 w-9 flex-shrink-0 rounded-lg bg-zinc-100" />
                    <div className="flex-1 space-y-1.5">
                        <div className="h-3 w-40 rounded bg-zinc-100" />
                        <div className="h-2.5 w-24 rounded bg-zinc-100" />
                    </div>
                    <div className="h-6 w-20 rounded-full bg-zinc-100" />
                </div>
            ))}
        </div>
    );
}

function BorrowRequestPage({ user }) {
    const canManageBorrow = user?.role === 'admin' || user?.role === 'system_admin';
    const isUser = user?.role === 'user';

    const [items, setItems] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [form, setForm] = useState(DEFAULT_FORM);
    const [submitting, setSubmitting] = useState(false);
    const [submitAttempted, setSubmitAttempted] = useState(false);
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [selectedRequest, setSelectedRequest] = useState(null);
    const [activeTab, setActiveTab] = useState('in_process');

    const [equipmentOptions, setEquipmentOptions] = useState([]);
    const [loadingEquipment, setLoadingEquipment] = useState(true);
    const [equipmentError, setEquipmentError] = useState('');

    const firstFieldRef = useRef(null);

    const loadBorrowRequests = async () => {
        try {
            setLoading(true);
            setError('');
            const data = await listBorrowRequests();
            setItems(Array.isArray(data) ? data : []);
        } catch (loadError) {
            setError(loadError?.message || 'Unable to load borrow requests');
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
            setEquipmentError(loadError?.message || 'ไม่สามารถโหลดรายการอุปกรณ์ได้');
        } finally {
            setLoadingEquipment(false);
        }
    };

    useEffect(() => {
        loadBorrowRequests();
        // Loaded for every role now, not just `user` — the list rows below
        // need product_name/asset_tag to show equipment names instead of
        // bare equipment_id, in addition to the create-request picker.
        loadEquipmentOptions();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // Lock body scroll and support Escape-to-close while the modal is open.
    useEffect(() => {
        if (!showCreateModal) return undefined;
        const previousOverflow = document.body.style.overflow;
        document.body.style.overflow = 'hidden';
        function handleKeyDown(event) {
            if (event.key === 'Escape' && !submitting) closeModal();
        }
        document.addEventListener('keydown', handleKeyDown);
        requestAnimationFrame(() => firstFieldRef.current?.focus());
        return () => {
            document.body.style.overflow = previousOverflow;
            document.removeEventListener('keydown', handleKeyDown);
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [showCreateModal, submitting]);

    const equipmentById = useMemo(() => {
        const map = new Map();
        equipmentOptions.forEach((eq) => map.set(eq.id, eq));
        return map;
    }, [equipmentOptions]);

    const visibleItems = useMemo(() => {
        let roleFiltered = items;

        // User เห็นเฉพาะคำขอของตัวเอง
        // Admin / System Admin เห็นคำขอทั้งหมด
        if (isUser) {
            roleFiltered = items.filter(
                (item) => item?.created_by_user_id === user?.id
            );
        }

        const activeStatuses = TABS.find((tab) => tab.key === activeTab)?.statuses;

        if (!activeStatuses) return roleFiltered;

        return roleFiltered.filter((item) => activeStatuses.includes(item?.status));
    }, [items, isUser, user?.id, activeTab]);

    const formErrors = useMemo(() => validateForm(form), [form]);

    function openModal() {
        setForm({ ...DEFAULT_FORM, request_date: todayIsoDate() });
        setSubmitAttempted(false);
        setShowCreateModal(true);
    }

    function closeModal() {
        if (submitting) return;
        setShowCreateModal(false);
        setForm(DEFAULT_FORM);
        setSubmitAttempted(false);
    }

    function openDetailModal(item) {
        setSelectedRequest(item);
    }

    function closeDetailModal() {
        setSelectedRequest(null);
    }

    async function handleCreate(event) {
        event.preventDefault();
        if (!isUser) return;

        setSubmitAttempted(true);
        if (Object.keys(formErrors).length > 0) return;

        try {
            setSubmitting(true);
            setError('');
            await createBorrowRequest({
                equipment_id: Number(form.equipment_id),
                borrower_name: form.borrower_name.trim(),
                emp_id: form.emp_id.trim(),
                department: form.department.trim(),
                it_request_no: form.it_request_no.trim(),
                request_date: form.request_date,
                borrow_type: form.borrow_type,
                // chk_return_date_matches_type requires this to be NULL for
                // เบิกถาวร and NOT NULL for เบิกชั่วคราว — keep both sides in sync.
                return_date: form.borrow_type === 'เบิกชั่วคราว' ? form.return_date : null,
                reason: form.reason,
            });
            closeModal();
            await loadBorrowRequests();
        } catch (submitError) {
            setError(submitError?.message || 'Unable to create borrow request');
        } finally {
            setSubmitting(false);
        }
    }

    async function handleApprove(id) {
        if (!canManageBorrow) return;

        try {
            setError('');
            await approveBorrowRequest(id);
            await loadBorrowRequests();
        } catch (approveError) {
            setError(approveError?.message || 'Unable to approve borrow request');
        }
    }

    async function handleReject(id) {
        if (!canManageBorrow) return;

        try {
            setError('');
            await rejectBorrowRequest(id);
            await loadBorrowRequests();
        } catch (rejectError) {
            setError(rejectError?.message || 'Unable to reject borrow request');
        }
    }

    async function handleReturn(id) {
        if (!isUser) return;

        try {
            setError('');
            await returnBorrowRequest(id);
            await loadBorrowRequests();
        } catch (returnError) {
            setError(returnError?.message || 'Unable to return borrow request');
        }
    }

    async function handleConfirmReturn(id) {
        if (!canManageBorrow) return;

        try {
            setError('');
            await confirmReturnBorrowRequest(id);
            await loadBorrowRequests();
        } catch (confirmError) {
            setError(confirmError?.message || 'Unable to confirm return');
        }
    }

    return (
        <div className="p-4 sm:p-6">
            {error ? (
                <div className="mb-4 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3.5 py-2.5 text-sm text-red-700">
                    <ExclamationTriangleIcon className="mt-0.5 h-4 w-4 flex-shrink-0" />
                    <span>{error}</span>
                </div>
            ) : null}

            <div className="rounded-xl border border-zinc-200 bg-white p-4 shadow-sm sm:p-5">
                {/* Header */}
                <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
                    <div className="flex items-center gap-3">
                        <div>
                            <h2 className="text-sm font-semibold text-zinc-800">รายการเบิก-คืนอุปกรณ์</h2>
                            <p className="mt-0.5 text-xs text-zinc-400">ทั้งหมด {visibleItems.length} รายการ</p>
                        </div>
                    </div>
                    {isUser ? (
                        <button
                            type="button"
                            onClick={openModal}
                            className="inline-flex h-[34px] items-center gap-1.5 rounded-lg bg-red-600 px-3.5 text-sm font-medium text-white shadow-sm transition hover:bg-red-700 active:bg-red-800"
                        >
                            <PlusIcon className="h-4 w-4" />
                            สร้างคำขอเบิก
                        </button>
                    ) : null}
                </div>

                {/* Tabs */}
                <div className="mb-4 flex gap-1 overflow-x-auto border-b border-zinc-200">
                    {TABS.map((tab) => {
                        const count = items.filter((item) => {
                            let roleOk = !isUser || item?.created_by_user_id === user?.id;
                            if (!roleOk) return false;
                            return !tab.statuses || tab.statuses.includes(item?.status);
                        }).length;
                        return (
                            <button
                                key={tab.key}
                                type="button"
                                onClick={() => setActiveTab(tab.key)}
                                className={`-mb-px flex flex-shrink-0 items-center gap-1.5 border-b-2 px-3.5 py-2.5 text-[13px] transition-colors ${activeTab === tab.key
                                    ? 'border-red-600 font-medium text-zinc-800'
                                    : 'border-transparent text-zinc-500 hover:text-zinc-700'
                                    }`}
                            >
                                {tab.label}
                                <span
                                    className={`rounded-full px-1.5 py-0.5 text-[11px] ${activeTab === tab.key ? 'bg-red-50 text-red-700' : 'bg-zinc-100 text-zinc-500'
                                        }`}
                                >
                                    {count}
                                </span>
                            </button>
                        );
                    })}
                </div>

                {/* List */}
                {loading ? (
                    <ListSkeleton />
                ) : visibleItems.length === 0 ? (
                    <div className="py-9 text-center">
                        <ClipboardDocumentListIcon className="mx-auto mb-2.5 h-8 w-8 text-zinc-300" />
                        <p className="text-[13.5px] font-medium text-zinc-700">ไม่มีรายการในหมวดนี้</p>
                        <span className="text-xs text-zinc-400">ไม่มีรายการที่ต้องดำเนินการในหมวดนี้</span>
                    </div>
                ) : (
                    <div className="space-y-2">
                        {visibleItems.map((item) => {
                            const equipment = equipmentById.get(item.equipment_id);
                            const title = equipment
                                ? `${equipment.product_name}${equipment.asset_tag ? ` · ${equipment.asset_tag}` : ''}`
                                : loadingEquipment
                                    ? 'กำลังโหลดข้อมูลอุปกรณ์...'
                                    : `อุปกรณ์รหัส ${item.equipment_id}`;
                            const statusDisplay = getStatusDisplay(item.status);
                            const isDone = item.status === 'คืนแล้ว';

                            return (
                                <div
                                    key={item.id}
                                    className={`flex flex-wrap items-center justify-between gap-3 rounded-lg border border-zinc-200 px-3.5 py-3 transition hover:border-zinc-300 ${
                                        isDone ? 'opacity-60' : ''
                                    }`}
                                >
                                    <div className="flex items-center gap-3">
                                        <div className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-zinc-50 text-zinc-500">
                                            <ArchiveBoxIcon className="h-[18px] w-[18px]" />
                                        </div>

                                        <div>
                                            <p className="text-[13px] font-medium text-zinc-800">{title}</p>
                                            <p className="mt-0.5 text-xs text-zinc-500">{buildMetaLine(item)}</p>
                                        </div>
                                    </div>

                                    <div className="flex flex-wrap items-center gap-2">
                                        <button
                                            type="button"
                                            onClick={() => openDetailModal(item)}
                                            className="h-[30px] rounded-lg border border-zinc-200 bg-white px-3 text-xs font-medium text-zinc-600 transition hover:bg-zinc-50 hover:text-zinc-800"
                                        >
                                            ดูรายละเอียด
                                        </button>

                                        <span
                                            className={`inline-flex rounded-full px-2.5 py-1 text-xs font-medium ${statusDisplay.className}`}
                                        >
                                            {statusDisplay.label}
                                        </span>

                                        {canManageBorrow && item.status === 'รออนุมัติ' ? (
                                            <>
                                                <button
                                                    type="button"
                                                    onClick={() => handleApprove(item.id)}
                                                    className="h-[30px] rounded-lg border border-zinc-200 bg-white px-3 text-xs font-medium text-zinc-700 transition hover:bg-zinc-50"
                                                >
                                                    อนุมัติ
                                                </button>
                                                <button
                                                    type="button"
                                                    onClick={() => handleReject(item.id)}
                                                    className="h-[30px] rounded-lg border border-red-200 bg-white px-3 text-xs font-medium text-red-600 transition hover:border-red-300 hover:bg-red-50"
                                                >
                                                    ปฏิเสธ
                                                </button>
                                            </>
                                        ) : null}

                                        {isUser && (item.status === 'อนุมัติ' || item.status === 'เกินกำหนดคืน') ? (
                                            <button
                                                type="button"
                                                onClick={() => handleReturn(item.id)}
                                                className="h-[30px] rounded-lg border border-zinc-200 bg-white px-3 text-xs font-medium text-zinc-700 transition hover:bg-zinc-50"
                                            >
                                                คืนอุปกรณ์
                                            </button>
                                        ) : null}

                                        {canManageBorrow && item.status === 'รอตรวจรับคืน' ? (
                                            <button
                                                type="button"
                                                onClick={() => handleConfirmReturn(item.id)}
                                                className="h-[30px] rounded-lg border border-zinc-200 bg-white px-3 text-xs font-medium text-zinc-700 transition hover:bg-zinc-50"
                                            >
                                                รับคืน
                                            </button>
                                        ) : null}
                                    </div>
                                </div>
                            );
                        })}
                    </div>
                )}
            </div>

            {/* Request detail modal */}
            {selectedRequest ? (() => {
                const equipment = equipmentById.get(selectedRequest.equipment_id);
                const statusDisplay = getStatusDisplay(selectedRequest.status);
                const equipmentTitle = equipment
                    ? `${equipment.product_name}${equipment.asset_tag ? ` · ${equipment.asset_tag}` : ''}`
                    : `อุปกรณ์รหัส ${selectedRequest.equipment_id}`;

                return (
                    <div
                        className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-[2px]"
                        onClick={(event) => {
                            if (event.target === event.currentTarget) closeDetailModal();
                        }}
                        role="presentation"
                    >
                        <div
                            role="dialog"
                            aria-modal="true"
                            aria-labelledby="borrow-detail-title"
                            className="w-[520px] max-w-full overflow-hidden rounded-2xl bg-white shadow-2xl"
                        >
                            <div className="flex items-start justify-between border-b border-zinc-100 px-6 py-4">
                                <div>
                                    <h3 id="borrow-detail-title" className="text-base font-semibold text-zinc-800">
                                        รายละเอียดคำขอ #{selectedRequest.id}
                                    </h3>
                                    <p className="mt-0.5 text-xs text-zinc-400">ข้อมูลคำขอเบิกอุปกรณ์</p>
                                </div>
                                <button type="button" onClick={closeDetailModal} className="rounded-lg p-1.5 text-zinc-400 transition hover:bg-zinc-100 hover:text-zinc-600" aria-label="ปิดรายละเอียด">
                                    <XMarkIcon className="h-5 w-5" />
                                </button>
                            </div>

                            <div className="space-y-4 px-6 py-5 text-sm">
                                <div className="flex items-center justify-between gap-3 rounded-lg bg-zinc-50 px-3 py-2.5">
                                    <span className="text-zinc-500">สถานะ</span>
                                    <span className={`inline-flex rounded-full px-2.5 py-1 text-xs font-medium ${statusDisplay.className}`}>
                                        {statusDisplay.label}
                                    </span>
                                </div>

                                <div className="grid gap-3 sm:grid-cols-2">
                                    <div>
                                        <p className="text-xs text-zinc-400">ชื่อผู้ขอเบิก</p>
                                        <p className="mt-1 font-medium text-zinc-800">{selectedRequest.borrower_name || '-'}</p>
                                    </div>
                                    <div>
                                        <p className="text-xs text-zinc-400">รหัสพนักงาน</p>
                                        <p className="mt-1 font-medium text-zinc-800">{selectedRequest.emp_id || '-'}</p>
                                    </div>
                                </div>

                                <div className="grid gap-3 sm:grid-cols-2">
                                    <div>
                                        <p className="text-xs text-zinc-400">แผนก</p>
                                        <p className="mt-1 font-medium text-zinc-800">{selectedRequest.department || '-'}</p>
                                    </div>
                                    <div>
                                        <p className="text-xs text-zinc-400">IT Request/On Call no.</p>
                                        <p className="mt-1 font-medium text-zinc-800">{selectedRequest.it_request_no || '-'}</p>
                                    </div>
                                </div>

                                <div>
                                    <p className="text-xs text-zinc-400">ประเภทการเบิก</p>
                                    <p className="mt-1 font-medium text-zinc-800">{selectedRequest.borrow_type || '-'}</p>
                                </div>

                                <div>
                                    <p className="text-xs text-zinc-400">อุปกรณ์</p>
                                    <p className="mt-1 font-medium text-zinc-800">{equipmentTitle}</p>
                                    {equipment?.asset_name ? <p className="mt-0.5 text-xs text-zinc-500">{equipment.asset_name}</p> : null}
                                </div>

                                <div className="grid gap-3 sm:grid-cols-2">
                                    <div>
                                        <p className="text-xs text-zinc-400">วันที่ยื่นคำขอ</p>
                                        <p className="mt-1 text-zinc-700">
                                            {formatThaiDate(selectedRequest.request_date || selectedRequest.created_at) || '-'}
                                        </p>
                                    </div>
                                    <div>
                                        <p className="text-xs text-zinc-400">วันที่ต้องการคืน</p>
                                        <p className="mt-1 text-zinc-700">
                                            {selectedRequest.borrow_type === 'เบิกชั่วคราว'
                                                ? (formatThaiDate(selectedRequest.return_date) || '-')
                                                : 'ไม่ระบุ (เบิกถาวร)'}
                                        </p>
                                    </div>
                                </div>

                                <div>
                                    <p className="text-xs text-zinc-400">รายละเอียด/เหตุผลการเบิก</p>
                                    <p className="mt-1 whitespace-pre-wrap rounded-lg border border-zinc-100 bg-zinc-50 px-3 py-2.5 text-zinc-700">
                                        {selectedRequest.reason || '-'}
                                    </p>
                                </div>
                            </div>

                            <div className="flex justify-end border-t border-zinc-100 bg-zinc-50/60 px-6 py-4">
                                <button type="button" onClick={closeDetailModal} className="h-9 rounded-lg border border-zinc-200 bg-white px-4 text-sm text-zinc-700 transition hover:bg-zinc-50">
                                    ปิด
                                </button>
                            </div>
                        </div>
                    </div>
                );
            })() : null}

            {/* Create request modal */}
            {showCreateModal ? (
                <div
                    className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-[2px]"
                    onClick={(event) => {
                        if (event.target === event.currentTarget) closeModal();
                    }}
                    role="presentation"
                >
                    <div
                        role="dialog"
                        aria-modal="true"
                        aria-labelledby="create-borrow-title"
                        className="flex max-h-[88vh] w-[460px] max-w-full flex-col overflow-hidden rounded-2xl bg-white shadow-2xl"
                    >
                        {/* Header */}
                        <div className="flex items-start justify-between border-b border-zinc-100 px-6 py-4">
                            <div className="flex items-start gap-3">
                                <div>
                                    <h3 id="create-borrow-title" className="text-base font-semibold text-zinc-800">
                                        สร้างคำขอเบิกอุปกรณ์
                                    </h3>
                                    <p className="mt-0.5 text-xs text-zinc-400">คำขอจะถูกส่งไปรออนุมัติจาก Admin</p>
                                </div>
                            </div>
                            <button
                                type="button"
                                onClick={closeModal}
                                disabled={submitting}
                                className="flex-shrink-0 rounded-lg p-1.5 text-zinc-400 transition hover:bg-zinc-100 hover:text-zinc-600 disabled:opacity-50"
                                aria-label="ปิดหน้าต่าง"
                            >
                                <XMarkIcon className="h-5 w-5" />
                            </button>
                        </div>

                        {/* Body */}
                        <form id="create-borrow-form" onSubmit={handleCreate} className="flex-1 overflow-y-auto px-6 py-5">
                            {/* Part#1: User */}
                            <div className="mb-4 grid grid-cols-2 gap-3">
                                <div>
                                    <FieldLabel icon={CalendarDaysIcon} required>
                                        วันที่
                                    </FieldLabel>
                                    <input
                                        ref={firstFieldRef}
                                        type="date"
                                        value={form.request_date}
                                        onChange={(event) => setForm({ ...form, request_date: event.target.value })}
                                        disabled={submitting}
                                        className={inputClass(submitAttempted && formErrors.request_date)}
                                    />
                                    {submitAttempted ? <FieldError message={formErrors.request_date} /> : null}
                                </div>

                                <div>
                                    <FieldLabel icon={HashtagIcon}>
                                        IT Request/On Call no.
                                    </FieldLabel>
                                    <input
                                        type="text"
                                        value={form.it_request_no}
                                        onChange={(event) => setForm({ ...form, it_request_no: event.target.value })}
                                        placeholder="ถ้ามี"
                                        disabled={submitting}
                                        className={inputClass(false)}
                                    />
                                </div>
                            </div>

                            <div className="mb-4">
                                <FieldLabel icon={UserIcon} required>
                                    ชื่อผู้ขอเบิก (Requester)
                                </FieldLabel>
                                <input
                                    type="text"
                                    value={form.borrower_name}
                                    onChange={(event) => setForm({ ...form, borrower_name: event.target.value })}
                                    placeholder="กรอกชื่อผู้ที่ต้องการเบิก"
                                    disabled={submitting}
                                    className={inputClass(submitAttempted && formErrors.borrower_name)}
                                />
                                {submitAttempted ? <FieldError message={formErrors.borrower_name} /> : null}
                            </div>

                            <div className="mb-4 grid grid-cols-2 gap-3">
                                <div>
                                    <FieldLabel icon={IdentificationIcon} required>
                                        Emp.ID
                                    </FieldLabel>
                                    <input
                                        type="text"
                                        value={form.emp_id}
                                        onChange={(event) => setForm({ ...form, emp_id: event.target.value })}
                                        placeholder="รหัสพนักงาน"
                                        disabled={submitting}
                                        className={inputClass(submitAttempted && formErrors.emp_id)}
                                    />
                                    {submitAttempted ? <FieldError message={formErrors.emp_id} /> : null}
                                </div>

                                <div>
                                    <FieldLabel icon={BuildingOfficeIcon} required>
                                        แผนก (Department)
                                    </FieldLabel>
                                    <input
                                        type="text"
                                        value={form.department}
                                        onChange={(event) => setForm({ ...form, department: event.target.value })}
                                        placeholder="เช่น Call.4"
                                        disabled={submitting}
                                        className={inputClass(submitAttempted && formErrors.department)}
                                    />
                                    {submitAttempted ? <FieldError message={formErrors.department} /> : null}
                                </div>
                            </div>

                            {/* Part#2: Equipment Information — picked from the
                                catalog; details render inside the picker once
                                an item is selected. */}
                            <div className="mb-4">
                                <FieldLabel icon={ArchiveBoxIcon} required>
                                    อุปกรณ์ที่ต้องการเบิก
                                </FieldLabel>
                                <EquipmentPicker
                                    value={form.equipment_id}
                                    onChange={(id) => setForm({ ...form, equipment_id: id })}
                                    disabled={submitting}
                                    equipmentOptions={equipmentOptions}
                                    loadingOptions={loadingEquipment}
                                    optionsError={equipmentError}
                                    hasError={submitAttempted && formErrors.equipment_id}
                                />
                                {submitAttempted ? <FieldError message={formErrors.equipment_id} /> : null}
                            </div>

                            <div className="mb-4 grid grid-cols-2 gap-3">
                                <div>
                                    <FieldLabel icon={TagIcon}>ประเภทการเบิก</FieldLabel>
                                    <select
                                        value={form.borrow_type}
                                        onChange={(event) => setForm({
                                            ...form,
                                            borrow_type: event.target.value,
                                            // clear any leftover date so a stale value can't
                                            // slip through if the user switches back and forth
                                            return_date: event.target.value === 'เบิกถาวร' ? '' : form.return_date,
                                        })}
                                        disabled={submitting}
                                        className={inputClass(false)}
                                    >
                                        <option value="เบิกถาวร">เบิกถาวร</option>
                                        <option value="เบิกชั่วคราว">เบิกชั่วคราว</option>
                                    </select>
                                </div>

                                <div>
                                    <FieldLabel icon={CalendarDaysIcon} required={form.borrow_type === 'เบิกชั่วคราว'}>
                                        วันที่ต้องการคืน
                                    </FieldLabel>
                                    {form.borrow_type === 'เบิกชั่วคราว' ? (
                                        <>
                                            <input
                                                type="date"
                                                value={form.return_date}
                                                onChange={(event) => setForm({ ...form, return_date: event.target.value })}
                                                disabled={submitting}
                                                min={new Date().toISOString().slice(0, 10)}
                                                className={inputClass(submitAttempted && formErrors.return_date)}
                                            />
                                            {submitAttempted ? <FieldError message={formErrors.return_date} /> : null}
                                        </>
                                    ) : (
                                        <div className="flex h-10 items-center rounded-lg border border-dashed border-zinc-200 px-3 text-xs text-zinc-400">
                                            ไม่ต้องระบุ (เบิกถาวร)
                                        </div>
                                    )}
                                </div>
                            </div>

                            <div>
                                <FieldLabel icon={ChatBubbleLeftEllipsisIcon} required>
                                    รายละเอียด/เหตุผลการเบิก (Detail/Proposed)
                                </FieldLabel>
                                <textarea
                                    value={form.reason}
                                    onChange={(event) => setForm({ ...form, reason: event.target.value })}
                                    placeholder="รายละเอียดหรือเหตุผลการเบิก"
                                    rows={3}
                                    disabled={submitting}
                                    className={`w-full resize-none rounded-lg border px-3 py-2 text-sm outline-none transition focus:ring-2 ${
                                        submitAttempted && formErrors.reason
                                            ? 'border-red-300 focus:border-red-500 focus:ring-red-100'
                                            : 'border-zinc-200 focus:border-red-500 focus:ring-red-100'
                                    }`}
                                />
                                {submitAttempted ? <FieldError message={formErrors.reason} /> : null}
                            </div>
                        </form>

                        {/* Footer */}
                        <div className="flex justify-end gap-2 border-t border-zinc-100 bg-zinc-50/60 px-6 py-4">
                            <button
                                type="button"
                                onClick={closeModal}
                                disabled={submitting}
                                className="h-9 rounded-lg border border-zinc-200 bg-white px-4 text-sm text-zinc-700 transition hover:bg-zinc-50 disabled:opacity-60"
                            >
                                ยกเลิก
                            </button>
                            <button
                                type="submit"
                                form="create-borrow-form"
                                disabled={submitting}
                                className="inline-flex h-9 items-center gap-1.5 rounded-lg bg-red-600 px-4 text-sm font-medium text-white transition hover:bg-red-700 disabled:opacity-60"
                            >
                                {submitting ? <ArrowPathIcon className="h-4 w-4 animate-spin" /> : null}
                                {submitting ? 'กำลังส่ง...' : 'ส่งคำขอ'}
                            </button>
                        </div>
                    </div>
                </div>
            ) : null}
        </div>
    );
}

export default BorrowRequestPage;
