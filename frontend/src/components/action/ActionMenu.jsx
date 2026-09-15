import React, { useEffect, useRef, useState } from 'react';
import {
  EllipsisVerticalIcon,
  EyeIcon,
  PencilSquareIcon,
  TrashIcon,
} from '@heroicons/react/24/outline';

function ActionMenu({
  item,
  canManage,
  onView,
  onEdit,
  onDelete,
}) {
  const [open, setOpen] = useState(false);
  const [position, setPosition] = useState({
    top: 0,
    right: 0,
  });

  const buttonRef = useRef(null);
  const menuRef = useRef(null);

  function handleToggle() {
    if (!open && buttonRef.current) {
      const rect =
        buttonRef.current.getBoundingClientRect();

      setPosition({
        top: rect.bottom + 6,
        right: window.innerWidth - rect.right,
      });
    }

    setOpen((prev) => !prev);
  }

  useEffect(() => {
    function handleClickOutside(event) {
      if (
        menuRef.current &&
        !menuRef.current.contains(event.target) &&
        buttonRef.current &&
        !buttonRef.current.contains(event.target)
      ) {
        setOpen(false);
      }
    }

    function handleClose() {
      setOpen(false);
    }

    document.addEventListener(
      'mousedown',
      handleClickOutside
    );

    window.addEventListener(
      'resize',
      handleClose
    );

    window.addEventListener(
      'scroll',
      handleClose,
      true
    );

    return () => {
      document.removeEventListener(
        'mousedown',
        handleClickOutside
      );

      window.removeEventListener(
        'resize',
        handleClose
      );

      window.removeEventListener(
        'scroll',
        handleClose,
        true
      );
    };
  }, []);

  return (
    <>
      <button
        ref={buttonRef}
        type="button"
        onClick={handleToggle}
        aria-label="actions"
        className="rounded-lg p-2 text-zinc-400 transition hover:bg-zinc-100 hover:text-zinc-700"
      >
        <EllipsisVerticalIcon className="h-5 w-5" />
      </button>

      {open && (
        <div
          ref={menuRef}
          className="fixed z-[9999] w-48 overflow-hidden rounded-xl border border-zinc-200 bg-white py-1.5 shadow-xl"
          style={{
            top: position.top,
            right: position.right,
          }}
        >
          <button
            type="button"
            onClick={() => {
              setOpen(false);
              onView(item);
            }}
            className="flex w-full items-center gap-3 px-4 py-2.5 text-left text-sm text-zinc-700 transition hover:bg-zinc-50"
          >
            <EyeIcon className="h-4 w-4 text-zinc-400" />
            ดูรายละเอียด
          </button>

          {canManage && (
            <>
              <button
                type="button"
                onClick={() => {
                  setOpen(false);
                  onEdit(item);
                }}
                className="flex w-full items-center gap-3 px-4 py-2.5 text-left text-sm text-zinc-700 transition hover:bg-zinc-50"
              >
                <PencilSquareIcon className="h-4 w-4 text-zinc-400" />
                แก้ไขข้อมูล
              </button>

              <div className="my-1 border-t border-zinc-100" />

              <button
                type="button"
                onClick={() => {
                  setOpen(false);
                  onDelete(item);
                }}
                className="flex w-full items-center gap-3 px-4 py-2.5 text-left text-sm text-red-600 transition hover:bg-red-50"
              >
                <TrashIcon className="h-4 w-4" />
                ลบอุปกรณ์
              </button>
            </>
          )}
        </div>
      )}
    </>
  );
}

export default ActionMenu;