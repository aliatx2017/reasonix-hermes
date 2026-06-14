import { Bot, Network } from "lucide-react";
import type { CouncilDashboardView } from "../../lib/types";

interface CouncilPanelProps {
  council: CouncilDashboardView | null;
}

export function CouncilPanel({ council }: CouncilPanelProps) {
  if (!council || !council.enabled) {
    return (
      <div className="hermes-panel" style={{ padding: 12, color: "var(--color-text-muted)", fontSize: 12 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 6 }}>
          <Network size={14} style={{ opacity: 0.5 }} />
          <span>Multi-model council is {council?.status || "disabled"}.</span>
        </div>
        <span>Configure in reasonix.toml: [mesh] + [[mesh.peers]]</span>
      </div>
    );
  }

  return (
    <div className="hermes-panel" style={{ padding: 12, fontSize: 12 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 8 }}>
        <Network size={14} style={{ color: "var(--color-accent)" }} />
        <span style={{ fontWeight: 600 }}>Council</span>
        <span style={{ color: "var(--color-text-muted)", fontFamily: "monospace", fontSize: 10 }}>
          {council.peers.length} peer{council.peers.length !== 1 ? "s" : ""}
        </span>
      </div>
      {council.peers.map((p) => (
        <div
          key={p.name}
          style={{
            display: "flex",
            alignItems: "center",
            gap: 6,
            padding: "3px 8px",
            marginBottom: 2,
            borderRadius: 4,
            background: "var(--color-surface-raised)",
          }}
        >
          <Bot size={12} style={{ color: p.enabled ? "var(--color-green)" : "var(--color-text-muted)" }} />
          <span style={{ fontWeight: 500 }}>{p.name}</span>
          {p.url && (
            <span style={{ color: "var(--color-text-muted)", fontFamily: "monospace", fontSize: 10, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
              {p.url}
            </span>
          )}
        </div>
      ))}
      {council.status && (
        <div style={{ marginTop: 6, color: "var(--color-text-muted)", fontSize: 10 }}>
          {council.status}
        </div>
      )}
    </div>
  );
}
