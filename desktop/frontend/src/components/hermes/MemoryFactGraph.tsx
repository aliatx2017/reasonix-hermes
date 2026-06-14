import { useState, useEffect, useRef } from "react";
import { Network, Tag, GitGraph, X } from "lucide-react";
import { app } from "../../lib/bridge";
import * as d3 from "d3";

interface MemoryFactView {
  title: string;
  type: string;
  description: string;
  hasDenseEmbedding: boolean;
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

const ALL_TYPES = Object.keys(TYPE_COLORS);

function typeColor(kind: string): string {
  for (const [k, v] of Object.entries(TYPE_COLORS)) {
    if (kind.startsWith(k)) return v;
  }
  return "var(--color-text-muted)";
}

// --- TF-IDF cosine similarity for vector links ---

function tokenize(text: string): string[] {
  return text.toLowerCase().replace(/[^a-z0-9\s]/g, "").split(/\s+/).filter((w) => w.length > 2);
}

function computeTFIDF(docs: string[][]): number[][] {
  // Document frequency.
  const df: Record<string, number> = {};
  for (const doc of docs) {
    const seen = new Set(doc);
    for (const w of seen) df[w] = (df[w] || 0) + 1;
  }
  const N = docs.length;

  // TF-IDF vectors.
  const vocab = Object.keys(df);
  return docs.map((doc) => {
    const tf: Record<string, number> = {};
    for (const w of doc) tf[w] = (tf[w] || 0) + 1;
    return vocab.map((w) => (tf[w] || 0) * Math.log((N + 1) / (df[w] + 1)));
  });
}

function cosineSimilarity(a: number[], b: number[]): number {
  let dot = 0, normA = 0, normB = 0;
  for (let i = 0; i < a.length; i++) {
    dot += a[i] * b[i];
    normA += a[i] * a[i];
    normB += b[i] * b[i];
  }
  if (normA === 0 || normB === 0) return 0;
  return dot / (Math.sqrt(normA) * Math.sqrt(normB));
}

// --- D3 Force-Directed Graph ---

interface GraphNode extends d3.SimulationNodeDatum {
  id: number;
  title: string;
  type: string;
  description: string;
}

function MemoryForceGraph({
  facts,
  onSelect,
  selectedId,
}: {
  facts: MemoryFactView[];
  onSelect: (fact: MemoryFactView | null) => void;
  selectedId: number | null;
}) {
  const svgRef = useRef<SVGSVGElement | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!svgRef.current || !containerRef.current || facts.length === 0) return;

    const container = containerRef.current;
    const width = container.clientWidth;
    const height = Math.max(300, container.clientHeight);

    const nodes: GraphNode[] = facts.map((f, i) => ({
      id: i,
      title: f.title,
      type: f.type.split(":")[0] || f.type,
      description: f.description,
    }));

    // Compute TF-IDF vectors for similarity edges.
    const docs = nodes.map((n) => tokenize(n.title + " " + n.description));
    const vectors = computeTFIDF(docs);

    // Build links: intra-type cohesion + high cosine similarity.
    const links: { source: number; target: number; similar: boolean }[] = [];
    const typeGroups: Record<string, number[]> = {};
    nodes.forEach((n) => {
      if (!typeGroups[n.type]) typeGroups[n.type] = [];
      typeGroups[n.type].push(n.id);
    });
    for (const ids of Object.values(typeGroups)) {
      for (let i = 0; i < ids.length - 1; i++) {
        links.push({ source: ids[i], target: ids[i + 1], similar: false });
      }
    }
    // Add high-similarity cross-type edges.
    for (let i = 0; i < nodes.length; i++) {
      for (let j = i + 1; j < nodes.length; j++) {
        if (nodes[i].type === nodes[j].type) continue;
        const sim = cosineSimilarity(vectors[i], vectors[j]);
        if (sim > 0.3) {
          links.push({ source: i, target: j, similar: true });
        }
      }
    }

    const svg = d3.select(svgRef.current);
    svg.selectAll("*").remove();
    svg.attr("viewBox", `0 0 ${width} ${height}`);

    const color = (t: string) => TYPE_COLORS[t] || "var(--color-text-muted)";

    const link = svg.append("g")
      .selectAll("line")
      .data(links)
      .join("line")
      .attr("stroke", (d) => d.similar ? "var(--color-accent)" : "var(--color-border)")
      .attr("stroke-width", (d) => d.similar ? 1.5 : 1)
      .attr("stroke-opacity", (d) => d.similar ? 0.5 : 0.35)
      .attr("stroke-dasharray", (d) => d.similar ? "4 2" : "none");

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

    node.on("click", (_event, d) => {
      onSelect(facts[d.id]);
    });

    node.append("circle")
      .attr("r", (d) => Math.max(8, Math.min(18, d.title.length * 1.2)))
      .attr("fill", (d) => color(d.type))
      .attr("fill-opacity", 0.85)
      .attr("stroke", (d) => d.id === selectedId ? "#fff" : color(d.type))
      .attr("stroke-width", (d) => d.id === selectedId ? 2.5 : 1.5);

    node.append("title")
      .text((d) => `${d.title}\n${d.description}`);

    node.append("text")
      .attr("text-anchor", "middle")
      .attr("dy", "0.35em")
      .attr("font-size", 8)
      .attr("fill", "#fff")
      .attr("pointer-events", "none")
      .text((d) => d.title.length > 12 ? d.title.slice(0, 11) + "…" : d.title);

    const zoom = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.3, 4])
      .on("zoom", (event) => {
        svg.selectAll("g").attr("transform", event.transform);
      });
    svg.call(zoom);

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
  }, [facts, selectedId]);

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
  const [typeFilter, setTypeFilter] = useState<Set<string>>(new Set(ALL_TYPES));
  const [selectedFact, setSelectedFact] = useState<MemoryFactView | null>(null);

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

  // Apply type filter.
  const filteredFacts = facts.filter((f) => {
    const base = f.type.split(":")[0] || f.type;
    return typeFilter.has(base);
  });

  if (facts.length === 0) {
    return (
      <div style={{ fontSize: 13, color: "var(--color-text-muted)", fontStyle: "italic", padding: "8px 0" }}>
        No memory facts yet. Facts are saved via <code>remember</code>, <code>#</code> quick-add, or auto-memory.
      </div>
    );
  }

  // Group by type for the badges view.
  const groups: Record<string, MemoryFactView[]> = {};
  for (const f of filteredFacts) {
    const baseType = f.type.split(":")[0] || f.type;
    if (!groups[baseType]) groups[baseType] = [];
    groups[baseType].push(f);
  }

  const toggleType = (t: string) => {
    const next = new Set(typeFilter);
    if (next.has(t)) next.delete(t); else next.add(t);
    setTypeFilter(next);
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
      {/* Header + toggle */}
      <div style={{ display: "flex", gap: 12, fontSize: 11, color: "var(--color-text-muted)", alignItems: "center", justifyContent: "space-between" }}>
        <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
          <span><Network size={11} style={{ marginRight: 3, verticalAlign: -1 }} />{filteredFacts.length} facts</span>
          <span><Tag size={11} style={{ marginRight: 3, verticalAlign: -1 }} />{Object.keys(groups).length} types</span>
          {/* Type filter chips */}
          {viewMode === "graph" && ALL_TYPES.map((t) => (
            <button
              key={t}
              onClick={() => toggleType(t)}
              style={{
                padding: "1px 6px", borderRadius: 3, fontSize: 10, border: "1px solid",
                borderColor: typeFilter.has(t) ? typeColor(t) : "var(--color-border)",
                background: typeFilter.has(t) ? `${typeColor(t)}20` : "transparent",
                color: typeFilter.has(t) ? typeColor(t) : "var(--color-text-muted)",
                cursor: "pointer", opacity: typeFilter.has(t) ? 1 : 0.5,
              }}
            >
              {t}
            </button>
          ))}
        </div>
        <button
          onClick={() => { setViewMode(viewMode === "badges" ? "graph" : "badges"); setSelectedFact(null); }}
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
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          <MemoryForceGraph
            facts={filteredFacts}
            onSelect={setSelectedFact}
            selectedId={filteredFacts.findIndex((f) => f === selectedFact)}
          />
          {/* Selected fact detail panel */}
          {selectedFact && (
            <div style={{
              display: "flex", gap: 8, padding: "8px 10px",
              background: "var(--bg-soft)", borderRadius: 6,
              border: `1px solid ${typeColor(selectedFact.type)}40`,
              fontSize: 12, position: "relative",
            }}>
              <button
                onClick={() => setSelectedFact(null)}
                style={{
                  position: "absolute", top: 4, right: 4, background: "none", border: "none",
                  cursor: "pointer", color: "var(--color-text-muted)", padding: 0,
                }}
              >
                <X size={12} />
              </button>
              <div style={{ width: 8, height: 8, borderRadius: "50%", background: typeColor(selectedFact.type), marginTop: 3, flexShrink: 0 }} />
              <div>
                <div style={{ fontWeight: 600, color: typeColor(selectedFact.type) }}>{selectedFact.title}</div>
                <div style={{ fontSize: 11, color: "var(--color-text-muted)", marginTop: 2 }}>{selectedFact.description}</div>
                <div style={{ fontSize: 10, color: "var(--color-text-3)", marginTop: 2 }}>type: {selectedFact.type}</div>
              </div>
            </div>
          )}
        </div>
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
