const BASE = '/api'
const SERVER_KEY = import.meta.env.VITE_SERVER_KEY || ''

async function fetchApi(endpoint) {
  const headers = {}
  if (SERVER_KEY) {
    headers['Authorization'] = `Bearer ${SERVER_KEY}`
  }
  const res = await fetch(`${BASE}${endpoint}`, { headers })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Request failed' }))
    throw new Error(err.error || 'Request failed')
  }
  return res.json()
}

export const api = {
  getStats: () => fetchApi('/stats'),
  getCompanies: (params = {}) => {
    const q = new URLSearchParams(params).toString()
    return fetchApi(`/companies?${q}`)
  },
  getCompany: (ticker) => fetchApi(`/companies/${encodeURIComponent(ticker)}`),
  getCompanyGraph: (ticker) => fetchApi(`/companies/${encodeURIComponent(ticker)}/graph`),
  getSectors: () => fetchApi('/sectors'),
  getNews: (params = {}) => {
    const q = new URLSearchParams(params).toString()
    return fetchApi(`/news?${q}`)
  },
  getNewsStats: () => fetchApi('/news/stats'),
  getScans: (limit = 10) => fetchApi(`/scans?limit=${limit}`),
  getAnalysisScans: (params = {}) => {
    const q = new URLSearchParams(params).toString()
    return fetchApi(`/analysis/scans?${q}`)
  },
  getAnalysisScan: (id) => fetchApi(`/analysis/scans/${id}`),
  getSignals: (params = {}) => {
    const q = new URLSearchParams(params).toString()
    return fetchApi(`/analysis/signals?${q}`)
  },
  getAnalysisStats: () => fetchApi('/analysis/stats'),
}
