import React from 'react';
import Link from 'next/link';
import useJobs from '../hooks/useJobs';
import LoadingSkeleton from './LoadingSkeleton';

/**
 * Props for JobTable component.
 * - pipelineId: identifier of the pipeline whose jobs are displayed.
 * - page?: current page (defaults to 1).
 * - pageSize?: number of jobs per page (defaults to 10).
 */
interface JobTableProps {
  pipelineId: string;
  page?: number;
  pageSize?: number;
}

/**
 * Simple status chip that follows the design system colors.
 */
const StatusChip: React.FC<{ status: string }> = ({ status }) => {
  const base = 'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium';
  const variants: Record<string, string> = {
    success: 'bg-success-100 text-success-800',
    failed: 'bg-danger-100 text-danger-800',
    running: 'bg-primary-100 text-primary-800',
    queued: 'bg-secondary-100 text-secondary-800',
    canceled: 'bg-gray-100 text-gray-800',
  };
  const className = `${base} ${variants[status] ?? 'bg-gray-100 text-gray-800'}`;
  const label = status.charAt(0).toUpperCase() + status.slice(1);
  return <span className={className}>{label}</span>;
};

/**
 * Formats a duration in seconds to a human‑readable string.
 * Example: 125 => "2m 5s"
 */
const formatDuration = (seconds: number): string => {
  if (seconds < 60) return `${seconds}s`;
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}m ${secs}s`;
};

/**
 * JobTable component – renders a responsive, accessible table of recent jobs.
 * It uses the `useJobs` hook for data fetching, shows loading skeletons,
 * handles errors, and provides a link to the AI suggestion when available.
 */
const JobTable: React.FC<JobTableProps> = ({
  pipelineId,
  page = 1,
  pageSize = 10,
}) => {
  const { data, error, isLoading, isFetching, refetch } = useJobs(pipelineId, {
    page,
    pageSize,
  });

  // Pagination state – simple client‑side handling (could be expanded)
  const [currentPage, setCurrentPage] = React.useState(page);
  const totalPages = data?.totalPages ?? 1;

  const handlePrev = () => {
    if (currentPage > 1) {
      setCurrentPage(prev => prev - 1);
    }
  };
  const handleNext = () => {
    if (currentPage < totalPages) {
      setCurrentPage(prev => prev + 1);
    }
  };

  // Refetch when page changes
  React.useEffect(() => {
    refetch();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentPage]);

  if (error) {
    return (
      <div className="p-4 text-center text-danger-600" role="alert">
        Failed to load jobs. Please try again.
      </div>
    );
  }

  return (
    <div className="w-full overflow-x-auto">
      <table className="min-w-full divide-y divide-border bg-card" aria-label="Recent jobs">
        <thead className="bg-background">
          <tr>
            <th
              scope="col"
              className="px-4 py-2 text-left text-sm font-medium text-secondary-700"
            >
              Job
            </th>
            <th
              scope="col"
              className="px-4 py-2 text-left text-sm font-medium text-secondary-700"
            >
              Status
            </th>
            <th
              scope="col"
              className="px-4 py-2 text-left text-sm font-medium text-secondary-700"
            >
              Duration
            </th>
            <th
              scope="col"
              className="px-4 py-2 text-left text-sm font-medium text-secondary-700"
            >
              AI Suggestion
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {isLoading || isFetching ? (
            Array.from({ length: pageSize }).map((_, idx) => (
              <tr key={idx}>
                <td className="px-4 py-2">
                  <LoadingSkeleton className="h-4 w-32" />
                </td>
                <td className="px-4 py-2">
                  <LoadingSkeleton className="h-4 w-20" />
                </td>
                <td className="px-4 py-2">
                  <LoadingSkeleton className="h-4 w-12" />
                </td>
                <td className="px-4 py-2">
                  <LoadingSkeleton className="h-4 w-24" />
                </td>
              </tr>
            ))
          ) : (
            data?.jobs?.map(job => (
              <tr key={job.id} className="hover:bg-background transition-colors">
                <td className="px-4 py-2 text-sm text-secondary-900">
                  {job.name ?? `Job #${job.id}`}
                </td>
                <td className="px-4 py-2">
                  <StatusChip status={job.status} />
                </td>
                <td className="px-4 py-2 text-sm text-secondary-900">
                  {formatDuration(job.duration)}
                </td>
                <td className="px-4 py-2">
                  {job.aiSuggestionId ? (
                    <Link
                      href={`/pipelines/${pipelineId}/jobs/${job.id}`}
                      className="text-primary-600 hover:underline focus:outline-none focus:ring-2 focus:ring-primary-500 rounded"
                    >
                      View suggestion
                    </Link>
                  ) : (
                    <span className="text-gray-500 text-sm">—</span>
                  )}
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>

      {/* Pagination controls */}
      <div className="flex items-center justify-between mt-4 px-4">
        <p className="text-sm text-secondary-600">
          Page {currentPage} of {totalPages}
        </p>
        <div className="flex space-x-2">
          <button
            type="button"
            onClick={handlePrev}
            disabled={currentPage === 1}
            className={`px-3 py-1 text-sm rounded border ${
              currentPage === 1
                ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
                : 'bg-white text-primary-600 hover:bg-primary-50 border-primary-300'
            } focus:outline-none focus:ring-2 focus:ring-primary-500`}
          >
            Previous
          </button>
          <button
            type="button"
            onClick={handleNext}
            disabled={currentPage === totalPages}
            className={`px-3 py-1 text-sm rounded border ${
              currentPage === totalPages
                ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
                : 'bg-white text-primary-600 hover:bg-primary-50 border-primary-300'
            } focus:outline-none focus:ring-2 focus:ring-primary-500`}
          >
            Next
          </button>
        </div>
      </div>
    </div>
  );
};

export default JobTable;