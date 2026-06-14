import { Download, Search, Star, Tag } from "lucide-react";
import type { CouncilDashboardView } from "../../lib/types"; // reuse: extend later

interface MarketplaceSkill {
  name: string;
  description: string;
  author: string;
  tags: string[];
  rating: number;
  url: string;
}

interface MarketplacePanelProps {
  council?: CouncilDashboardView | null; // placeholder — standalone panel
}

const MOCK_SKILLS: MarketplaceSkill[] = [
  { name: "golang-patterns", description: "Idiomatic Go patterns, best practices, and conventions.", author: "reasonix-hermes", tags: ["go", "patterns"], rating: 4.8, url: "" },
  { name: "code-review", description: "Universal code review checklist.", author: "reasonix-hermes", tags: ["review", "quality"], rating: 4.7, url: "" },
  { name: "golang-testing", description: "Go testing patterns including table-driven tests.", author: "reasonix-hermes", tags: ["go", "testing"], rating: 4.6, url: "" },
  { name: "api-design", description: "REST API design patterns.", author: "reasonix-hermes", tags: ["api", "design"], rating: 4.6, url: "" },
  { name: "evidence-first-reasoning", description: "Evidence-first diagnostic reasoning.", author: "reasonix-hermes", tags: ["reasoning", "diagnosis"], rating: 4.5, url: "" },
  { name: "frontend-patterns", description: "React, Next.js, state management patterns.", author: "reasonix-hermes", tags: ["react", "frontend"], rating: 4.5, url: "" },
];

export function MarketplacePanel(_props: MarketplacePanelProps) {
  return (
    <div className="hermes-panel" style={{ padding: 12, fontSize: 12 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 8 }}>
        <Search size={14} style={{ color: "var(--color-accent)" }} />
        <span style={{ fontWeight: 600 }}>Skill Marketplace</span>
        <span style={{ color: "var(--color-text-muted)", fontSize: 10 }}>
          {MOCK_SKILLS.length} skills · agentskills.io-compatible
        </span>
      </div>

      <div style={{ marginBottom: 8 }}>
        <div style={{ display: "flex", gap: 4, flexWrap: "wrap", marginBottom: 8 }}>
          {["go", "python", "testing", "review", "api", "react"].map((tag) => (
            <span
              key={tag}
              style={{
                padding: "1px 6px",
                borderRadius: 4,
                background: "var(--color-surface-raised)",
                border: "1px solid var(--color-border)",
                fontSize: 10,
                cursor: "pointer",
                color: "var(--color-text-muted)",
              }}
            >
              {tag}
            </span>
          ))}
        </div>
      </div>

      <div style={{ display: "flex", flexDirection: "column", gap: 4, maxHeight: 300, overflowY: "auto" }}>
        {MOCK_SKILLS.map((skill) => (
          <div
            key={skill.name}
            style={{
              display: "flex",
              alignItems: "flex-start",
              gap: 8,
              padding: "6px 8px",
              borderRadius: 6,
              background: "var(--color-surface-raised)",
              border: "1px solid var(--color-border)",
            }}
          >
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ display: "flex", alignItems: "center", gap: 4 }}>
                <span style={{ fontWeight: 600, fontSize: 12 }}>{skill.name}</span>
                <span style={{ display: "flex", alignItems: "center", gap: 2, color: "var(--color-accent)", fontSize: 10 }}>
                  <Star size={10} /> {skill.rating}
                </span>
              </div>
              <div style={{ color: "var(--color-text-muted)", fontSize: 10, marginTop: 2, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                {skill.description}
              </div>
              <div style={{ display: "flex", gap: 3, marginTop: 3, flexWrap: "wrap" }}>
                {skill.tags.map((tag) => (
                  <span key={tag} style={{ display: "inline-flex", alignItems: "center", gap: 2, fontSize: 9, color: "var(--color-text-muted)" }}>
                    <Tag size={8} />{tag}
                  </span>
                ))}
              </div>
            </div>
            <button
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: 3,
                padding: "3px 8px",
                borderRadius: 4,
                border: "1px solid var(--color-accent)",
                background: "transparent",
                color: "var(--color-accent)",
                cursor: "pointer",
                fontSize: 10,
                fontWeight: 500,
                flexShrink: 0,
              }}
              title={`Install ${skill.name}`}
            >
              <Download size={10} /> Install
            </button>
          </div>
        ))}
      </div>

      <div style={{ marginTop: 8, color: "var(--color-text-muted)", fontSize: 10 }}>
        CLI: <code style={{ fontFamily: "monospace" }}>reasonix marketplace search &lt;query&gt;</code>
      </div>
    </div>
  );
}
