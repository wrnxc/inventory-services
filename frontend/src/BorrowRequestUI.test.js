import { render, screen } from '@testing-library/react';
import EquipmentPage from './pages/EquipmentPage';

describe('Borrow request UI checklist (T-20)', () => {
  it('shows return action for user and hides it for admin', () => {
    render(
      <EquipmentPage
        user={{ id: 2, username: 'service01', role: 'user' }}
        initialState="success"
      />
    );

    expect(
      screen.getByRole('button', { name: /คืนอุปกรณ์/i })
    ).toBeInTheDocument();

    expect(
      screen.queryByRole('button', { name: /รับคืน/i })
    ).not.toBeInTheDocument();
  });

  it('hides borrow request creation for admin and shows confirm-return action', () => {
    render(
      <EquipmentPage
        user={{ id: 1, username: 'admin', role: 'admin' }}
        initialState="success"
      />
    );

    expect(
      screen.queryByRole('button', { name: /สร้างคำขอเบิก/i })
    ).not.toBeInTheDocument();

    expect(
      screen.getByRole('button', { name: /รับคืน/i })
    ).toBeInTheDocument();
  });
});
