import { Radio, Users } from "lucide-react";
import type { CollabView } from "../../lib/types";

interface CollabPanelProps {
  collab: CollabView | null;
}

export function CollabPanel({ collab }: CollabPanelProps) {
  if (!collab || !collab.enabled) {
    return (
      <div className="hermes-panel" style={{ padding: 12, color: "var(--color-text-muted)", fontSize: 12 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 6 }}>
          <Radio size={14} style={{ opacity: 0.5 }} />
          <span>Live collaboration is disabled.</span>
        </div>
        <span>Enable in reasonix.toml: [collab] enabled = true</span>
      </div>
    );
  }

  return (
    <div className="hermes-panel" style={{ padding: 12, fontSize: 12 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 8 }}>
        <Radio size={14} style={{ color: "var(--color-green)" }} />
        <span style={{ fontWeight: 600, color: "var(--color-green)" }}>Live</span>
        <span style={{ color: "var(--color-text-muted)", fontFamily: "monospace", fontSize: 11 }}>
          {collab.listenAddr}
        </span>
      </div>

      <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 8 }}>
        <Users size={14} style={{ color: "var(--color-accent)" }} />
        <span style={{ fontWeight: 500 }}>{collab.watchers}</span>
        <span style={{ color: "var(--color-text-muted)" }}>
          watcher{collab.watchers !== 1 ? "s" : ""} connected
        </span>
      </div>

      {collab.sessions.length > 0 && (
        <div style={{ marginTop: 6 }}>
          <div style={{ fontWeight: 600, marginBottom: 4, color: "var(--color-text-muted)", fontSize: 10, textTransform: "uppercase", letterSpacing: "0.05em" }}>
            Active Sessions
          </div>
          {collab.sessions.map((sid) => (
            <div
              key={sid}
              style={{
                padding: "3px 8px",
                marginBottom: 2,
                borderRadius: 4,
                background: "var(--color-surface-raised)",
                fontFamily: "monospace",
                fontSize: 10,
                color: "var(--color-text-muted)",
                overflow: "hidden",
                textOverflow: "ellipsis",
                whiteSpace: "nowrap",
              }}
              title={sid}
            >
              {sid.length > 50 ? sid.slice(0, 47) + "…" : sid}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
