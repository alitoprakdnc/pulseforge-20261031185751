import useSWR, { mutate as globalMutate } from 'swr';
import { useState, useCallback } from 'react';
import api from '../utils/api';

/**
 * Job entity returned by the backend.
 */
export type Job = {
  id: string;
  name?: string;
  /** Pipeline step status */
  status: 'queued' | 'running' | 'success' | 'failed' | 'canceled';
  /** Duration in seconds */
  duration: number;
  /** ISO‑8601 timestamp when the job started */
  started_at: string;
  /** ISO‑8601 timestamp when the job finished (optional) */
  finished_at?: string;
  /** Small excerpt of the raw log (optional) */
  log_excerpt?: string;
  /** Linked AI suggestion identifier (optional) */
  ai_suggestion_id?: string;
};

/**
 * Shape of the paginated response from `/api/pipelines/:id/jobs`.
 */
type JobsResponse = {
  items: Job[];
  total: number;
  page: number;
  pageSize: number;
};

type UseJobsOptions = {
  /** First page to load (default: 1) */
  initialPage?: number;
  /** Number of jobs per page (default: 20) */
  pageSize?: number;
  /** Automatic revalidation interval in milliseconds (optional) */
  refreshInterval?: number;
};

/**
 * Hook to fetch jobs for a given pipeline.
 *
 * Features:
 *  - Pagination (page, pageSize, `loadMore`, `hasMore`)
 *  - Real‑time refresh via SWR `refreshInterval` or manual `refresh`
 *  - Loading / error states
 *  - Type‑safe response handling
 *
 * @param pipelineId The UUID of the pipeline whose jobs we want to list.
 * @param options Hook configuration.
 */
export const useJobs = (
  pipelineId: string,
  options?: UseJobsOptions
) => {
  const {
    initialPage = 1,
    pageSize = 20,
    refreshInterval,
  } = options ?? {};

  const [page, setPage] = useState<number>(initialPage);

  // Only fire the request when a valid pipelineId is supplied.
  const shouldFetch = Boolean(pipelineId);

  const fetcher = async (url: string) => {
    const { data } = await api.get<JobsResponse>(url);
    return data;
  };

  const { data, error, isValidating, mutate } = useSWR<JobsResponse>(
    shouldFetch
      ? `/api/pipelines/${pipelineId}/jobs?page=${page}&pageSize=${pageSize}`
      : null,
    fetcher,
    {
      revalidateOnFocus: true,
      refreshInterval,
      keepPreviousData: true, // keep old page while fetching the next one
    }
  );

  const loading = !data && !error;

  const setPageAndRefresh = useCallback((newPage: number) => {
    setPage(newPage);
  }, []);

  const loadMore = useCallback(() => {
    if (hasMore) {
      setPage((p) => p + 1);
    }
  }, [hasMore]);

  const refresh = useCallback(() => {
    mutate();
  }, [mutate]);

  const hasMore = data ? page * pageSize < data.total : false;

  return {
    /** List of jobs for the current page */
    jobs: data?.items ?? [],
    /** Total number of jobs across all pages */
    total: data?.total ?? 0,
    /** Current page index (1‑based) */
    page,
    /** Page size used for the request */
    pageSize,
    /** Set a specific page (triggers a re‑fetch) */
    setPage: setPageAndRefresh,
    /** Load the next page if available */
    loadMore,
    /** Whether more pages exist */
    hasMore,
    /** Loading flag for the initial request */
    loading,
    /** SWR error object (if any) */
    error,
    /** Indicates if a background revalidation is in progress */
    isValidating,
    /** Manually trigger a refresh */
    refresh,
    /** Direct access to SWR's mutate for advanced use‑cases */
    mutate,
  };
};

export default useJobs;