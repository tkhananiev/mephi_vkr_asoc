import type { ReactNode } from 'react'

export function PageFrame({
  title,
  lead,
  badge,
  children,
}: {
  title: string
  lead?: string
  badge?: string
  children: ReactNode
}) {
  return (
    <>
      <header className="topbar">
        <span className="topbar-title">ASPM Console</span>
        {badge ? <span className="topbar-pill">{badge}</span> : null}
      </header>
      <main className="page">
        <h1>{title}</h1>
        {lead ? <p className="page-lead">{lead}</p> : null}
        {children}
      </main>
    </>
  )
}
