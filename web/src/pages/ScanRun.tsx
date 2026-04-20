import { useState } from 'react'
import { runSemgrepScenario } from '../api/client'
import type { PassportResponse } from '../api/types'
import { PageFrame } from '../layout/PageFrame'

const defaultBody = {
  scanner_name: 'semgrep',
  target_path: '/app/demo/vulnerable-app',
  semgrep_config: '/app/demo/semgrep-rules.yml',
}

export function ScanRun() {
  const [targetPath, setTargetPath] = useState(defaultBody.target_path)
  const [semgrepConfig, setSemgrepConfig] = useState(defaultBody.semgrep_config)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<PassportResponse | null>(null)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError(null)
    const out = await runSemgrepScenario({
      scanner_name: 'semgrep',
      target_path: targetPath.trim() || undefined,
      semgrep_config: semgrepConfig.trim() || undefined,
    })
    setLoading(false)
    if (!out.ok) {
      setResult(null)
      setError(out.error)
      return
    }
    setResult(out.data)
  }

  return (
    <PageFrame
      title="Сканирование"
      lead="Запуск полного сценария: Semgrep → обработка → группы → тикеты (как POST /api/v1/scans/semgrep на api-service)."
      badge="POST :8080"
    >
      <div className="split">
        <div>
          <form className="form-grid" onSubmit={onSubmit}>
            <label className="field">
              target_path (внутри контейнера semgrep)
              <input
                value={targetPath}
                onChange={(e) => setTargetPath(e.target.value)}
                autoComplete="off"
              />
            </label>
            <label className="field">
              semgrep_config
              <input
                value={semgrepConfig}
                onChange={(e) => setSemgrepConfig(e.target.value)}
                autoComplete="off"
              />
            </label>
            <button className="btn btn-primary" type="submit" disabled={loading}>
              {loading ? 'Выполняется…' : 'Запустить сценарий'}
            </button>
            {error ? <p className="err">{error}</p> : null}
          </form>
        </div>
        <div className="card" style={{ margin: 0 }}>
          <h2 className="card-title">Ответ (паспорт)</h2>
          <textarea
            className="code-preview"
            readOnly
            value={result ? JSON.stringify(result, null, 2) : '—'}
            spellCheck={false}
          />
        </div>
      </div>
    </PageFrame>
  )
}
