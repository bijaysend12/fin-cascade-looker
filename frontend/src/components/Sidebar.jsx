import { useState } from 'react'
import { NavLink } from 'react-router-dom'
import { Home, CandlestickChart, Radar, Newspaper, LayoutDashboard, Building2, LogOut, Menu, X, User } from 'lucide-react'
import { logOut } from '../services/firebase'
import { useAuth } from './AuthProvider'
import '../styles/sidebar.css'

const mainLinks = [
  { to: '/', icon: Home, label: 'Home' },
  { to: '/stocks', icon: CandlestickChart, label: 'Stocks' },
  { to: '/analysis', icon: Radar, label: 'Scans' },
  { to: '/news', icon: Newspaper, label: 'News' },
]

const adminLinks = [
  { to: '/admin', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/companies', icon: Building2, label: 'Companies' },
]

export default function Sidebar() {
  const [open, setOpen] = useState(false)
  const profile = useAuth()
  const isAdmin = profile?.is_admin

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
          <p>Market Intelligence</p>
          <button className="sidebar-close-btn" onClick={close}>
            <X size={20} />
          </button>
        </div>
        <nav className="sidebar-nav">
          <div className="nav-section-label">Main</div>
          {mainLinks.map(({ to, icon: Icon, label }) => (
            <NavLink key={to} to={to} end={to === '/'} className={({ isActive }) => isActive ? 'active' : ''} onClick={close}>
              <Icon size={18} />
              {label}
            </NavLink>
          ))}
          {isAdmin && (
            <>
              <div className="nav-section-label" style={{ marginTop: 16 }}>Admin</div>
              {adminLinks.map(({ to, icon: Icon, label }) => (
                <NavLink key={to} to={to} className={({ isActive }) => isActive ? 'active' : ''} onClick={close}>
                  <Icon size={18} />
                  {label}
                </NavLink>
              ))}
            </>
          )}
        </nav>
        <div className="sidebar-footer">
          {profile && (
            <div className="sidebar-user">
              <div className="sidebar-avatar">
                <User size={14} />
              </div>
              <span>{profile.name || profile.email}</span>
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
