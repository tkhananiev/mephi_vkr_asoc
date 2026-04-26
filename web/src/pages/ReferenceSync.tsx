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

  async function run(path: '/api/v1/sync/bdu' | '/api/v1/sync/nvd' | '/api/v1/sync/all', query = '', label: string) {
    setBusy(label)
    setMsg(null)
    const r = await postSync(path, query)
    setBusy(null)
    if (!r.ok) {
      setMsg(r.error)
      return
    }
    setMsg(`Принято (HTTP ${r.status}). Актуализируйте список прогонов.`)
    await loadRuns()
  }

  return (
    <PageFrame
      eyebrow="Справочник"
      title="БДУ · NVD"
      lead="Ручной синк с ФСТЭК и NIST. Полный NVD может идти долго — для демо достаточно одного CVE."
      badge="reference-data :8081"
    >
      {msg ? <div className="msg-banner">{msg}</div> : null}
      {busy ? (
        <div className="msg-banner" style={{ borderColor: 'rgba(99, 102, 241, 0.4)' }}>
          Выполняется: {busy}…
        </div>
      ) : null}

      <div className="action-grid">
        <button
          type="button"
          className="action-card accent-bdu"
          disabled={!!busy}
          onClick={() => run('/api/v1/sync/bdu', '', 'БДУ (RSS)')}
        >
          <h4>БДУ ФСТЭК</h4>
          <p>RSS-лента, запись в catalog / raw / audit</p>
        </button>
        <button
          type="button"
          className="action-card accent-nvd"
          disabled={!!busy}
          onClick={() => run('/api/v1/sync/nvd', '?cve_id=CVE-2021-44228', 'NVD одна CVE')}
        >
          <h4>NVD · одна CVE</h4>
          <p>Быстрый тест, например Log4Shell</p>
        </button>
        <button
          type="button"
          className="action-card accent-nvd"
          disabled={!!busy}
          onClick={() => run('/api/v1/sync/nvd', '', 'NVD полный')}
        >
          <h4>NVD · полный прогон</h4>
          <p>Долго, нужен API key и сеть</p>
        </button>
        <button
          type="button"
          className="action-card accent-warn"
          disabled={!!busy}
          onClick={() => run('/api/v1/sync/all', '', 'БДУ + NVD')}
        >
          <h4>БДУ + NVD</h4>
          <p>Оба источника подряд</p>
        </button>
        <button type="button" className="action-card" onClick={() => loadRuns()}>
          <h4>Обновить прогоны</h4>
          <p>GET /api/v1/sync/runs</p>
        </button>
      </div>

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
                <td colSpan={6} className="table-empty">
                  Нажмите «Обновить прогоны», чтобы подтянуть audit.reference_sync_runs
                </td>
              </tr>
            ) : runs.length === 0 ? (
              <tr>
                <td colSpan={6} className="table-empty">
                  Записей ещё нет
                </td>
              </tr>
            ) : (
              runs.map((row) => (
                <tr key={row.id}>
                  <td>{row.id}</td>
                  <td>
                    <code style={{ color: 'var(--accent-2)' }}>{row.source_code}</code>
                  </td>
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
