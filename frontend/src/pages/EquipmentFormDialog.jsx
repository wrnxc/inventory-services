import React, { useEffect, useMemo, useState } from 'react';

import ConfirmModal from '../components/modal/ConfirmModal';
import { listEquipmentTypes } from '../api/equipmentTypesClient';

const STATUS_OPTIONS = ['ในคลัง', 'กำลังใช้งาน', 'เสียหาย', 'เลิกใช้งาน'];

const TITLE_BY_MODE = {
  add: 'เพิ่มอุปกรณ์ใหม่',
  edit: 'แก้ไขข้อมูลอุปกรณ์',
  view: 'รายละเอียดอุปกรณ์',
};

const DESCRIPTION_BY_MODE = {
  add: 'กรอกรายละเอียดอุปกรณ์ที่จะนำเข้าระบบ',
  edit: (name) => `แก้ไขข้อมูล ${name}`,
  view: 'ข้อมูลอุปกรณ์ปัจจุบัน',
};

const EMPTY_FORM = {
  type_id: '',
  product_name: '',
  asset_name: '',
  asset_tag: '',
  asset_serial_no: '',
  bar_code: '',
  vendor_name: '',
  location: '',
  assigned_to_department: '',
  site: '',
  asset_no: '',
  budget: '',
  remark: '',
  username: '',
  req_no: '',
  status: 'ในคลัง',
};

const INPUT_CLASS =
  'h-10 w-full rounded-lg border border-zinc-200 px-3 text-sm outline-none transition focus:border-red-500 focus:ring-2 focus:ring-red-100 disabled:cursor-not-allowed disabled:border-zinc-100 disabled:bg-zinc-50 disabled:text-zinc-500';

const TEXTAREA_CLASS =
  'w-full resize-none rounded-lg border border-zinc-200 px-3 py-2.5 text-sm outline-none transition focus:border-red-500 focus:ring-2 focus:ring-red-100 disabled:cursor-not-allowed disabled:border-zinc-100 disabled:bg-zinc-50 disabled:text-zinc-500';

function EquipmentFormDialog({ mode, item, onClose, onSubmit }) {
  // ---------- state ----------
  const [form, setForm] = useState({
    ...EMPTY_FORM,
    type_id: item?.type_id ? String(item.type_id) : '',
    product_name: item?.product_name || '',
    asset_name: item?.asset_name || '',
    asset_tag: item?.asset_tag || '',
    asset_serial_no: item?.asset_serial_no || '',
    bar_code: item?.bar_code || '',
    vendor_name: item?.vendor_name || '',
    location: item?.location || '',
    assigned_to_department: item?.assigned_to_department || '',
    site: item?.site || '',
    asset_no: item?.asset_no || '',
    budget: item?.budget || '',
    remark: item?.remark || '',
    username: item?.username || '',
    req_no: item?.req_no || '',
    status: item?.status || 'ในคลัง',
  });

  const [equipmentTypes, setEquipmentTypes] = useState([]);
  const [typesLoading, setTypesLoading] = useState(true);
  const [typesError, setTypesError] = useState('');

  const [saving, setSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [pendingPayload, setPendingPayload] = useState(null);

  // ---------- effects ----------
  useEffect(() => {
    let cancelled = false;

    async function loadTypes() {
      try {
        setTypesLoading(true);
        setTypesError('');

        const data = await listEquipmentTypes();
        const items = Array.isArray(data) ? data : [];

        if (!cancelled) setEquipmentTypes(items);
      } catch (error) {
        if (!cancelled) {
          setEquipmentTypes([]);
          setTypesError(error?.message || 'ไม่สามารถโหลดประเภทอุปกรณ์ได้');
        }
      } finally {
        if (!cancelled) setTypesLoading(false);
      }
    }

    loadTypes();
    return () => {
      cancelled = true;
    };
  }, []);

  // ---------- derived values ----------
  const isView = mode === 'view';
  const isAdd = mode === 'add';
  const isEdit = mode === 'edit';

  const typeOptions = useMemo(() => {
    const options = equipmentTypes.map((type) => ({
      value: String(type.id),
      label: type.name,
    }));

    const hasCurrentType = options.some(
      (option) => option.value === String(item?.type_id)
    );

    if (item?.type_id && item?.type_name && !hasCurrentType) {
      options.push({ value: String(item.type_id), label: item.type_name });
    }

    return options;
  }, [equipmentTypes, item]);

  const typeSelectDisabled = isView || typesLoading || Boolean(typesError);
  const typeSelectPlaceholder = typesLoading
    ? 'กำลังโหลดประเภทอุปกรณ์...'
    : '— เลือกประเภท —';

  const dialogTitle = TITLE_BY_MODE[mode];
  const dialogDescription = isEdit
    ? DESCRIPTION_BY_MODE.edit(item?.asset_name || '')
    : DESCRIPTION_BY_MODE[mode];

  const submitDisabled = saving || typesLoading || Boolean(typesError);
  const submitLabel = isEdit ? 'บันทึกการแก้ไข' : 'บันทึกอุปกรณ์';
  const cancelLabel = isView ? 'ปิด' : 'ยกเลิก';

  const confirmTitle = isEdit ? 'ยืนยันการแก้ไขอุปกรณ์' : 'ยืนยันการเพิ่มอุปกรณ์';
  const confirmDescription = isEdit
    ? `ต้องการบันทึกการแก้ไข ${form.asset_name} หรือไม่?`
    : `ต้องการเพิ่ม ${form.asset_name} เข้าระบบหรือไม่?`;
  const confirmLabel = saving
    ? 'กำลังบันทึก...'
    : isEdit
      ? 'ยืนยันการแก้ไข'
      : 'ยืนยันเพิ่มอุปกรณ์';

  const showStatusBadge = isView && Boolean(item?.status);
  const showMetaFooter = isView;
  const showAddStatusHint = isAdd;

  // ---------- handlers ----------
  function updateField(field, value) {
    setForm((current) => ({ ...current, [field]: value }));
  }

  function handleClose() {
    if (saving) return;
    onClose();
  }

  function buildPayload() {
    return {
      type_id: Number(form.type_id),
      product_name: form.product_name.trim(),
      asset_name: form.asset_name.trim(),
      asset_tag: form.asset_tag.trim(),
      asset_serial_no: form.asset_serial_no.trim(),
      bar_code: form.bar_code.trim(),
      vendor_name: form.vendor_name.trim(),
      location: form.location.trim(),
      assigned_to_department: form.assigned_to_department.trim(),
      site: form.site.trim(),
      asset_no: form.asset_no.trim(),
      budget: form.budget.trim(),
      remark: form.remark.trim(),
      username: form.username.trim(),
      req_no: form.req_no.trim(),
      status: form.status,
    };
  }

  function validateForm() {
    if (!form.type_id) return 'กรุณาเลือกประเภทอุปกรณ์';
    if (!form.product_name.trim()) return 'กรุณากรอก Product Name';
    if (!form.asset_name.trim()) return 'กรุณากรอก Asset Name';
    return '';
  }

  function handleSubmit(event) {
    event.preventDefault();

    const validationError = validateForm();

    if (validationError) {
      setErrorMessage(validationError);
      return;
    }

    setErrorMessage('');
    setPendingPayload(buildPayload());
    setConfirmOpen(true);
  }

  async function handleConfirmSubmit() {
    if (!pendingPayload) return;

    try {
      setSaving(true);
      setErrorMessage('');

      await onSubmit(pendingPayload);

      setConfirmOpen(false);
      setPendingPayload(null);
    } catch (error) {
      setErrorMessage(error?.message || 'ไม่สามารถบันทึกข้อมูลได้');
      setConfirmOpen(false);
    } finally {
      setSaving(false);
    }
  }

  function handleConfirmCancel() {
    if (saving) return;

    setConfirmOpen(false);
    setPendingPayload(null);
  }

  // ---------- render ----------
  return (
    <>
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4">
        <div className="flex max-h-[90vh] w-full max-w-3xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl">
          <div className="flex-shrink-0 border-b border-zinc-100 px-6 py-5">
            <div className="flex items-start justify-between gap-4">
              <div>
                <h3 className="text-lg font-semibold text-zinc-800">{dialogTitle}</h3>
                <p className="mt-1 text-xs text-zinc-400">{dialogDescription}</p>
              </div>

              {showStatusBadge ? (
                <span className="flex-shrink-0 rounded-full bg-zinc-100 px-3 py-1 text-xs font-medium text-zinc-600">
                  {item.status}
                </span>
              ) : null}
            </div>
          </div>

          <form id="equipment-form" onSubmit={handleSubmit} className="flex-1 overflow-y-auto px-6 py-5">
            <FormSection title="ข้อมูลอุปกรณ์">
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                <Field label="Product Type" required>
                  <select
                    value={form.type_id}
                    disabled={typeSelectDisabled}
                    onChange={(event) => updateField('type_id', event.target.value)}
                    className={INPUT_CLASS}
                  >
                    <option value="">{typeSelectPlaceholder}</option>
                    {typeOptions.map((option) => (
                      <option key={option.value} value={option.value}>
                        {option.label}
                      </option>
                    ))}
                  </select>
                  {typesError ? <p className="mt-1.5 text-xs text-red-600">{typesError}</p> : null}
                </Field>

                <Field label="Product Name" required>
                  <input
                    value={form.product_name}
                    disabled={isView}
                    onChange={(event) => updateField('product_name', event.target.value)}
                    placeholder="เช่น HP ProBook"
                    className={INPUT_CLASS}
                  />
                </Field>

                <Field label="Asset Name" required>
                  <input
                    value={form.asset_name}
                    disabled={isView}
                    onChange={(event) => updateField('asset_name', event.target.value)}
                    placeholder="เช่น NB-0356"
                    className={INPUT_CLASS}
                  />
                </Field>

                <Field label="Asset Tag">
                  <input
                    value={form.asset_tag}
                    disabled={isView}
                    onChange={(event) => updateField('asset_tag', event.target.value)}
                    className={INPUT_CLASS}
                  />
                </Field>

                <Field label="Serial Number">
                  <input
                    value={form.asset_serial_no}
                    disabled={isView}
                    onChange={(event) => updateField('asset_serial_no', event.target.value)}
                    className={INPUT_CLASS}
                  />
                </Field>

                <Field label="Barcode">
                  <input
                    value={form.bar_code}
                    disabled={isView}
                    onChange={(event) => updateField('bar_code', event.target.value)}
                    className={INPUT_CLASS}
                  />
                </Field>
              </div>
            </FormSection>

            <FormSection title="ตำแหน่งจัดเก็บ / ผู้จำหน่าย">
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                <Field label="Vendor">
                  <input
                    value={form.vendor_name}
                    disabled={isView}
                    onChange={(event) => updateField('vendor_name', event.target.value)}
                    className={INPUT_CLASS}
                  />
                </Field>

                <Field label="Location">
                  <input
                    value={form.location}
                    disabled={isView}
                    onChange={(event) => updateField('location', event.target.value)}
                    className={INPUT_CLASS}
                  />
                </Field>

                <Field label="Site">
                  <input
                    value={form.site}
                    disabled={isView}
                    onChange={(event) => updateField('site', event.target.value)}
                    className={INPUT_CLASS}
                  />
                </Field>

                <Field label="Department">
                  <input
                    value={form.assigned_to_department}
                    disabled={isView}
                    onChange={(event) => updateField('assigned_to_department', event.target.value)}
                    className={INPUT_CLASS}
                  />
                </Field>
              </div>
            </FormSection>

            <FormSection title="การเบิกจ่ายและงบประมาณ">
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                <Field label="Asset No.">
                  <input
                    value={form.asset_no}
                    disabled={isView}
                    onChange={(event) => updateField('asset_no', event.target.value)}
                    className={INPUT_CLASS}
                  />
                </Field>

                <Field label="Budget">
                  <input
                    value={form.budget}
                    disabled={isView}
                    maxLength={10}
                    onChange={(event) => updateField('budget', event.target.value)}
                    className={INPUT_CLASS}
                  />
                </Field>

                <Field label="User Name">
                  <input
                    value={form.username}
                    disabled={isView}
                    onChange={(event) => updateField('username', event.target.value)}
                    placeholder="ชื่อผู้ใช้งานอุปกรณ์"
                    className={INPUT_CLASS}
                  />
                </Field>

                <Field label="Request No.">
                  <input
                    value={form.req_no}
                    disabled={isView}
                    onChange={(event) => updateField('req_no', event.target.value)}
                    className={INPUT_CLASS}
                  />
                </Field>

                <Field label="สถานะ">
                  <select
                    value={form.status}
                    disabled={isView || isAdd}
                    onChange={(event) => updateField('status', event.target.value)}
                    className={INPUT_CLASS}
                  >
                    {STATUS_OPTIONS.map((status) => (
                      <option key={status} value={status}>
                        {status}
                      </option>
                    ))}
                  </select>
                  {showAddStatusHint ? (
                    <p className="mt-1.5 text-xs text-zinc-400">
                      อุปกรณ์ใหม่จะเริ่มต้นด้วยสถานะ "ในคลัง"
                    </p>
                  ) : null}
                </Field>
              </div>
            </FormSection>

            <FormSection title="หมายเหตุ" last>
              <textarea
                rows={3}
                value={form.remark}
                disabled={isView}
                onChange={(event) => updateField('remark', event.target.value)}
                placeholder="รายละเอียดเพิ่มเติม (ถ้ามี)"
                className={TEXTAREA_CLASS}
              />
            </FormSection>

            {showMetaFooter ? (
              <div className="mt-5 flex flex-wrap gap-x-6 gap-y-1 border-t border-zinc-100 pt-4 text-xs text-zinc-400">
                <span>สร้างเมื่อ: {item?.created_at || '-'}</span>
                <span>แก้ไขล่าสุด: {item?.updated_at || '-'}</span>
              </div>
            ) : null}

            {errorMessage ? (
              <div role="alert" className="mt-4 rounded-lg bg-red-50 px-3 py-2.5 text-xs text-red-700">
                {errorMessage}
              </div>
            ) : null}
          </form>

          <div className="flex flex-shrink-0 justify-end gap-2 border-t border-zinc-100 bg-white px-6 py-4">
            <button
              type="button"
              onClick={handleClose}
              disabled={saving}
              className="h-9 rounded-lg border border-zinc-200 bg-white px-4 text-sm hover:bg-zinc-50 disabled:opacity-60"
            >
              {cancelLabel}
            </button>

            {!isView ? (
              <button
                type="submit"
                form="equipment-form"
                disabled={submitDisabled}
                className="h-9 rounded-lg bg-red-600 px-4 text-sm font-medium text-white transition hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {submitLabel}
              </button>
            ) : null}
          </div>
        </div>
      </div>

      {confirmOpen ? (
        <ConfirmModal
          title={confirmTitle}
          description={confirmDescription}
          confirmLabel={confirmLabel}
          loading={saving}
          onConfirm={handleConfirmSubmit}
          onCancel={handleConfirmCancel}
        />
      ) : null}
    </>
  );
}

function FormSection({ title, children, last = false }) {
  return (
    <div className={last ? '' : 'mb-6 border-b border-zinc-100 pb-6'}>
      <h4 className="mb-3 text-[11px] font-semibold uppercase tracking-wide text-zinc-400">{title}</h4>
      {children}
    </div>
  );
}

function Field({ label, required = false, children }) {
  return (
    <div>
      <label className="mb-1.5 block text-xs font-medium text-zinc-600">
        {label}
        {required ? <span className="ml-1 text-red-600">*</span> : null}
      </label>
      {children}
    </div>
  );
}

export default EquipmentFormDialog;