import { useEffect, useRef, useState } from 'react'
import * as d3 from 'd3'
import '../styles/graph.css'

const NODE_COLORS = {
  Company: '#6366f1',
  Competitor: '#ef4444',
  Plant: '#22c55e',
  Sector: '#f59e0b',
  Supplier: '#8b5cf6',
  Customer: '#06b6d4',
  RawMaterial: '#ec4899',
  Location: '#64748b',
}

const NODE_RADIUS = {
  Company: 24,
  Competitor: 16,
  Supplier: 16,
  Customer: 16,
  Sector: 18,
  Plant: 12,
  RawMaterial: 13,
  Location: 12,
}

export default function KnowledgeGraph({ data }) {
  const svgRef = useRef()
  const [selected, setSelected] = useState(null)

  useEffect(() => {
    if (!data?.nodes?.length) return

    const svg = d3.select(svgRef.current)
    svg.selectAll('*').remove()

    const rect = svgRef.current.getBoundingClientRect()
    const width = rect.width || 900
    const height = rect.height || 600

    const g = svg.append('g')

    const zoom = d3.zoom()
      .scaleExtent([0.2, 4])
      .on('zoom', (event) => g.attr('transform', event.transform))
    svg.call(zoom)

    const nodes = data.nodes.map(d => ({ ...d }))
    const links = data.links.map(d => ({ ...d }))

    const simulation = d3.forceSimulation(nodes)
      .force('link', d3.forceLink(links).id(d => d.id).distance(120))
      .force('charge', d3.forceManyBody().strength(-300))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collision', d3.forceCollide().radius(d => (NODE_RADIUS[d.type] || 14) + 8))

    const link = g.append('g')
      .selectAll('line')
      .data(links)
      .join('line')
      .attr('stroke', '#2a2a3a')
      .attr('stroke-width', 1.5)
      .attr('stroke-opacity', 0.6)

    const linkLabel = g.append('g')
      .selectAll('text')
      .data(links)
      .join('text')
      .text(d => d.type)
      .attr('font-size', 8)
      .attr('fill', '#6a6a7a')
      .attr('text-anchor', 'middle')

    const node = g.append('g')
      .selectAll('circle')
      .data(nodes)
      .join('circle')
      .attr('r', d => NODE_RADIUS[d.type] || 14)
      .attr('fill', d => NODE_COLORS[d.type] || '#6366f1')
      .attr('stroke', d => d.isCenter ? '#fff' : 'none')
      .attr('stroke-width', d => d.isCenter ? 2 : 0)
      .attr('cursor', 'pointer')
      .on('click', (event, d) => setSelected(d))
      .call(d3.drag()
        .on('start', (event, d) => {
          if (!event.active) simulation.alphaTarget(0.3).restart()
          d.fx = d.x; d.fy = d.y
        })
        .on('drag', (event, d) => { d.fx = event.x; d.fy = event.y })
        .on('end', (event, d) => {
          if (!event.active) simulation.alphaTarget(0)
          d.fx = null; d.fy = null
        })
      )

    const label = g.append('g')
      .selectAll('text')
      .data(nodes)
      .join('text')
      .text(d => d.label?.length > 18 ? d.label.slice(0, 16) + '..' : d.label)
      .attr('font-size', d => d.isCenter ? 12 : 10)
      .attr('font-weight', d => d.isCenter ? 700 : 400)
      .attr('fill', '#f0f0f5')
      .attr('text-anchor', 'middle')
      .attr('dy', d => (NODE_RADIUS[d.type] || 14) + 14)
      .attr('pointer-events', 'none')

    simulation.on('tick', () => {
      link
        .attr('x1', d => d.source.x).attr('y1', d => d.source.y)
        .attr('x2', d => d.target.x).attr('y2', d => d.target.y)

      linkLabel
        .attr('x', d => (d.source.x + d.target.x) / 2)
        .attr('y', d => (d.source.y + d.target.y) / 2)

      node.attr('cx', d => d.x).attr('cy', d => d.y)
      label.attr('x', d => d.x).attr('y', d => d.y)
    })

    return () => simulation.stop()
  }, [data])

  const legendTypes = [...new Set(data?.nodes?.map(n => n.type) || [])]

  return (
    <div className="graph-container">
      <div className="graph-legend">
        {legendTypes.map(t => (
          <div key={t} className="graph-legend-item">
            <div className="graph-legend-dot" style={{ background: NODE_COLORS[t] || '#6366f1' }} />
            {t}
          </div>
        ))}
      </div>
      <svg ref={svgRef} className="graph-svg" />
      {selected && (
        <div className="graph-details">
          <h4>{selected.label}</h4>
          <div className="graph-detail-row">
            <span className="graph-detail-key">Type</span>
            <span className="graph-detail-value">{selected.type}</span>
          </div>
          <div className="graph-detail-row">
            <span className="graph-detail-key">ID</span>
            <span className="graph-detail-value">{selected.id}</span>
          </div>
          {selected.properties && Object.entries(selected.properties).map(([k, v]) => (
            <div key={k} className="graph-detail-row">
              <span className="graph-detail-key">{k}</span>
              <span className="graph-detail-value">{String(v)}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
