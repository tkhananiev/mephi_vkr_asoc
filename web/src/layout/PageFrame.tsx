import type { ReactNode } from 'react'

export function PageFrame({
  title,
  lead,
  badge,
  eyebrow,
  children,
}: {
  title: string
  lead?: string
  badge?: string
  /** Короткий ярлык над заголовком */
  eyebrow?: string
  children: ReactNode
}) {
  return (
    <>
      <header className="topbar">
        <div className="topbar-left">ASOC</div>
        {badge ? <span className="topbar-pill">{badge}</span> : null}
      </header>
      <main className="page">
        <div className="page-hero">
          {eyebrow ? <div className="page-eyebrow">{eyebrow}</div> : null}
          <h1 className="page-title">{title}</h1>
          {lead ? <p className="page-lead">{lead}</p> : null}
        </div>
        {children}
      </main>
    </>
  )
}
