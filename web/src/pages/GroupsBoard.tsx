import { useCallback, useEffect, useState } from 'react'
import { fetchGroups } from '../api/client'
import type { GroupRow } from '../api/types'
import { PageFrame } from '../layout/PageFrame'
import { severityTone } from '../lib/severityStyle'

function sevClass(t: ReturnType<typeof severityTone>) {
  switch (t) {
    case 'crit':
      return 'badge sev-crit'
    case 'high':
      return 'badge sev-high'
    case 'med':
      return 'badge sev-med'
    case 'low':
      return 'badge sev-low'
    default:
      return 'badge sev-unk'
  }
}

export function GroupsBoard() {
  const [rows, setRows] = useState<GroupRow[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    const r = await fetchGroups(200)
    setLoading(false)
    if (!r.ok) {
      setRows(null)
      setError(r.error)
      return
    }
    setRows(r.data)
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  return (
    <PageFrame
      eyebrow="Операционка"
      title="Группы уязвимостей"
      lead="Агрегаты после ingest и корреляции. Одна группа — один логический тикет. Обновите список после скана."
      badge="processing :8082"
    >
      <div className="toolbar-row">
        <button type="button" className="btn btn-ghost" onClick={() => void load()}>
          Обновить
        </button>
        {loading ? <span style={{ color: 'var(--text-faint)', fontSize: '0.85rem' }}>Загрузка…</span> : null}
      </div>
      {error ? <p className="err">{error}</p> : null}
      <div className="table-wrap">
        <table className="data">
          <thead>
            <tr>
              <th>id</th>
              <th>severity</th>
              <th>assets</th>
              <th>status</th>
              <th>group_key</th>
              <th>rule</th>
            </tr>
          </thead>
          <tbody>
            {rows === null && !error ? (
              <tr>
                <td colSpan={6} className="table-empty">
                  …
                </td>
              </tr>
            ) : rows && rows.length === 0 ? (
              <tr>
                <td colSpan={6} className="table-empty">
                  Групп нет. Сначала выполните сценарий на странице «Сканирование».
                </td>
              </tr>
            ) : (
              rows?.map((g) => (
                <tr key={g.id}>
                  <td>
                    <code style={{ color: 'var(--accent-bright)' }}>{g.id}</code>
                  </td>
                  <td>
                    <span className={sevClass(severityTone(g.severity_max))}>{g.severity_max}</span>
                  </td>
                  <td>{g.assets_count}</td>
                  <td>
                    <span className="badge">{g.status}</span>
                  </td>
                  <td
                    style={{
                      maxWidth: 380,
                      wordBreak: 'break-all',
                      fontSize: '0.76rem',
                      fontFamily: 'var(--font-mono)',
                      color: 'var(--text-muted)',
                    }}
                  >
                    {g.group_key}
                  </td>
                  <td>
                    <span style={{ color: 'var(--text-faint)', fontSize: '0.78rem' }}>{g.grouping_rule}</span>
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
