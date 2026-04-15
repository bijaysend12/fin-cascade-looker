import { useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'
import '../styles/news.css'

const PAGE_SIZE = 20
const EVENT_TYPES = ['', 'natural_disaster', 'policy_change', 'corporate_action', 'supply_chain', 'regulatory', 'commodity', 'earnings', 'geopolitical', 'infrastructure', 'sector_news', 'other']

export default function NewsFeed() {
  const [articles, setArticles] = useState([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(0)
  const [classification, setClassification] = useState('')
  const [eventType, setEventType] = useState('')
  const [newsStats, setNewsStats] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.getNewsStats().then(setNewsStats)
  }, [])

  const load = useCallback(async () => {
    setLoading(true)
    const params = { limit: PAGE_SIZE, offset: page * PAGE_SIZE }
    if (classification) params.classification = classification
    if (eventType) params.type = eventType
    const data = await api.getNews(params)
    setArticles(data.articles || [])
    setTotal(data.total || 0)
    setLoading(false)
  }, [classification, eventType, page])

  useEffect(() => { load() }, [load])

  const totalPages = Math.ceil(total / PAGE_SIZE)

  return (
    <div>
      <h1 className="page-title">News</h1>

      {newsStats && (
        <div className="news-stats">
          <div className="news-stat">
            <div className="news-stat-value">{Object.values(newsStats.byClassification || {}).reduce((a, b) => a + b, 0)}</div>
            <div className="news-stat-label">Total</div>
          </div>
          <div className="news-stat">
            <div className="news-stat-value" style={{ color: 'var(--danger)' }}>{newsStats.byClassification?.HIGH || 0}</div>
            <div className="news-stat-label">HIGH</div>
          </div>
          <div className="news-stat">
            <div className="news-stat-value" style={{ color: 'var(--warning)' }}>{newsStats.byClassification?.MEDIUM || 0}</div>
            <div className="news-stat-label">MEDIUM</div>
          </div>
          <div className="news-stat">
            <div className="news-stat-value" style={{ color: 'var(--success)' }}>{newsStats.byClassification?.LOW || 0}</div>
            <div className="news-stat-label">LOW</div>
          </div>
        </div>
      )}

      <div style={{ display: 'flex', gap: 8, marginBottom: 16, flexWrap: 'wrap' }}>
        {['', 'HIGH', 'MEDIUM', 'LOW'].map(c => (
          <button key={c} className={`signal-filter ${classification === c ? 'active' : ''}`}
            onClick={() => { setClassification(c); setPage(0) }}>
            {c || 'All'}
          </button>
        ))}
      </div>

      <div className="news-toolbar">
        <select value={eventType} onChange={e => { setEventType(e.target.value); setPage(0) }}>
          <option value="">All Event Types</option>
          {EVENT_TYPES.filter(Boolean).map(t => (
            <option key={t} value={t}>{t.replace(/_/g, ' ')}</option>
          ))}
        </select>
        <span style={{ fontSize: 13, color: 'var(--text-muted)', marginLeft: 'auto' }}>{total} articles</span>
      </div>

      {loading ? <div className="loading">Loading...</div> : (
        <div className="news-list">
          {articles.map(a => (
            <div key={a.hash} className="news-card" data-classification={a.classification}>
              <span className={`badge badge-${(a.classification || 'low').toLowerCase()}`}>
                {a.classification || '?'}
              </span>
              <div className="news-card-body">
                <div className="news-card-title">
                  {a.link ? <a href={a.link} target="_blank" rel="noopener noreferrer">{a.title || 'Untitled'}</a> : (a.title || 'Untitled')}
                </div>
                <div className="news-card-meta">
                  <span>{a.source}</span>
                  {a.event_type && <span className="event-type-tag">{a.event_type.replace(/_/g, ' ')}</span>}
                  <span>{formatDate(a.processed_at)}</span>
                </div>
              </div>
            </div>
          ))}
          {articles.length === 0 && <div className="empty">No articles found</div>}
        </div>
      )}

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

function formatDate(d) {
  if (!d) return ''
  try {
    return new Date(d + 'Z').toLocaleString('en-IN', { timeZone: 'Asia/Kolkata' })
  } catch {
    return d
  }
}
