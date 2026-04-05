export default function StatsCard({ label, value, color }) {
  return (
    <div className="card" style={{ borderLeft: `3px solid ${color || 'var(--accent)'}` }}>
      <div style={{ fontSize: 28, fontWeight: 700, color: color || 'var(--text-primary)' }}>{value}</div>
      <div style={{ fontSize: 13, color: 'var(--text-muted)', marginTop: 4 }}>{label}</div>
    </div>
  )
}
