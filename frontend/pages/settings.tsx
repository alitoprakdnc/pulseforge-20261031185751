import { useState, useEffect, FormEvent } from "react";
import Layout from "@/components/Layout";
import AlertBanner from "@/components/AlertBanner";
import LoadingSkeleton from "@/components/LoadingSkeleton";
import { useAuth } from "@/context/AuthContext";
import api from "@/utils/api";
import useSWR, { mutate } from "swr";
import { useToast } from "@/components/Toast";

type ApiToken = {
  id: string;
  name: string;
  token: string;
  createdAt: string;
};

type SecretManagerConfig = {
  provider: "gcp" | "vault" | "";
  config: Record<string, string>;
};

type NotificationPrefs = {
  emailEnabled: boolean;
  email: string;
  slackWebhookUrl: string;
};

type Settings = {
  apiTokens: ApiToken[];
  secretManager: SecretManagerConfig;
  notifications: NotificationPrefs;
};

const fetcher = (url: string) => api.get(url).then((res) => res.data);

export default function SettingsPage() {
  const { user } = useAuth();
  const { data, error, isLoading } = useSWR<Settings>("/api/settings", fetcher);
  const toast = useToast();

  // Local form state
  const [newTokenName, setNewTokenName] = useState("");
  const [smProvider, setSmProvider] = useState<"gcp" | "vault" | "">("");
  const [smConfig, setSmConfig] = useState<Record<string, string>>({});
  const [notifEmailEnabled, setNotifEmailEnabled] = useState(false);
  const [notifEmail, setNotifEmail] = useState("");
  const [notifSlackUrl, setNotifSlackUrl] = useState("");

  // Populate form state when data loads
  useEffect(() => {
    if (data) {
      setSmProvider(data.secretManager.provider);
      setSmConfig(data.secretManager.config);
      setNotifEmailEnabled(data.notifications.emailEnabled);
      setNotifEmail(data.notifications.email);
      setNotifSlackUrl(data.notifications.slackWebhookUrl);
    }
  }, [data]);

  // ---------- API Token Handlers ----------
  const handleAddToken = async (e: FormEvent) => {
    e.preventDefault();
    if (!newTokenName.trim()) return;
    try {
      await api.post("/api/settings/api-tokens", { name: newTokenName.trim() });
      toast.success("API token created successfully");
      setNewTokenName("");
      mutate("/api/settings");
    } catch (err: any) {
      toast.error(err?.response?.data?.message || "Failed to create token");
    }
  };

  const handleDeleteToken = async (id: string) => {
    if (!confirm("Are you sure you want to revoke this token?")) return;
    try {
      await api.delete(`/api/settings/api-tokens/${id}`);
      toast.success("Token revoked");
      mutate("/api/settings");
    } catch (err: any) {
      toast.error(err?.response?.data?.message || "Failed to revoke token");
    }
  };

  // ---------- Secret Manager Handlers ----------
  const handleSecretManagerSubmit = async (e: FormEvent) => {
    e.preventDefault();
    try {
      await api.put("/api/settings/secret-manager", {
        provider: smProvider,
        config: smConfig,
      });
      toast.success("Secret manager configuration saved");
      mutate("/api/settings");
    } catch (err: any) {
      toast.error(err?.response?.data?.message || "Failed to save secret manager");
    }
  };

  // ---------- Notification Preferences Handlers ----------
  const handleNotificationsSubmit = async (e: FormEvent) => {
    e.preventDefault();
    try {
      await api.put("/api/settings/notifications", {
        emailEnabled: notifEmailEnabled,
        email: notifEmail,
        slackWebhookUrl: notifSlackUrl,
      });
      toast.success("Notification preferences updated");
      mutate("/api/settings");
    } catch (err: any) {
      toast.error(err?.response?.data?.message || "Failed to update notifications");
    }
  };

  // ---------- Render ----------
  if (isLoading) {
    return (
      <Layout>
        <div className="p-6 space-y-6">
          <LoadingSkeleton className="h-8 w-48" />
          <LoadingSkeleton className="h-64 w-full" />
          <LoadingSkeleton className="h-48 w-full" />
        </div>
      </Layout>
    );
  }

  if (error) {
    return (
      <Layout>
        <AlertBanner type="error" className="m-6">
          Failed to load settings. Please try again later.
        </AlertBanner>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="max-w-4xl mx-auto p-6 space-y-10">
        {/* API Tokens Section */}
        <section className="space-y-4">
          <h2 className="text-xl font-semibold text-gray-800">API Tokens</h2>
          <p className="text-sm text-gray-600">
            Create tokens for external tools or scripts. Tokens are shown only once
            after creation.
          </p>

          <form onSubmit={handleAddToken} className="flex items-center space-x-3">
            <label htmlFor="new-token-name" className="sr-only">
              Token name
            </label>
            <input
              id="new-token-name"
              type="text"
              placeholder="Token name"
              value={newTokenName}
              onChange={(e) => setNewTokenName(e.target.value)}
              className="flex-1 rounded border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-600"
              required
            />
            <button
              type="submit"
              className="rounded bg-primary-600 px-4 py-2 text-white hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-600"
            >
              Create
            </button>
          </form>

          <div className="mt-4 grid gap-4 md:grid-cols-2">
            {data?.apiTokens?.length ? (
              data.apiTokens.map((token) => (
                <div
                  key={token.id}
                  className="rounded border border-gray-200 bg-white p-4 shadow-sm"
                >
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="font-medium text-gray-800">{token.name}</p>
                      <p className="text-xs text-gray-500">
                        Created {new Date(token.createdAt).toLocaleDateString()}
                      </p>
                    </div>
                    <button
                      onClick={() => handleDeleteToken(token.id)}
                      className="text-sm text-danger-600 hover:underline"
                    >
                      Revoke
                    </button>
                  </div>
                </div>
              ))
            ) : (
              <p className="text-gray-500 col-span-2">No API tokens created yet.</p>
            )}
          </div>
        </section>

        {/* Secret Manager Integration */}
        <section className="space-y-4 border-t border-gray-200 pt-8">
          <h2 className="text-xl font-semibold text-gray-800">Secret Manager Integration</h2>
          <p className="text-sm text-gray-600">
            Connect to a secret manager to securely store CI/CD tokens and credentials.
          </p>

          <form onSubmit={handleSecretManagerSubmit} className="space-y-4">
            <div className="flex items-center space-x-4">
              <label htmlFor="sm-provider" className="w-32 font-medium text-gray-700">
                Provider
              </label>
              <select
                id="sm-provider"
                value={smProvider}
                onChange={(e) => setSmProvider(e.target.value as any)}
                className="rounded border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-600"
                required
              >
                <option value="">Select provider</option>
                <option value="gcp">Google Cloud Secret Manager</option>
                <option value="vault">HashiCorp Vault</option>
              </select>
            </div>

            {smProvider === "gcp" && (
              <div className="space-y-2">
                <label htmlFor="gcp-project" className="block font-medium text-gray-700">
                  GCP Project ID
                </label>
                <input
                  id="gcp-project"
                  type="text"
                  value={smConfig["projectId"] || ""}
                  onChange={(e) =>
                    setSmConfig((prev) => ({ ...prev, projectId: e.target.value }))
                  }
                  className="w-full rounded border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-600"
                  required
                />
              </div>
            )}

            {smProvider === "vault" && (
              <div className="space-y-2">
                <label htmlFor="vault-address" className="block font-medium text-gray-700">
                  Vault Address
                </label>
                <input
                  id="vault-address"
                  type="url"
                  value={smConfig["address"] || ""}
                  onChange={(e) =>
                    setSmConfig((prev) => ({ ...prev, address: e.target.value }))
                  }
                  className="w-full rounded border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-600"
                  required
                />
                <label htmlFor="vault-token" className="block font-medium text-gray-700">
                  Vault Token
                </label>
                <input
                  id="vault-token"
                  type="password"
                  value={smConfig["token"] || ""}
                  onChange={(e) =>
                    setSmConfig((prev) => ({ ...prev, token: e.target.value }))
                  }
                  className="w-full rounded border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-600"
                  required
                />
              </div>
            )}

            <button
              type="submit"
              className="rounded bg-primary-600 px-4 py-2 text-white hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-600"
            >
              Save Configuration
            </button>
          </form>
        </section>

        {/* Notification Preferences */}
        <section className="space-y-4 border-t border-gray-200 pt-8">
          <h2 className="text-xl font-semibold text-gray-800">Notification Preferences</h2>
          <p className="text-sm text-gray-600">
            Choose how you’d like to receive alerts about pipeline health and queue
            starvation.
          </p>

          <form onSubmit={handleNotificationsSubmit} className="space-y-4">
            <div className="flex items-center space-x-4">
              <label htmlFor="email-enabled" className="w-32 font-medium text-gray-700">
                Email alerts
              </label>
              <input
                id="email-enabled"
                type="checkbox"
                checked={notifEmailEnabled}
                onChange={(e) => setNotifEmailEnabled(e.target.checked)}
                className="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-600"
              />
            </div>

            {notifEmailEnabled && (
              <div className="flex items-center space-x-4">
                <label htmlFor="email-address" className="w-32 font-medium text-gray-700">
                  Email address
                </label>
                <input
                  id="email-address"
                  type="email"
                  value={notifEmail}
                  onChange={(e) => setNotifEmail(e.target.value)}
                  className="flex-1 rounded border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-600"
                  required
                />
              </div>
            )}

            <div className="flex items-center space-x-4">
              <label htmlFor="slack-webhook" className="w-32 font-medium text-gray-700">
                Slack webhook
              </label>
              <input
                id="slack-webhook"
                type="url"
                placeholder="https://hooks.slack.com/services/..."
                value={notifSlackUrl}
                onChange={(e) => setNotifSlackUrl(e.target.value)}
                className="flex-1 rounded border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-600"
              />
            </div>

            <button
              type="submit"
              className="rounded bg-primary-600 px-4 py-2 text-white hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-600"
            >
              Save Preferences
            </button>
          </form>
        </section>
      </div>
    </Layout>
  );
}