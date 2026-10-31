import React, { useState, useRef, useEffect, FormEvent } from "react";

type Role = "user" | "assistant";

interface Message {
  role: Role;
  content: string;
}

/**
 * Renders a single message bubble.
 * Handles plain text and fenced code blocks.
 */
const MessageBubble: React.FC<{ message: Message }> = ({ message }) => {
  const renderParts = (content: string) => {
    // Split by fenced code blocks (```lang\ncode\n```), keep delimiters
    const parts = content.split(/(```[\s\S]*?```)/g);
    return parts.map((part, idx) => {
      const codeMatch = part.match(/^```(\w*)\n([\s\S]*?)```$/);
      if (codeMatch) {
        const language = codeMatch[1] || "text";
        const code = codeMatch[2];
        return (
          <CodeBlock key={idx} language={language} code={code} />
        );
      }
      // Normal text – preserve line breaks
      return (
        <p key={idx} className="whitespace-pre-wrap text-sm text-gray-800">
          {part}
        </p>
      );
    });
  };

  const bubbleClasses =
    message.role === "user"
      ? "bg-primary text-white self-end"
      : "bg-card text-gray-800 self-start";

  const containerJustify = message.role === "user" ? "justify-end" : "justify-start";

  return (
    <div className={`flex ${containerJustify} mb-3`}>
      <div
        className={`${bubbleClasses} rounded-lg p-3 max-w-[80%] shadow-sm break-words`}
      >
        {renderParts(message.content)}
      </div>
    </div>
  );
};

/**
 * Code block component with copy‑to‑clipboard functionality.
 */
const CodeBlock: React.FC<{ language: string; code: string }> = ({
  language,
  code,
}) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Copy failed", err);
    }
  };

  return (
    <div className="relative my-2 rounded-md bg-gray-900 text-gray-100 overflow-hidden">
      <pre className="p-4 text-sm overflow-x-auto">
        <code>{code}</code>
      </pre>
      <button
        type="button"
        onClick={handleCopy}
        className="absolute top-2 right-2 rounded bg-gray-800 px-2 py-1 text-xs text-gray-200 hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-primary"
        aria-label="Copy code to clipboard"
      >
        {copied ? "Copied!" : "Copy"}
      </button>
      {language && (
        <div className="absolute bottom-0 left-0 bg-gray-800 bg-opacity-75 px-2 py-0.5 text-xs text-gray-200">
          {language}
        </div>
      )}
    </div>
  );
};

/**
 * Main AI chat component.
 * Streams LLM responses, displays them, and allows user interaction.
 */
const AIChat: React.FC = () => {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const chatContainerRef = useRef<HTMLDivElement>(null);

  // Auto‑scroll to newest message
  useEffect(() => {
    if (chatContainerRef.current) {
      chatContainerRef.current.scrollTop =
        chatContainerRef.current.scrollHeight;
    }
  }, [messages, loading]);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    const trimmed = input.trim();
    if (!trimmed) return;

    // Add user message
    const userMsg: Message = { role: "user", content: trimmed };
    setMessages((prev) => [...prev, userMsg]);
    setInput("");
    setLoading(true);

    try {
      const response = await fetch("/api/ai/chat", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ message: trimmed }),
      });

      if (!response.ok) {
        throw new Error(`Server error ${response.status}`);
      }

      const reader = response.body?.getReader();
      if (!reader) {
        throw new Error("ReadableStream not supported by browser.");
      }

      // Placeholder assistant message that will be updated incrementally
      const assistantMsg: Message = { role: "assistant", content: "" };
      setMessages((prev) => [...prev, assistantMsg]);

      const decoder = new TextDecoder();
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        const chunk = decoder.decode(value);
        setMessages((prev) => {
          const last = prev[prev.length - 1];
          if (last.role !== "assistant") return prev;
          const updated: Message = {
            ...last,
            content: last.content + chunk,
          };
          return [...prev.slice(0, -1), updated];
        });
      }
    } catch (err) {
      console.error(err);
      const errorMsg: Message = {
        role: "assistant",
        content: "⚠️ Sorry, I couldn’t process your request. Please try again.",
      };
      setMessages((prev) => [...prev, errorMsg]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-col h-full max-h-[80vh] bg-background p-4 rounded-lg shadow">
      <div
        ref={chatContainerRef}
        className="flex-1 overflow-y-auto space-y-2 mb-4"
        aria-live="polite"
      >
        {messages.map((msg, idx) => (
          <MessageBubble key={idx} message={msg} />
        ))}
        {loading && (
          <div className="flex justify-start mb-3">
            <div className="bg-card text-gray-800 rounded-lg p-3 shadow-sm max-w-[80%]">
              <div className="flex items-center space-x-2">
                <svg
                  className="animate-spin h-5 w-5 text-primary"
                  xmlns="http://www.w3.org/2000/svg"
                  fill="none"
                  viewBox="0 0 24 24"
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
                    d="M4 12a8 8 0 018-8v8z"
                  />
                </svg>
                <span className="text-sm text-gray-600">Thinking...</span>
              </div>
            </div>
          </div>
        )}
      </div>

      <form onSubmit={handleSubmit} className="flex items-end space-x-2">
        <label htmlFor="ai-chat-input" className="sr-only">
          Ask PulseForge
        </label>
        <textarea
          id="ai-chat-input"
          rows={1}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          disabled={loading}
          placeholder="Ask a question about your pipelines…"
          className="flex-1 resize-none rounded-md border border-gray-300 p-2 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent disabled:opacity-50"
        />
        <button
          type="submit"
          disabled={loading || !input.trim()}
          className={`rounded-md px-4 py-2 text-sm font-medium text-white ${
            loading || !input.trim()
              ? "bg-gray-400 cursor-not-allowed"
              : "bg-primary hover:bg-primary-dark"
          } focus:outline-none focus:ring-2 focus:ring-primary`}
        >
          Send
        </button>
      </form>
    </div>
  );
};

export default AIChat;