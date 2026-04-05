import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { Search } from 'lucide-react'
import { api } from '../services/api'
import '../styles/companies.css'

const PAGE_SIZE = 20

export default function Companies() {
  const [companies, setCompanies] = useState([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(0)
  const [search, setSearch] = useState('')
  const [sector, setSector] = useState('')
  const [cap, setCap] = useState('')
  const [sectors, setSectors] = useState([])
  const [loading, setLoading] = useState(true)
  const navigate = useNavigate()

  useEffect(() => {
    api.getSectors().then(setSectors)
  }, [])

  const load = useCallback(async () => {
    setLoading(true)
    const params = { limit: PAGE_SIZE, offset: page * PAGE_SIZE }
    if (search) params.search = search
    if (sector) params.sector = sector
    if (cap) params.cap = cap
    const data = await api.getCompanies(params)
    setCompanies(data.companies || [])
    setTotal(data.total || 0)
    setLoading(false)
  }, [search, sector, cap, page])

  useEffect(() => { load() }, [load])

  const totalPages = Math.ceil(total / PAGE_SIZE)

  function deClass(val) {
    if (val == null) return ''
    if (val > 1.5) return 'de-high'
    if (val > 0.5) return 'de-medium'
    return 'de-low'
  }

  return (
    <div>
      <h1 className="page-title">Companies</h1>

      <div className="companies-toolbar">
        <div className="search-box">
          <Search size={16} />
          <input
            placeholder="Search by name or ticker..."
            value={search}
            onChange={e => { setSearch(e.target.value); setPage(0) }}
          />
        </div>
        <select value={sector} onChange={e => { setSector(e.target.value); setPage(0) }}>
          <option value="">All Sectors</option>
          {sectors.map(s => <option key={s.name} value={s.name}>{s.name} ({s.companyCount})</option>)}
        </select>
        <select value={cap} onChange={e => { setCap(e.target.value); setPage(0) }}>
          <option value="">All Caps</option>
          <option value="large">Large</option>
          <option value="mid">Mid</option>
          <option value="small">Small</option>
        </select>
      </div>

      <div className="companies-table-wrap">
        {loading ? <div className="loading">Loading...</div> : (
          <table>
            <thead>
              <tr>
                <th>Ticker</th>
                <th>Name</th>
                <th>Sector</th>
                <th>Industry</th>
                <th>Market Cap</th>
                <th>D/E Ratio</th>
              </tr>
            </thead>
            <tbody>
              {companies.map(c => (
                <tr key={c.ticker} onClick={() => navigate(`/company/${encodeURIComponent(c.ticker)}`)}>
                  <td className="ticker-cell">{c.ticker}</td>
                  <td style={{ color: 'var(--text-primary)' }}>{c.name}</td>
                  <td>{c.sector}</td>
                  <td>{c.industry}</td>
                  <td><span className={`badge badge-${c.marketCapCategory}`}>{c.marketCapCategory}</span></td>
                  <td className={`de-ratio ${deClass(c.debtToEquity)}`}>{c.debtToEquity?.toFixed(2) ?? '-'}</td>
                </tr>
              ))}
              {companies.length === 0 && (
                <tr><td colSpan={6} className="empty">No companies found</td></tr>
              )}
            </tbody>
          </table>
        )}
      </div>

      {totalPages > 1 && (
        <div className="pagination">
          <button disabled={page === 0} onClick={() => setPage(p => p - 1)}>Previous</button>
          <span>Page {page + 1} of {totalPages}</span>
          <button disabled={page >= totalPages - 1} onClick={() => setPage(p => p + 1)}>Next</button>
        </div>
      )}
    </div>
  )
}
