import React, { useMemo } from "react";
import useSWR from "swr";
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from "recharts";

import PipelineCard from "./PipelineCard";
import LoadingSkeleton from "./LoadingSkeleton";
import AlertBanner from "./AlertBanner";

import { usePipelines } from "../hooks/usePipelines";
import { api } from "../utils/api";

type QueueHeatmapPoint = {
  timestamp: string; // ISO string
  queueLength: number;
};

type AISuggestion = {
  id: string;
  pipelineId: string;
  text: string;
  confidence: number; // 0‑1
  createdAt: string;
};

const fetcher = (url: string) => api.get(url).then((res) => res.data);

/**
 * Dashboard component
 *
 * - Shows a responsive grid of PipelineCard components.
 * - Renders a queue‑length heat‑map using Recharts.
 * - Displays an AI‑suggestion carousel aggregated across pipelines.
 *
 * All data fetching is done with SWR for caching, revalidation and
 * graceful error handling. Loading states use the shared LoadingSkeleton
 * component and errors are surfaced via the AlertBanner component.
 */
const Dashboard: React.FC = () => {
  // Pipelines list (hook already uses SWR internally)
  const { pipelines, isLoading: pipelinesLoading, error: pipelinesError } = usePipelines();

  // Queue heat‑map data
  const {
    data: queueData,
    error: queueError,
    isLoading: queueLoading,
  } = useSWR<QueueHeatmapPoint[]>("/api/queues/heatmap", fetcher, {
    refreshInterval: 30_000, // refresh every 30s
    dedupingInterval: 10_000,
  });

  // AI suggestions (latest 10)
  const {
    data: suggestions,
    error: suggestionsError,
    isLoading: suggestionsLoading,
  } = useSWR<AISuggestion[]>("/api/ai/suggestions?limit=10", fetcher, {
    refreshInterval: 60_000,
    dedupingInterval: 20_000,
  });

  // Memoised chart data to avoid re‑render churn
  const chartData = useMemo(() => {
    if (!queueData) return [];
    return queueData.map((pt) => ({
      time: new Date(pt.timestamp).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
      queue: pt.queueLength,
    }));
  }, [queueData]);

  // Render helpers
  const renderPipelineGrid = () => {
    if (pipelinesLoading) {
      return (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <LoadingSkeleton key={i} className="h-48 w-full rounded-lg" />
          ))}
        </div>
      );
    }

    if (pipelinesError) {
      return <AlertBanner type="error" message="Failed to load pipelines. Please try again later." />;
    }

    if (!pipelines?.length) {
      return (
        <div className="text-center text-gray-600 py-12">
          <p className="text-lg">No pipelines have been connected yet.</p>
          <a
            href="/settings"
            className="mt-4 inline-block bg-primary-600 text-white px-4 py-2 rounded hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500"
          >
            Add a Pipeline
          </a>
        </div>
      );
    }

    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {pipelines.map((pipeline) => (
          <PipelineCard key={pipeline.id} pipeline={pipeline} />
        ))}
      </div>
    );
  };

  const renderQueueHeatmap = () => {
    if (queueLoading) {
      return <LoadingSkeleton className="h-64 w-full rounded-lg" />;
    }

    if (queueError) {
      return <AlertBanner type="error" message="Unable to load queue metrics." />;
    }

    if (!chartData.length) {
      return (
        <p className="text-gray-500 text-center py-8">No queue data available.</p>
      );
    }

    return (
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={chartData} margin={{ top: 20, right: 30, left: 0, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#E5E7EB" />
          <XAxis dataKey="time" stroke="#6B7280" />
          <YAxis stroke="#6B7280" />
          <Tooltip
            contentStyle={{ backgroundColor: "#FFFFFF", border: "1px solid #E5E7EB" }}
            labelStyle={{ color: "#111827" }}
          />
          <Line type="monotone" dataKey="queue" stroke="#0F62FE" strokeWidth={2} dot={false} />
        </LineChart>
      </ResponsiveContainer>
    );
  };

  const renderSuggestionCarousel = () => {
    if (suggestionsLoading) {
      return (
        <div className="flex space-x-4 overflow-x-auto py-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <LoadingSkeleton key={i} className="w-80 h-48 rounded-lg flex-shrink-0" />
          ))}
        </div>
      );
    }

    if (suggestionsError) {
      return <AlertBanner type="error" message="Failed to load AI suggestions." />;
    }

    if (!suggestions?.length) {
      return (
        <p className="text-gray-500 text-center py-8">
          No AI suggestions yet. They will appear here once analysis runs.
        </p>
      );
    }

    return (
      <div
        className="flex space-x-4 overflow-x-auto py-2"
        role="region"
        aria-label="AI suggestions carousel"
      >
        {suggestions.map((s) => (
          <div
            key={s.id}
            className="w-80 flex-shrink-0 bg-white rounded-lg shadow-sm border border-gray-200 p-4 hover:shadow-md focus-within:ring-2 focus-within:ring-primary-500"
            tabIndex={0}
          >
            <p className="text-sm text-gray-800 line-clamp-4">{s.text}</p>
            <div className="mt-2 flex items-center justify-between">
              <span className="text-xs text-gray-500">
                {new Date(s.createdAt).toLocaleDateString()}
              </span>
              <span
                className={`text-xs font-medium px-2 py-0.5 rounded ${
                  s.confidence >= 0.8
                    ? "bg-success-100 text-success-800"
                    : s.confidence >= 0.5
                    ? "bg-primary-100 text-primary-800"
                    : "bg-danger-100 text-danger-800"
                }`}
              >
                Confidence: {(s.confidence * 100).toFixed(0)}%
              </span>
            </div>
          </div>
        ))}
      </div>
    );
  };

  return (
    <main className="p-6 bg-background-100 min-h-screen">
      {/* Page Header */}
      <header className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Dashboard</h1>
        <p className="mt-1 text-sm text-gray-600">
          Overview of your CI/CD pipelines, queue health, and AI‑driven recommendations.
        </p>
      </header>

      {/* Pipelines Section */}
      <section className="mb-10" aria-labelledby="pipelines-heading">
        <h2 id="pipelines-heading" className="text-xl font-medium text-gray-800 mb-4">
          Your Pipelines
        </h2>
        {renderPipelineGrid()}
      </section>

      {/* Queue Heat‑Map Section */}
      <section className="mb-10" aria-labelledby="queue-heading">
        <h2 id="queue-heading" className="text-xl font-medium text-gray-800 mb-4">
          Queue Length Heat‑Map
        </h2>
        <div className="bg-card-100 p-4 rounded-lg shadow-sm border border-gray-200">{renderQueueHeatmap()}</div>
      </section>

      {/* AI Suggestions Section */}
      <section aria-labelledby="suggestions-heading">
        <h2 id="suggestions-heading" className="text-xl font-medium text-gray-800 mb-4">
          AI‑Generated Recommendations
        </h2>
        <div className="bg-card-100 p-4 rounded-lg shadow-sm border border-gray-200">{renderSuggestionCarousel()}</div>
      </section>
    </main>
  );
};

export default Dashboard;