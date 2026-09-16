const API_BASE_URL =
  process.env.REACT_APP_API_BASE_URL ||
  'http://localhost:8080/api/v1';

async function parseError(response, fallbackMessage) {
  const payload = await response
    .json()
    .catch(() => null);

  const error = new Error(
    payload?.error?.message ||
      fallbackMessage
  );

  error.code =
    payload?.error?.code;

  error.status =
    response.status;

  throw error;
}

export async function previewImportEquipment(file) {
  const body = new FormData();

  body.append('file', file);

  const response = await fetch(
    `${API_BASE_URL}/import/preview`,
    {
      method: 'POST',
      credentials: 'include',
      body,
    }
  );

  if (!response.ok) {
    await parseError(
      response,
      'ไม่สามารถอ่านตัวอย่างข้อมูลได้'
    );
  }

  return response.json();
}

export async function importEquipment(file) {
  const body = new FormData();

  body.append('file', file);

  const response = await fetch(
    `${API_BASE_URL}/import`,
    {
      method: 'POST',
      credentials: 'include',
      body,
    }
  );

  if (!response.ok) {
    await parseError(
      response,
      'ไม่สามารถนำเข้าข้อมูลได้'
    );
  }

  return response.json();
}

export async function getImportHistory() {
  const response = await fetch(
    `${API_BASE_URL}/import/history`,
    {
      credentials: 'include',
    }
  );

  if (!response.ok) {
    await parseError(
      response,
      'ไม่สามารถโหลดประวัติการนำเข้าได้'
    );
  }

  return response.json();
}