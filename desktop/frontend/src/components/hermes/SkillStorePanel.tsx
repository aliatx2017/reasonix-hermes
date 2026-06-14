import { useState, useEffect } from "react";
import { Globe, ShoppingBag, Server, BookOpen } from "lucide-react";
import { SkillsHubBrowser } from "./SkillsHubBrowser";
import { MarketplacePanel } from "./MarketplacePanel";
import { app } from "../../lib/bridge";

type TabId = "lobehub" | "market" | "mcp" | "custom";

interface TabDef {
  id: TabId;
  label: string;
  icon: React.ReactNode;
}

const TABS: TabDef[] = [
  { id: "lobehub", label: "LobeHub", icon: <Globe size={14} /> },
  { id: "market", label: "Market", icon: <ShoppingBag size={14} /> },
  { id: "mcp", label: "MCP", icon: <Server size={14} /> },
  { id: "custom", label: "Custom", icon: <BookOpen size={14} /> },
];

export function SkillStorePanel() {
  const [activeTab, setActiveTab] = useState<TabId>("market");

  return (
    <div className="hermes-panel" style={{ padding: "4px 0" }}>
      {/* Tab bar */}
      <div style={{ display: "flex", gap: 2, marginBottom: 8, borderBottom: "1px solid var(--color-border)", paddingBottom: 0 }}>
        {TABS.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            style={{
              display: "flex", alignItems: "center", gap: 5,
              padding: "6px 12px", fontSize: 12, fontWeight: 500,
              border: "none", borderBottom: activeTab === tab.id ? "2px solid var(--color-accent)" : "2px solid transparent",
              background: "transparent",
              color: activeTab === tab.id ? "var(--color-accent)" : "var(--color-text-muted)",
              cursor: "pointer",
              borderRadius: "4px 4px 0 0",
            }}
          >
            {tab.icon}
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div>
        {activeTab === "lobehub" && <LobeHubTab />}
        {activeTab === "market" && <MarketplacePanel />}
        {activeTab === "mcp" && <MCPTab />}
        {activeTab === "custom" && <SkillsHubBrowser />}
      </div>
    </div>
  );
}

/** LobeHub — the live sync/search interface from the marketplace. */
function LobeHubTab() {
  const [lastSync, setLastSync] = useState<{ lastSync: string; fetched: number; added: number } | null>(null);

  useEffect(() => {
    app.LastLobeHubSync().then((s) => {
      if (s.lastSync) setLastSync(s);
    }).catch(() => {});
  }, []);

  return (
    <div>
      <p style={{ fontSize: 12, color: "var(--color-text-muted)", margin: "0 0 12px" }}>
        Browse and sync skills from the <strong>LobeHub</strong> community marketplace
        (360,000+ community skills). Use the sync button below to pull the latest skills.
      </p>
      {lastSync && (
        <div style={{
          padding: "6px 10px", marginBottom: 10, borderRadius: 6, fontSize: 11,
          background: "var(--color-accent-subtle)", color: "var(--color-accent)",
        }}>
          Last synced: {new Date(lastSync.lastSync).toLocaleDateString()} — {lastSync.fetched} skills fetched, {lastSync.added} new.
        </div>
      )}
      <MarketplacePanel />
    </div>
  );
}

/** MCP — installed MCP servers overview. */
function MCPTab() {
  return (
    <div style={{ padding: "12px 0" }}>
      <p style={{ fontSize: 12, color: "var(--color-text-muted)", margin: "0 0 12px" }}>
        MCP servers extend Reasonix with external tools, data sources, and integrations.
        Manage installed servers in <strong>Settings → MCP</strong>.
      </p>
      <div
        style={{
          padding: 16, borderRadius: 8,
          background: "var(--color-surface-raised)",
          border: "1px solid var(--color-border)",
        }}
      >
        <h4 style={{ margin: "0 0 8px", fontSize: 13, fontWeight: 600 }}>
          <Server size={14} style={{ marginRight: 4, verticalAlign: "middle" }} />
          Built-in MCP Servers
        </h4>
        <ul style={{ margin: 0, paddingLeft: 18, fontSize: 12, color: "var(--color-text-muted)", lineHeight: 1.8 }}>
          <li><strong>codegraph</strong> — code intelligence: search, context, trace</li>
          <li><strong>time</strong> — current time and timezone utilities</li>
          <li><strong>context7</strong> — up-to-date documentation lookup</li>
          <li><strong>memory</strong> — persistent cross-session memory (Hindsight)</li>
        </ul>
        <p style={{ margin: "12px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
          Connect additional servers via <code>Settings → MCP → Add Server</code> or
          <code> reasonix install-source install --kind mcp ...</code>
        </p>
      </div>
    </div>
  );
}
