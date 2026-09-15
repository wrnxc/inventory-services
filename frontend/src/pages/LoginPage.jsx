import React, { useEffect, useRef, useState } from 'react';
import { login } from '../api/authClient';
import { UserIcon, LockClosedIcon, ExclamationCircleIcon } from '@heroicons/react/24/outline';
import { CheckIcon } from '@heroicons/react/24/solid';
import './LoginPage.css';

export default function LoginPage({ onLoginSuccess }) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [successFlash, setSuccessFlash] = useState(false);

  const userRef = useRef(null);
  const timerRef = useRef(null);

  useEffect(() => {
    userRef.current?.focus();

    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, []);

  async function handleSubmit(event) {
    event.preventDefault();

    if (!username.trim() || !password) {
      setErrorMessage('กรุณากรอกชื่อผู้ใช้และรหัสผ่าน');
      userRef.current?.focus();
      return;
    }
    setIsSubmitting(true);
    setErrorMessage('');

    try {
      const result = await login(username.trim(), password);
      
      setSuccessFlash(true);

      timerRef.current = setTimeout(() => {
        onLoginSuccess?.(result?.user || null);
      }, 400);
    } catch (error) {
      setErrorMessage(
        error?.message ||
          'เข้าสู่ระบบไม่สำเร็จ กรุณาลองใหม่อีกครั้ง'
      );

      setIsSubmitting(false);
    }
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-neutral-50 px-4 py-10 font-[Sarabun,sans-serif]">
      <div className="relative w-full max-w-[392px]">
        {/* Brand */}
        <div className="mb-8 flex flex-col items-center text-center">
          <div className="relative mb-4">
            <div className="flex h-[72px] w-[72px] items-center justify-center rounded-2xl bg-white shadow-lg ring-1 ring-stone-200">
              <img
                src="/images/easybuy-logo.png"
                alt="EasyBuy"
                className="h-11 w-auto object-contain"
              />
            </div>
          </div>

          <h1 className="text-[22px] font-bold tracking-wide text-stone-900">
            EasyBuy Inventory
          </h1>

          <p className="mt-1.5 text-sm text-stone-500">
            ระบบจัดการอุปกรณ์สำหรับแผนก Service
          </p>
        </div>

        {/* Login Card */}
        <div className="login-fade-in overflow-hidden rounded-2xl border border-stone-200 bg-white shadow-xl">
          {/* Top accent */}
          <div className="h-[3px] w-full bg-gradient-to-r from-red-600 via-red-600 to-amber-400" />

          <div className="px-6 pb-6 pt-5">
            {successFlash ? (
              /* Success */
              <div className="flex flex-col items-center py-6 text-center">
                <div className="login-pop-in mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-emerald-50 ring-8 ring-emerald-50/60">
                  <CheckIcon
                    className="h-6 w-6 text-emerald-600"
                    strokeWidth={2.5}
                  />
                </div>

                <p className="text-[15px] font-semibold text-stone-900">
                  เข้าสู่ระบบสำเร็จ
                </p>

                <p className="mt-1 text-xs text-stone-400">
                  กำลังพาไปหน้าหลัก…
                </p>
              </div>
            ) : (
              <>
                {/* Header */}
                <div className="mb-6 flex items-baseline justify-between">
                  <div>
                    <h2
                      aria-label="Login"
                      className="text-[17px] font-semibold text-stone-900"
                    >
                      เข้าสู่ระบบ
                    </h2>
                  </div>

                  <span className="shrink-0 rounded-full bg-amber-50 px-2.5 py-1 text-[10.5px] font-semibold tracking-wide text-amber-700">
                    SVC
                  </span>
                </div>

                <form
                  onSubmit={handleSubmit}
                  noValidate
                  className="space-y-4"
                >
                  {/* Username */}
                  <div>
                    <label
                      htmlFor="username"
                      className="mb-1.5 block text-xs font-medium text-stone-600"
                    >
                      ชื่อผู้ใช้
                    </label>

                    <div className="relative">
                      <UserIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-stone-400" />

                      <input
                        id="username"
                        ref={userRef}
                        name="username"
                        type="text"
                        autoComplete="username"
                        value={username}
                        disabled={isSubmitting}
                        onChange={(event) =>
                          setUsername(event.target.value)
                        }
                        placeholder="กรอกชื่อผู้ใช้"
                        className="w-full rounded-xl border border-stone-200 bg-white py-2.5 pl-9 pr-3 text-[13.5px] text-stone-900 outline-none transition placeholder:text-stone-400 focus:border-red-600 focus:ring-4 focus:ring-red-100 disabled:bg-stone-50 disabled:text-stone-400"
                      />
                    </div>
                  </div>

                  {/* Password */}
                  <div>
                    <label
                      htmlFor="password"
                      className="mb-1.5 block text-xs font-medium text-stone-600"
                    >
                      รหัสผ่าน
                    </label>

                    <div className="relative">
                      <LockClosedIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-stone-400" />

                      <input
                        id="password"
                        name="password"
                        type="password"
                        autoComplete="current-password"
                        value={password}
                        disabled={isSubmitting}
                        onChange={(event) =>
                          setPassword(event.target.value)
                        }
                        placeholder="••••••••"
                        className="w-full rounded-xl border border-stone-200 bg-white py-2.5 pl-9 pr-3 text-[13.5px] text-stone-900 outline-none transition placeholder:text-stone-400 focus:border-red-600 focus:ring-4 focus:ring-red-100 disabled:bg-stone-50 disabled:text-stone-400"
                      />
                    </div>
                  </div>

                  {/* Error */}
                  <div
                    role="alert"
                    aria-live="polite"
                    className="min-h-[1px]"
                  >
                    {errorMessage && (
                      <div className="login-slide-down flex items-start gap-2 rounded-xl bg-red-50 px-3 py-2.5 text-xs text-red-700">
                        <ExclamationCircleIcon className="mt-0.5 h-3.5 w-3.5 flex-shrink-0" />

                        <span>{errorMessage}</span>
                      </div>
                    )}
                  </div>

                  {/* Submit */}
                  <button
                    type="submit"
                    disabled={isSubmitting}
                    className="flex h-11 w-full items-center justify-center gap-2 rounded-xl border border-red-600 bg-red-600 text-[13.5px] font-medium text-white shadow-sm transition hover:enabled:border-red-700 hover:enabled:bg-red-700 hover:enabled:shadow-md active:enabled:scale-[0.99] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500 disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    {isSubmitting ? (
                      <>
                        <span className="h-[13px] w-[13px] animate-spin rounded-full border-2 border-white/45 border-t-white" />
                        กำลังเข้าสู่ระบบ…
                      </>
                    ) : (
                      'เข้าสู่ระบบ'
                    )}
                  </button>
                </form>
              </>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="mt-6 flex items-center justify-center gap-2 text-[11px] tracking-wide text-stone-400">
          <span
            aria-hidden="true"
            className="h-3 w-px bg-stone-300"
          />

          <span>EBI · SERVICE DEPARTMENT</span>

          <span
            aria-hidden="true"
            className="h-3 w-px bg-stone-300"
          />
        </div>
      </div>
    </div>
  );
}