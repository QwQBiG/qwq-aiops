export function isCancellation(error) {
  return error === 'cancel' || error === 'close' || error?.code === 'ERR_CANCELED' || error?.name === 'AbortError'
}

export function errorMessage(error, fallback) {
  let body = error?.response?.data
  if (typeof body === 'string' && body.trim() && !body.trim().startsWith('<')) {
    try {
      body = JSON.parse(body)
    } catch {
      return body.trim().slice(0, 500)
    }
  }
  for (const value of [body?.message, body?.msg, body?.error]) {
    if (typeof value === 'string' && value.trim()) return value
  }
  return fallback
}
