import { useState, useEffect, useMemo } from 'react'
import { api } from '../services/api'
import '../styles/analysis.css'
import '../styles/home.css'

export default function StocksPage() {
  const [signals, setSignals] = useState([])
  const [filter, setFilter] = useState('')
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.getSignals({ limit: 500, ...(filter ? { signal: filter } : {}) })
      .then(data => setSignals(data.signals || []))
      .catch(() => setSignals([]))
      .finally(() => setLoading(false))
  }, [filter])

  const grouped = useMemo(() => {
    const map = {}
    signals.forEach(s => {
      if (!map[s.ticker]) {
        map[s.ticker] = { ticker: s.ticker, signals: [], buyCount: 0, sellCount: 0, watchCount: 0, totalConfidence: 0 }
      }
      map[s.ticker].signals.push(s)
      map[s.ticker].totalConfidence += s.confidence || 0
      if (s.signal === 'BUY') map[s.ticker].buyCount++
      else if (s.signal === 'SELL') map[s.ticker].sellCount++
      else if (s.signal === 'WATCH') map[s.ticker].watchCount++
    })
    return Object.values(map)
      .map(g => ({
        ...g,
        avgConfidence: Math.round(g.totalConfidence / g.signals.length),
        dominantSignal: g.buyCount >= g.sellCount && g.buyCount >= g.watchCount ? 'BUY'
          : g.sellCount >= g.watchCount ? 'SELL' : 'WATCH',
      }))
      .filter(g => !search || g.ticker.toLowerCase().includes(search.toLowerCase()))
      .sort((a, b) => b.avgConfidence - a.avgConfidence)
  }, [signals, search])

  if (loading) return <div className="loading">Loading stocks...</div>

  return (
    <div>
      <h1 className="page-title">Stocks</h1>

      <div style={{ display: 'flex', gap: 12, marginBottom: 20, flexWrap: 'wrap' }}>
        <div className="search-box" style={{ flex: 1, minWidth: 200 }}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/></svg>
          <input placeholder="Search ticker..." value={search} onChange={e => setSearch(e.target.value)} />
        </div>
        <div className="signal-toolbar" style={{ margin: 0 }}>
          {['', 'BUY', 'SELL', 'WATCH', 'SKIP'].map(f => (
            <button key={f} className={`signal-filter ${filter === f ? 'active' : ''}`}
              onClick={() => { setFilter(f); setLoading(true) }}>
              {f || 'All'}
            </button>
          ))}
        </div>
      </div>

      <div className="stocks-grid">
        {grouped.map(stock => {
          const colorMap = { BUY: 'var(--success)', SELL: 'var(--danger)', WATCH: 'var(--warning)' }
          const color = colorMap[stock.dominantSignal] || 'var(--text-muted)'
          return (
            <div key={stock.ticker} className="stock-card">
              <div className="stock-card-header">
                <span className="stock-card-ticker">{stock.ticker}</span>
                <span className={`signal-badge signal-${stock.dominantSignal.toLowerCase()}`}>{stock.dominantSignal}</span>
              </div>
              <div className="stock-card-stats">
                <span>{stock.signals.length} signals</span>
                <span style={{ color }}>{stock.avgConfidence}% avg</span>
              </div>
              <div className="stock-card-bar-track">
                <div className="stock-card-bar-fill" style={{
                  width: `${stock.avgConfidence}%`,
                  background: `linear-gradient(90deg, ${color}60, ${color})`,
                }} />
              </div>
              <div className="stock-card-breakdown">
                {stock.buyCount > 0 && <span style={{ color: 'var(--success)' }}>{stock.buyCount} BUY</span>}
                {stock.sellCount > 0 && <span style={{ color: 'var(--danger)' }}>{stock.sellCount} SELL</span>}
                {stock.watchCount > 0 && <span style={{ color: 'var(--warning)' }}>{stock.watchCount} WATCH</span>}
              </div>
            </div>
          )
        })}
        {grouped.length === 0 && <div className="empty">No stocks found</div>}
      </div>
    </div>
  )
}
