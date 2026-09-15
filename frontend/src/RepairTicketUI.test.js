import { render, screen } from '@testing-library/react';
import RepairTicketPage from './pages/RepairTicketPage';

describe('Repair ticket UI checklist (T-24)', () => {
  it('shows the repair ticket list and create action for a user', () => {
    render(
      <RepairTicketPage
        user={{ id: 2, username: 'service01', role: 'user' }}
        initialState="success"
      />
    );

    expect(
      screen.getByRole('heading', {
        name: /รายการแจ้งซ่อม/i,
      })
    ).toBeInTheDocument();

    expect(
      screen.getByRole('button', {
        name: /เพิ่มใบแจ้งซ่อม/i,
      })
    ).toBeInTheDocument();

    expect(
      screen.queryByRole('button', {
        name: /อัปเดตสถานะ/i,
      })
    ).not.toBeInTheDocument();
  });

  it('shows the status update action only for admin and system admin', () => {
    const { rerender } = render(
      <RepairTicketPage
        user={{ id: 1, username: 'admin', role: 'admin' }}
        initialState="success"
      />
    );

    expect(
      screen.getByRole('button', {
        name: /อัปเดตสถานะ/i,
      })
    ).toBeInTheDocument();

    rerender(
      <RepairTicketPage
        user={{ id: 3, username: 'root', role: 'system_admin' }}
        initialState="success"
      />
    );

    expect(
      screen.getByRole('button', {
        name: /อัปเดตสถานะ/i,
      })
    ).toBeInTheDocument();
  });
});
