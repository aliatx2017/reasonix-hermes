import { useState, useEffect, useRef } from "react";
import { Network, Tag, GitGraph } from "lucide-react";
import { app } from "../../lib/bridge";
import * as d3 from "d3";

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

// --- D3 Force-Directed Graph ---

interface GraphNode extends d3.SimulationNodeDatum {
  id: number;
  title: string;
  type: string;
  description: string;
}

function MemoryForceGraph({ facts }: { facts: MemoryFactView[] }) {
  const svgRef = useRef<SVGSVGElement | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!svgRef.current || !containerRef.current || facts.length === 0) return;

    const container = containerRef.current;
    const width = container.clientWidth;
    const height = Math.max(300, container.clientHeight);

    // Build nodes.
    const nodes: GraphNode[] = facts.map((f, i) => ({
      id: i,
      title: f.title,
      type: f.type.split(":")[0] || f.type,
      description: f.description,
    }));

    // Build links between nodes of the same type (cluster edges).
    const links: { source: number; target: number }[] = [];
    const typeGroups: Record<string, number[]> = {};
    nodes.forEach((n) => {
      if (!typeGroups[n.type]) typeGroups[n.type] = [];
      typeGroups[n.type].push(n.id);
    });
    // Link consecutive nodes within each type group to create cluster cohesion.
    for (const ids of Object.values(typeGroups)) {
      for (let i = 0; i < ids.length - 1; i++) {
        links.push({ source: ids[i], target: ids[i + 1] });
      }
    }

    const svg = d3.select(svgRef.current);
    svg.selectAll("*").remove();
    svg.attr("viewBox", `0 0 ${width} ${height}`);

    const color = (t: string) => TYPE_COLORS[t] || "var(--color-text-muted)";

    // Arrowhead marker.
    svg.append("defs").selectAll("marker")
      .data(["arrow"])
      .join("marker")
      .attr("id", "arrow")
      .attr("viewBox", "0 -5 10 10")
      .attr("refX", 24)
      .attr("refY", 0)
      .attr("markerWidth", 6)
      .attr("markerHeight", 6)
      .attr("orient", "auto")
      .append("path")
      .attr("fill", "var(--color-border)")
      .attr("d", "M0,-5L10,0L0,5");

    // Links.
    const link = svg.append("g")
      .selectAll("line")
      .data(links)
      .join("line")
      .attr("stroke", "var(--color-border)")
      .attr("stroke-width", 1)
      .attr("stroke-opacity", 0.4);

    // Nodes.
    const node = svg.append("g")
      .selectAll("g")
      .data(nodes)
      .join("g")
      .attr("cursor", "pointer") as d3.Selection<SVGGElement, GraphNode, SVGGElement, unknown>;

    node.call(d3.drag<SVGGElement, GraphNode>()
        .on("start", (event, d) => {
          if (!event.active) simulation.alphaTarget(0.3).restart();
          d.fx = d.x;
          d.fy = d.y;
        })
        .on("drag", (event, d) => {
          d.fx = event.x;
          d.fy = event.y;
        })
        .on("end", (event, d) => {
          if (!event.active) simulation.alphaTarget(0);
          d.fx = null;
          d.fy = null;
        }));

    node.append("circle")
      .attr("r", (d) => Math.max(8, Math.min(18, d.title.length * 1.2)))
      .attr("fill", (d) => color(d.type))
      .attr("fill-opacity", 0.85)
      .attr("stroke", (d) => color(d.type))
      .attr("stroke-width", 1.5);

    node.append("title")
      .text((d) => `${d.title}\n${d.description}`);

    node.append("text")
      .attr("text-anchor", "middle")
      .attr("dy", "0.35em")
      .attr("font-size", 8)
      .attr("fill", "#fff")
      .attr("pointer-events", "none")
      .text((d) => d.title.length > 12 ? d.title.slice(0, 11) + "…" : d.title);

    // Zoom.
    const zoom = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.3, 4])
      .on("zoom", (event) => {
        svg.selectAll("g").attr("transform", event.transform);
      });
    svg.call(zoom);

    // Force simulation.
    const simulation = d3.forceSimulation<GraphNode>(nodes)
      .force("link", d3.forceLink<GraphNode, { source: number; target: number }>(links)
        .id((d) => d.id)
        .distance(60)
        .strength(0.3))
      .force("charge", d3.forceManyBody().strength(-120))
      .force("center", d3.forceCenter(width / 2, height / 2))
      .force("collide", d3.forceCollide(22));

    simulation.on("tick", () => {
      link
        .attr("x1", (d: any) => d.source.x)
        .attr("y1", (d: any) => d.source.y)
        .attr("x2", (d: any) => d.target.x)
        .attr("y2", (d: any) => d.target.y);

      node.attr("transform", (d) => `translate(${d.x},${d.y})`);
    });

    return () => { simulation.stop(); };
  }, [facts]);

  return (
    <div ref={containerRef} style={{ width: "100%", height: "100%", minHeight: 300 }}>
      <svg ref={svgRef} style={{ width: "100%", height: "100%" }} />
    </div>
  );
}

// --- Main Component ---

export function MemoryFactGraph() {
  const [facts, setFacts] = useState<MemoryFactView[]>([]);
  const [viewMode, setViewMode] = useState<"badges" | "graph">("badges");

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

  // Group by type for the badges view.
  const groups: Record<string, MemoryFactView[]> = {};
  for (const f of facts) {
    const baseType = f.type.split(":")[0] || f.type;
    if (!groups[baseType]) groups[baseType] = [];
    groups[baseType].push(f);
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
      {/* Header + toggle */}
      <div style={{ display: "flex", gap: 12, fontSize: 11, color: "var(--color-text-muted)", alignItems: "center", justifyContent: "space-between" }}>
        <div style={{ display: "flex", gap: 12 }}>
          <span><Network size={11} style={{ marginRight: 3, verticalAlign: -1 }} />{facts.length} facts</span>
          <span><Tag size={11} style={{ marginRight: 3, verticalAlign: -1 }} />{Object.keys(groups).length} types</span>
        </div>
        <button
          onClick={() => setViewMode(viewMode === "badges" ? "graph" : "badges")}
          style={{
            display: "inline-flex", alignItems: "center", gap: 3,
            padding: "2px 6px", fontSize: 10, border: "1px solid var(--color-border)",
            borderRadius: 4, background: "var(--bg)", color: "var(--fg)", cursor: "pointer",
          }}
        >
          <GitGraph size={10} />
          {viewMode === "badges" ? "Graph" : "Badges"}
        </button>
      </div>

      {viewMode === "graph" ? (
        <MemoryForceGraph facts={facts} />
      ) : (
        /* Clustered fact display (badges) */
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
      )}
    </div>
  );
}
