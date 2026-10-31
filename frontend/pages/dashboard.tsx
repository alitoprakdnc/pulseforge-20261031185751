import React, { useEffect, useContext } from 'react';
import type { NextPage } from 'next';
import Head from 'next/head';
import { useRouter } from 'next/router';
import Layout from '@/components/Layout';
import Dashboard from '@/components/Dashboard';
import LoadingSkeleton from '@/components/LoadingSkeleton';
import AlertBanner from '@/components/AlertBanner';
import { AuthContext } from '@/context/AuthContext';
import { usePipelines } from '@/hooks/usePipelines';
import type { Pipeline } from '@/types';

const DashboardPage: NextPage = () => {
  const router = useRouter();
  const { user, loading: authLoading } = useContext(AuthContext);
  const { pipelines, error, isLoading } = usePipelines();

  // Redirect unauthenticated users to the login page
  useEffect(() => {
    if (!authLoading && !user) {
      router.replace('/login');
    }
  }, [authLoading, user, router]);

  // Render loading state while auth or data is being fetched
  const showLoading = authLoading || isLoading;

  // Empty state when there are no pipelines
  const renderEmptyState = () => (
    <div className="flex flex-col items-center justify-center py-12 text-center">
      <svg
        className="w-16 h-16 text-gray-400 mb-4"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
        aria-hidden="true"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M9 17v-6a2 2 0 012-2h2a2 2 0 012 2v6m-6 0h6"
        />
      </svg>
      <h2 className="text-lg font-medium text-gray-900 mb-2">
        No pipelines connected
      </h2>
      <p className="text-sm text-gray-600 max-w-md">
        Connect a repository from GitHub, GitLab or CircleCI to start
        monitoring your CI/CD pipelines. Visit the Settings page to add a new
        integration.
      </p>
    </div>
  );

  return (
    <>
      <Head>
        <title>PulseForge • Dashboard</title>
        <meta name="description" content="CI/CD observability dashboard" />
      </Head>
      <Layout>
        {showLoading && (
          <div className="space-y-4">
            <LoadingSkeleton count={3} height={200} className="rounded-lg" />
          </div>
        )}

        {!showLoading && error && (
          <AlertBanner
            type="error"
            message={error?.message ?? 'Failed to load pipelines.'}
          />
        )}

        {!showLoading && !error && pipelines?.length === 0 && renderEmptyState()}

        {!showLoading && !error && pipelines && pipelines.length > 0 && (
          <Dashboard pipelines={pipelines as Pipeline[]} />
        )}
      </Layout>
    </>
  );
};

export default DashboardPage;