import { useState, useEffect } from "react";
import { Shield, AlertTriangle, CheckCircle2, FileText } from "lucide-react";
import { app } from "../../lib/bridge";
import type { ConstitutionHealthView } from "../../lib/types";

const SEVERITY_ICONS: Record<string, React.ReactNode> = {
  error: <AlertTriangle size={10} color="var(--color-warn)" />,
  warn: <AlertTriangle size={10} color="var(--color-yellow)" />,
  warning: <AlertTriangle size={10} color="var(--color-yellow)" />,
  info: <CheckCircle2 size={10} color="var(--color-green)" />,
};

export function ConstitutionHealthPanel() {
  const [data, setData] = useState<ConstitutionHealthView | null>(null);

  useEffect(() => {
    // Prefer push events; fall back to fetch-once.
    try {
      const w = window as any;
      if (w.runtime?.EventsOn) {
        const unsub = w.runtime.EventsOn("hermes:dashboard", (payload: any) => {
          if (payload?.constitution) setData(payload.constitution);
        });
        app.ConstitutionHealth().then(setData).catch(() => {});
        return () => { try { unsub(); } catch { /* ignore */ } };
      }
    } catch { /* fall through */ }
    app.ConstitutionHealth().then(setData).catch(() => {});
  }, []);

  if (!data) return null;

  if (!data.loaded) {
    return (
      <div style={{ fontSize: 13, color: "var(--color-text-muted)", padding: "8px 0" }}>
        <div style={{ fontStyle: "italic", marginBottom: 8 }}>
          No constitution found. Create <code>.reasonix/constitution.json</code> in your project root to define project invariants.
        </div>
        <pre style={{
          fontSize: 11, background: "var(--color-surface)", padding: "8px 12px", borderRadius: 4,
          color: "var(--color-text-muted)", lineHeight: 1.5,
        }}>
{`{
  "version": 1,
  "conventions": { "language": "Go" },
  "principles": ["Config-driven core", "Single binary"],
  "constraints": ["No hardcoded models", "Spec-first changes"],
  "rules": [
    { "id": "spec-first", "description": "Change spec before code", "severity": "error", "scope": "*.go" }
  ]
}`}
        </pre>
      </div>
    );
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
      {(data.principles || []).length > 0 && (
        <div>
          <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 4, color: "var(--color-text-muted)" }}>Principles</div>
          {data.principles.map((p, i) => (
            <div key={i} style={{ fontSize: 12, padding: "2px 0", display: "flex", alignItems: "center", gap: 6 }}>
              <Shield size={12} style={{ color: "var(--color-accent)", flexShrink: 0 }} />
              {p}
            </div>
          ))}
        </div>
      )}

      {(data.constraints || []).length > 0 && (
        <div>
          <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 4, color: "var(--color-text-muted)" }}>Hard Constraints</div>
          {data.constraints.map((c, i) => (
            <div key={i} style={{ fontSize: 12, padding: "2px 0", display: "flex", alignItems: "center", gap: 6 }}>
              <AlertTriangle size={12} style={{ color: "var(--color-warn)", flexShrink: 0 }} />
              {c}
            </div>
          ))}
        </div>
      )}

      {(data.rules || []).length > 0 && (
        <div>
          <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 4, color: "var(--color-text-muted)" }}>Rules ({(data.rules || []).length})</div>
          {(data.rules || []).map((r) => (
            <div
              key={r.id}
              style={{
                display: "flex", alignItems: "center", gap: 8,
                padding: "4px 8px", borderRadius: 4, fontSize: 12,
                background: r.severity === "error" ? "var(--color-warn-subtle)" : "var(--color-surface-raised)",
                border: `1px solid ${r.severity === "error" ? "var(--color-warn)" : "var(--color-border)"}`,
                marginBottom: 4,
              }}
            >
              {SEVERITY_ICONS[r.severity] || SEVERITY_ICONS.info}
              <span style={{ fontWeight: 600 }}>{r.id}</span>
              <span style={{ color: "var(--color-text-muted)", flex: 1 }}>{r.description}</span>
              {r.scope && (
                <span style={{ fontSize: 10, padding: "0 4px", borderRadius: 3, background: "var(--color-surface)", color: "var(--color-text-muted)" }}>
                  {r.scope}
                </span>
              )}
            </div>
          ))}
        </div>
      )}

      <div style={{ fontSize: 11, color: "var(--color-text-muted)", marginTop: 4, display: "flex", alignItems: "center", gap: 4 }}>
        <FileText size={11} />
        {data.path}
      </div>
    </div>
  );
}
