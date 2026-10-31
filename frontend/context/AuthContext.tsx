import React, {
  createContext,
  useContext,
  useEffect,
  useState,
  ReactNode,
  useCallback,
} from "react";
import api from "../utils/api";

type Role = "admin" | "manager" | "engineer" | "viewer";

export interface User {
  id: string;
  email: string;
  name: string;
  role: Role;
}

interface AuthContextValue {
  /** Currently authenticated user (null when not logged in) */
  user: User | null;
  /** True while an auth‑related request is in flight */
  loading: boolean;
  /** Last error that occurred during login / logout / refresh */
  error: string | null;
  /** Perform login with email & password */
  login: (email: string, password: string) => Promise<void>;
  /** Perform logout */
  logout: () => Promise<void>;
  /** Force a refresh of the current session (e.g. after token rotation) */
  refresh: () => Promise<void>;
}

/** Default (unauthenticated) context value – will be overridden by the provider */
const AuthContext = createContext<AuthContextValue>({
  user: null,
  loading: false,
  error: null,
  login: async () => {},
  logout: async () => {},
  refresh: async () => {},
});

interface AuthProviderProps {
  children: ReactNode;
}

/**
 * AuthProvider loads the current session on mount, exposes login / logout helpers,
 * and keeps the user object in sync with the backend.
 *
 * The backend stores the JWT in an HttpOnly cookie, therefore we never expose the
 * raw token to the client. All requests are automatically authenticated via the
 * cookie; the provider only needs to know *who* is logged in.
 */
export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  /**
   * Fetch the current authenticated user.
   * The endpoint returns 200 with user data when the session cookie is valid,
   * otherwise 401.
   */
  const fetchCurrentUser = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await api.get<User>("/api/auth/me");
      setUser(response.data);
    } catch (err: any) {
      // 401 means no valid session – treat as logged out.
      if (err.response?.status === 401) {
        setUser(null);
      } else {
        setError(
          err.response?.data?.message ||
            err.message ||
            "Failed to fetch authentication status"
        );
      }
    } finally {
      setLoading(false);
    }
  }, []);

  // Load user on component mount
  useEffect(() => {
    fetchCurrentUser();
  }, [fetchCurrentUser]);

  /**
   * Login using email & password.
   * The backend sets an HttpOnly cookie on success.
   */
  const login = useCallback(
    async (email: string, password: string) => {
      setLoading(true);
      setError(null);
      try {
        await api.post(
          "/api/auth/login",
          { email, password },
          {
            // Ensure the cookie is stored; withCredentials tells the browser to
            // include and accept cookies for cross‑origin requests.
            withCredentials: true,
          }
        );
        // After a successful login the server has set the session cookie.
        // Refresh the user state to reflect the new session.
        await fetchCurrentUser();
      } catch (err: any) {
        setError(
          err.response?.data?.message ||
            err.message ||
            "Invalid email or password"
        );
        setUser(null);
        throw err; // re‑throw for callers that need to react (e.g. UI)
      } finally {
        setLoading(false);
      }
    },
    [fetchCurrentUser]
  );

  /**
   * Logout clears the session cookie on the server.
   */
  const logout = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await api.post(
        "/api/auth/logout",
        {},
        {
          withCredentials: true,
        }
      );
      setUser(null);
    } catch (err: any) {
      setError(
        err.response?.data?.message ||
          err.message ||
          "Failed to log out"
      );
      // Even if the request fails we still clear local state to avoid a stale UI.
      setUser(null);
    } finally {
      setLoading(false);
    }
  }, []);

  /**
   * Refresh forces the backend to rotate the JWT (if it implements a refresh
   * endpoint) and updates the local user state.
   */
  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await api.post(
        "/api/auth/refresh",
        {},
        {
          withCredentials: true,
        }
      );
      await fetchCurrentUser();
    } catch (err: any) {
      setError(
        err.response?.data?.message ||
          err.message ||
          "Failed to refresh session"
      );
      setUser(null);
    } finally {
      setLoading(false);
    }
  }, [fetchCurrentUser]);

  const value: AuthContextValue = {
    user,
    loading,
    error,
    login,
    logout,
    refresh,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

/**
 * Hook to access authentication state and actions.
 *
 * Example:
 *   const { user, login, logout, loading } = useAuth();
 */
export const useAuth = (): AuthContextValue => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
};