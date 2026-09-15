import React, { useEffect, useState, useCallback } from 'react';
import { getDashboard } from '../api/dashboardClient';

// ---------------------------------------------------------------------------
// Inline icons — plain SVG, no external icon package required.
// ---------------------------------------------------------------------------
const iconProps = (className) => ({
  className,
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 2,
  strokeLinecap: 'round',
  strokeLinejoin: 'round',
  'aria-hidden': 'true',
});

const BarChartIcon = ({ className }) => (
  <svg {...iconProps(className)}>
    <line x1="12" y1="20" x2="12" y2="10" />
    <line x1="18" y1="20" x2="18" y2="4" />
    <line x1="6" y1="20" x2="6" y2="16" />
  </svg>
);

const PieChartIcon = ({ className }) => (
  <svg {...iconProps(className)}>
    <path d="M21.21 15.89A10 10 0 1 1 8 2.83" />
    <path d="M22 12A10 10 0 0 0 12 2v10z" />
  </svg>
);

const ActivityIcon = ({ className }) => (
  <svg {...iconProps(className)}>
    <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
  </svg>
);

const ClockIcon = ({ className }) => (
  <svg {...iconProps(className)}>
    <circle cx="12" cy="12" r="10" />
    <polyline points="12 6 12 12 16 14" />
  </svg>
);

// ---------------------------------------------------------------------------
// Design tokens — lifted 1:1 from the EasyBuy mockup's :root custom
// properties so the React page matches the HTML mockup exactly.
// ---------------------------------------------------------------------------
const COLORS = {
  bg: '#F4F4F5',
  surface: '#FFFFFF',
  surfaceAlt: '#FAFAFA',
  border: '#E4E4E7',
  text: '#27272A',
  text2: '#71717A',
  textMuted: '#A1A1AA',
  red: '#D6272C',
  redDark: '#B0181D',
  redBg: '#FDEBEC',
  yellow: '#F5A623',
  yellowDark: '#92600A',
  yellowBg: '#FDF3E0',
  grayBadgeBg: '#F0F0F1',
  grayBadgeText: '#52525B',
  green: '#1F8A4C',
  greenBg: '#E9F6EE',
};

const ACCENTS = {
  red: { border: COLORS.red, num: COLORS.redDark },
  yellow: { border: COLORS.yellow, num: COLORS.yellowDark },
  green: { border: COLORS.green, num: COLORS.green },
  gray: { border: COLORS.grayBadgeText, num: COLORS.text },
};

// key, label, accent, page this KPI drills into
const KPI_FIELDS = [
  ['pending_borrow_requests', 'คำขอเบิกรออนุมัติ', 'yellow', 'borrow'],
  ['pending_return_confirmations', 'รายการคืนรอยืนยัน', 'yellow', 'borrow'],
  ['pending_repair_tickets', 'แจ้งซ่อมรอดำเนินการ', 'yellow', 'repair'],
  ['equipment_below_minimum', 'อุปกรณ์ต่ำกว่าระดับขั้นต่ำ', 'red', 'reports'],
  ['total_equipment', 'อุปกรณ์ทั้งหมด', 'red', 'equipment'],
  ['equipment_in_use', 'อุปกรณ์ที่ถูกเบิก', 'gray', 'borrow'],
  ['equipment_under_repair', 'อุปกรณ์อยู่ระหว่างซ่อม', 'yellow', 'repair'],
  ['successful_borrows', 'รายการเบิกสำเร็จ', 'green', 'borrow'],
  ['under_repair', 'รายการซ่อมของฉัน', 'yellow', 'repair'],
  ['in_progress_requests', 'คำขอที่กำลังดำเนินการ', 'yellow', 'borrow'],
];

const ROLE_KEYS = {
  user: ['pending_borrow_requests', 'successful_borrows', 'under_repair', 'in_progress_requests'],
  system_admin: ['total_equipment', 'equipment_in_use', 'equipment_under_repair', 'equipment_below_minimum'],
  admin: ['total_equipment', 'pending_borrow_requests', 'pending_return_confirmations', 'pending_repair_tickets', 'equipment_below_minimum'],
};

const ROLE_LABELS = {
  user: 'ผู้ใช้งาน',
  admin: 'ผู้ดูแลระบบ',
  system_admin: 'ผู้ดูแลระบบสูงสุด',
};

// ---------------------------------------------------------------------------
// Mocked chart data — stands in for real analytics until those endpoints
// exist. Shapes mirror the mockup's bar chart / donut widgets.
// ---------------------------------------------------------------------------
const EQUIPMENT_BY_TYPE = [
  { label: 'Thin Client', pct: 35 },
  { label: 'Notebook', pct: 44 },
  { label: 'Card reader', pct: 52 },
  { label: 'Switch', pct: 60 },
  { label: 'Printer', pct: 70 },
  { label: 'Monitor', pct: 80 },
  { label: 'Server', pct: 90 },
  { label: 'Router', pct: 94 },
];

const BORROW_SPLIT = [
  { label: 'เบิกถาวร', pct: 63, color: COLORS.red },
  { label: 'เบิกชั่วคราว', pct: 37, color: COLORS.yellow },
];

const REPAIR_SPLIT = [
  { label: 'กำลังดำเนินการซ่อม', pct: 56, color: COLORS.yellow },
  { label: 'ซ่อมเสร็จแล้ว', pct: 44, color: COLORS.green },
];

const ACTIVITY_LOG = [
  { title: 'มีการเบิกอุปกรณ์ Notebook 2 เครื่อง — โดย admin02', time: '10/06/2569 · 23:20', dot: COLORS.grayBadgeText },
  { title: 'แจ้งซ่อม Switch SW-014 — โดย service01', time: '10/06/2569 · 21:05', dot: COLORS.yellow },
  { title: 'อนุมัติคำขอเบิก Dell Latitude 5420 — โดย admin01', time: '09/06/2569 · 16:42', dot: COLORS.green },
];

// ---------------------------------------------------------------------------
// Small presentational building blocks
// ---------------------------------------------------------------------------
function Card({ title, icon, children, className = '' }) {
  return (
    <div
      className={`rounded-xl bg-white p-4 shadow-sm sm:p-5 ${className}`}
      style={{ border: `1px solid ${COLORS.border}` }}
    >
      {title && (
        <p className="mb-4 flex items-center gap-1.5 text-[13px] font-medium" style={{ color: COLORS.text2 }}>
          {icon}
          {title}
        </p>
      )}
      {children}
    </div>
  );
}

function KpiCard({ label, value, accent, onClick }) {
  const style = ACCENTS[accent] || ACCENTS.gray;
  const interactive = typeof onClick === 'function';

  return (
    <div
      role={interactive ? 'button' : undefined}
      tabIndex={interactive ? 0 : undefined}
      aria-label={interactive ? `ดูรายละเอียด ${label}` : undefined}
      onClick={interactive ? onClick : undefined}
      onKeyDown={
        interactive
          ? (e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                onClick();
              }
            }
          : undefined
      }
      className={`group rounded-r-xl bg-white py-3.5 pl-4 pr-4 transition-all duration-150 ${
        interactive
          ? 'cursor-pointer hover:shadow-md hover:-translate-y-0.5 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-400 focus-visible:ring-offset-1'
          : ''
      }`}
      style={{ borderLeft: `4px solid ${style.border}` }}
    >
      <p className="text-xs" style={{ color: COLORS.text2 }}>
        {label}
      </p>
      <p className="mt-2 text-2xl font-semibold" style={{ color: style.num }}>
        {value}
      </p>
      {interactive && (
        <p
          className="mt-1.5 flex items-center gap-1 text-[11px] transition-colors group-hover:opacity-100"
          style={{ color: COLORS.textMuted }}
        >
          ดูรายละเอียด <span aria-hidden="true">→</span>
        </p>
      )}
    </div>
  );
}

// SVG-based donut so each segment's arc can be "drawn in" on mount by
// animating stroke-dasharray from 0 to its full length (stroke-dashoffset
// stays fixed, so the arc's start position never moves — only its length
// grows). Because this component only exists in the tree while
// `state === 'success'` (see DashboardPage below), it unmounts while
// loading and mounts fresh on every successful load, so the sweep-in
// replays every time the dashboard (re)loads without any extra key/reset
// plumbing.
function Donut({ segments, size = 80 }) {
  const [progress, setProgress] = useState(false);

  useEffect(() => {
    // Two rAFs: the first lets the browser paint the 0-length state,
    // the second flips the flag so the CSS transition actually animates
    // from that painted 0 state instead of jumping straight to the target.
    let raf2;
    const raf1 = requestAnimationFrame(() => {
      raf2 = requestAnimationFrame(() => setProgress(true));
    });
    return () => {
      cancelAnimationFrame(raf1);
      if (raf2) cancelAnimationFrame(raf2);
    };
  }, []);

  const strokeWidth = size * 0.175;
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;

  let cumulativePct = 0;
  const arcs = segments.map((s, i) => {
    const startPct = cumulativePct;
    cumulativePct += s.pct;
    const fullLength = (s.pct / 100) * circumference;
    return {
      ...s,
      dashoffset: -((startPct / 100) * circumference),
      dasharray: progress ? `${fullLength} ${circumference - fullLength}` : `0 ${circumference}`,
      delay: i * 150,
    };
  });

  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      className="flex-shrink-0 -rotate-90 transition-transform duration-300 hover:scale-110"
    >
      <circle
        cx={size / 2}
        cy={size / 2}
        r={radius}
        fill="none"
        stroke={COLORS.border}
        strokeWidth={strokeWidth}
        opacity={0.35}
      />
      {arcs.map((a) => (
        <circle
          key={a.label}
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke={a.color}
          strokeWidth={strokeWidth}
          strokeDasharray={a.dasharray}
          strokeDashoffset={a.dashoffset}
          style={{
            transition: 'stroke-dasharray 0.9s cubic-bezier(0.34, 1.56, 0.64, 1)',
            transitionDelay: `${a.delay}ms`,
          }}
        />
      ))}
    </svg>
  );
}

function DonutCard({ title, segments }) {
  return (
    <Card title={title} icon={<PieChartIcon className="h-3.5 w-3.5 flex-shrink-0" />}>
      <div className="flex items-center justify-center gap-6 sm:justify-start">
        <Donut segments={segments} size={92} />
        <div className="flex flex-col gap-1 text-xs">
          {segments.map((s) => (
            <span
              key={s.label}
              className="-mx-1.5 flex cursor-default items-center gap-2 rounded-md px-1.5 py-1 transition-colors hover:bg-zinc-50"
              style={{ color: COLORS.text }}
            >
              <span
                className="inline-block h-2.5 w-2.5 flex-shrink-0 rounded-full"
                style={{ background: s.color }}
                aria-hidden="true"
              />
              <span>
                {s.label} <span className="font-semibold">· {s.pct}%</span>
              </span>
            </span>
          ))}
        </div>
      </div>
    </Card>
  );
}

// Same mount-triggered animation pattern as Donut above: bars start at
// height 0 and transition up to their real percentage right after mount,
// staggered per bar so they visibly rise up from the bottom left-to-right.
// Unmounts/remounts with the rest of the "success" block, so it replays on
// every dashboard load.
function EquipmentBarChart({ className = '' }) {
  const [grown, setGrown] = useState(false);

  useEffect(() => {
    const raf1 = requestAnimationFrame(() => {
      requestAnimationFrame(() => setGrown(true));
    });
    return () => cancelAnimationFrame(raf1);
  }, []);

  return (
    <Card
      title="จำนวนอุปกรณ์ในแต่ละประเภท"
      icon={<BarChartIcon className="h-3.5 w-3.5 flex-shrink-0" />}
      className={`flex h-full flex-col ${className}`}
    >
      <div className="flex flex-1 flex-col justify-center overflow-x-auto">
        <div
          className="flex items-end justify-between gap-2 sm:gap-3"
          style={{ minWidth: 380, height: 160, borderBottom: `1px solid ${COLORS.border}`, paddingBottom: 4 }}
        >
          {EQUIPMENT_BY_TYPE.map((b, i) => (
            <div key={b.label} className="group flex h-full flex-1 cursor-default flex-col items-center justify-end">
              <span
                className="mb-1.5 scale-75 rounded-full px-1.5 py-0.5 text-[10px] font-semibold opacity-0 transition-all duration-200 group-hover:scale-100 group-hover:opacity-100"
                style={{ background: COLORS.redBg, color: COLORS.redDark, transitionTimingFunction: 'cubic-bezier(0.34, 1.56, 0.64, 1)' }}
              >
                {b.pct}%
              </span>
              <div
                className="w-full max-w-[34px] origin-bottom rounded-t-md group-hover:scale-x-125 group-hover:shadow-lg sm:max-w-[44px]"
                style={{
                  height: `${grown ? b.pct : 0}%`,
                  background: COLORS.red,
                  opacity: 0.85,
                  transitionProperty: 'height, transform, box-shadow',
                  transitionDuration: '700ms, 300ms, 300ms',
                  transitionTimingFunction: 'cubic-bezier(0.34, 1.56, 0.64, 1)',
                  transitionDelay: `${i * 60}ms, 0ms, 0ms`,
                }}
                title={`${b.label} · ${b.pct}%`}
              />
            </div>
          ))}
        </div>
        <div className="mt-2 flex justify-between gap-2 sm:gap-3" style={{ minWidth: 380 }}>
          {EQUIPMENT_BY_TYPE.map((b) => (
            <span
              key={b.label}
              className="flex-1 truncate text-center text-[10px] sm:text-[11px]"
              style={{ color: COLORS.textMuted }}
            >
              {b.label}
            </span>
          ))}
        </div>
      </div>
    </Card>
  );
}

function ActivityLogCard() {
  return (
    <Card title="Activity logs" icon={<ActivityIcon className="h-3.5 w-3.5 flex-shrink-0" />}>
      <ul className="flex flex-col">
        {ACTIVITY_LOG.map((item, i) => (
          <li
            key={i}
            className="flex items-start gap-2.5 py-2"
            style={{ borderTop: i === 0 ? 'none' : `1px solid ${COLORS.border}` }}
          >
            <span
              className="mt-1.5 h-1.5 w-1.5 flex-shrink-0 rounded-full"
              style={{ background: item.dot }}
              aria-hidden="true"
            />
            <div className="min-w-0">
              <p className="truncate text-[13px] font-medium" style={{ color: COLORS.text }}>
                {item.title}
              </p>
              <p className="text-xs" style={{ color: COLORS.text2 }}>
                {item.time}
              </p>
            </div>
          </li>
        ))}
      </ul>
    </Card>
  );
}

function KpiSkeleton({ count = 5 }) {
  return (
    <div
      className="grid gap-3"
      style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))' }}
    >
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="animate-pulse rounded-xl bg-white p-4" style={{ border: `1px solid ${COLORS.border}` }}>
          <div className="h-3 w-20 rounded bg-zinc-200" />
          <div className="mt-3 h-6 w-12 rounded bg-zinc-200" />
        </div>
      ))}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
function DashboardPage({ user, initialState = 'loading', onNavigate }) {
  const [state, setState] = useState(initialState);
  const [data, setData] = useState({});
  const [updatedAt, setUpdatedAt] = useState(null);

  const load = useCallback(() => {
    let cancelled = false;
    setState('loading');
    getDashboard()
      .then((result) => {
        if (cancelled) return;
        setData(result || {});
        setUpdatedAt(new Date());
        setState('success');
      })
      .catch(() => {
        if (!cancelled) setState('error');
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (initialState === 'error') return undefined;
    return load();
  }, [initialState, load]);

  const role = user?.role || 'user';
  const roleKeys = ROLE_KEYS[role] || ROLE_KEYS.user;
  const fields = KPI_FIELDS.filter(([key]) => roleKeys.includes(key));

  const goTo = (page) => {
    if (typeof onNavigate === 'function') onNavigate(page);
  };

  if (state === 'error') {
    return (
      <div className="min-h-screen w-full p-4 sm:p-6" style={{ background: COLORS.bg }}>
        <div className="w-full">
          <div
            className="rounded-xl bg-white p-6 sm:p-8"
            style={{ border: `1px solid ${COLORS.redBg}` }}
          >
            <p className="text-sm font-medium" style={{ color: COLORS.redDark }}>
              โหลดข้อมูลแดชบอร์ดไม่สำเร็จ
            </p>
            <p className="mt-1 text-xs" style={{ color: COLORS.text2 }}>
              ตรวจสอบการเชื่อมต่อของคุณแล้วลองอีกครั้ง
            </p>
            <button
              type="button"
              onClick={load}
              className="mt-4 rounded-lg px-4 py-2 text-sm font-medium text-white"
              style={{ background: COLORS.red }}
            >
              ลองอีกครั้ง
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen w-full p-4 sm:p-6" style={{ background: COLORS.bg, color: COLORS.text }}>
      <main className="w-full">
        <div className="mb-5 flex flex-wrap items-end justify-between gap-2">
          <div>
            <p className="text-xs" style={{ color: COLORS.textMuted }}>
              Portal · สวัสดี, {user?.username || 'ผู้ใช้งาน'}
              {user?.org ? ` · ${user.org}` : ''}
            </p>
          </div>
          {updatedAt && state === 'success' && (
            <span className="flex items-center gap-1 text-[11px]" style={{ color: COLORS.textMuted }}>
              <ClockIcon className="h-3 w-3" />
              อัปเดตล่าสุด {updatedAt.toLocaleTimeString('th-TH', { hour: '2-digit', minute: '2-digit' })}
            </span>
          )}
        </div>

        {state === 'loading' ? (
          <KpiSkeleton count={fields.length || 5} />
        ) : fields.length === 0 ? (
          <div className="rounded-xl bg-white p-6 text-center" style={{ border: `1px solid ${COLORS.border}` }}>
            <p className="text-sm" style={{ color: COLORS.text2 }}>
              ไม่มีข้อมูลสรุปสำหรับบทบาทของคุณ
            </p>
          </div>
        ) : (
          <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))' }}>
            {fields.map(([key, label, accent, page]) => (
              <KpiCard
                key={key}
                label={label}
                value={data[key] ?? 0}
                accent={accent}
                onClick={() => goTo(page)}
              />
            ))}
          </div>
        )}

        {state === 'success' && (
          // Bar chart column is narrower relative to donut column than a
          // plain 2:1 split (xl:grid-cols-3 + col-span-2/1) would give, and
          // xl:items-stretch makes both columns match the tallest one so the
          // bar chart card grows to equal the combined height of the two
          // stacked donut cards. Tune `2.6fr` to taste.
          //
          // This whole block (and therefore EquipmentBarChart/DonutCard) is
          // only in the tree while state === 'success', so it unmounts
          // while loading and mounts fresh every time load() succeeds —
          // that's what makes the entrance animations replay on every
          // dashboard load without extra reset logic.
          <div className="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-[2.6fr_1fr] xl:items-stretch">
            <EquipmentBarChart />
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-1">
              <DonutCard title="อุปกรณ์ถูกยืม" segments={BORROW_SPLIT} />
              <DonutCard title="อุปกรณ์ส่งซ่อม" segments={REPAIR_SPLIT} />
            </div>
            <div className="xl:col-span-2">
              <ActivityLogCard />
            </div>
          </div>
        )}
      </main>
    </div>
  );
}

export default DashboardPage;
