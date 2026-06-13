import { useState, useEffect } from "react";
import { Network, Tag } from "lucide-react";
import { app } from "../../lib/bridge";

interface MemoryFactView {
  title: string;
  type: string;
  description: string;
}

interface HermesDashboardPayload {
  memoryFacts?: MemoryFactView[];
}

const TYPE_COLORS: Record<string, string> = {
  user: "#8b7cff",
  project: "#d4a853",
  feedback: "#38d6a8",
  reference: "#4d8df6",
  local: "#ff6a3d",
};

function typeColor(kind: string): string {
  for (const [k, v] of Object.entries(TYPE_COLORS)) {
    if (kind.startsWith(k)) return v;
  }
  return "var(--color-text-muted)";
}

export function MemoryFactGraph() {
  const [facts, setFacts] = useState<MemoryFactView[]>([]);

  useEffect(() => {
    try {
      const w = window as any;
      if (w.runtime?.EventsOn) {
        const unsub = w.runtime.EventsOn("hermes:dashboard", (payload: HermesDashboardPayload) => {
          if (payload?.memoryFacts) setFacts(payload.memoryFacts);
        });
        app.MemoryFacts().then(setFacts).catch(() => {});
        return () => { try { unsub(); } catch { /* ignore */ } };
      }
    } catch { /* fall through */ }

    app.MemoryFacts().then(setFacts).catch(() => {});
    const id = setInterval(() => app.MemoryFacts().then(setFacts).catch(() => {}), 5000);
    return () => clearInterval(id);
  }, []);

  if (facts.length === 0) {
    return (
      <div style={{ fontSize: 13, color: "var(--color-text-muted)", fontStyle: "italic", padding: "8px 0" }}>
        No memory facts yet. Facts are saved via <code>remember</code>, <code>#</code> quick-add, or auto-memory.
      </div>
    );
  }

  // Group by type for a clustered display.
  const groups: Record<string, MemoryFactView[]> = {};
  for (const f of facts) {
    const baseType = f.type.split(":")[0] || f.type;
    if (!groups[baseType]) groups[baseType] = [];
    groups[baseType].push(f);
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
      <div style={{ display: "flex", gap: 12, fontSize: 11, color: "var(--color-text-muted)" }}>
        <span><Network size={11} style={{ marginRight: 3, verticalAlign: -1 }} />{facts.length} facts</span>
        <span><Tag size={11} style={{ marginRight: 3, verticalAlign: -1 }} />{Object.keys(groups).length} types</span>
      </div>

      {/* Clustered fact display */}
      <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
        {Object.entries(groups).map(([kind, items]) => (
          <div key={kind}>
            <div style={{
              display: "flex", alignItems: "center", gap: 6, marginBottom: 6,
              fontSize: 11, fontWeight: 600, color: typeColor(kind),
            }}>
              <span style={{
                width: 8, height: 8, borderRadius: "50%",
                background: typeColor(kind), display: "inline-block",
              }} />
              {kind}
              <span style={{ color: "var(--color-text-muted)", fontWeight: 400 }}>
                ({items.length})
              </span>
            </div>
            <div style={{ display: "flex", flexWrap: "wrap", gap: 6 }}>
              {items.map((f, i) => (
                <div
                  key={i}
                  title={f.description}
                  style={{
                    padding: "4px 8px", borderRadius: 4, fontSize: 11,
                    border: `1px solid ${typeColor(kind)}40`,
                    background: `${typeColor(kind)}10`,
                    color: "var(--fg)", maxWidth: 200,
                    overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
                  }}
                >
                  {f.title}
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
