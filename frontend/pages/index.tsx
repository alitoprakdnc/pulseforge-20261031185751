import { useEffect } from 'react';
import { useRouter } from 'next/router';
import { GetServerSideProps } from 'next';
import { parse } from 'cookie';
import { useAuth } from '../context/AuthContext';

const IndexPage = () => {
  const router = useRouter();
  const { user, loading } = useAuth();

  useEffect(() => {
    if (!loading) {
      if (user) {
        router.replace('/dashboard');
      } else {
        router.replace('/login');
      }
    }
  }, [loading, user, router]);

  return (
    <div className="flex items-center justify-center min-h-screen bg-background">
      {loading && (
        <svg
          className="animate-spin h-8 w-8 text-primary"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          aria-label="Loading"
        >
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          />
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8v8H4z"
          />
        </svg>
      )}
    </div>
  );
};

export const getServerSideProps: GetServerSideProps = async (context) => {
  const cookies = context.req.headers.cookie
    ? parse(context.req.headers.cookie)
    : {};

  // The auth token is stored in an HttpOnly cookie (e.g., "auth_token")
  const token = cookies['auth_token'] || cookies['token'];

  if (token) {
    // Token exists – assume the user is authenticated and redirect to dashboard
    return {
      redirect: {
        destination: '/dashboard',
        permanent: false,
      },
    };
  }

  // No token – redirect to the login page
  return {
    redirect: {
      destination: '/login',
      permanent: false,
    },
  };
};

export default IndexPage;