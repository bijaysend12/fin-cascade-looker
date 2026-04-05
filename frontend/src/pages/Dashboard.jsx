import { useState, useEffect } from 'react'
import { api } from '../services/api'
import StatsCard from '../components/StatsCard'
import '../styles/dashboard.css'

export default function Dashboard() {
  const [stats, setStats] = useState(null)
  const [sectors, setSectors] = useState([])
  const [recentNews, setRecentNews] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      api.getStats(),
      api.getSectors(),
      api.getNews({ limit: 5 }),
    ]).then(([s, sec, news]) => {
      setStats(s)
      setSectors(sec)
      setRecentNews(news.articles || [])
    }).finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="loading">Loading dashboard...</div>
  if (!stats) return <div className="empty">Failed to load stats</div>

  const maxCount = Math.max(...sectors.map(s => s.companyCount || 0), 1)

  return (
    <div>
      <h1 className="page-title">Dashboard</h1>

      <div className="dashboard-stats">
        <StatsCard label="Companies" value={stats.neo4j?.companies || 0} color="var(--accent)" />
        <StatsCard label="Plants" value={stats.neo4j?.plants || 0} color="var(--success)" />
        <StatsCard label="Sectors" value={stats.neo4j?.sectors || 0} color="var(--warning)" />
        <StatsCard label="Locations" value={stats.neo4j?.locations || 0} color="var(--cyan)" />
        <StatsCard label="Raw Materials" value={stats.neo4j?.materials || 0} color="var(--pink)" />
      </div>

      <div className="dashboard-grid">
        <div className="card">
          <h3 style={{ marginBottom: 16, fontSize: 15, color: 'var(--text-secondary)' }}>Companies by Sector</h3>
          <div className="sector-bars">
            {sectors.map(s => (
              <div key={s.name} className="sector-bar-row">
                <span className="sector-bar-label">{s.name}</span>
                <div className="sector-bar-track">
                  <div className="sector-bar-fill" style={{ width: `${(s.companyCount / maxCount) * 100}%` }}>
                    {s.companyCount}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="card">
          <h3 style={{ marginBottom: 16, fontSize: 15, color: 'var(--text-secondary)' }}>Relationships</h3>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <RelStat label="COMPETES_WITH" value={stats.relationships?.competes || 0} color="var(--danger)" />
            <RelStat label="SUPPLIES_TO" value={stats.relationships?.supplies || 0} color="var(--accent)" />
            <RelStat label="CONSUMES" value={stats.relationships?.consumes || 0} color="var(--warning)" />
          </div>

          <h3 style={{ marginBottom: 12, marginTop: 24, fontSize: 15, color: 'var(--text-secondary)' }}>News Summary</h3>
          <div style={{ display: 'flex', gap: 12 }}>
            <MiniStat label="Total" value={stats.news?.total || 0} />
            <MiniStat label="HIGH" value={stats.news?.high || 0} color="var(--danger)" />
            <MiniStat label="MEDIUM" value={stats.news?.medium || 0} color="var(--warning)" />
            <MiniStat label="LOW" value={stats.news?.low || 0} color="var(--success)" />
          </div>

          {recentNews.length > 0 && (
            <>
              <h3 style={{ marginBottom: 10, marginTop: 24, fontSize: 15, color: 'var(--text-secondary)' }}>Recent Articles</h3>
              <div className="recent-articles">
                {recentNews.map(a => (
                  <div key={a.hash} className="recent-article">
                    <span className="recent-article-title">{a.title || 'Untitled'}</span>
                    <span className={`badge badge-${(a.classification || 'low').toLowerCase()}`}>{a.classification}</span>
                  </div>
                ))}
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}

function RelStat({ label, value, color }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '10px 14px', background: 'var(--bg-hover)', borderRadius: 'var(--radius-sm)' }}>
      <span style={{ fontSize: 13, color: 'var(--text-muted)', fontFamily: 'monospace' }}>{label}</span>
      <span style={{ fontSize: 18, fontWeight: 700, color }}>{value}</span>
    </div>
  )
}

function MiniStat({ label, value, color }) {
  return (
    <div style={{ flex: 1, textAlign: 'center', padding: '10px', background: 'var(--bg-hover)', borderRadius: 'var(--radius-sm)' }}>
      <div style={{ fontSize: 20, fontWeight: 700, color: color || 'var(--text-primary)' }}>{value}</div>
      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{label}</div>
    </div>
  )
}
