import React, { useState, useEffect, useCallback } from "react";
import { XMarkIcon } from "@heroicons/react/24/outline";

export type AlertType = "info" | "warning" | "error" | "success";

export interface AlertBannerProps {
  /** Optional identifier – useful when the parent maintains a list of alerts */
  id?: string;
  /** Visual style of the alert */
  type: AlertType;
  /** Optional short title, rendered in bold */
  title?: string;
  /** Main message body */
  message: string;
  /** Called when the user dismisses the banner */
  onDismiss?: (id?: string) => void;
}

/**
 * Dismissable banner displayed at the top of the UI for system‑wide alerts
 * (e.g. queue starvation, failed health‑checks). It follows the PulseForge
 * design system: accessible, responsive and colour‑coded per alert type.
 */
export const AlertBanner: React.FC<AlertBannerProps> = ({
  id,
  type,
  title,
  message,
  onDismiss,
}) => {
  const [visible, setVisible] = useState(true);

  // Auto‑dismiss after a configurable timeout for non‑critical alerts
  useEffect(() => {
    if (type === "info" || type === "success") {
      const timer = setTimeout(() => handleClose(), 10_000);
      return () => clearTimeout(timer);
    }
    return;
  }, [type]);

  const handleClose = useCallback(() => {
    setVisible(false);
    if (onDismiss) {
      onDismiss(id);
    }
  }, [id, onDismiss]);

  if (!visible) {
    return null;
  }

  // Tailwind colour mapping per alert type
  const colourMap: Record<AlertType, { bg: string; border: string; text: string; icon: JSX.Element }> = {
    info: {
      bg: "bg-primary-50",
      border: "border-primary-200",
      text: "text-primary-800",
      icon: (
        <svg
          className="h-5 w-5 text-primary-600"
          fill="currentColor"
          viewBox="0 0 20 20"
          aria-hidden="true"
        >
          <path d="M9 9V5a1 1 0 112 0v4a1 1 0 01-2 0zm0 6a1 1 0 112 0 1 1 0 01-2 0z" />
        </svg>
      ),
    },
    warning: {
      bg: "bg-yellow-50",
      border: "border-yellow-200",
      text: "text-yellow-800",
      icon: (
        <svg
          className="h-5 w-5 text-yellow-600"
          fill="currentColor"
          viewBox="0 0 20 20"
          aria-hidden="true"
        >
          <path d="M8.257 3.099c.765-1.36 2.721-1.36 3.486 0l6.518 11.6c.75 1.334-.213 2.999-1.743 2.999H3.482c-1.53 0-2.493-1.665-1.743-2.999l6.518-11.6zM11 13a1 1 0 10-2 0 1 1 0 002 0zm-1-8a1 1 0 00-.993.883L9 6v4a1 1 0 001.993.117L11 10V6a1 1 0 00-1-1z" />
        </svg>
      ),
    },
    error: {
      bg: "bg-danger-50",
      border: "border-danger-200",
      text: "text-danger-800",
      icon: (
        <svg
          className="h-5 w-5 text-danger-600"
          fill="currentColor"
          viewBox="0 0 20 20"
          aria-hidden="true"
        >
          <path
            fillRule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-10.293a1 1 0 00-1.414-1.414L10 8.586 7.707 6.293a1 1 0 00-1.414 1.414L8.586 10l-2.293 2.293a1 1 0 101.414 1.414L10 11.414l2.293 2.293a1 1 0 001.414-1.414L11.414 10l2.293-2.293z"
            clipRule="evenodd"
          />
        </svg>
      ),
    },
    success: {
      bg: "bg-success-50",
      border: "border-success-200",
      text: "text-success-800",
      icon: (
        <svg
          className="h-5 w-5 text-success-600"
          fill="currentColor"
          viewBox="0 0 20 20"
          aria-hidden="true"
        >
          <path
            fillRule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.707a1 1 0 00-1.414-1.414L9 11.586 7.707 10.293a1 1 0 00-1.414 1.414L9 14.414l4.707-4.707z"
            clipRule="evenodd"
          />
        </svg>
      ),
    },
  };

  const { bg, border, text, icon } = colourMap[type];

  return (
    <div
      role="alert"
      aria-live="assertive"
      className={`w-full max-w-4xl mx-auto my-4 px-4 py-3 rounded-md border ${bg} ${border} flex items-start sm:items-center space-x-3 sm:space-x-4`}
    >
      <div className="flex-shrink-0">{icon}</div>
      <div className={`flex-1 ${text} text-sm`}>
        {title && <p className="font-medium">{title}</p>}
        <p>{message}</p>
      </div>
      <button
        type="button"
        onClick={handleClose}
        className={`flex-shrink-0 rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 ${text} hover:${text} hover:bg-opacity-20`}
        aria-label="Dismiss alert"
      >
        <XMarkIcon className="h-5 w-5" aria-hidden="true" />
      </button>
    </div>
  );
};

export default AlertBanner;