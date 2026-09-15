import { render, screen } from '@testing-library/react';
import EquipmentPage from './pages/EquipmentPage';

import { listEquipment } from './api/equipmentClient';

jest.mock('./api/equipmentClient', () => ({
  listEquipment: jest.fn(),
  createEquipment: jest.fn(),
  updateEquipment: jest.fn(),
  deleteEquipment: jest.fn(),
}));

const adminUser = {
  id: 1,
  username: 'admin',
  role: 'admin',
};

beforeEach(() => {
  jest.clearAllMocks();
});

describe('Equipment page checklist (authenticated)', () => {
  it('shows a loading state while the equipment list is being fetched', () => {
    listEquipment.mockReturnValue(
      new Promise(() => {})
    );

    render(
      <EquipmentPage
        user={adminUser}
        initialState="loading"
      />
    );

    expect(
      screen.getByText(/loading equipment/i)
    ).toBeInTheDocument();
  });

  it('shows an empty state when there are no equipment records', () => {
    render(
      <EquipmentPage
        user={adminUser}
        initialState="empty"
      />
    );

    expect(
      screen.getByText(/no equipment found/i)
    ).toBeInTheDocument();
  });

  it('shows an error state when loading the equipment list fails', () => {
    render(
      <EquipmentPage
        user={adminUser}
        initialState="error"
      />
    );

    expect(
      screen.getByText(/unable to load equipment/i)
    ).toBeInTheDocument();
  });

  it('renders the equipment page on success', () => {
    render(
      <EquipmentPage
        user={adminUser}
        initialState="success"
      />
    );

    expect(
      screen.getByRole('heading', {
        name: /รายการอุปกรณ์ทั้งหมด/i,
      })
    ).toBeInTheDocument();

    expect(
      screen.getByRole('heading', {
        name: /create equipment/i,
      })
    ).toBeInTheDocument();

    expect(
      screen.getByRole('heading', {
        name: /edit equipment/i,
      })
    ).toBeInTheDocument();
  });

  it('shows equipment management actions for admin', () => {
    render(
      <EquipmentPage
        user={adminUser}
        initialState="success"
      />
    );

    expect(
      screen.getByRole('button', {
        name: /เพิ่มอุปกรณ์/i,
      })
    ).toBeInTheDocument();
  });

  it('hides equipment management actions for user role', () => {
    render(
      <EquipmentPage
        user={{
          id: 2,
          username: 'service01',
          role: 'user',
        }}
        initialState="success"
      />
    );

    expect(
      screen.queryByRole('button', {
        name: /เพิ่มอุปกรณ์/i,
      })
    ).not.toBeInTheDocument();
  });
});