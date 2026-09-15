import React, { useEffect, useState } from 'react';
import { ClipboardDocumentListIcon } from '@heroicons/react/24/outline';
import { listActivityLogs } from '../api/activityLogClient';

function ActivityLogPage({ initialState = 'loading' }) {
  const [state, setState] = useState(initialState);
  const [logs, setLogs] = useState([]);

  useEffect(() => {
    if (initialState === 'error') return undefined;

    let cancelled = false;
    setState('loading');
    listActivityLogs()
      .then((items) => {
        if (!cancelled) {
          setLogs(Array.isArray(items) ? items : []);
          setState('success');
        }
      })
      .catch(() => {
        if (!cancelled) setState('error');
      });

    return () => {
      cancelled = true;
    };
  }, [initialState]);

  if (state === 'loading') return <div className="p-6 text-sm text-zinc-500">Loading activity logs...</div>;
  if (state === 'error') return <div className="p-6 text-sm text-red-700">Unable to load activity logs</div>;

  return (
    <div className="p-6">
      <div className="mb-5 flex items-center gap-3">
        <ClipboardDocumentListIcon className="h-6 w-6 text-red-600" />
        <h2 className="text-xl font-semibold text-zinc-800">ประวัติการใช้งาน</h2>
      </div>
      <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white">
        {logs.length === 0 ? <p className="p-5 text-sm text-zinc-500">ไม่มีประวัติการใช้งาน</p> : logs.map((log) => (
          <div key={log.id} className="border-b border-zinc-100 px-5 py-3 last:border-b-0">
            <p className="text-sm font-medium text-zinc-800">{log.action}</p>
            <p className="mt-1 text-xs text-zinc-500">{log.resource_type} #{log.resource_id}</p>
          </div>
        ))}
      </div>
    </div>
  );
}

export default ActivityLogPage;
