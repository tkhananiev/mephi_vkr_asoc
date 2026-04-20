import type { GroupRow, PassportResponse, ScanRequestBody, SyncRunRow } from './types'

async function parseJSON<T>(res: Response): Promise<T> {
  const text = await res.text()
  if (!text) return {} as T
  return JSON.parse(text) as T
}

export async function runSemgrepScenario(
  body: ScanRequestBody,
): Promise<{ ok: true; data: PassportResponse } | { ok: false; error: string }> {
  const res = await fetch('/api/v1/scans/semgrep', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const err = await parseJSON<{ error?: string }>(res).catch(() => ({}) as { error?: string })
    return { ok: false, error: err.error ?? `HTTP ${res.status}` }
  }
  const data = await parseJSON<PassportResponse>(res)
  return { ok: true, data }
}

export async function fetchGroups(
  limit = 50,
): Promise<{ ok: true; data: GroupRow[] } | { ok: false; error: string }> {
  const res = await fetch(`/api/v1/groups?limit=${limit}`)
  if (!res.ok) {
    return { ok: false, error: `HTTP ${res.status}` }
  }
  const data = await parseJSON<GroupRow[]>(res)
  return { ok: true, data }
}

export async function postSync(
  path: '/api/v1/sync/bdu' | '/api/v1/sync/nvd' | '/api/v1/sync/all',
  query = '',
): Promise<{ ok: true; status: number } | { ok: false; error: string }> {
  const res = await fetch(path + query, { method: 'POST' })
  if (!res.ok) {
    const err = await parseJSON<{ error?: string }>(res).catch(() => ({}) as { error?: string })
    return { ok: false, error: err.error ?? `HTTP ${res.status}` }
  }
  return { ok: true, status: res.status }
}

export async function fetchSyncRuns(
  limit = 15,
): Promise<{ ok: true; data: SyncRunRow[] } | { ok: false; error: string }> {
  const res = await fetch(`/api/v1/sync/runs?limit=${limit}`)
  if (!res.ok) {
    return { ok: false, error: `HTTP ${res.status}` }
  }
  const data = await parseJSON<SyncRunRow[]>(res)
  return { ok: true, data }
}
