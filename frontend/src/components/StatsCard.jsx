export default function StatsCard({ label, value, color }) {
  const c = color || 'var(--accent)'
  return (
    <div className="stats-card" style={{
      borderLeft: `3px solid ${c}`,
    }}>
      <div className="stats-card-value" style={{ color: c }}>{value}</div>
      <div className="stats-card-label">{label}</div>
      <div className="stats-card-glow" style={{ background: c, opacity: 0.06 }} />
    </div>
  )
}
