/** Минималистичные SVG-иконки навигации (fill=currentColor) */

export function IconDashboard({ className }: { className?: string }) {
  return (
    <svg className={className} width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden>
      <path
        d="M4 4h7v7H4V4zm9 0h7v4h-7V4zM4 13h7v7H4v-7zm9 3h7v4h-7v-4zm0-7h4v3h-4V9z"
        fill="currentColor"
        opacity="0.92"
      />
    </svg>
  )
}

export function IconScan({ className }: { className?: string }) {
  return (
    <svg className={className} width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden>
      <path
        d="M4 8a4 4 0 0 1 4-4h2v2H8a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2v-2h2v2a4 4 0 0 1-4 4H8a4 4 0 0 1-4-4V8Z"
        fill="currentColor"
        opacity="0.88"
      />
      <path
        d="M15 3h6v6h-2V7.41l-7.3 7.3-1.4-1.42L17.6 6H15V3Z"
        fill="currentColor"
        opacity="0.95"
      />
    </svg>
  )
}

export function IconDatabase({ className }: { className?: string }) {
  return (
    <svg className={className} width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden>
      <ellipse cx="12" cy="5" rx="7" ry="3" stroke="currentColor" strokeWidth="1.5" fill="none" />
      <path
        d="M5 5v4c0 1.2 2.5 2.2 7 2.2s7-1 7-2.2V5M5 9v4c0 1.2 2.5 2.2 7 2.2s7-1 7-2.2V9M5 13v2c0 1.2 2.5 2.2 7 2.2s7-1 7-2.2v-2"
        stroke="currentColor"
        strokeWidth="1.5"
        fill="none"
        strokeLinecap="round"
      />
    </svg>
  )
}

export function IconLayers({ className }: { className?: string }) {
  return (
    <svg className={className} width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden>
      <path
        d="M12 2 3 7l9 5 9-5-9-5Zm-9 9 9 5 9-5M3 12l9 5 9-5M3 17l9 5 9-5"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        fill="none"
      />
    </svg>
  )
}
