import { BarChart3, BookOpen, Calendar, Download, FileText, GitBranch, History, Keyboard, Network, Shield, Sliders, Zap } from "lucide-react";
import type { SettingsView, HotbarView, ProfileView, CostSummaryView, ScheduleDashboardView } from "../../lib/types";
import { CacheEconomyGauge } from "./CacheEconomyGauge";
import { CostWidget } from "./CostWidget";
import { DiscordMonitor } from "./DiscordMonitor";
import { GoalProgressWidget } from "./GoalProgressWidget";
import { PublishWidget } from "./PublishWidget";
import { ScheduleWidget } from "./ScheduleWidget";
import { SkillsHubBrowser } from "./SkillsHubBrowser";
import { SubagentTreePanel } from "./SubagentTreePanel";
import { ConstitutionHealthPanel } from "./ConstitutionHealthPanel";
import { TokenBreakdownChart } from "./TokenBreakdownChart";
import { CompactionTimeline } from "./CompactionTimeline";
import { CheckpointFileList } from "./CheckpointFileList";
import { MemoryFactGraph } from "./MemoryFactGraph";

interface HermesSettingsProps {
  s: SettingsView;
  onHotbarChange: (hotbar: HotbarView) => void;
  onProfileSelect: (name: string) => void;
  cache?: { hitTokens: number; missTokens: number; totalTokens: number; hitRate: number } | null;
  discord?: { running: boolean; platform: string; activeSessions: number; status: string } | null;
  goal?: { active: boolean; goal: string; status: string; turns: number; blocks: number } | null;
  cost?: CostSummaryView | null;
  schedule?: ScheduleDashboardView | null;
}

const HOTBAR_KEYS = [
  { key: "key1" as const, digit: "1", defaultAction: "palette" },
  { key: "key2" as const, digit: "2", defaultAction: "workspace" },
  { key: "key3" as const, digit: "3", defaultAction: "new" },
  { key: "key4" as const, digit: "4", defaultAction: "history" },
  { key: "key5" as const, digit: "5", defaultAction: "dock" },
  { key: "key6" as const, digit: "6", defaultAction: "sidebar" },
  { key: "key7" as const, digit: "7", defaultAction: "settings" },
];

const ACTION_LABELS: Record<string, string> = {
  "": "unbound",
  palette: "Command Palette",
  workspace: "Workspace Panel",
  new: "New Chat",
  history: "History",
  dock: "Toggle Dock",
  sidebar: "Toggle Sidebar",
  settings: "Settings",
};

export function HermesSettings({ s, onHotbarChange: _onHotbar, onProfileSelect: _onProfile, cache, discord, goal, cost, schedule }: HermesSettingsProps) {
  const profiles = Object.entries(s.profiles ?? {}) as [string, ProfileView][];
  const activeProfile = s.activeProfile || "";

  return (
    <div className="hermes-settings" style={{ padding: "16px 0" }}>
      {/* ═══════════ LIVE DASHBOARD ═══════════ */}
      <section className="settings-section">
        <h3 className="settings-section__title">
          <Zap size={16} style={{ marginRight: 6 }} />
          Live Dashboard
        </h3>
        <div style={{ display: "flex", gap: 8, marginTop: 8, flexWrap: "wrap", alignItems: "center" }}>
          <CacheEconomyGauge cache={cache ?? null} />
          <DiscordMonitor status={discord ?? null} />
          <div style={{ flex: 1, minWidth: 200 }}>
            <GoalProgressWidget goal={goal ?? null} compact />
          </div>
        </div>
      </section>

      <hr style={{ margin: "24px 0", border: "none", borderTop: "1px solid var(--color-border)" }} />

      <section className="settings-section">
        <h3 className="settings-section__title">
          <BarChart3 size={16} style={{ marginRight: 6 }} />
          Token Breakdown
        </h3>
        <TokenBreakdownChart />
      </section>

      <hr style={{ margin: "24px 0", border: "none", borderTop: "1px solid var(--color-border)" }} />

      <section className="settings-section">
        <h3 className="settings-section__title">
          <History size={16} style={{ marginRight: 6 }} />
          Compaction Timeline
        </h3>
        <CompactionTimeline />
      </section>

      <hr style={{ margin: "24px 0", border: "none", borderTop: "1px solid var(--color-border)" }} />

      <section className="settings-section">
        <h3 className="settings-section__title">
          <FileText size={16} style={{ marginRight: 6 }} />
          Checkpoint Files
        </h3>
        <p className="settings-section__desc" style={{ fontSize: 12, color: "var(--color-text-muted)", marginBottom: 8 }}>
          Per-turn file snapshots captured before the agent writes or edits. Click a turn to see pre-edit contents.
        </p>
        {/* Checkpoints are fetched from the rewind data; pass empty for standalone section */}
        <CheckpointFileList checkpoints={[]} />
      </section>

      <hr style={{ margin: "24px 0", border: "none", borderTop: "1px solid var(--color-border)" }} />

      <section className="settings-section">
        <h3 className="settings-section__title">
          <Network size={16} style={{ marginRight: 6 }} />
          Memory Fact Graph
        </h3>
        <MemoryFactGraph />
      </section>

      <hr style={{ margin: "24px 0", border: "none", borderTop: "1px solid var(--color-border)" }} />

      {/* ═══════════ COST + SCHEDULE + PUBLISH ═══════════ */}
      <section className="settings-section">
        <h3 className="settings-section__title">
          <Calendar size={16} style={{ marginRight: 6 }} />
          Session Cost & Scheduling
        </h3>
        <div style={{ display: "flex", gap: 12, marginTop: 8, flexWrap: "wrap" }}>
          <div style={{ flex: "1 1 200px", minWidth: 180 }}>
            <CostWidget data={cost ?? null} />
          </div>
          <div style={{ flex: "2 1 350px", minWidth: 300 }}>
            <ScheduleWidget data={schedule ?? null} />
          </div>
        </div>
      </section>

      <hr style={{ margin: "24px 0", border: "none", borderTop: "1px solid var(--color-border)" }} />

      <section className="settings-section">
        <h3 className="settings-section__title">
          <Download size={16} style={{ marginRight: 6 }} />
          Publish Transcript
        </h3>
        <PublishWidget />
      </section>

      <hr style={{ margin: "24px 0", border: "none", borderTop: "1px solid var(--color-border)" }} />

      {/* ═══════════ HOTBAR ═══════════ */}
      <section className="settings-section">
        <h3 className="settings-section__title">
          <Keyboard size={16} style={{ marginRight: 6 }} />
          Hotbar
        </h3>
        <p className="settings-section__desc">
          Keyboard digit keys 1–7 trigger actions. Edit in <code>[desktop.hotbar]</code> in{" "}
          <code>{s.configPath || "reasonix.toml"}</code>.
        </p>
        <div className="hotbar-grid" style={{
          display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(180px, 1fr))",
          gap: 8, marginTop: 12,
        }}>
          {HOTBAR_KEYS.map(({ key, digit, defaultAction }) => {
            const action = s.hotbar?.[key] || "";
            const label = ACTION_LABELS[action] || action;
            const isDefault = !action || action === defaultAction;
            return (
              <div key={key} style={{
                display: "flex", alignItems: "center", gap: 8,
                padding: "6px 10px", borderRadius: 6,
                background: "var(--color-surface-raised)",
                border: "1px solid var(--color-border)",
              }}>
                <kbd style={{
                  display: "inline-flex", alignItems: "center", justifyContent: "center",
                  minWidth: 24, height: 24, borderRadius: 4, padding: "0 4px",
                  background: "var(--color-surface)", border: "1px solid var(--color-border)",
                  fontSize: 13, fontWeight: 700,
                }}>
                  {digit}
                </kbd>
                <span style={{ fontSize: 13, color: isDefault ? "var(--color-text-muted)" : "var(--color-text)" }}>
                  {label}
                  {isDefault && <span style={{ fontSize: 10, marginLeft: 4 }}>(default)</span>}
                </span>
              </div>
            );
          })}
        </div>
      </section>

      <hr style={{ margin: "24px 0", border: "none", borderTop: "1px solid var(--color-border)" }} />

      {/* ═══════════ PROFILES ═══════════ */}
      <section className="settings-section">
        <h3 className="settings-section__title">
          <Sliders size={16} style={{ marginRight: 6 }} />
          Harness Profiles
        </h3>
        <p className="settings-section__desc">
          Named bundles of model, effort, and approval settings. Define in{" "}
          <code>[profiles.&lt;name&gt;]</code> blocks in{" "}
          <code>{s.configPath || "reasonix.toml"}</code>. Switch via{" "}
          <code>/profile &lt;name&gt;</code> in chat.
        </p>
        {profiles.length === 0 ? (
          <div style={{
            marginTop: 12, padding: "16px", borderRadius: 8,
            background: "var(--color-surface-raised)", border: "1px dashed var(--color-border)",
            textAlign: "center",
          }}>
            <p style={{ color: "var(--color-text-muted)", fontStyle: "italic", margin: 0 }}>
              No profiles configured.
            </p>
            <pre style={{
              marginTop: 8, fontSize: 12, color: "var(--color-text-muted)",
              background: "var(--color-surface)", padding: "8px 12px", borderRadius: 4,
              textAlign: "left",
            }}>
{`[profiles.code-review]
description = "Deep review with high effort"
model = "deepseek-pro"
effort = "high"
tool_approve_mode = "ask"
auto_plan = "on"

[profiles.quick-fix]
description = "Fast edits with auto-approve"
model = "deepseek-flash"
effort = "medium"
tool_approve_mode = "yolo"`}
            </pre>
          </div>
        ) : (
          <div style={{ marginTop: 12, display: "flex", flexDirection: "column", gap: 8 }}>
            {profiles.map(([name, p]) => {
              const isActive = name === activeProfile;
              return (
                <div
                  key={name}
                  style={{
                    display: "flex", alignItems: "center", justifyContent: "space-between",
                    padding: "10px 14px", borderRadius: 8,
                    background: isActive ? "var(--color-accent-subtle)" : "var(--color-surface-raised)",
                    border: `1px solid ${isActive ? "var(--color-accent)" : "var(--color-border)"}`,
                  }}
                >
                  <div>
                    <div style={{ fontWeight: 600, display: "flex", alignItems: "center", gap: 8 }}>
                      {name}
                      {isActive && (
                        <span style={{
                          fontSize: 11, padding: "1px 6px", borderRadius: 4,
                          background: "var(--color-accent)", color: "#fff", fontWeight: 600,
                        }}>
                          active
                        </span>
                      )}
                    </div>
                    <div style={{ fontSize: 13, color: "var(--color-text-muted)", marginTop: 2 }}>
                      {p.description || "—"}
                    </div>
                    <div style={{ fontSize: 12, color: "var(--color-text-muted)", marginTop: 4, display: "flex", gap: 12, flexWrap: "wrap" }}>
                      {p.model && <span>model: {p.model}</span>}
                      {p.effort && <span>effort: {p.effort}</span>}
                      {p.toolApproveMode && <span>mode: {p.toolApproveMode}</span>}
                      {p.autoPlan && <span>auto-plan: {p.autoPlan}</span>}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </section>

      <hr style={{ margin: "24px 0", border: "none", borderTop: "1px solid var(--color-border)" }} />

      {/* ═══════════ SKILLS HUB ═══════════ */}
      <section className="settings-section">
        <h3 className="settings-section__title">
          <BookOpen size={16} style={{ marginRight: 6 }} />
          Skills Hub
        </h3>
        <p className="settings-section__desc">
          Browse our 17-skill community registry. Install via the command line or
          use <code>/install</code> in chat.
        </p>
        <SkillsHubBrowser />
      </section>

      <hr style={{ margin: "24px 0", border: "none", borderTop: "1px solid var(--color-border)" }} />

      {/* ═══════════ SUB-AGENT TREE ═══════════ */}
      <section className="settings-section">
        <h3 className="settings-section__title">
          <GitBranch size={16} style={{ marginRight: 6 }} />
          Sub-Agent Tasks
        </h3>
        <p className="settings-section__desc">
          Sub-agent tasks spawned by the model via the <code>task</code> tool or
          subagent skills. Shows active and completed sub-agents for this session.
        </p>
        <SubagentTreePanel />
      </section>

      <hr style={{ margin: "24px 0", border: "none", borderTop: "1px solid var(--color-border)" }} />

      {/* ═══════════ CONSTITUTION ═══════════ */}
      <section className="settings-section">
        <h3 className="settings-section__title">
          <Shield size={16} style={{ marginRight: 6 }} />
          Constitution Health
        </h3>
        <p className="settings-section__desc">
          Reads <code>.reasonix/constitution.json</code> from the project root.
          Shows principles, hard constraints, and code-level rules with severity.
        </p>
        <ConstitutionHealthPanel />
      </section>

      <hr style={{ margin: "24px 0", border: "none", borderTop: "1px solid var(--color-border)" }} />

      {/* ═══════════ BRANDING ═══════════ */}
      <section className="settings-section">
        <h3 className="settings-section__title">
          <Zap size={16} style={{ marginRight: 6 }} />
          About Reasonix-Hermes
        </h3>
        <p className="settings-section__desc">
          Extended fork of{" "}
          <a href="https://github.com/esengine/deepseek-reasonix" target="_blank" rel="noreferrer">
            esengine/deepseek-reasonix
          </a>{" "}
          (v1.6.0). Adds Discord bot, MCP bridge server, Hindsight memory, curated
          skills hub, native hook runner, portable mode, and desktop hotbar +
          harness profiles.
        </p>
        <div style={{ marginTop: 12, fontSize: 13, color: "var(--color-text-muted)" }}>
          <div>
            <FileText size={12} style={{ marginRight: 4, verticalAlign: "middle" }} />
            Config: <code>{s.configPath || "reasonix.toml"}</code>
          </div>
        </div>
      </section>
    </div>
  );
}
