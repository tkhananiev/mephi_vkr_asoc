import { PageFrame } from '../layout/PageFrame'

const ports = [
  { id: 'api', name: 'api-service', port: '8080', hint: 'внешний сценарий' },
  { id: 'ref', name: 'reference-data', port: '8081', hint: 'БДУ · NVD' },
  { id: 'prc', name: 'processing', port: '8082', hint: 'ingest · корреляция' },
  { id: 'jir', name: 'jira-integration', port: '8083', hint: 'тикеты' },
] as const

export function Dashboard() {
  return (
    <PageFrame
      eyebrow="Поток ASOC"
      title="Сводка контура"
      lead="Один публичный сценарий: Semgrep → нормализация → сопоставление с CVE/БДУ в каталоге → группы → задача в трекере. Справочник наполняется независимо от сканов."
      badge="Vite dev"
    >
      <div className="grid-stats">
        {ports.map((s) => (
          <div key={s.id} className="stat">
            <div className="stat-label">{s.name}</div>
            <div className="stat-desc">{s.hint}</div>
            <div className="stat-value">:{s.port}</div>
          </div>
        ))}
      </div>

      <div className="flow-section">
        <div className="card" style={{ marginTop: 0 }}>
          <h2 className="card-title">
            <span className="card-title-dot" aria-hidden />
            Скан → обработка → тикет
          </h2>
          <div className="flow-strip">
            <div className="flow-node">
              <h3>Клиент</h3>
              <p>UI / Postman — POST /scans/semgrep</p>
            </div>
            <span className="flow-connector" aria-hidden>
              →
            </span>
            <div className="flow-node">
              <h3>api + semgrep</h3>
              <p>Оркестратор, SAST, JSON</p>
            </div>
            <span className="flow-connector" aria-hidden>
              →
            </span>
            <div className="flow-node">
              <h3>ingest</h3>
              <p>Kafka или HTTP, корреляция</p>
            </div>
            <span className="flow-connector" aria-hidden>
              →
            </span>
            <div className="flow-node">
              <h3>Groups</h3>
              <p>Ключ по CVE, CWE, коду, версии</p>
            </div>
            <span className="flow-connector" aria-hidden>
              →
            </span>
            <div className="flow-node">
              <h3>Jira</h3>
              <p>REST, мок на стенде</p>
            </div>
          </div>
        </div>
      </div>

      <div className="flow-section">
        <div className="card">
          <h2 className="card-title">
            <span className="card-title-dot" aria-hidden />
            Справочник
          </h2>
          <div className="flow-grid flow-2" style={{ maxWidth: 720 }}>
            <div className="flow-node">
              <h3>reference-data</h3>
              <p>БДУ ФСТЭК (RSS) и NVD (API) в PostgreSQL</p>
            </div>
            <span className="flow-connector" aria-hidden>
              →
            </span>
            <div className="flow-node">
              <h3>Каталог</h3>
              <p>catalog · raw · audit — читается processing при корреляции</p>
            </div>
          </div>
        </div>
      </div>
    </PageFrame>
  )
}
