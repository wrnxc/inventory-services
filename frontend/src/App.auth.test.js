import {
  render,
  screen,
} from '@testing-library/react';

import userEvent from '@testing-library/user-event';

import App from './App';

import {
  getCurrentUser,
  login,
  logout,
} from './api/authClient';

const mockEquipmentList = [
  {
    id: 1,
    asset_name: 'Laptop-001',
    asset_serial_no: 'SN-001',
    type_id: 1,
    status: 'ในคลัง',
  },
];

const mockUser = {
  id: 1,
  username: 'active-user',
  role: 'user',
};

jest.mock('./api/equipmentClient', () => ({
  listEquipment: jest.fn(() =>
    Promise.resolve(mockEquipmentList)
  ),

  createEquipment: jest.fn(),
  updateEquipment: jest.fn(),
  deleteEquipment: jest.fn(),
}));

jest.mock('./api/authClient', () => ({
  getCurrentUser: jest.fn(),
  login: jest.fn(),
  logout: jest.fn(),
}));

function navigate(path) {
  window.history.replaceState(
    {},
    '',
    path
  );
}

beforeEach(() => {
  jest.clearAllMocks();

  window.history.replaceState(
    {},
    '',
    '/'
  );

  // default = ไม่มี session
  getCurrentUser.mockRejectedValue(
    new Error('missing session')
  );

  logout.mockResolvedValue({
    success: true,
  });
});

describe('Authentication flow checklist', () => {
  it('shows the login page at /login', async () => {
    navigate('/login');

    render(<App />);

    expect(
      await screen.findByRole(
        'heading',
        {
          name: /^login$/i,
        }
      )
    ).toBeInTheDocument();

    expect(
      screen.getByLabelText(
        /username/i
      )
    ).toBeInTheDocument();

    expect(
      screen.getByLabelText(
        /password/i
      )
    ).toBeInTheDocument();

    expect(
      screen.getByRole(
        'button',
        {
          name: /sign in/i,
        }
      )
    ).toBeInTheDocument();
  });

  it('redirects protected equipment route to /login when there is no session', async () => {
    navigate('/equipment');

    render(<App />);

    expect(
      await screen.findByRole(
        'heading',
        {
          name: /^login$/i,
        }
      )
    ).toBeInTheDocument();

    expect(
      window.location.pathname
    ).toBe('/login');
  });

  it('keeps the session after login and lands on the equipment page', async () => {
    navigate('/login');

    login.mockResolvedValue({
      user: mockUser,
    });

    render(<App />);

    await screen.findByRole(
      'heading',
      {
        name: /^login$/i,
      }
    );

    await userEvent.type(
      screen.getByLabelText(
        /username/i
      ),
      'active-user'
    );

    await userEvent.type(
      screen.getByLabelText(
        /password/i
      ),
      'correct-password'
    );

    await userEvent.click(
      screen.getByRole(
        'button',
        {
          name: /sign in/i,
        }
      )
    );

    expect(
      await screen.findByRole(
        'heading',
        {
          name: /รายการอุปกรณ์ทั้งหมด/i,
        }
      )
    ).toBeInTheDocument();

    expect(
      window.location.pathname
    ).toBe('/equipment');

    expect(login).toHaveBeenCalledWith(
      'active-user',
      'correct-password'
    );
  });

  it('restores an authenticated session from /me', async () => {
    getCurrentUser.mockResolvedValue({
      user: mockUser,
    });

    navigate('/equipment');

    render(<App />);

    expect(
      await screen.findByRole(
        'heading',
        {
          name: /รายการอุปกรณ์ทั้งหมด/i,
        }
      )
    ).toBeInTheDocument();

    expect(
      window.location.pathname
    ).toBe('/equipment');

    expect(
      getCurrentUser
    ).toHaveBeenCalledTimes(1);
  });

  it('redirects "/" to /login when there is no session', async () => {
    navigate('/');

    render(<App />);

    expect(
      await screen.findByRole(
        'heading',
        {
          name: /^login$/i,
        }
      )
    ).toBeInTheDocument();

    expect(
      window.location.pathname
    ).toBe('/login');
  });

  it('redirects "/" to /equipment when a session already exists', async () => {
    getCurrentUser.mockResolvedValue({
      user: mockUser,
    });

    navigate('/');

    render(<App />);

    expect(
      await screen.findByRole(
        'heading',
        {
          name: /รายการอุปกรณ์ทั้งหมด/i,
        }
      )
    ).toBeInTheDocument();

    expect(
      window.location.pathname
    ).toBe('/equipment');
  });

  it('redirects an already-authenticated user away from /login to /equipment', async () => {
    getCurrentUser.mockResolvedValue({
      user: mockUser,
    });

    navigate('/login');

    render(<App />);

    expect(
      await screen.findByRole(
        'heading',
        {
          name: /รายการอุปกรณ์ทั้งหมด/i,
        }
      )
    ).toBeInTheDocument();

    expect(
      window.location.pathname
    ).toBe('/equipment');
  });
});