const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080/api/v1';

async function request(path) {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    credentials: 'include',
  });

  if (!response.ok) {
    const payload = await response.json().catch(() => null);
    throw new Error(payload?.error?.message || 'ไม่สามารถโหลดแดชบอร์ดได้');
  }

  return response.json();
}

export function getDashboard() {
  return request('/dashboard');
}
