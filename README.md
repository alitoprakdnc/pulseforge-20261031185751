# PulseForge

PulseForge is an AI‑powered CI/CD observability and optimization platform. It continuously monitors pipelines across GitHub Actions, GitLab CI, and CircleCI, detects queue starvation, health‑check failures, and performance regressions, and surfaces actionable AI‑driven recommendations. Teams get a real‑time dashboard, secure webhook ingestion with constant‑time HMAC validation, secret‑manager integration, and an in‑product AI chat that can rewrite flaky steps, suggest caching strategies, and generate OpenAPI‑compatible health endpoints automatically.

## Technologies

- Go (Gin)
- PostgreSQL
- Redis
- RabbitMQ
- Docker / Docker‑Compose
- Next.js (React, TypeScript)
- Tailwind CSS
- OpenAI / Ollama (LLM adapters)
- Google Cloud Secret Manager (optional)
- GitHub GraphQL API

## Features

- Unified pipeline ingestion from GitHub Actions, GitLab CI, CircleCI via secure webhooks
- Constant‑time HMAC validation for webhook authenticity
- Real‑time health‑check endpoint generator (OpenAPI 3.0) for each pipeline
- AI engine that analyses recent job logs, detects patterns, and suggests concrete refactorings (e.g., caching, parallelism, timeout adjustments)
- Queue starvation detector that monitors pending jobs across providers and raises alerts
- Dashboard with pipeline overview, job timelines, error heat‑maps, and AI‑generated remediation cards
- In‑product AI chat (LLM‑backed) that can answer “why did this job fail?” and rewrite YAML snippets
- Integration with Google Cloud Secret Manager / HashiCorp Vault for secure storage of tokens
- Alerting via email, Slack, and webhook to external incident tools
- Server‑less task runner (RabbitMQ + Go workers) that processes heavy log analysis and LLM calls asynchronously
- Exportable OpenAPI spec for the health‑check API and webhook contract

## Architecture

The system follows a micro‑service‑ish monolith pattern.

**Backend (Go)**
- `gin-gonic` HTTP server exposing REST + GraphQL endpoints.
- Middleware stack: authentication (JWT), HMAC validation, request logging, panic recovery.
- PostgreSQL stores users, pipelines, jobs, AI suggestions, and alert rules.
- Redis caches recent job logs and AI responses for low latency.
- RabbitMQ drives two background workers: `queue_monitor` (polls CI providers for queue length) and `ai_suggester` (feeds logs to LLM and stores suggestions).
- Secret manager client abstracts Google Secret Manager or Vault; secrets are never persisted in DB.
- OpenAPI 3.0 spec is generated from annotated handlers and served via Swagger UI.

**Frontend (Next.js)**
- TypeScript React SPA with server‑side rendering for SEO‑critical pages.
- Tailwind‑CSS design system (spacing, colors, typography) ensures a premium look.
- Auth handled via JWT stored in HttpOnly cookie; React context provides user state.
- Data fetching via SWR hooks; real‑time updates via WebSocket (optional) or polling.
- AI chat component streams LLM responses.

**Deployment**
- Docker‑Compose for local dev (services: api, web, db, redis, rabbitmq).
- Production can be deployed to Kubernetes; Helm chart omitted for brevity.
- Health‑check endpoint `/api/health` returns JSON with DB, Redis, and worker status.

**Security**
- All external webhook endpoints require HMAC signature verified in constant time.
- Rate limiting per IP using Redis token bucket.
- CSP, XSS protection, and secure cookie flags.
- Auditing logs written to PostgreSQL and forwarded to external SIEM via syslog.

**Scalability**
- Horizontal scaling of API containers behind a load balancer.
- Workers can be scaled independently; job queue persists in RabbitMQ.
- Read‑replica of PostgreSQL optional for analytics.


## Product Design

PulseForge adopts a clean, enterprise‑grade UI built on Tailwind CSS and a custom component library:
- **Typography**: Inter (400/500/600) for body, headings; 14‑px base, 1.5 line‑height.
- **Color palette**: Primary #0F62FE, Secondary #6F6F6F, Success #24A148, Danger #DA1E28, Background #F4F7FA, Card #FFFFFF, Border #E0E0E0.
- **Spacing**: 4‑px grid (4,8,12,16,24,32,48).
- **Components**: Card, Table, Badge, Avatar, Modal, Tabs, Toast, LoadingSkeleton, AIChatBubble.
- **Interaction states**: hover/active shadows, focus rings, disabled opacity.
- **Responsive layout**: Sidebar collapses to hamburger on <768px; dashboard grid reflows from 4‑col to 1‑col.
- **Accessibility**: ARIA labels, keyboard navigation, color contrast ≥4.5:1.
- **Charts**: Recharts (line for queue length, heat‑map for failure frequency).
- **Empty/Loading/Error states**: Illustrated placeholders with concise copy and CTA buttons.
The UI is fully functional; components are coded as reusable React/TSX files with props typed for strict contracts.

## Testing

Comprehensive testing pyramid:
- **Unit**: Go `testing` + `testify` for services, repositories, utils; Jest + React Testing Library for UI components.
- **Integration**: Test API routes with `httptest` server and a Docker‑Compose‑based PostgreSQL/Redis fixture.
- **End‑to‑End**: Cypress tests covering login, pipeline ingestion, dashboard rendering, AI chat flow, and alert creation.
- **Load/Performance**: k6 scripts simulating 500 concurrent webhook posts and 200 dashboard users.
- **Security**: Automated scans for HMAC timing attacks, JWT tampering, and secret leakage using `gosec` and `npm audit`.
- **CI**: GitHub Actions pipeline runs lint, unit, integration, and Cypress on each PR; nightly k6 load test.
All test artifacts are versioned in the repo under `tests/`.

## Status

Generated by QuarkForge. Review and test before deployment. Automated test status is recorded in the generation logs.
