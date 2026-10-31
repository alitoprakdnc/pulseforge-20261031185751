import React, { useMemo, useState } from "react";
import { useRouter } from "next/router";
import useSWR from "swr";
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from "recharts";
import Layout from "@/components/Layout";
import JobTable from "@/components/JobTable";
import AIChat from "@/components/AIChat";
import LoadingSkeleton from "@/components/LoadingSkeleton";
import AlertBanner from "@/components/AlertBanner";
import { api } from "@/utils/api";
import { ClipboardIcon, CheckIcon } from "@heroicons/react/24/outline";

type Pipeline = {
  id: string;
  name: string;
  provider: "github" | "gitlab" | "circleci";
  repo: string;
  health_endpoint: string;
  alerts: string[];
};

type Job = {
  id: string;
  name: string;
  status: "success" | "failed" | "running" | "canceled";
  duration_seconds: number;
  started_at: string; // ISO
};

type AISuggestion = {
  id: string;
  job_id: string;
  suggestion: string;
  confidence: number; // 0‑1
  created_at: string;
};

type HealthInfo = {
  endpoint: string;
  sample_response: Record<string, unknown>;
};

const fetcher = (url: string) => api.get(url).then((res) => res.data);

export default function PipelineDetailPage() {
  const router = useRouter();
  const { id } = router.query as { id: string };

  const {
    data: pipeline,
    error: pipelineError,
    isValidating: pipelineLoading,
  } = useSWR<Pipeline>(id ? `/api/pipelines/${id}` : null, fetcher);

  const {
    data: jobs,
    error: jobsError,
    isValidating: jobsLoading,
  } = useSWR<Job[]>(id ? `/api/pipelines/${id}/jobs?limit=50` : null, fetcher);

  const {
    data: suggestions,
    error: suggestionsError,
    isValidating: suggestionsLoading,
  } = useSWR<AISuggestion[]>(id ? `/api/pipelines/${id}/ai-suggestions` : null, fetcher);

  const {
    data: healthInfo,
    error: healthError,
    isValidating: healthLoading,
  } = useSWR<HealthInfo>(id ? `/api/pipelines/${id}/health` : null, fetcher);

  const [copiedId, setCopiedId] = useState<string | null>(null);

  const chartData = useMemo(() => {
    if (!jobs) return [];
    // Sort by start time ascending
    const sorted = [...jobs].sort(
      (a, b) => new Date(a.started_at).getTime() - new Date(b.started_at).getTime()
    );
    return sorted.map((job) => ({
      name: job.name,
      duration: job.duration_seconds,
      start: new Date(job.started_at).toLocaleString(),
      status: job.status,
    }));
  }, [jobs]);

  const handleCopy = (text: string, suggestionId: string) => {
    navigator.clipboard.writeText(text).then(() => {
      setCopiedId(suggestionId);
      setTimeout(() => setCopiedId(null), 2000);
    });
  };

  if (pipelineLoading || jobsLoading || suggestionsLoading || healthLoading) {
    return (
      <Layout>
        <div className="p-6 space-y-4">
          <LoadingSkeleton className="h-8 w-48" />
          <LoadingSkeleton className="h-64 w-full" />
          <LoadingSkeleton className="h-96 w-full" />
        </div>
      </Layout>
    );
  }

  if (pipelineError) {
    return (
      <Layout>
        <AlertBanner type="danger" message="Failed to load pipeline data." />
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="p-6 space-y-8">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <h1 className="text-2xl font-semibold text-gray-900">{pipeline?.name}</h1>
          <span className="inline-flex items-center rounded-md bg-gray-100 px-3 py-1 text-sm font-medium text-gray-800">
            {pipeline?.provider?.toUpperCase()}
          </span>
        </div>

        {/* Alerts */}
        {pipeline?.alerts?.length ? (
          <div className="space-y-2">
            {pipeline.alerts.map((msg, idx) => (
              <AlertBanner key={idx} type="warning" message={msg} />
            ))}
          </div>
        ) : null}

        {/* Job Timeline Chart */}
        <section aria-labelledby="timeline-heading">
          <h2 id="timeline-heading" className="text-lg font-medium text-gray-900 mb-4">
            Job Duration Timeline
          </h2>
          {chartData.length ? (
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={chartData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="start" tick={{ fontSize: 12 }} />
                <YAxis
                  label={{ value: "seconds", angle: -90, position: "insideLeft", fontSize: 12 }}
                />
                <Tooltip />
                <Line
                  type="monotone"
                  dataKey="duration"
                  stroke="#0F62FE"
                  strokeWidth={2}
                  dot={{ r: 3 }}
                />
              </LineChart>
            </ResponsiveContainer>
          ) : (
            <p className="text-sm text-gray-500">No job data available for chart.</p>
          )}
        </section>

        {/* Recent Jobs Table */}
        <section aria-labelledby="jobs-heading">
          <h2 id="jobs-heading" className="text-lg font-medium text-gray-900 mb-4">
            Recent Jobs
          </h2>
          {jobsError ? (
            <AlertBanner type="danger" message="Failed to load jobs." />
          ) : (
            <JobTable jobs={jobs ?? []} />
          )}
        </section>

        {/* AI Suggestions */}
        <section aria-labelledby="ai-heading">
          <h2 id="ai-heading" className="text-lg font-medium text-gray-900 mb-4">
            AI‑Generated Suggestions
          </h2>
          {suggestionsError ? (
            <AlertBanner type="danger" message="Failed to load AI suggestions." />
          ) : suggestions && suggestions.length ? (
            <div className="grid gap-4 md:grid-cols-2">
              {suggestions.map((s) => (
                <div
                  key={s.id}
                  className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm"
                >
                  <div className="flex items-start justify-between">
                    <p className="text-sm text-gray-800 whitespace-pre-wrap">{s.suggestion}</p>
                    <button
                      type="button"
                      onClick={() => handleCopy(s.suggestion, s.id)}
                      className="ml-2 inline-flex items-center rounded-md p-1 text-gray-500 hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-primary-500"
                      aria-label="Copy suggestion"
                    >
                      {copiedId === s.id ? (
                        <CheckIcon className="h-5 w-5 text-green-600" />
                      ) : (
                        <ClipboardIcon className="h-5 w-5" />
                      )}
                    </button>
                  </div>
                  <div className="mt-2 flex items-center text-xs text-gray-600">
                    <span className="mr-2">Confidence:</span>
                    <span className="font-medium">
                      {(s.confidence * 100).toFixed(0)}%
                    </span>
                    <span className="ml-4 text-gray-400">Generated {new Date(s.created_at).toLocaleString()}</span>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-gray-500">No AI suggestions available.</p>
          )}
        </section>

        {/* Health Endpoint Preview */}
        <section aria-labelledby="health-heading">
          <h2 id="health-heading" className="text-lg font-medium text-gray-900 mb-4">
            Health‑Check Endpoint
          </h2>
          {healthError ? (
            <AlertBanner type="danger" message="Failed to load health‑check info." />
          ) : healthInfo ? (
            <div className="rounded-lg border border-gray-200 bg-gray-50 p-4 font-mono text-sm text-gray-800 overflow-x-auto">
              <div className="mb-2">
                <span className="font-medium text-gray-900">GET </span>
                <span className="text-primary-600">{healthInfo.endpoint}</span>
              </div>
              <pre className="whitespace-pre-wrap">
                {JSON.stringify(healthInfo.sample_response, null, 2)}
              </pre>
            </div>
          ) : (
            <p className="text-sm text-gray-500">Health‑check information not available.</p>
          )}
        </section>

        {/* AI Chat */}
        <section aria-labelledby="chat-heading">
          <h2 id="chat-heading" className="text-lg font-medium text-gray-900 mb-4">
            Ask the AI
          </h2>
          <AIChat pipelineId={pipeline?.id ?? ""} />
        </section>
      </div>
    </Layout>
  );
}