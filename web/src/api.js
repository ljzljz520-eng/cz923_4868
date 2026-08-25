async function request(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      'X-Operator': 'web-pharmacist',
      ...(options.headers || {})
    }
  })
  const payload = await response.json()
  if (!response.ok) {
    throw new Error(payload.error || '操作失败')
  }
  return payload.data
}

export const api = {
  dashboard: () => request('/api/dashboard'),
  medicines: () => request('/api/medicines'),
  register: command => request('/api/orders', { method: 'POST', body: JSON.stringify(command) }),
  call: (id, command) => request(`/api/orders/${id}/call`, { method: 'POST', body: JSON.stringify(command) }),
  dispense: (id, command) => request(`/api/orders/${id}/dispense`, { method: 'POST', body: JSON.stringify(command) })
}
