-- 002_add_indexes.sql
-- Adds performance‑critical indexes for the CI/CD observability data model.
-- Indexes are created IF NOT EXISTS to make the migration idempotent.
-- All statements are wrapped in a transaction to ensure atomicity.

BEGIN;

-- Indexes for the jobs table
CREATE INDEX IF NOT EXISTS idx_jobs_pipeline_id
    ON jobs (pipeline_id);

CREATE INDEX IF NOT EXISTS idx_jobs_status
    ON jobs (status);

-- Most queries filter recent jobs, so store created_at in descending order
CREATE INDEX IF NOT EXISTS idx_jobs_created_at_desc
    ON jobs (created_at DESC);

-- Composite index to efficiently fetch the latest jobs for a given pipeline
CREATE INDEX IF NOT EXISTS idx_jobs_pipeline_created_at
    ON jobs (pipeline_id, created_at DESC);

COMMIT;