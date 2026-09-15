const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080/api/v1';

async function request(path, { method = 'GET', body, headers } = {}) {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method,
    credentials: 'include', //Frontend ส่ง cookie อัตโนมัติ
    headers: {
      'Content-Type': 'application/json',
      ...headers,
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    throw new Error(`Request failed: ${response.status}`);
  }

  return response.json();
}

export async function listEquipment() {
  return request('/equipment');
}

export async function createEquipment(payload) {
  return request('/equipment', {
    method: 'POST',
    body: payload,
  });
}

export async function updateEquipment(id, payload) {
  return request(`/equipment/${id}`, {
    method: 'PUT',
    body: payload,
  });
}

export async function deleteEquipment(id) {
  return request(`/equipment/${id}`, {
    method: 'DELETE',
  });
}