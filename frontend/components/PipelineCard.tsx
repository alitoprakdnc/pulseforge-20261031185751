import React, { useCallback } from "react";
import Link from "next/link";
import { useRouter } from "next/router";
import { HiOutlineTrash, HiOutlinePencilAlt, HiOutlineExternalLink } from "react-icons/hi";
import { api } from "../utils/api";
import { useToast } from "../hooks/useToast";

type Provider = "github" | "gitlab" | "circleci";

interface Pipeline {
  id: string;
  name: string;
  provider: Provider;
  healthStatus: "healthy" | "degraded" | "critical";
  /** ISO timestamp of the most recent run */
  lastRunAt?: string;
}

/**
 * Props for the PipelineCard component.
 */
interface PipelineCardProps {
  /** Pipeline data to render */
  pipeline: Pipeline;
}

/**
 * Mapping of provider to display name and Tailwind colors.
 */
const providerMeta: Record<Provider, { label: string; bg: string; icon: JSX.Element }> = {
  github: {
    label: "GitHub Actions",
    bg: "bg-[#0F62FE]/10 text-[#0F62FE]",
    icon: (
      <svg
        className="w-4 h-4"
        viewBox="0 0 24 24"
        fill="currentColor"
        aria-hidden="true"
      >
        <path d="M12 .5C5.37.5 0 5.87 0 12.5c0 5.3 3.44 9.8 8.21 11.39.6.11.82-.26.82-.58 0-.29-.01-1.05-.02-2.06-3.34.73-4.04-1.61-4.04-1.61-.55-1.4-1.34-1.77-1.34-1.77-1.09-.75.08-.74.08-.74 1.21.09 1.85 1.24 1.85 1.24 1.07 1.84 2.81 1.31 3.5 1 .11-.78.42-1.31.76-1.61-2.66-.3-5.46-1.33-5.46-5.93 0-1.31.47-2.38 1.24-3.22-.13-.3-.54-1.52.12-3.17 0 0 1.01-.32 3.3 1.23a11.48 11.48 0 013-.4c1.02.01 2.05.14 3 .4 2.28-1.55 3.29-1.23 3.29-1.23.66 1.65.25 2.87.12 3.17.77.84 1.23 1.91 1.23 3.22 0 4.61-2.81 5.62-5.48 5.92.43.37.81 1.1.81 2.22 0 1.6-.01 2.89-.01 3.28 0 .32.22.7.83.58A12.01 12.01 0 0024 12.5c0-6.63-5.37-12-12-12z" />
      </svg>
    ),
  },
  gitlab: {
    label: "GitLab CI",
    bg: "bg-[#FC6D26]/10 text-[#FC6D26]",
    icon: (
      <svg
        className="w-4 h-4"
        viewBox="0 0 24 24"
        fill="currentColor"
        aria-hidden="true"
      >
        <path d="M22.545 12.09l-1.73-5.33c-.13-.41-.5-.71-.93-.71h-3.84l1.44-4.44c.13-.41-.02-.86-.38-1.09-.36-.23-.83-.18-1.13.13L9.5 12.09c-.31.31-.31.81 0 1.12l5.02 5.02c.31.31.81.31 1.12 0l6.9-6.9c.31-.31.31-.81 0-1.12z" />
      </svg>
    ),
  },
  circleci: {
    label: "CircleCI",
    bg: "bg-[#343434]/10 text-[#343434]",
    icon: (
      <svg
        className="w-4 h-4"
        viewBox="0 0 24 24"
        fill="currentColor"
        aria-hidden="true"
      >
        <path d="M12 0C5.373 0 0 5.373 0 12c0 5.302 3.438 9.8 8.205 11.385.6.111.82-.261.82-.58 0-.286-.01-1.04-.015-2.04-3.338.726-4.042-1.61-4.042-1.61-.546-1.386-1.334-1.756-1.334-1.756-1.09-.745.083-.73.083-.73 1.206.086 1.84 1.24 1.84 1.24 1.07 1.834 2.81 1.304 3.5.997.108-.775.418-1.304.76-1.603-2.665-.304-5.466-1.332-5.466-5.93 0-1.31.466-2.38 1.236-3.22-.124-.304-.535-1.527.117-3.183 0 0 1.008-.322 3.306 1.23a11.48 11.48 0 013-.4c1.02.01 2.05.14 3 .4 2.298-1.552 3.306-1.23 3.306-1.23.652 1.656.242 2.879.118 3.183.77.84 1.236 1.91 1.236 3.22 0 4.61-2.803 5.624-5.475 5.922.43.37.81 1.1.81 2.22 0 1.603-.01 2.894-.01 3.285 0 .32.22.69.82.58A12.015 12.015 0 0024 12c0-6.627-5.373-12-12-12z" />
      </svg>
    ),
  },
};

/**
 * Mapping of health status to Tailwind colors.
 */
const healthMeta: Record<Pipeline["healthStatus"], { label: string; bg: string }> = {
  healthy: { label: "Healthy", bg: "bg-success-100 text-success-800" },
  degraded: { label: "Degraded", bg: "bg-amber-100 text-amber-800" },
  critical: { label: "Critical", bg: "bg-danger-100 text-danger-800" },
};

export const PipelineCard: React.FC<PipelineCardProps> = ({ pipeline }) => {
  const router = useRouter();
  const toast = useToast();

  const handleDelete = useCallback(async () => {
    const confirmed = window.confirm(
      `Are you sure you want to delete pipeline "${pipeline.name}"? This action cannot be undone.`
    );
    if (!confirmed) return;

    try {
      await api.delete(`/pipelines/${pipeline.id}`);
      toast.success(`Pipeline "${pipeline.name}" deleted.`);
      // Refresh the page or inform parent via an event (implementation‑specific)
      router.replace(router.asPath);
    } catch (err) {
      console.error(err);
      toast.error("Failed to delete pipeline. Please try again.");
    }
  }, [pipeline.id, pipeline.name, router, toast]);

  const providerInfo = providerMeta[pipeline.provider];
  const healthInfo = healthMeta[pipeline.healthStatus];

  return (
    <div
      className="bg-card rounded-lg shadow-sm border border-border hover:shadow-md transition-shadow flex flex-col"
      role="region"
      aria-labelledby={`pipeline-${pipeline.id}-title`}
    >
      <div className="p-4 flex flex-col gap-3">
        {/* Header */}
        <div className="flex items-center justify-between">
          <h3
            id={`pipeline-${pipeline.id}-title`}
            className="text-lg font-medium text-gray-900 truncate"
          >
            {pipeline.name}
          </h3>
          <Link href={`/pipelines/${pipeline.id}`} legacyBehavior>
            <a
              className="text-primary-600 hover:text-primary-800 focus:outline-none focus:ring-2 focus:ring-primary-500 rounded"
              aria-label={`View details for ${pipeline.name}`}
            >
              <HiOutlineExternalLink className="w-5 h-5" />
            </a>
          </Link>
        </div>

        {/* Badges */}
        <div className="flex items-center gap-2">
          {/* Provider badge */}
          <span
            className={`inline-flex items-center gap-1 px-2 py-0.5 rounded ${providerInfo.bg} text-xs font-medium`}
          >
            {providerInfo.icon}
            {providerInfo.label}
          </span>

          {/* Health status badge */}
          <span
            className={`inline-flex items-center px-2 py-0.5 rounded ${healthInfo.bg} text-xs font-medium`}
          >
            {healthInfo.label}
          </span>
        </div>

        {/* Last run timestamp */}
        {pipeline.lastRunAt && (
          <p className="text-sm text-gray-600">
            Last run:{" "}
            <time dateTime={pipeline.lastRunAt}>
              {new Date(pipeline.lastRunAt).toLocaleString(undefined, {
                dateStyle: "short",
                timeStyle: "short",
              })}
            </time>
          </p>
        )}

        {/* Action buttons */}
        <div className="flex items-center gap-2 mt-2">
          <Link href={`/pipelines/${pipeline.id}/edit`} legacyBehavior>
            <a
              className="flex items-center gap-1 px-3 py-1.5 bg-gray-100 text-gray-800 rounded hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-primary-500"
              aria-label={`Edit ${pipeline.name}`}
            >
              <HiOutlinePencilAlt className="w-4 h-4" />
              <span className="text-sm">Edit</span>
            </a>
          </Link>

          <button
            type="button"
            onClick={handleDelete}
            className="flex items-center gap-1 px-3 py-1.5 bg-danger-100 text-danger-800 rounded hover:bg-danger-200 focus:outline-none focus:ring-2 focus:ring-danger-500"
            aria-label={`Delete ${pipeline.name}`}
          >
            <HiOutlineTrash className="w-4 h-4" />
            <span className="text-sm">Delete</span>
          </button>
        </div>
      </div>
    </div>
  );
};

export default PipelineCard;