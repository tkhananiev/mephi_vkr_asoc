export function severityTone(sev: string): 'crit' | 'high' | 'med' | 'low' | 'unk' {
  const s = sev.toLowerCase()
  if (s.includes('crit')) return 'crit'
  if (s === 'high' || s === 'h') return 'high'
  if (s.includes('med') || s === 'medium') return 'med'
  if (s === 'low') return 'low'
  return 'unk'
}
