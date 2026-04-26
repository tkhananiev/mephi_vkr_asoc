import type { ReactNode } from 'react'
import { NavLink, Outlet } from 'react-router-dom'
import { BrandLogo } from '../components/BrandLogo'
import { IconDashboard, IconDatabase, IconLayers, IconScan } from '../components/Icons'

const nav: { to: string; label: string; end?: boolean; icon: ReactNode }[] = [
  { to: '/', label: 'Обзор', end: true, icon: <IconDashboard className="nav-icon" /> },
  { to: '/scan', label: 'Сканирование', icon: <IconScan className="nav-icon" /> },
  { to: '/reference', label: 'Справочник', icon: <IconDatabase className="nav-icon" /> },
  { to: '/groups', label: 'Группы', icon: <IconLayers className="nav-icon" /> },
]

export function AppShell() {
  return (
    <div className="app-root">
      <aside className="sidebar">
        <div className="sidebar-brand">
          <BrandLogo size={40} />
          <div className="sidebar-brand-text">
            <div className="sidebar-brand-title">ASOC</div>
            <div className="sidebar-brand-sub">orchestrated security findings → tickets</div>
          </div>
        </div>
        <nav>
          {nav.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end === true}
              className={({ isActive }) => 'nav-link' + (isActive ? ' active' : '')}
            >
              {item.icon}
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-footer">
          dev UI · прокси <code>/api</code> на стенд
        </div>
      </aside>
      <div className="main-wrap">
        <Outlet />
      </div>
    </div>
  )
}
