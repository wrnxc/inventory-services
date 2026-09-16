import React from 'react';

const RESPONSE_STYLE = {
  success: {
    title: 'สำเร็จ',
    badge: 'bg-emerald-50 text-emerald-700',
    dot: 'bg-emerald-500',
  },
  error: {
    title: 'เกิดข้อผิดพลาด',
    badge: 'bg-red-50 text-red-700',
    dot: 'bg-red-500',
  },
  warning: {
    title: 'แจ้งเตือน',
    badge: 'bg-amber-50 text-amber-700',
    dot: 'bg-amber-500',
  },
  info: {
    title: 'แจ้งข้อมูล',
    badge: 'bg-zinc-100 text-zinc-700',
    dot: 'bg-zinc-400',
  },
};

function ResponseModal({
  type = 'info',
  title,
  description,
  confirmLabel = 'ตกลง',
  onClose,
}) {
  const style =
    RESPONSE_STYLE[type] ||
    RESPONSE_STYLE.info;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4">
      <div className="w-full max-w-sm rounded-xl bg-white p-6 shadow-2xl">

        <div
          className={`inline-flex items-center gap-2 rounded-full px-2.5 py-1 text-xs font-medium ${style.badge}`}
        >
          <span
            className={`h-1.5 w-1.5 rounded-full ${style.dot}`}
          />

          {style.title}
        </div>

        <h3 className="mt-4 text-base font-semibold text-zinc-800">
          {title}
        </h3>

        {description && (
          <div className="mt-2 text-sm leading-6 text-zinc-500">
            {description}
          </div>
        )}

        <div className="mt-6 flex justify-end">
          <button
            type="button"
            onClick={onClose}
            className="h-9 rounded-lg bg-red-600 px-4 text-sm font-medium text-white hover:bg-red-700"
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}

export default ResponseModal;