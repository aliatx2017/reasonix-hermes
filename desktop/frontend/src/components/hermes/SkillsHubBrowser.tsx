import { useState } from "react";
import { BookOpen, Search, Tag } from "lucide-react";

interface SkillEntry {
  name: string;
  description: string;
  category: string;
  author?: string;
  runAs?: string;
}

// Static snapshot of our 17-skill registry — avoids network fetch at runtime.
const SKILLS_HUB: SkillEntry[] = [
  { name: "adversarial-review", description: "Adversarial code review with structured BLOCK/ALLOW output and risk scoring.", category: "security", runAs: "subagent" },
  { name: "code-review", description: "Comprehensive code review covering correctness, security, performance, and style.", category: "quality", runAs: "subagent" },
  { name: "test-generator", description: "Generate unit tests with TDD methodology — table-driven, subtests, mocks.", category: "testing" },
  { name: "refactoring", description: "Systematic refactoring with safety checks — extract method, rename, inline.", category: "refactoring" },
  { name: "api-design", description: "REST API design patterns: resource naming, status codes, pagination, versioning.", category: "architecture" },
  { name: "git-commit", description: "Generate conventional commit messages from staged diffs following best practices.", category: "git" },
  { name: "debugger", description: "Systematic debugging workflow: reproduce, isolate, fix, verify, prevent.", category: "debugging" },
  { name: "documentation", description: "Generate or improve code documentation — docstrings, READMEs, API docs.", category: "documentation" },
  { name: "council", description: "Multi-agent discussion and decision-making pattern with role simulation.", category: "collaboration" },
  { name: "deep-research", description: "In-depth research on a topic using web search and codebase analysis.", category: "research", runAs: "subagent" },
  { name: "security-audit", description: "Security-focused audit: injection, authz, secrets, deserialization, path traversal.", category: "security", runAs: "subagent" },
  { name: "migration-assistant", description: "Assist with framework/library migrations: analyze breaking changes, generate plans.", category: "migration" },
  { name: "performance-profiler", description: "Identify performance bottlenecks, suggest optimizations, benchmark comparisons.", category: "performance" },
  { name: "ci-cd-helper", description: "Generate and fix CI/CD pipeline configurations for GitHub Actions, GitLab CI, etc.", category: "devops" },
  { name: "database-helper", description: "SQL query optimization, schema design review, migration generation.", category: "database" },
  { name: "frontend-builder", description: "Build components from specs with accessibility, responsive design, and tests.", category: "frontend" },
  { name: "explore", description: "Wide-net read-only codebase exploration in an isolated subagent.", category: "exploration", runAs: "subagent" },
];

const CATEGORIES = [...new Set(SKILLS_HUB.map((s) => s.category))].sort();

export function SkillsHubBrowser() {
  const [filter, setFilter] = useState("");
  const [category, setCategory] = useState("");

  const filtered = SKILLS_HUB.filter((s) => {
    if (category && s.category !== category) return false;
    if (filter && !s.name.includes(filter.toLowerCase()) && !s.description.toLowerCase().includes(filter.toLowerCase())) return false;
    return true;
  });

  return (
    <div style={{ padding: "8px 0" }}>
      <div style={{ display: "flex", gap: 8, marginBottom: 12, alignItems: "center" }}>
        <div style={{ position: "relative", flex: 1 }}>
          <Search size={14} style={{ position: "absolute", left: 8, top: "50%", transform: "translateY(-50%)", color: "var(--color-text-muted)" }} />
          <input
            type="text"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            placeholder="Search 17 skills..."
            style={{
              width: "100%", padding: "6px 6px 6px 28px", fontSize: 13, borderRadius: 6,
              border: "1px solid var(--color-border)", background: "var(--color-surface)", color: "var(--color-text)",
            }}
          />
        </div>
        <select
          value={category}
          onChange={(e) => setCategory(e.target.value)}
          style={{
            padding: "6px 8px", fontSize: 13, borderRadius: 6,
            border: "1px solid var(--color-border)", background: "var(--color-surface)", color: "var(--color-text)",
          }}
        >
          <option value="">All categories</option>
          {CATEGORIES.map((c) => (
            <option key={c} value={c}>{c}</option>
          ))}
        </select>
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(260px, 1fr))", gap: 8 }}>
        {filtered.map((skill) => (
          <div
            key={skill.name}
            style={{
              padding: "10px 12px", borderRadius: 8,
              background: "var(--color-surface-raised)", border: "1px solid var(--color-border)",
              cursor: "default",
            }}
          >
            <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 4 }}>
              <BookOpen size={14} style={{ color: "var(--color-accent)", flexShrink: 0 }} />
              <span style={{ fontWeight: 600, fontSize: 13 }}>{skill.name}</span>
              {skill.runAs === "subagent" && (
                <span style={{ fontSize: 10, padding: "0 4px", borderRadius: 3, background: "var(--color-accent-subtle)", color: "var(--color-accent)" }}>
                  subagent
                </span>
              )}
            </div>
            <div style={{ fontSize: 12, color: "var(--color-text-muted)", lineHeight: 1.4, marginBottom: 6 }}>
              {skill.description}
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
              <span style={{
                fontSize: 10, padding: "1px 6px", borderRadius: 3,
                background: "var(--color-surface)", border: "1px solid var(--color-border)",
                color: "var(--color-text-muted)", display: "inline-flex", alignItems: "center", gap: 3,
              }}>
                <Tag size={10} />
                {skill.category}
              </span>
              {skill.author && (
                <span style={{ fontSize: 10, color: "var(--color-text-muted)" }}>
                  by {skill.author}
                </span>
              )}
            </div>
          </div>
        ))}
      </div>

      {filtered.length === 0 && (
        <div style={{ textAlign: "center", padding: 24, color: "var(--color-text-muted)", fontStyle: "italic" }}>
          No skills match your search.
        </div>
      )}

      <div style={{ marginTop: 12, fontSize: 12, color: "var(--color-text-muted)", textAlign: "center" }}>
        {filtered.length} of {SKILLS_HUB.length} skills · install via{" "}
        <code style={{ fontSize: 11 }}>reasonix install-source install --source https://github.com/aliatx2017/reasonix-hermes/tree/main/skills-hub/skills</code>
      </div>
    </div>
  );
}
