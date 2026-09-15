const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080/api/v1';

export async function importEquipment(file) {
  const body = new FormData();
  body.append('file', file);

  const response = await fetch(`${API_BASE_URL}/import`, {
    method: 'POST',
    credentials: 'include',
    body,
  });

  if (!response.ok) {
    const payload = await response.json().catch(() => null);
    throw new Error(payload?.error?.message || 'ไม่สามารถนำเข้าข้อมูลได้');
  }

  return response.json();
}
