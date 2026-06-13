import { useState } from "react";
import { FileText, ChevronDown, ChevronRight } from "lucide-react";
import { app } from "../../lib/bridge";
import type { CheckpointMeta, CheckpointFileSnap } from "../../lib/types";

export function CheckpointFileList({ checkpoints }: { checkpoints: CheckpointMeta[] }) {
  const [expandedTurns, setExpandedTurns] = useState<Set<number>>(new Set());
  const [fileContents, setFileContents] = useState<Record<number, CheckpointFileSnap[]>>({});
  const [loading, setLoading] = useState<Set<number>>(new Set());

  const toggle = async (turn: number) => {
    const next = new Set(expandedTurns);
    if (next.has(turn)) {
      next.delete(turn);
      setExpandedTurns(next);
      return;
    }
    next.add(turn);
    setExpandedTurns(next);

    if (!fileContents[turn]) {
      setLoading((s) => new Set(s).add(turn));
      try {
        const snaps = await app.CheckpointFileList(turn);
        setFileContents((prev) => ({ ...prev, [turn]: snaps }));
      } catch { /* ignore */ }
      setLoading((s) => { const ns = new Set(s); ns.delete(turn); return ns; });
    }
  };

  if (checkpoints.length === 0) {
    return (
      <div style={{ fontSize: 13, color: "var(--color-text-muted)", fontStyle: "italic", padding: "8px 0" }}>
        No checkpoints yet. Checkpoints are created when the agent writes or edits files.
      </div>
    );
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 2, fontSize: 12 }}>
      {checkpoints.map((cp) => {
        const isExpanded = expandedTurns.has(cp.turn);
        const isLoading = loading.has(cp.turn);
        const snaps = fileContents[cp.turn];

        return (
          <div key={cp.turn} style={{ borderBottom: "1px solid var(--color-border-soft)" }}>
            <button
              onClick={() => toggle(cp.turn)}
              style={{
                display: "flex", alignItems: "center", gap: 6, width: "100%",
                padding: "6px 4px", background: "none", border: "none",
                cursor: "pointer", color: "var(--fg)", fontSize: 12, textAlign: "left",
              }}
            >
              {isExpanded ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
              <span style={{ fontWeight: 600 }}>Turn {cp.turn}</span>
              <span style={{ color: "var(--color-text-muted)" }}>
                · {cp.files.length} file{cp.files.length !== 1 ? "s" : ""}
              </span>
              <span style={{ color: "var(--color-text-3)", marginLeft: "auto", fontSize: 11 }}>
                {cp.prompt.length > 60 ? cp.prompt.slice(0, 60) + "…" : cp.prompt}
              </span>
            </button>

            {isExpanded && (
              <div style={{ padding: "0 4px 8px 20px" }}>
                {isLoading && <div style={{ color: "var(--color-text-muted)", padding: 4 }}>Loading…</div>}
                {snaps?.map((s) => (
                  <div key={s.path} style={{ marginTop: 4 }}>
                    <div style={{
                      display: "flex", alignItems: "center", gap: 4,
                      color: "var(--accent)", fontSize: 11, fontWeight: 500, marginBottom: 2,
                    }}>
                      <FileText size={10} />
                      {s.path}
                      {!s.content && <span style={{ color: "var(--color-text-muted)", fontStyle: "italic" }}> (new file)</span>}
                    </div>
                    {s.content && (
                      <pre style={{
                        margin: 0, padding: "4px 8px", background: "var(--code-bg)",
                        borderRadius: 4, fontSize: 11, lineHeight: 1.4,
                        maxHeight: 120, overflow: "auto", color: "var(--fg)",
                      }}>
                        {s.content.length > 500 ? s.content.slice(0, 500) + "\n…" : s.content}
                      </pre>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
