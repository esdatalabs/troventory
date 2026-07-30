// Thin fetch wrapper around the Troventory REST API (cmd/api). Every
// function returns parsed JSON on success and throws an Error carrying the
// API's own message on failure, so callers can show it directly.

// Default to the API on the same host the dashboard itself was loaded
// from, just on port 8080 — so a build/dev-server reachable from another
// device (a different hostname/IP than "localhost") still finds the API
// without VITE_API_BASE having to be baked in for that specific address.
// Set VITE_API_BASE explicitly to override.
const BASE = import.meta.env.VITE_API_BASE || `${window.location.protocol}//${window.location.hostname}:8080`

async function request(method, path, body) {
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  if (res.status === 204) return null

  const isJSON = (res.headers.get('content-type') || '').includes('application/json')
  const payload = isJSON ? await res.json() : await res.text()

  if (!res.ok) {
    const message = isJSON && payload && payload.error ? payload.error : `request failed (${res.status})`
    throw new Error(message)
  }
  return payload
}

const encode = (segment) => encodeURIComponent(segment)

export const api = {
  // Locations
  listLocations: () => request('GET', '/locations'),
  createLocation: (body) => request('POST', '/locations', body),
  renameLocation: (name, newName) => request('POST', `/locations/${encode(name)}/rename`, { name: newName }),
  moveLocation: (name, parent) => request('POST', `/locations/${encode(name)}/move`, { parent }),
  archiveLocation: (name) => request('POST', `/locations/${encode(name)}/archive`),

  // Items
  listItems: () => request('GET', '/items'),
  getItem: (description) => request('GET', `/items/${encode(description)}`),
  createItem: (body) => request('POST', '/items', body),
  updateItem: (description, patch) => request('PATCH', `/items/${encode(description)}`, patch),
  archiveItem: (description) => request('POST', `/items/${encode(description)}/archive`),
  scanItem: (barcode) => request('POST', '/items/scan', { barcode }),
  enrichItem: (barcode, target) => request('POST', '/items/enrich', { barcode, target: target || undefined }),

  // Valuation
  recordPrice: (description, body) => request('POST', `/items/${encode(description)}/value/price`, body),
  recordAppraisal: (description, body) => request('POST', `/items/${encode(description)}/value/appraisals`, body),
  setDepreciationRate: (description, ratePercent) =>
    request('PUT', `/items/${encode(description)}/value/depreciation-rate`, { rate_percent: ratePercent }),
  getCurrentValue: (description, date) =>
    request('GET', `/items/${encode(description)}/value/current${date ? `?date=${encode(date)}` : ''}`),

  // Search
  search: (filters) => {
    const params = new URLSearchParams()
    if (filters.desc) params.set('desc', filters.desc)
    if (filters.category) params.set('category', filters.category)
    if (filters.location) params.set('location', filters.location)
    if (filters.min !== '' && filters.min != null) params.set('min', filters.min)
    if (filters.max !== '' && filters.max != null) params.set('max', filters.max)
    const qs = params.toString()
    return request('GET', `/search${qs ? `?${qs}` : ''}`)
  },

  // Export
  exportURL: (format) => `${BASE}/export?format=${encode(format)}`,
}
