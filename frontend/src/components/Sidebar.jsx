import { NavLink } from 'react-router-dom'
import { LayoutDashboard, Building2, Newspaper } from 'lucide-react'
import '../styles/sidebar.css'

const links = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/companies', icon: Building2, label: 'Companies' },
  { to: '/news', icon: Newspaper, label: 'News Feed' },
]

export default function Sidebar() {
  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <h1>Fin Cascade</h1>
        <p>Knowledge Graph Dashboard</p>
      </div>
      <nav className="sidebar-nav">
        {links.map(({ to, icon: Icon, label }) => (
          <NavLink key={to} to={to} end={to === '/'} className={({ isActive }) => isActive ? 'active' : ''}>
            <Icon size={18} />
            {label}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
