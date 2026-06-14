import { FileDown, FileJson } from "lucide-react";
import { useState } from "react";
import { app } from "../../lib/bridge";

export function PublishWidget() {
  const [loading, setLoading] = useState<"html" | "json" | null>(null);
  const [preview, setPreview] = useState<string | null>(null);

  const handleExport = async (format: "html" | "json") => {
    setLoading(format);
    try {
      const content = format === "html"
        ? await app.PublishSessionHTML()
        : await app.PublishSessionJSON();
      if (content) {
        setPreview(content);
        // Copy to clipboard
        try {
          await navigator.clipboard.writeText(content);
        } catch { /* ignore */ }
        // Also trigger download
        const blob = new Blob([content], {
          type: format === "html" ? "text/html" : "application/json",
        });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `session-export.${format}`;
        a.click();
        URL.revokeObjectURL(url);
      }
    } catch {
      // silent
    } finally {
      setLoading(null);
    }
  };

  return (
    <div className="hermes-widget" style={{ padding: "8px 0" }}>
      <h4 style={{ fontSize: 13, fontWeight: 600, margin: "0 0 8px", color: "var(--color-text-muted)" }}>
        <FileDown size={13} style={{ marginRight: 4, verticalAlign: "middle" }} />
        Export Session
      </h4>
      <p style={{ fontSize: 11, color: "var(--color-text-muted)", marginBottom: 8 }}>
        Export the current session transcript as a self-contained HTML document or
        JSON data file. The HTML includes inline CSS and syntax-highlighted code blocks.
      </p>
      <div style={{ display: "flex", gap: 8 }}>
        <button
          className="btn btn-sm"
          onClick={() => handleExport("html")}
          disabled={loading !== null}
          style={{ fontSize: 12, display: "flex", alignItems: "center", gap: 4 }}
        >
          <FileDown size={14} />
          {loading === "html" ? "Exporting..." : "Export HTML"}
        </button>
        <button
          className="btn btn-sm"
          onClick={() => handleExport("json")}
          disabled={loading !== null}
          style={{ fontSize: 12, display: "flex", alignItems: "center", gap: 4 }}
        >
          <FileJson size={14} />
          {loading === "json" ? "Exporting..." : "Export JSON"}
        </button>
      </div>
      {preview && (
        <div
          style={{
            marginTop: 8,
            padding: 8,
            borderRadius: 6,
            background: "var(--color-bg-secondary)",
            fontSize: 11,
            color: "var(--color-success, #22c55e)",
          }}
        >
          Exported and copied to clipboard ✓
        </div>
      )}
    </div>
  );
}
