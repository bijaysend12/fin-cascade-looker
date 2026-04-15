import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { AlertTriangle } from 'lucide-react'
import { api } from '../services/api'
import '../styles/analysis.css'
import '../styles/home.css'

export default function HomePage() {
  const [stats, setStats] = useState(null)
  const [signals, setSignals] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      api.getAnalysisStats().catch(() => null),
      api.getSignals({ limit: 100 }).catch(() => ({ signals: [] })),
    ]).then(([st, sig]) => {
      setStats(st)
      setSignals(sig.signals || [])
    }).finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="loading">Loading...</div>

  const criticalEvents = signals.filter(s => s.signal === 'BUY' || s.signal === 'SELL')
    .sort((a, b) => (b.confidence || 0) - (a.confidence || 0))

  const topMovers = (() => {
    const map = {}
    signals.forEach(s => {
      if (!map[s.ticker]) map[s.ticker] = { ticker: s.ticker, signals: [], totalConfidence: 0 }
      map[s.ticker].signals.push(s)
      map[s.ticker].totalConfidence += s.confidence || 0
    })
    return Object.values(map)
      .map(g => ({
        ...g,
        avgConfidence: Math.round(g.totalConfidence / g.signals.length),
        dominantSignal: g.signals.filter(s => s.signal === 'BUY').length >= g.signals.filter(s => s.signal === 'SELL').length ? 'BUY' : 'SELL',
      }))
      .sort((a, b) => b.avgConfidence - a.avgConfidence)
      .slice(0, 5)
  })()

  return (
    <div>
      <h1 className="page-title">Home</h1>

      {stats && (
        <>
          {/* Stats Grid */}
          <div className="analysis-stats">
            <div className="analysis-stat">
              <div className="analysis-stat-value">{stats.total_scans}</div>
              <div className="analysis-stat-label">Scans</div>
            </div>
            <div className="analysis-stat">
              <div className="analysis-stat-value">{stats.total_events}</div>
              <div className="analysis-stat-label">Events</div>
            </div>
            <div className="analysis-stat">
              <div className="analysis-stat-value">{stats.total_signals}</div>
              <div className="analysis-stat-label">Signals</div>
            </div>
            <div className="analysis-stat">
              <div className="analysis-stat-value" style={{ color: 'var(--success)' }}>{stats.by_signal?.BUY || 0}</div>
              <div className="analysis-stat-label">Buy</div>
            </div>
            <div className="analysis-stat">
              <div className="analysis-stat-value" style={{ color: 'var(--danger)' }}>{stats.by_signal?.SELL || 0}</div>
              <div className="analysis-stat-label">Sell</div>
            </div>
            <div className="analysis-stat">
              <div className="analysis-stat-value" style={{ color: 'var(--warning)' }}>{stats.by_signal?.WATCH || 0}</div>
              <div className="analysis-stat-label">Watch</div>
            </div>
          </div>

          {/* Signal Distribution */}
          <div className="section-title" style={{ marginTop: 28 }}>Signal Distribution</div>
          <div className="signal-distribution">
            {['BUY', 'SELL', 'WATCH', 'SKIP'].map(signal => {
              const count = stats.by_signal?.[signal] || 0
              const total = stats.total_signals || 1
              const pct = Math.round((count / total) * 100)
              const colorMap = { BUY: 'var(--success)', SELL: 'var(--danger)', WATCH: 'var(--warning)', SKIP: '#6b7280' }
              return (
                <div key={signal} className="distribution-row">
                  <span className="distribution-label">{signal}</span>
                  <div className="distribution-track">
                    <div className="distribution-fill" style={{ width: `${pct}%`, background: `linear-gradient(90deg, ${colorMap[signal]}60, ${colorMap[signal]})` }} />
                  </div>
                  <span className="distribution-value" style={{ color: colorMap[signal] }}>{count}</span>
                </div>
              )
            })}
          </div>
        </>
      )}

      {/* Critical Events */}
      {criticalEvents.length > 0 && (
        <>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 28 }}>
            <div className="critical-icon-wrap">
              <AlertTriangle size={16} />
            </div>
            <div className="section-title" style={{ margin: 0 }}>Critical Events</div>
          </div>
          <div className="critical-events-list">
            {criticalEvents.slice(0, 5).map((s, i) => {
              const isSell = s.signal === 'SELL'
              return (
                <div key={i} className="critical-event-card" data-signal={s.signal}>
                  <div className="critical-event-header">
                    <span className={`signal-badge signal-${s.signal?.toLowerCase()}`}>{s.signal}</span>
                    <span className="critical-event-ticker">{s.ticker}</span>
                    <span style={{ marginLeft: 'auto', fontSize: 12, color: 'var(--text-muted)' }}>{s.confidence}%</span>
                  </div>
                  <div className="critical-event-headline">{s.event_headline}</div>
                  <div className="critical-event-impact">{s.impact_range} &middot; {s.direction}</div>
                </div>
              )
            })}
          </div>
        </>
      )}

      {/* Top Movers */}
      {topMovers.length > 0 && (
        <>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 28 }}>
            <div className="section-title" style={{ margin: 0 }}>Top Movers</div>
            <Link to="/stocks" style={{ fontSize: 12, color: 'var(--accent)', textDecoration: 'none' }}>View all</Link>
          </div>
          <div className="top-signals-grid">
            {topMovers.map((stock, i) => {
              const colorMap = { BUY: 'var(--success)', SELL: 'var(--danger)' }
              const color = colorMap[stock.dominantSignal] || 'var(--text-muted)'
              return (
                <div key={stock.ticker} className="top-signal-card">
                  <div className="top-signal-rank">#{i + 1}</div>
                  <div className="top-signal-info">
                    <div className="top-signal-ticker">{stock.ticker}</div>
                    <div className="top-signal-event">{stock.signals.length} signals &middot; {stock.avgConfidence}% avg</div>
                  </div>
                  <span className={`signal-badge signal-${stock.dominantSignal.toLowerCase()}`}>{stock.dominantSignal}</span>
                </div>
              )
            })}
          </div>
        </>
      )}
    </div>
  )
}
