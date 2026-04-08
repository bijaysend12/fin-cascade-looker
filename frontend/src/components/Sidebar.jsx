import { NavLink } from 'react-router-dom'
import { LayoutDashboard, Building2, Newspaper, Activity, LogOut } from 'lucide-react'
import { logOut } from '../services/firebase'
import { useAuth } from './AuthProvider'
import '../styles/sidebar.css'

const publicLinks = [
  { to: '/news', icon: Newspaper, label: 'News Feed' },
  { to: '/analysis', icon: Activity, label: 'Analysis' },
]

const adminLinks = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/companies', icon: Building2, label: 'Companies' },
  { to: '/news', icon: Newspaper, label: 'News Feed' },
  { to: '/analysis', icon: Activity, label: 'Analysis' },
]

export default function Sidebar() {
  const profile = useAuth()
  const isAdmin = profile?.is_admin
  const links = isAdmin ? adminLinks : publicLinks

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
      <div className="sidebar-footer">
        {profile && (
          <div className="sidebar-user">
            {profile.name || profile.email}
          </div>
        )}
        <button className="logout-btn" onClick={() => logOut()}>
          <LogOut size={16} />
          Sign out
        </button>
      </div>
    </aside>
  )
}
