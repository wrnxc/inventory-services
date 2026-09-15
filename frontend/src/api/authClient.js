const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080/api/v1';

async function request(path, options = {}) {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  });

  let data = null;

  try {
    data = await response.json();
  } catch {
    data = null;
  }

  if (!response.ok) {
    const errorCode = data?.error?.code;

    switch (errorCode) {
      case 'INVALID_CREDENTIALS':
        throw new Error('ชื่อผู้ใช้หรือรหัสผ่านไม่ถูกต้อง');

      case 'ACCOUNT_INACTIVE':
        throw new Error('บัญชีผู้ใช้นี้ถูกระงับการใช้งาน');

      case 'VALIDATION_ERROR':
        throw new Error('กรุณากรอกชื่อผู้ใช้และรหัสผ่าน');

      case 'UNAUTHORIZED':
        throw new Error('กรุณาเข้าสู่ระบบใหม่อีกครั้ง');

      case 'FORBIDDEN':
        throw new Error('คุณไม่มีสิทธิ์ดำเนินการนี้');

      default:
        throw new Error(
          data?.error?.message ||
            'ไม่สามารถดำเนินการได้ กรุณาลองใหม่อีกครั้ง'
        );
    }
  }

  if (response.status === 204) {
    return null;
  }

  return data;
}

export async function login(username, password) {
  return request('/login', {
    method: 'POST',
    body: JSON.stringify({
      username,
      password,
    }),
  });
}

export async function logout() {
  return request('/logout', {
    method: 'POST',
  });
}

export async function getCurrentUser() {
  return request('/me');
}