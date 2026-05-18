/** 从 axios 错误中解析可读文案（含 responseType: blob 时的 JSON 错误体） */
export async function readApiErrorMessage(error, fallback = '请求失败') {
  const data = error?.response?.data
  if (!data) return error?.message || fallback
  if (data instanceof Blob) {
    try {
      const text = await data.text()
      if (text.trim().startsWith('{')) {
        const json = JSON.parse(text)
        return json.error || json.message || fallback
      }
      const trimmed = text.trim()
      if (trimmed) return trimmed.length > 160 ? `${trimmed.slice(0, 160)}…` : trimmed
    } catch {
      /* ignore */
    }
    return fallback
  }
  if (typeof data === 'object' && data !== null) {
    return data.error || data.message || fallback
  }
  return String(data) || fallback
}
