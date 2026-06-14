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

/** MCP — live installed MCP servers overview. */
function MCPTab() {
  const [servers, setServers] = useState<Array<{
    name: string; transport: string; status: string; builtIn?: boolean;
    tools: number; prompts: number; resources: number; error?: string;
    url?: string; command?: string;
  }> | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    const fetch = () => {
      app.Capabilities().then((cap) => {
        if (cancelled) return;
        setServers(cap.servers ?? []);
        setLoading(false);
      }).catch(() => {
        if (cancelled) return;
        setLoading(false);
      });
    };
    fetch();
    const id = setInterval(fetch, 15000);
    return () => { cancelled = true; clearInterval(id); };
  }, []);

  const statusColor = (s: string): string => {
    switch (s) {
      case "connected": return "var(--color-green)";
      case "failed": return "var(--color-warn)";
      case "initializing": return "var(--color-yellow)";
      default: return "var(--color-text-muted)";
    }
  };

  return (
    <div style={{ padding: "12px 0" }}>
      <p style={{ fontSize: 12, color: "var(--color-text-muted)", margin: "0 0 12px" }}>
        MCP servers extend Reasonix with external tools, data sources, and integrations.
        Manage installed servers in <strong>Settings → MCP</strong>.
      </p>
      {loading ? (
        <div style={{ padding: 16, textAlign: "center", color: "var(--color-text-muted)", fontSize: 12 }}>Loading…</div>
      ) : servers && servers.length > 0 ? (
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          {servers.map((srv) => (
            <div
              key={srv.name}
              style={{
                padding: "10px 12px", borderRadius: 8,
                background: "var(--color-surface-raised)",
                border: "1px solid var(--color-border)",
                display: "flex", alignItems: "center", gap: 10,
              }}
            >
              <span
                style={{
                  width: 8, height: 8, borderRadius: "50%", flexShrink: 0,
                  background: statusColor(srv.status),
                }}
                title={srv.status + (srv.error ? ": " + srv.error : "")}
              />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 12, fontWeight: 600, display: "flex", alignItems: "center", gap: 6 }}>
                  {srv.name}
                  {srv.builtIn && (
                    <span style={{ fontSize: 9, padding: "0 4px", borderRadius: 3, background: "var(--color-accent)", color: "#fff" }}>built-in</span>
                  )}
                </div>
                <div style={{ fontSize: 10, color: "var(--color-text-muted)", marginTop: 2 }}>
                  {srv.transport}
                  {srv.url && ` · ${srv.url.replace(/\/$/, "")}`}
                  {srv.command && ` · ${srv.command}`}
                </div>
              </div>
              <div style={{ display: "flex", gap: 6, flexShrink: 0 }}>
                {srv.tools > 0 && (
                  <span style={{ fontSize: 10, color: "var(--color-text-muted)", background: "var(--color-surface)", padding: "1px 5px", borderRadius: 4 }}>
                    {srv.tools} tools
                  </span>
                )}
                {srv.prompts > 0 && (
                  <span style={{ fontSize: 10, color: "var(--color-text-muted)", background: "var(--color-surface)", padding: "1px 5px", borderRadius: 4 }}>
                    {srv.prompts} prompts
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div style={{ padding: 16, textAlign: "center", color: "var(--color-text-muted)", fontSize: 12 }}>
          No MCP servers installed. Add one in <strong>Settings → MCP</strong>.
        </div>
      )}
      <p style={{ margin: "12px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
        Connect additional servers via <code>Settings → MCP → Add Server</code> or{" "}
        <code>reasonix install-source install --kind mcp ...</code>
      </p>
    </div>
  );
}
