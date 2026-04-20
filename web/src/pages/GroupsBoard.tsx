import { useEffect, useState } from 'react'
import { fetchGroups } from '../api/client'
import type { GroupRow } from '../api/types'
import { PageFrame } from '../layout/PageFrame'

export function GroupsBoard() {
  const [rows, setRows] = useState<GroupRow[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      const r = await fetchGroups(100)
      if (cancelled) return
      if (!r.ok) {
        setRows(null)
        setError(r.error)
        return
      }
      setError(null)
      setRows(r.data)
    })()
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <PageFrame
      title="Группы уязвимостей"
      lead="Список из processing-service после ingest и корреляции (GET /api/v1/groups)."
      badge="GET :8082"
    >
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
                <td colSpan={6} style={{ color: 'var(--text-muted)' }}>
                  Загрузка…
                </td>
              </tr>
            ) : rows && rows.length === 0 ? (
              <tr>
                <td colSpan={6} style={{ color: 'var(--text-muted)' }}>
                  Групп нет — сначала выполните сканирование
                </td>
              </tr>
            ) : (
              rows?.map((g) => (
                <tr key={g.id}>
                  <td>{g.id}</td>
                  <td>
                    <span className="badge">{g.severity_max}</span>
                  </td>
                  <td>{g.assets_count}</td>
                  <td>{g.status}</td>
                  <td style={{ maxWidth: 360, wordBreak: 'break-all' }}>{g.group_key}</td>
                  <td>{g.grouping_rule}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </PageFrame>
  )
}
