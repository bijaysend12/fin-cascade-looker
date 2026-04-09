import { useState } from 'react'
import { NavLink } from 'react-router-dom'
import { LayoutDashboard, Building2, Newspaper, Activity, LogOut, Menu, X } from 'lucide-react'
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
  const [open, setOpen] = useState(false)
  const profile = useAuth()
  const isAdmin = profile?.is_admin
  const links = isAdmin ? adminLinks : publicLinks

  const close = () => setOpen(false)

  return (
    <>
      <button className="mobile-menu-btn" onClick={() => setOpen(true)}>
        <Menu size={22} />
      </button>

      {open && <div className="sidebar-backdrop" onClick={close} />}

      <aside className={`sidebar ${open ? 'sidebar-open' : ''}`}>
        <div className="sidebar-header">
          <h1>Fin Cascade</h1>
          <p>Knowledge Graph Dashboard</p>
          <button className="sidebar-close-btn" onClick={close}>
            <X size={20} />
          </button>
        </div>
        <nav className="sidebar-nav">
          {links.map(({ to, icon: Icon, label }) => (
            <NavLink key={to} to={to} end={to === '/'} className={({ isActive }) => isActive ? 'active' : ''} onClick={close}>
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
    </>
  )
}
