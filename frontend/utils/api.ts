import axios, { AxiosInstance, AxiosError, AxiosResponse } from 'axios';
import Router from 'next/router';

/**
 * Shape of an error that will be thrown from the API client.
 * All UI layers should rely on this structure for consistent handling.
 */
export interface APIError {
  /** HTTP status code (0 when the request never reached the server). */
  status: number;
  /** Human‑readable message for the UI. */
  message: string;
  /** Optional payload returned by the backend (validation errors, stack traces, …). */
  details?: unknown;
}

/**
 * Retrieve the JWT used for authorisation.
 *
 * The token is stored in `localStorage` after a successful login.
 * The backend also sets an HttpOnly cookie, which is automatically sent
 * with `withCredentials: true`. Keeping the token in localStorage allows us
 * to add the `Authorization: Bearer …` header for APIs that expect it.
 */
function getAccessToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('accessToken');
}

/**
 * Remove any persisted auth artefacts – used when we receive a 401.
 */
function clearAuth(): void {
  if (typeof window !== 'undefined') {
    localStorage.removeItem('accessToken');
  }
}

/**
 * Create a pre‑configured Axios instance.
 *
 * - `baseURL` is taken from `NEXT_PUBLIC_API_URL` (defaults to `/api` for dev).
 * - `withCredentials` ensures HttpOnly cookies are sent on every request.
 * - A request interceptor injects the JWT when present.
 * - A response interceptor normalises errors into `APIError` and redirects on 401.
 */
const api: AxiosInstance = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL ?? '/api',
  timeout: 15000,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
    Accept: 'application/json',
  },
});

/* -------------------------------------------------------------------------- */
/* Request interceptor – attach JWT if we have one                           */
/* -------------------------------------------------------------------------- */
api.interceptors.request.use(
  (config) => {
    const token = getAccessToken();
    if (token) {
      // Preserve any existing headers while adding Authorization
      config.headers = {
        ...config.headers,
        Authorization: `Bearer ${token}`,
      };
    }
    return config;
  },
  (error) => Promise.reject(error)
);

/* -------------------------------------------------------------------------- */
/* Response interceptor – normalise errors & handle auth expiry               */
/* -------------------------------------------------------------------------- */
api.interceptors.response.use(
  (response: AxiosResponse) => response,
  (error: AxiosError) => {
    const transformed: APIError = {
      status: error.response?.status ?? 0,
      message: '',
    };

    if (error.response?.data) {
      const payload = error.response.data as Record<string, unknown>;
      transformed.message =
        (payload.message as string) ||
        (payload.error as string) ||
        'An unexpected error occurred';
      transformed.details = payload;
    } else if (error.request) {
      transformed.message = 'No response received from the server';
    } else {
      transformed.message = error.message;
    }

    // If the token is no longer valid, clear local state and force a login.
    if (transformed.status === 401) {
      clearAuth();
      Router.replace('/login');
    }

    return Promise.reject(transformed);
  }
);

export default api;