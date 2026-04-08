import { getIdToken } from './firebase'

const BASE = '/api'

async function fetchApi(endpoint) {
  const headers = {}
  const token = await getIdToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const res = await fetch(`${BASE}${endpoint}`, { headers })
  if (res.status === 401) {
    throw new Error('Unauthorized')
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Request failed' }))
    throw new Error(err.error || 'Request failed')
  }
  return res.json()
}

export const api = {
  getMe: () => fetchApi('/me'),
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
