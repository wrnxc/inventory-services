const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080/api/v1';

async function request(path, { method = 'GET', body, headers } = {}) {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...headers,
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    const payload = await response.json().catch(() => null);
    const message = payload?.error?.message || 'ไม่สามารถดำเนินการได้';
    throw new Error(message);
  }

  if (response.status === 204) {
    return null;
  }

  return response.json();
}

export async function listBorrowRequests(params = {}) {
  const query = new URLSearchParams();

  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      query.append(key, String(value));
    }
  });

  const suffix = query.toString() ? `?${query.toString()}` : '';
  return request(`/borrow-requests${suffix}`);
}

export async function createBorrowRequest(payload) {
  return request('/borrow-requests', {
    method: 'POST',
    body: payload,
  });
}

export async function approveBorrowRequest(id) {
  return request(`/borrow-requests/${id}/approve`, {
    method: 'PUT',
  });
}

export async function rejectBorrowRequest(id) {
  return request(`/borrow-requests/${id}/reject`, {
    method: 'PUT',
  });
}

export async function returnBorrowRequest(id) {
  return request(`/borrow-requests/${id}/return`, {
    method: 'POST',
  });
}

export async function confirmReturnBorrowRequest(id) {
  return request(`/borrow-requests/${id}/confirm-return`, {
    method: 'PUT',
  });
}
