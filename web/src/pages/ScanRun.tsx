import { useState, type FormEvent } from 'react'
import { runSemgrepScenario } from '../api/client'
import type { PassportResponse } from '../api/types'
import { PageFrame } from '../layout/PageFrame'

const defaultBody = {
  target_path: '/app/demo/vulnerable-app',
  semgrep_config: '/app/demo/semgrep-rules.yml',
}

export function ScanRun() {
  const [targetPath, setTargetPath] = useState(defaultBody.target_path)
  const [semgrepConfig, setSemgrepConfig] = useState(defaultBody.semgrep_config)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<PassportResponse | null>(null)

  async function onSubmit(e: FormEvent) {
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
      eyebrow="Сценарий"
      title="Скан и паспорт"
      lead="Полный проход: Semgrep → processing → группы → тикеты. Пути target и правила — внутри контейнера semgrep."
      badge="api-service :8080"
    >
      <div className="split">
        <div className="panel-elevated">
          <h2 className="card-title" style={{ marginTop: 0 }}>
            <span className="card-title-dot" aria-hidden />
            Параметры
          </h2>
          <form className="form-grid" onSubmit={onSubmit}>
            <label className="field">
              <span>target_path</span>
              <span className="hint">каталог с кодом внутри контейнера semgrep</span>
              <input
                value={targetPath}
                onChange={(e) => setTargetPath(e.target.value)}
                autoComplete="off"
                spellCheck={false}
              />
            </label>
            <label className="field">
              <span>semgrep_config</span>
              <span className="hint">путь к YAML или набор, например p/java</span>
              <input
                value={semgrepConfig}
                onChange={(e) => setSemgrepConfig(e.target.value)}
                autoComplete="off"
                spellCheck={false}
              />
            </label>
            <button className="btn btn-primary" type="submit" disabled={loading}>
              {loading ? 'Пайплайн выполняется…' : 'Запустить сценарий'}
            </button>
            {error ? <p className="err">{error}</p> : null}
          </form>
        </div>
        <div>
          <div className="code-window">
            <div className="code-window-hd">
              <span className="code-window-dots" aria-hidden>
                <i />
                <i />
                <i />
              </span>
              <span>response.json</span>
            </div>
            <textarea
              className="code-preview"
              readOnly
              placeholder="// ответ API появится здесь"
              value={result ? JSON.stringify(result, null, 2) : ''}
              spellCheck={false}
            />
          </div>
        </div>
      </div>
    </PageFrame>
  )
}
