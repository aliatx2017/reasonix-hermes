import { useState, useEffect, useCallback, useRef } from "react";
import { FileText, Save, Plus, RefreshCw, Eye, Edit3, X, Wand2, Brain } from "lucide-react";
import { app } from "../lib/bridge";
import type { MemoryFactView } from "../lib/types";

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

interface MarkdownFileEntry {
  name: string;
  path: string;
  relPath: string;
  size: number;
}

// Simple markdown-to-HTML renderer (headers, bold, italic, code, links, lists).
function renderMarkdown(md: string): string {
  if (!md) return "";
  let html = md
    .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")
    .replace(/^### (.+)$/gm, "<h3>$1</h3>")
    .replace(/^## (.+)$/gm, "<h2>$1</h2>")
    .replace(/^# (.+)$/gm, "<h1>$1</h1>")
    .replace(/\*\*\*(.+?)\*\*\*/g, "<strong><em>$1</em></strong>")
    .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
    .replace(/\*(.+?)\*/g, "<em>$1</em>")
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener">$1</a>')
    .replace(/^- (.+)$/gm, "<li>$1</li>")
    .replace(/\n\n/g, "</p><p>")
    .replace(/\n/g, "<br>");
  html = "<p>" + html + "</p>";
  html = html.replace(/((?:<li>.*?<\/li><br>?)+)/g, (match) => {
    const items = match.replace(/<br>/g, "");
    return "<ul>" + items + "</ul>";
  });
  return html;
}

// --- CodeMirror editor wrapper ---

import { EditorView, keymap, placeholder as cmPlaceholder } from "@codemirror/view";
import { EditorState, type Extension } from "@codemirror/state";
import { markdown, markdownLanguage } from "@codemirror/lang-markdown";
import { defaultKeymap, history, historyKeymap, indentWithTab } from "@codemirror/commands";
import { autocompletion, type CompletionContext, type CompletionResult } from "@codemirror/autocomplete";
import { syntaxHighlighting, defaultHighlightStyle } from "@codemirror/language";

function fimCompletionSource(context: CompletionContext, relPath: string, _getContent: () => string): Promise<CompletionResult | null> | null {
  // Only trigger on explicit keybind (Ctrl+Space), not auto.
  if (!context.explicit) return null;
  const pos = context.pos;
  // Only trigger if there's at least some prefix context.
  if (pos < 10) return null;

  return app.FIMComplete(relPath, pos).then((result) => {
    if (!result?.text) return null;
    const text = result.text;
    return {
      from: pos,
      options: [{
        label: text.length > 60 ? text.slice(0, 60) + "…" : text,
        detail: "FIM",
        apply: text,
      }],
    };
  }).catch(() => null);
}

interface CodeMirrorEditorProps {
  initialValue: string;
  relPath: string;
  onChange: (value: string) => void;
  onSave: () => void;
}

function CodeMirrorEditor({ initialValue, relPath, onChange, onSave }: CodeMirrorEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const contentRef = useRef(initialValue);

  // Keep a ref to the latest relPath for the completion source.
  const relPathRef = useRef(relPath);
  relPathRef.current = relPath;

  useEffect(() => {
    if (!containerRef.current) return;

    const updateListener = EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        const newValue = update.state.doc.toString();
        contentRef.current = newValue;
        onChange(newValue);
      }
    });

    const extensions: Extension[] = [
      markdown({ base: markdownLanguage }),
      syntaxHighlighting(defaultHighlightStyle),
      history(),
      keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
      cmPlaceholder("Start writing…"),
      updateListener,
      autocompletion({
        override: [(ctx) => fimCompletionSource(ctx, relPathRef.current, () => contentRef.current)],
      }),
      EditorView.lineWrapping,
      EditorView.theme({
        "&": { height: "100%", fontSize: "13px" },
        ".cm-scroller": { overflow: "auto", fontFamily: "var(--mono, monospace)", lineHeight: "1.6" },
        ".cm-content": { padding: "16px", caretColor: "var(--fg)" },
        ".cm-cursor": { borderLeftColor: "var(--fg)" },
        ".cm-activeLine": { background: "var(--bg-soft)" },
        ".cm-selectionBackground": { background: "var(--sidebar-active)" },
        ".cm-gutters": { display: "none" },
      }),
      // Ctrl+S save
      keymap.of([{
        key: "Mod-s",
        run: () => { onSave(); return true; },
        preventDefault: true,
      }]),
    ];

    const state = EditorState.create({
      doc: initialValue,
      extensions,
    });

    const view = new EditorView({ state, parent: containerRef.current });
    viewRef.current = view;

    return () => { view.destroy(); viewRef.current = null; };
  }, []); // mount once; content syncs via initialValue handled by doc

  // Sync initialValue into the editor when a new file is opened.
  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;
    const current = view.state.doc.toString();
    if (current !== initialValue) {
      view.dispatch({
        changes: { from: 0, to: current.length, insert: initialValue },
      });
      contentRef.current = initialValue;
    }
  }, [initialValue]);

  return <div ref={containerRef} style={{ height: "100%", overflow: "hidden" }} />;
}

export function WriteMode() {
  const [files, setFiles] = useState<MarkdownFileEntry[]>([]);
  const [openFiles, setOpenFiles] = useState<{relPath: string; name: string; content: string; dirty: boolean}[]>([]);
  const [activeFileIdx, setActiveFileIdx] = useState(-1);
  const selectedFile = activeFileIdx >= 0 ? openFiles[activeFileIdx]?.relPath : null;
  const content = activeFileIdx >= 0 ? openFiles[activeFileIdx]?.content ?? "" : "";
  const dirty = activeFileIdx >= 0 ? openFiles[activeFileIdx]?.dirty ?? false : false;
  const fileName = activeFileIdx >= 0 ? openFiles[activeFileIdx]?.name ?? "" : "";
  const [viewMode, setViewMode] = useState<"edit" | "split" | "preview">("edit");
  const [newFileName, setNewFileName] = useState("");
  const [showNewFile, setShowNewFile] = useState(false);
  const [fimBusy, setFimBusy] = useState(false);
  const [memoryOpen, setMemoryOpen] = useState(false);
  const [memoryFacts, setMemoryFacts] = useState<MemoryFactView[]>([]);

  const loadFiles = useCallback(async () => {
    try {
      const list = await app.ListMarkdownFiles();
      setFiles(list ?? []);
    } catch { /* ignore */ }
  }, []);

  const openFile = useCallback(async (relPath: string) => {
    // Check if already open.
    setOpenFiles((prev) => {
      const existing = prev.findIndex((f) => f.relPath === relPath);
      if (existing >= 0) { setActiveFileIdx(existing); return prev; }
      return prev;
    });
    // Load and add if not already open.
    try {
      const data = await app.ReadMarkdownFile(relPath);
      if (data) {
        setOpenFiles((prev) => {
          const already = prev.findIndex((f) => f.relPath === relPath);
          if (already >= 0) return prev;
          const next = [...prev, { relPath, name: data.name, content: data.content, dirty: false }];
          setActiveFileIdx(next.length - 1);
          return next;
        });
      }
    } catch { /* ignore */ }
  }, []);

  const closeFile = useCallback((idx: number) => {
    setOpenFiles((prev) => {
      const next = prev.filter((_, i) => i !== idx);
      if (next.length === 0) setActiveFileIdx(-1);
      else if (idx >= next.length) setActiveFileIdx(next.length - 1);
      else setActiveFileIdx(Math.max(0, idx));
      return next;
    });
  }, []);

  const saveFile = useCallback(async () => {
    const file = openFiles[activeFileIdx];
    if (!file) return;
    try {
      await app.SaveMarkdownFile(file.relPath, file.content);
      setOpenFiles((prev) => prev.map((f, i) => i === activeFileIdx ? { ...f, dirty: false } : f));
    } catch { /* ignore */ }
  }, [openFiles, activeFileIdx]);

  // Auto-save: debounced 2s after last edit.
  const autoSaveTimer = useRef<number>(0);
  const handleContentChange = useCallback((value: string) => {
    setOpenFiles((prev) => prev.map((f, i) => i === activeFileIdx ? { ...f, content: value, dirty: true } : f));
    if (autoSaveTimer.current) clearTimeout(autoSaveTimer.current);
    autoSaveTimer.current = setTimeout(() => {
      setOpenFiles((prev) => {
        const file = prev[activeFileIdx];
        if (file?.relPath) {
          app.SaveMarkdownFile(file.relPath, value).then(() => {
            setOpenFiles((p) => p.map((f, i) => i === activeFileIdx ? { ...f, dirty: false } : f));
          }).catch(() => {});
        }
        return prev;
      });
    }, 2000);
  }, [activeFileIdx]);

  const createFile = useCallback(async () => {
    if (!newFileName.trim()) return;
    try {
      const relPath = await app.CreateMarkdownFile(newFileName.trim(), "# " + newFileName.trim() + "\n\n");
      if (relPath) {
        setShowNewFile(false);
        setNewFileName("");
        await loadFiles();
        await openFile(relPath);
      }
    } catch { /* ignore */ }
  }, [newFileName, loadFiles, openFile]);

  useEffect(() => { loadFiles(); }, [loadFiles]);

  // Load memory facts filtered by relevance to the current file.
  useEffect(() => {
    if (!selectedFile) { setMemoryFacts([]); return; }
    let cancelled = false;
    app.MemoryFacts().then((all) => {
      if (cancelled || !all) return;
      // Score facts by keyword overlap with file name + content snippet.
      const fileWords = new Set(
        (selectedFile.replace(/[^a-zA-Z0-9]/g, " ").toLowerCase() + " " +
         content.slice(0, 500).replace(/[^a-zA-Z0-9]/g, " ").toLowerCase())
          .split(/\s+/).filter((w) => w.length > 2)
      );
      const scored = all.map((f) => {
        const factWords = (f.title + " " + f.description).toLowerCase().split(/\s+/);
        const score = factWords.filter((w) => fileWords.has(w)).length;
        return { fact: f, score };
      });
      scored.sort((a, b) => b.score - a.score);
      // Take top 10 with any overlap, or top 5 if none match.
      const threshold = scored.find((s) => s.score > 0) ? 1 : 0;
      const filtered = scored.filter((s) => s.score >= threshold).slice(0, 10);
      setMemoryFacts(filtered.map((s) => s.fact));
    }).catch(() => {});
    return () => { cancelled = true; };
  }, [selectedFile, content.slice(0, 500)]);

  const handleFIM = useCallback(async () => {
    if (!selectedFile || fimBusy) return;
    setFimBusy(true);
    try {
      // FIM is triggered via CodeMirror autocompletion (Ctrl+Space).
      // This button invokes it programmatically — find the editor view.
      const view = document.querySelector(".cm-editor") as HTMLElement | null;
      if (view) {
        // Dispatch Ctrl+Space to the CodeMirror editor.
        const cmContent = view.querySelector(".cm-content");
        if (cmContent) {
          cmContent.dispatchEvent(new KeyboardEvent("keydown", { key: " ", ctrlKey: true, bubbles: true }));
        }
      }
    } finally {
      setTimeout(() => setFimBusy(false), 500);
    }
  }, [selectedFile, fimBusy]);

  return (
    <div style={{ display: "flex", height: "100%", minHeight: 0 }}>
      {/* File browser sidebar */}
      <div style={{
        width: 200, borderRight: "1px solid var(--color-border)",
        display: "flex", flexDirection: "column", background: "var(--bg-soft)",
      }}>
        <div style={{
          display: "flex", alignItems: "center", justifyContent: "space-between",
          padding: "8px 10px", borderBottom: "1px solid var(--color-border)",
        }}>
          <span style={{ fontSize: 12, fontWeight: 600, color: "var(--fg)" }}>Files</span>
          <div style={{ display: "flex", gap: 4 }}>
            <button onClick={loadFiles} title="Refresh" style={iconBtnStyle}>
              <RefreshCw size={12} />
            </button>
            <button onClick={() => setShowNewFile(!showNewFile)} title="New file" style={iconBtnStyle}>
              <Plus size={12} />
            </button>
          </div>
        </div>

        {showNewFile && (
          <div style={{ padding: "6px 10px", display: "flex", gap: 4, borderBottom: "1px solid var(--color-border)" }}>
            <input
              type="text"
              placeholder="name.md"
              value={newFileName}
              onChange={(e) => setNewFileName(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter") createFile(); }}
              style={{
                flex: 1, fontSize: 11, padding: "2px 6px",
                background: "var(--bg)", border: "1px solid var(--color-border)",
                borderRadius: 3, color: "var(--fg)",
              }}
              autoFocus
            />
            <button onClick={createFile} style={iconBtnStyle}><Plus size={10} /></button>
            <button onClick={() => setShowNewFile(false)} style={iconBtnStyle}><X size={10} /></button>
          </div>
        )}

        <div style={{ flex: 1, overflow: "auto" }}>
          {files.map((f) => (
            <button
              key={f.relPath}
              onClick={() => openFile(f.relPath)}
              style={{
                display: "flex", alignItems: "center", gap: 6, width: "100%",
                padding: "5px 10px", border: "none", background: selectedFile === f.relPath ? "var(--sidebar-active)" : "none",
                cursor: "pointer", fontSize: 11, color: selectedFile === f.relPath ? "var(--accent)" : "var(--fg)",
                textAlign: "left",
              }}
            >
              <FileText size={11} />
              <span style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                {f.relPath}
              </span>
            </button>
          ))}
        </div>
      </div>

      {/* Editor / Preview */}
      <div style={{ flex: 1, display: "flex", flexDirection: "column", minWidth: 0 }}>
        {/* Tab bar */}
        {openFiles.length > 0 && (
          <div style={{
            display: "flex", borderBottom: "1px solid var(--color-border)",
            background: "var(--bg-soft)", overflow: "auto",
          }}>
            {openFiles.map((f, i) => (
              <button
                key={f.relPath}
                onClick={() => setActiveFileIdx(i)}
                title={f.relPath}
                style={{
                  display: "flex", alignItems: "center", gap: 4,
                  padding: "4px 10px", border: "none",
                  borderBottom: i === activeFileIdx ? "2px solid var(--accent)" : "2px solid transparent",
                  background: i === activeFileIdx ? "var(--bg)" : "none",
                  cursor: "pointer", fontSize: 11, color: i === activeFileIdx ? "var(--fg)" : "var(--color-text-muted)",
                  whiteSpace: "nowrap",
                }}
              >
                <FileText size={10} />
                {f.name}
                {f.dirty && <span style={{ color: "var(--color-warn)", marginLeft: 2 }}>●</span>}
                <span
                  onClick={(e) => { e.stopPropagation(); closeFile(i); }}
                  style={{ marginLeft: 4, cursor: "pointer", opacity: 0.5, fontSize: 10 }}
                  title="Close"
                >✕</span>
              </button>
            ))}
          </div>
        )}

        {/* Toolbar */}
        {selectedFile ? (
          <div style={{
            display: "flex", alignItems: "center", justifyContent: "space-between",
            padding: "6px 10px", borderBottom: "1px solid var(--color-border)",
            background: "var(--bg-soft)",
          }}>
            <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <FileText size={13} style={{ color: "var(--accent)" }} />
              <span style={{ fontSize: 12, fontWeight: 500 }}>{fileName}</span>
              {dirty && <span style={{ fontSize: 10, color: "var(--color-warn)" }}>• unsaved</span>}
            </div>
            <div style={{ display: "flex", gap: 4 }}>
              <button onClick={handleFIM} disabled={fimBusy} title="FIM completion (Ctrl+Space)" style={{
                ...iconBtnStyle, opacity: fimBusy ? 0.5 : 1,
              }}>
                <Wand2 size={12} />
                <span style={{ fontSize: 10 }}>FIM</span>
              </button>
              <button onClick={() => setMemoryOpen(!memoryOpen)} title="Memory facts" style={{
                ...iconBtnStyle, fontWeight: memoryOpen ? 700 : 400,
                color: memoryOpen ? "var(--accent)" : "var(--fg)",
              }}>
                <Brain size={12} />
                <span style={{ fontSize: 10 }}>Memory</span>
              </button>
              <button onClick={() => setViewMode(viewMode === "edit" ? "split" : viewMode === "split" ? "preview" : "edit")} style={{
                ...iconBtnStyle, fontWeight: viewMode !== "edit" ? 700 : 400,
                color: viewMode !== "edit" ? "var(--accent)" : "var(--fg)",
              }}>
                {viewMode === "edit" ? <Eye size={12} /> : viewMode === "split" ? <Edit3 size={12} /> : <Edit3 size={12} />}
                <span style={{ fontSize: 10 }}>{viewMode === "edit" ? "Preview" : viewMode === "split" ? "Edit" : "Edit"}</span>
              </button>
              <button onClick={saveFile} disabled={!dirty} style={{
                ...iconBtnStyle, opacity: dirty ? 1 : 0.4,
              }}>
                <Save size={12} />
                <span style={{ fontSize: 10 }}>Save</span>
              </button>
            </div>
          </div>
        ) : (
          <div style={{
            display: "flex", alignItems: "center", justifyContent: "center",
            flex: 1, color: "var(--color-text-muted)", fontSize: 13, fontStyle: "italic",
          }}>
            Select a markdown file to start editing.
          </div>
        )}

        {/* Content area */}
        {selectedFile && (
          <div style={{ flex: 1, minHeight: 0, overflow: "hidden", display: "flex" }}>
            {/* Editor pane */}
            {(viewMode === "edit" || viewMode === "split") && (
              <div style={{ flex: viewMode === "split" ? 1 : 1, minWidth: 0, borderRight: viewMode === "split" ? "1px solid var(--color-border)" : "none" }}>
                <CodeMirrorEditor
                  initialValue={content}
                  relPath={selectedFile}
                  onChange={handleContentChange}
                  onSave={saveFile}
                />
              </div>
            )}

            {/* Preview pane */}
            {(viewMode === "preview" || viewMode === "split") && (
              <div style={{ flex: viewMode === "split" ? 1 : 1, minWidth: 0 }}>
                <div
                  className="write-mode-preview"
                  dangerouslySetInnerHTML={{ __html: renderMarkdown(content) }}
                  style={{
                    padding: "16px 24px", overflow: "auto", height: "100%",
                    fontSize: 14, lineHeight: 1.7, color: "var(--fg)",
                  }}
                />
              </div>
            )}

            {/* Memory facts sidebar */}
            {memoryOpen && (
              <div style={{
                width: 220, borderLeft: "1px solid var(--color-border)",
                background: "var(--bg-soft)", display: "flex", flexDirection: "column",
                overflow: "hidden",
              }}>
                <div style={{
                  padding: "8px 10px", borderBottom: "1px solid var(--color-border)",
                  fontSize: 11, fontWeight: 600, color: "var(--fg)",
                  display: "flex", alignItems: "center", gap: 4,
                }}>
                  <Brain size={11} style={{ color: "#d4a853" }} />
                  Memory
                </div>
                <div style={{ flex: 1, overflow: "auto", padding: "6px 8px" }}>
                  {memoryFacts.length === 0 ? (
                    <div style={{ fontSize: 11, color: "var(--color-text-muted)", fontStyle: "italic", padding: "4px 0" }}>
                      No relevant facts.
                    </div>
                  ) : (
                    memoryFacts.map((f, i) => (
                      <div
                        key={i}
                        title={f.description}
                        style={{
                          padding: "3px 6px", marginBottom: 4, borderRadius: 3, fontSize: 10,
                          border: `1px solid ${typeColor(f.type)}40`,
                          background: `${typeColor(f.type)}10`,
                          color: "var(--fg)", lineHeight: 1.3,
                        }}
                      >
                        <div style={{ fontWeight: 600, color: typeColor(f.type) }}>{f.title}</div>
                        {f.description && (
                          <div style={{ color: "var(--color-text-muted)", marginTop: 1 }}>
                            {f.description.length > 80 ? f.description.slice(0, 80) + "…" : f.description}
                          </div>
                        )}
                      </div>
                    ))
                  )}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

const iconBtnStyle: React.CSSProperties = {
  display: "inline-flex", alignItems: "center", gap: 3,
  padding: "2px 6px", fontSize: 11, border: "1px solid var(--color-border)",
  borderRadius: 4, background: "var(--bg)", color: "var(--fg)", cursor: "pointer",
};
