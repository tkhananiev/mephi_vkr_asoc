import { PageFrame } from '../layout/PageFrame'

export function Dashboard() {
  return (
    <PageFrame
      title="Обзор"
      lead="Макет консоли: навигация слева, сверху контекст стенда. Ниже — логическая схема контура, с которой работает UI."
      badge="vite proxy → :8080–8083"
    >
      <div className="grid-stats">
        <div className="stat">
          <div className="stat-label">api-service</div>
          <div className="stat-value">8080</div>
        </div>
        <div className="stat">
          <div className="stat-label">reference-data</div>
          <div className="stat-value">8081</div>
        </div>
        <div className="stat">
          <div className="stat-label">processing</div>
          <div className="stat-value">8082</div>
        </div>
        <div className="stat">
          <div className="stat-label">jira-integration</div>
          <div className="stat-value">8083</div>
        </div>
      </div>

      <div className="card">
        <h2 className="card-title">Контур сканирования</h2>
        <div className="pipeline">
          <div className="pipeline-node">
            <strong>Клиент</strong>
            <span>эта консоль → POST /scans/semgrep</span>
          </div>
          <span className="pipeline-arrow" aria-hidden>
            →
          </span>
          <div className="pipeline-node">
            <strong>api-service</strong>
            <span>оркестратор, Semgrep, processing, тикеты</span>
          </div>
          <span className="pipeline-arrow" aria-hidden>
            →
          </span>
          <div className="pipeline-node">
            <strong>processing</strong>
            <span>ingest, корреляция, группы в PostgreSQL</span>
          </div>
          <span className="pipeline-arrow" aria-hidden>
            →
          </span>
          <div className="pipeline-node">
            <strong>jira-integration</strong>
            <span>задачи в mock / боевой Jira</span>
          </div>
        </div>
      </div>

      <div className="card">
        <h2 className="card-title">Справочник (параллельно)</h2>
        <div className="pipeline">
          <div className="pipeline-node">
            <strong>reference-data</strong>
            <span>синхронизация БДУ / NVD</span>
          </div>
          <span className="pipeline-arrow" aria-hidden>
            →
          </span>
          <div className="pipeline-node">
            <strong>PostgreSQL</strong>
            <span>схемы catalog, raw, audit</span>
          </div>
        </div>
      </div>

      <p className="page-lead" style={{ marginTop: '1.25rem', marginBottom: 0 }}>
        Дальше: страница «Сканирование» вызывает тот же сценарий, что curl к api-service; «Справочник» и «Группы» — прямые вызовы reference-data и processing через прокси Vite.
      </p>
    </PageFrame>
  )
}
