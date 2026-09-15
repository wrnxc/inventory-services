const API_BASE_URL =
  process.env.REACT_APP_API_BASE_URL ||
  'http://localhost:8080/api/v1';

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

  let data = null;

  try {
    data = await response.json();
  } catch {
    data = null;
  }

  if (!response.ok) {
    const message =
      data?.error?.message ||
      `Request failed: ${response.status}`;

    throw new Error(message);
  }

  return data;
}

export async function listEquipmentTypes() {
  return request('/equipment-types');
}