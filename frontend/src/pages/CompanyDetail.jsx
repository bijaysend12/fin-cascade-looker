import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { api } from '../services/api'
import KnowledgeGraph from '../components/KnowledgeGraph'
import '../styles/company-detail.css'

const TABS = ['Overview', 'Plants', 'Competitors', 'Supply Chain', 'Raw Materials', 'Graph']

export default function CompanyDetail() {
  const { ticker } = useParams()
  const navigate = useNavigate()
  const [data, setData] = useState(null)
  const [tab, setTab] = useState('Overview')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    api.getCompany(ticker).then(setData).finally(() => setLoading(false))
  }, [ticker])

  if (loading) return <div className="loading">Loading company data...</div>
  if (!data?.company) return <div className="empty">Company not found</div>

  const c = data.company

  return (
    <div>
      <div className="back-link" onClick={() => navigate('/companies')}>
        <ArrowLeft size={14} /> Back to Companies
      </div>

      <div className="company-header">
        <h1>{c.name}</h1>
        <span className="ticker">{c.ticker}</span>
        <span className={`badge badge-${c.marketCapCategory}`}>{c.marketCapCategory} cap</span>
        <span className="badge badge-mid">{c.sector}</span>
      </div>

      <div className="company-tabs">
        {TABS.map(t => {
          let count = ''
          if (t === 'Plants') count = ` (${data.plants?.length || 0})`
          if (t === 'Competitors') count = ` (${data.competitors?.length || 0})`
          if (t === 'Supply Chain') count = ` (${(data.suppliers?.length || 0) + (data.customers?.length || 0)})`
          if (t === 'Raw Materials') count = ` (${data.rawMaterials?.length || 0})`
          return (
            <button key={t} className={tab === t ? 'active' : ''} onClick={() => setTab(t)}>
              {t}{count}
            </button>
          )
        })}
      </div>

      {tab === 'Overview' && <OverviewTab company={c} sector={data.sector} />}
      {tab === 'Plants' && <PlantsTab plants={data.plants} />}
      {tab === 'Competitors' && <CompetitorsTab competitors={data.competitors} navigate={navigate} />}
      {tab === 'Supply Chain' && <SupplyChainTab suppliers={data.suppliers} customers={data.customers} navigate={navigate} />}
      {tab === 'Raw Materials' && <RawMaterialsTab materials={data.rawMaterials} />}
      {tab === 'Graph' && <GraphTab ticker={ticker} />}
    </div>
  )
}

function OverviewTab({ company, sector }) {
  return (
    <div className="card">
      <div className="info-grid">
        <InfoItem label="Industry" value={company.industry} />
        <InfoItem label="HQ City" value={company.hqCity} />
        <InfoItem label="HQ State" value={company.hqState} />
        <InfoItem label="Debt/Equity" value={company.debtToEquity?.toFixed(2)} />
        <InfoItem label="Export %" value={`${company.exportPct || 0}%`} />
        <InfoItem label="Market Cap" value={company.marketCapCategory} />
      </div>
      {company.description && (
        <div className="description-box">{company.description}</div>
      )}
      {sector && (
        <div style={{ marginTop: 16 }}>
          <div className="info-item-label" style={{ marginBottom: 8 }}>Sector Dependencies</div>
          <div style={{ display: 'flex', gap: 20 }}>
            <div>
              <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Upstream: </span>
              <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
                {(sector.upstreamDependencies || []).join(', ') || 'None'}
              </span>
            </div>
            <div>
              <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Downstream: </span>
              <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
                {(sector.downstreamSectors || []).join(', ') || 'None'}
              </span>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function InfoItem({ label, value }) {
  return (
    <div className="info-item">
      <div className="info-item-label">{label}</div>
      <div className="info-item-value">{value || '-'}</div>
    </div>
  )
}

function PlantsTab({ plants }) {
  if (!plants?.length) return <div className="empty">No plants found</div>
  return (
    <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>City</th>
            <th>State</th>
            <th>Type</th>
            <th>Capacity</th>
          </tr>
        </thead>
        <tbody>
          {plants.map((p, i) => (
            <tr key={i}>
              <td style={{ color: 'var(--text-primary)' }}>{p.name}</td>
              <td>{p.city}</td>
              <td>{p.state}</td>
              <td><span className="badge badge-mid">{p.type?.replace(/_/g, ' ')}</span></td>
              <td style={{ fontFamily: 'monospace', fontSize: 13 }}>{p.capacity || '-'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function CompetitorsTab({ competitors, navigate }) {
  if (!competitors?.length) return <div className="empty">No competitors found</div>
  return (
    <div className="relation-cards">
      {competitors.map(c => (
        <div key={c.ticker} className="relation-card" onClick={() => navigate(`/company/${encodeURIComponent(c.ticker)}`)}>
          <div className="relation-card-ticker">{c.ticker}</div>
          <div className="relation-card-name">{c.name}</div>
          {c.marketCapCategory && (
            <div className="relation-card-meta">
              <span className={`badge badge-${c.marketCapCategory}`}>{c.marketCapCategory}</span>
            </div>
          )}
        </div>
      ))}
    </div>
  )
}

function SupplyChainTab({ suppliers, customers, navigate }) {
  return (
    <div>
      <div className="supply-section">
        <h4>Suppliers ({suppliers?.length || 0})</h4>
        {suppliers?.length ? (
          <div className="relation-cards">
            {suppliers.map((s, i) => (
              <div key={i} className="relation-card" onClick={() => navigate(`/company/${encodeURIComponent(s.ticker)}`)}>
                <div className="relation-card-ticker">{s.ticker}</div>
                <div className="relation-card-name">{s.name}</div>
                {s.material && <div className="relation-card-meta">Supplies: {s.material}</div>}
              </div>
            ))}
          </div>
        ) : <div className="empty">No suppliers found</div>}
      </div>
      <div className="supply-section">
        <h4>Customers ({customers?.length || 0})</h4>
        {customers?.length ? (
          <div className="relation-cards">
            {customers.map((c, i) => (
              <div key={i} className="relation-card" onClick={() => navigate(`/company/${encodeURIComponent(c.ticker)}`)}>
                <div className="relation-card-ticker">{c.ticker}</div>
                <div className="relation-card-name">{c.name}</div>
                {c.material && <div className="relation-card-meta">Material: {c.material}</div>}
              </div>
            ))}
          </div>
        ) : <div className="empty">No customers found</div>}
      </div>
    </div>
  )
}

function RawMaterialsTab({ materials }) {
  if (!materials?.length) return <div className="empty">No raw materials found</div>
  return (
    <div className="relation-cards">
      {materials.map((m, i) => (
        <div key={i} className="material-card">
          <div className="material-name">{m.name}</div>
          <div className="material-meta">
            <span className={`imported-badge ${m.imported ? 'imported-yes' : 'imported-no'}`}>
              {m.imported ? 'Imported' : 'Domestic'}
            </span>
            {m.sourceCountry && <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>from {m.sourceCountry}</span>}
          </div>
        </div>
      ))}
    </div>
  )
}

function GraphTab({ ticker }) {
  const [graphData, setGraphData] = useState(null)

  useEffect(() => {
    api.getCompanyGraph(ticker).then(setGraphData)
  }, [ticker])

  if (!graphData) return <div className="loading">Loading graph...</div>
  return <KnowledgeGraph data={graphData} />
}
