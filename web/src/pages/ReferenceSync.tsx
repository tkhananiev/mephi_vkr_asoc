import { useCallback, useState } from 'react'
import { fetchSyncRuns, postSync } from '../api/client'
import type { SyncRunRow } from '../api/types'
import { PageFrame } from '../layout/PageFrame'

export function ReferenceSync() {
  const [runs, setRuns] = useState<SyncRunRow[] | null>(null)
  const [msg, setMsg] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)

  const loadRuns = useCallback(async () => {
    setMsg(null)
    const r = await fetchSyncRuns(20)
    if (!r.ok) {
      setRuns(null)
      setMsg(r.error)
      return
    }
    setRuns(r.data)
  }, [])

  async function run(path: '/api/v1/sync/bdu' | '/api/v1/sync/nvd' | '/api/v1/sync/all', query = '') {
    setBusy(path + query)
    setMsg(null)
    const r = await postSync(path, query)
    setBusy(null)
    if (!r.ok) {
      setMsg(r.error)
      return
    }
    setMsg(`Запрос принят (HTTP ${r.status}). Обновите таблицу прогонов.`)
    await loadRuns()
  }

  return (
    <PageFrame
      title="Справочник CVE / БДУ"
      lead="Управление синхронизацией reference-data-service. Полный NVD может занимать много времени; для проверки используйте один CVE."
      badge=":8081"
    >
      <div className="btn-row" style={{ marginBottom: '1rem' }}>
        <button
          type="button"
          className="btn btn-ghost"
          disabled={!!busy}
          onClick={() => run('/api/v1/sync/bdu')}
        >
          Синк БДУ
        </button>
        <button
          type="button"
          className="btn btn-ghost"
          disabled={!!busy}
          onClick={() => run('/api/v1/sync/nvd', '?cve_id=CVE-2021-44228')}
        >
          NVD: CVE-2021-44228
        </button>
        <button
          type="button"
          className="btn btn-ghost"
          disabled={!!busy}
          onClick={() => run('/api/v1/sync/nvd')}
        >
          NVD полный
        </button>
        <button
          type="button"
          className="btn btn-ghost"
          disabled={!!busy}
          onClick={() => run('/api/v1/sync/all')}
        >
          БДУ + NVD
        </button>
        <button type="button" className="btn btn-primary" onClick={() => loadRuns()}>
          Обновить прогоны
        </button>
      </div>
      {busy ? <p className="page-lead">Выполняется: {busy}…</p> : null}
      {msg ? <p className="page-lead">{msg}</p> : null}

      <div className="table-wrap">
        <table className="data">
          <thead>
            <tr>
              <th>id</th>
              <th>source</th>
              <th>status</th>
              <th>discovered</th>
              <th>processed</th>
              <th>started</th>
            </tr>
          </thead>
          <tbody>
            {runs === null ? (
              <tr>
                <td colSpan={6} style={{ color: 'var(--text-muted)' }}>
                  Нажмите «Обновить прогоны»
                </td>
              </tr>
            ) : runs.length === 0 ? (
              <tr>
                <td colSpan={6} style={{ color: 'var(--text-muted)' }}>
                  Пока нет записей
                </td>
              </tr>
            ) : (
              runs.map((row) => (
                <tr key={row.id}>
                  <td>{row.id}</td>
                  <td>{row.source_code}</td>
                  <td>
                    <span className="badge">{row.status}</span>
                  </td>
                  <td>{row.items_discovered ?? '—'}</td>
                  <td>{row.items_processed ?? '—'}</td>
                  <td style={{ whiteSpace: 'nowrap' }}>
                    {row.started_at ? String(row.started_at).replace('T', ' ').slice(0, 19) : '—'}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </PageFrame>
  )
}
