import type { AppProps, NextWebVitalsMetric } from 'next/app';
import Head from 'next/head';
import { AuthProvider } from '../context/AuthContext';
import Layout from '../components/Layout';
import '../styles/globals.css';

type NextPageWithLayout = {
  getLayout?: (page: React.ReactNode) => React.ReactNode;
} & typeof Component;

function MyApp({ Component, pageProps }: AppProps) {
  const getLayout =
    (Component as NextPageWithLayout).getLayout ??
    ((page: React.ReactNode) => <Layout>{page}</Layout>);

  return (
    <>
      <Head>
        <title>PulseForge</title>
        <meta
          name="description"
          content="AI‑powered CI/CD observability and optimization platform."
        />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
      </Head>
      <AuthProvider>{getLayout(<Component {...pageProps} />)}</AuthProvider>
    </>
  );
}

// Optional: report web vitals to an analytics endpoint
export function reportWebVitals(metric: NextWebVitalsMetric) {
  if (process.env.NODE_ENV === 'production') {
    // Example: send metrics to a backend endpoint
    fetch('/api/analytics', {
      method: 'POST',
      body: JSON.stringify(metric),
      keepalive: true,
    }).catch(() => {
      // Silently ignore analytics failures in production
    });
  }
}

export default MyApp;