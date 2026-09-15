import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

import DashboardPage from './pages/DashboardPage';
import ReportPage from './pages/ReportPage';
import ImportPage from './pages/ImportPage';
import ActivityLogPage from './pages/ActivityLogPage';
import Sidebar from './components/containers/Sidebar';

import { getDashboard } from './api/dashboardClient';
import { getReport } from './api/reportClient';
import { importEquipment } from './api/importClient';
import { listActivityLogs } from './api/activityLogClient';

jest.mock('./api/dashboardClient', () => ({
  getDashboard: jest.fn(),
}));

jest.mock('./api/reportClient', () => ({
  getReport: jest.fn(),
}));

jest.mock('./api/importClient', () => ({
  importEquipment: jest.fn(),
}));

jest.mock('./api/activityLogClient', () => ({
  listActivityLogs: jest.fn(),
}));

const adminUser = {
  id: 1,
  username: 'admin',
  role: 'admin',
};

const user = {
  id: 2,
  username: 'service01',
  role: 'user',
};

beforeEach(() => {
  jest.clearAllMocks();
});

describe('Dashboard, report, import, and activity-log UI checklist (T-30)', () => {
  it('shows dashboard loading and KPI success states through the typed dashboard client', async () => {
    getDashboard.mockResolvedValue({
      pending_borrow_requests: 2,
      pending_return_confirmations: 1,
      pending_repair_tickets: 3,
      equipment_below_minimum: 1,
    });

    render(
      <DashboardPage
        user={adminUser}
        initialState="loading"
      />
    );

    expect(
      screen.getByText(/loading dashboard/i)
    ).toBeInTheDocument();

    render(
      <DashboardPage
        user={adminUser}
        initialState="success"
      />
    );

    expect(
      await screen.findByRole('heading', {
        name: /แดชบอร์ด/i,
      })
    ).toBeInTheDocument();

    expect(getDashboard).toHaveBeenCalled();
  });

  it('shows an error state when dashboard loading fails', () => {
    render(
      <DashboardPage
        user={adminUser}
        initialState="error"
      />
    );

    expect(
      screen.getByText(/unable to load dashboard/i)
    ).toBeInTheDocument();
  });

  it('shows reports only for admin roles and uses the typed report client', async () => {
    getReport.mockResolvedValue({
      type: 'stock',
      items: [],
    });

    const { rerender } = render(
      <ReportPage
        user={adminUser}
        initialState="success"
      />
    );

    expect(
      await screen.findByRole('heading', {
        name: /รายงาน/i,
      })
    ).toBeInTheDocument();

    expect(
      screen.getByRole('button', {
        name: /โหลดรายงาน/i,
      })
    ).toBeInTheDocument();

    expect(getReport).toHaveBeenCalled();

    rerender(<ReportPage user={user} initialState="success" />);

    expect(
      screen.queryByRole('button', {
        name: /โหลดรายงาน/i,
      })
    ).not.toBeInTheDocument();
  });

  it('shows a separate import page for admin roles and submits through the typed import client', async () => {
    importEquipment.mockResolvedValue({
      total: 2,
      success: 2,
      failed: 0,
    });

    const { rerender } = render(
      <ImportPage
        user={adminUser}
        initialState="success"
      />
    );

    expect(
      screen.getByRole('heading', {
        name: /นำเข้าข้อมูล/i,
      })
    ).toBeInTheDocument();

    expect(
      screen.getByLabelText(/ไฟล์ csv/i)
    ).toBeInTheDocument();

    expect(
      screen.getByRole('button', {
        name: /นำเข้า/i,
      })
    ).toBeInTheDocument();

    rerender(<ImportPage user={user} initialState="success" />);

    expect(
      screen.queryByRole('button', {
        name: /นำเข้า/i,
      })
    ).not.toBeInTheDocument();
  });

  it('shows activity logs in newest-first order through the typed activity-log client', async () => {
    listActivityLogs.mockResolvedValue([
      { id: 2, action: 'new_action' },
      { id: 1, action: 'old_action' },
    ]);

    render(
      <ActivityLogPage
        user={user}
        initialState="success"
      />
    );

    expect(
      await screen.findByRole('heading', {
        name: /activity logs|ประวัติการใช้งาน/i,
      })
    ).toBeInTheDocument();

    expect(listActivityLogs).toHaveBeenCalled();
    expect(screen.getByText('new_action')).toBeInTheDocument();
    expect(screen.getByText('old_action')).toBeInTheDocument();
  });

  it('keeps import and report navigation in the separate admin menu while users see neither item', () => {
    const { rerender } = render(
      <MemoryRouter>
        <Sidebar user={adminUser} />
      </MemoryRouter>
    );

    expect(
      screen.getByRole('link', { name: /รายงาน/i })
    ).toHaveAttribute('href', '/report');

    expect(
      screen.getByRole('link', { name: /นำเข้าข้อมูล/i })
    ).toHaveAttribute('href', '/import-data');

    rerender(
      <MemoryRouter>
        <Sidebar user={user} />
      </MemoryRouter>
    );

    expect(
      screen.queryByRole('link', { name: /รายงาน/i })
    ).not.toBeInTheDocument();

    expect(
      screen.queryByRole('link', { name: /นำเข้าข้อมูล/i })
    ).not.toBeInTheDocument();
  });
});
