import React, { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '../services/api'
import '../styles/analysis.css'

export function AnalysisList() {
  const [scans, setScans] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.getAnalysisScans({ limit: 20 })
      .then(s => setScans(s.scans || []))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="loading">Loading scans...</div>

  return (
    <div>
      <h1 className="page-title">Scans</h1>

      <div className="scans-list">
        {scans.map(s => (
          <Link to={`/analysis/${s.id}`} key={s.id} className="scan-card">
            <div className="scan-card-header">
              <span className="scan-id">Scan #{s.id}</span>
              <span className="scan-time">{formatDate(s.ran_at)}</span>
            </div>
            <div className="scan-card-stats">
              <span className="scan-stat-item">{s.articles_new} new</span>
              <span className="scan-stat-item scan-stat-high">{s.high_count} HIGH</span>
              <span className="scan-stat-item scan-stat-medium">{s.medium_count} MED</span>
              <span className="scan-stat-item">{s.events_analyzed} events</span>
            </div>
          </Link>
        ))}
        {scans.length === 0 && (
          <div className="empty-state-card">
            <div className="empty-state-icon">
              <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
            </div>
            <p>No scans yet</p>
            <span>Pull to refresh</span>
          </div>
        )}
      </div>
    </div>
  )
}

export function AnalysisDetail() {
  const { id } = useParams()
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.getAnalysisScan(id).then(setData).finally(() => setLoading(false))
  }, [id])

  if (loading) return <div className="loading">Loading scan #{id}...</div>
  if (!data) return <div className="empty">Scan not found</div>

  const { scan, events } = data

  return (
    <div>
      <Link to="/analysis" className="back-link">&larr; Back to Scans</Link>
      <h1 className="page-title">Scan #{scan.id}</h1>

      <div className="scan-detail-header">
        <span className="scan-detail-date">{formatDate(scan.ran_at)}</span>
        <div className="scan-detail-stats">
          <span>{scan.articles_new} new articles</span>
          <span className="scan-stat-high">{scan.high_count} HIGH</span>
          <span className="scan-stat-medium">{scan.medium_count} MEDIUM</span>
          <span>{scan.events_analyzed} events</span>
        </div>
      </div>

      <div className="section-title">Events ({events.length})</div>

      {events.map((event, idx) => (
        <EventCard key={event.id} event={event} defaultExpanded={idx === 0} />
      ))}
    </div>
  )
}

function EventCard({ event, defaultExpanded = false }) {
  const [expanded, setExpanded] = useState(defaultExpanded)
  const signals = event.signals || []
  const buys = signals.filter(s => s.signal === 'BUY')
  const sells = signals.filter(s => s.signal === 'SELL')
  const topBuy = buys.sort((a, b) => (b.confidence || 0) - (a.confidence || 0))[0]
  const topSell = sells.sort((a, b) => (b.confidence || 0) - (a.confidence || 0))[0]

  return (
    <div className="event-card" data-severity={event.severity}>
      <div className="event-card-top" onClick={() => setExpanded(!expanded)}>
        <div className="event-header">
          <span className={`severity-badge severity-${event.severity?.toLowerCase()}`}>{event.severity}</span>
          <h3 className="event-headline">{event.headline}</h3>
        </div>
        <div className="event-meta">
          <span className="event-type-tag">{event.event_type?.replace(/_/g, ' ')}</span>
          {event.temporal && <span>Timeline: {event.temporal}</span>}
          {event.sectors && <span>Sectors: {Array.isArray(event.sectors) ? event.sectors.join(', ') : event.sectors}</span>}
        </div>

        {!expanded && signals.length > 0 && (
          <div className="event-preview">
            {topBuy && (
              <span className="preview-chip preview-buy">
                {topBuy.ticker} BUY {topBuy.impact_range} ({topBuy.confidence}%)
              </span>
            )}
            {topSell && (
              <span className="preview-chip preview-sell">
                {topSell.ticker} SELL {topSell.impact_range} ({topSell.confidence}%)
              </span>
            )}
            <span className="preview-count">{signals.length} signals</span>
          </div>
        )}

        <button className="expand-btn">{expanded ? 'Show less' : 'Show more'}</button>
      </div>

      {expanded && (
        <div className="event-card-body">
          {event.key_facts && (
            <div className="key-facts">
              <h4>Key Facts</h4>
              <ul>
                {(Array.isArray(event.key_facts) ? event.key_facts : [event.key_facts]).map((f, i) => (
                  <li key={i}>{f}</li>
                ))}
              </ul>
            </div>
          )}

          {event.analysis && (
            <div className="cascade-sections">
              {event.analysis.direct_impact && <AnalysisSection title="Direct Impact" data={event.analysis.direct_impact} />}
              {event.analysis.beneficiaries && <AnalysisSection title="Beneficiaries" data={event.analysis.beneficiaries} />}
              {event.analysis.demand_flow && <AnalysisSection title="Demand Flow" data={event.analysis.demand_flow} />}
              {event.analysis.supply_chain && <AnalysisSection title="Supply Chain" data={event.analysis.supply_chain} />}
              {event.analysis.sector_ripple && <AnalysisSection title="Sector Ripple" data={event.analysis.sector_ripple} />}
              {event.analysis.timeline && <AnalysisSection title="Timeline" data={event.analysis.timeline} />}
            </div>
          )}

          {event.signals?.length > 0 && (
            <div className="event-signals">
              <h4>Trading Signals</h4>
              <table className="signals-table compact">
                <thead>
                  <tr><th>Ticker</th><th>Signal</th><th>Direction</th><th>Impact</th><th>Confidence</th><th>Reason</th></tr>
                </thead>
                <tbody>
                  {event.signals.map((s, i) => (
                    <React.Fragment key={i}>
                      <tr>
                        <td className="ticker-cell">{s.ticker}</td>
                        <td><span className={`signal-badge signal-${s.signal?.toLowerCase()}`}>{s.signal}</span></td>
                        <td>{s.direction}</td>
                        <td>{s.impact_range}</td>
                        <td>{s.confidence}%</td>
                        <td className="reason-cell">{s.reason}</td>
                      </tr>
                      {s.reasoning_chain?.length > 0 && (
                        <tr className="reasoning-chain-row">
                          <td colSpan={6} className="reasoning-chain-cell">
                            <ol className="reasoning-chain">
                              {s.reasoning_chain.map((step, j) => (
                                <li key={j}>{step}</li>
                              ))}
                            </ol>
                          </td>
                        </tr>
                      )}
                    </React.Fragment>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {event.articles?.length > 0 && (
            <div className="event-articles">
              <h4>Source Articles ({event.articles.length})</h4>
              <div className="event-articles-list">
                {event.articles.map(a => (
                  <div key={a.hash} className="event-article-row">
                    <span className={`badge badge-${a.classification?.toLowerCase()}`}>{a.classification}</span>
                    {a.url ? (
                      <a href={a.url} target="_blank" rel="noopener noreferrer">{a.title}</a>
                    ) : (
                      <span>{a.title}</span>
                    )}
                    <span className="article-source">{a.source}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function AnalysisSection({ title, data }) {
  if (!data) return null
  return (
    <div className="analysis-section">
      <h5>{title}</h5>
      <div className="analysis-content">
        {typeof data === 'string' ? <p>{data}</p> :
          Array.isArray(data) ? <ul>{data.map((d, i) => <li key={i}>{typeof d === 'string' ? d : JSON.stringify(d)}</li>)}</ul> :
          typeof data === 'object' ? (
            <dl>{Object.entries(data).map(([k, v]) => (
              <div key={k}><dt>{k.replace(/_/g, ' ')}</dt><dd>{typeof v === 'string' ? v : JSON.stringify(v)}</dd></div>
            ))}</dl>
          ) : <p>{String(data)}</p>
        }
      </div>
    </div>
  )
}

function formatDate(d) {
  if (!d) return ''
  try {
    return new Date(d).toLocaleString('en-IN', { timeZone: 'Asia/Kolkata' })
  } catch {
    return d
  }
}
