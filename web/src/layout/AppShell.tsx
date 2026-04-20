import { NavLink, Outlet } from 'react-router-dom'

const nav: { to: string; label: string; icon: string; end?: boolean }[] = [
  { to: '/', label: 'Обзор', icon: '◆', end: true },
  { to: '/scan', label: 'Сканирование', icon: '◎' },
  { to: '/reference', label: 'Справочник', icon: '◇' },
  { to: '/groups', label: 'Группы', icon: '▣' },
]

export function AppShell() {
  return (
    <div className="app-root">
      <aside className="sidebar">
        <div className="sidebar-brand">
          <div className="sidebar-brand-title">ASPM Console</div>
          <div className="sidebar-brand-sub">оркестрация скана и тикетов</div>
        </div>
        <nav>
          {nav.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end === true}
              className={({ isActive }) => 'nav-link' + (isActive ? ' active' : '')}
            >
              <span className="nav-icon" aria-hidden>
                {item.icon}
              </span>
              {item.label}
            </NavLink>
          ))}
        </nav>
      </aside>
      <div className="main-wrap">
        <Outlet />
      </div>
    </div>
  )
}
