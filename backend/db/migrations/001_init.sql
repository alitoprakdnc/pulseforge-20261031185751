-- 001_init.sql: Initial schema for PulseForge
-- ------------------------------------------------------------
-- This migration creates the core tables used by the backend service:
--   users, pipelines, jobs, ai_suggestions
-- It also defines supporting enum types, extensions, and timestamp
-- maintenance triggers.

-- Enable required extensions (idempotent)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ------------------------------------------------------------------
-- Enum definitions
-- ------------------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'provider_enum') THEN
        CREATE TYPE provider_enum AS ENUM ('github', 'gitlab', 'circleci');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'job_status_enum') THEN
        CREATE TYPE job_status_enum AS ENUM ('queued', 'running', 'success', 'failed', 'canceled');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role_enum') THEN
        CREATE TYPE user_role_enum AS ENUM ('admin', 'user', 'viewer');
    END IF;
END$$;

-- ------------------------------------------------------------------
-- Timestamp helper trigger
-- ------------------------------------------------------------------
CREATE OR REPLACE FUNCTION set_updated_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ------------------------------------------------------------------
-- Table: users
-- ------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255),
    password_hash VARCHAR(255) NOT NULL,
    role user_role_enum NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Trigger to keep updated_at fresh
CREATE TRIGGER trg_users_updated
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_timestamp();

-- ------------------------------------------------------------------
-- Table: pipelines
-- ------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pipelines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider provider_enum NOT NULL,
    repo_owner VARCHAR(255) NOT NULL,
    repo_name VARCHAR(255) NOT NULL,
    -- The actual secret is stored in an external secret manager.
    -- We keep an opaque reference (e.g., secret name or version) for audit.
    secret_ref VARCHAR(255) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_pipeline_unique_repo UNIQUE (provider, repo_owner, repo_name)
);

CREATE INDEX IF NOT EXISTS idx_pipelines_user_id ON pipelines(user_id);
CREATE TRIGGER trg_pipelines_updated
BEFORE UPDATE ON pipelines
FOR EACH ROW EXECUTE FUNCTION set_updated_timestamp();

-- ------------------------------------------------------------------
-- Table: jobs
-- ------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_id UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    external_job_id VARCHAR(255) NOT NULL,
    status job_status_enum NOT NULL,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    duration_seconds NUMERIC(12,3),
    log_excerpt TEXT,               -- truncated log for quick UI view
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_job_external UNIQUE (pipeline_id, external_job_id)
);

CREATE INDEX IF NOT EXISTS idx_jobs_pipeline_id ON jobs(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE TRIGGER trg_jobs_updated
BEFORE UPDATE ON jobs
FOR EACH ROW EXECUTE FUNCTION set_updated_timestamp();

-- ------------------------------------------------------------------
-- Table: ai_suggestions
-- ------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ai_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    suggestion_text TEXT NOT NULL,
    confidence NUMERIC(5,2),        -- 0.00 – 100.00 percentage
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_suggestions_job_id ON ai_suggestions(job_id);

-- ------------------------------------------------------------------
-- End of 001_init.sql
-- ------------------------------------------------------------------