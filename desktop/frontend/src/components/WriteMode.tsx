import { useState, useEffect, useCallback } from "react";
import { FileText, Save, Plus, RefreshCw, Eye, Edit3, X } from "lucide-react";
import { app } from "../lib/bridge";

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
    // Escape HTML
    .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")
    // Headers
    .replace(/^### (.+)$/gm, "<h3>$1</h3>")
    .replace(/^## (.+)$/gm, "<h2>$1</h2>")
    .replace(/^# (.+)$/gm, "<h1>$1</h1>")
    // Bold and italic
    .replace(/\*\*\*(.+?)\*\*\*/g, "<strong><em>$1</em></strong>")
    .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
    .replace(/\*(.+?)\*/g, "<em>$1</em>")
    // Inline code
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    // Links
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener">$1</a>')
    // Unordered lists
    .replace(/^- (.+)$/gm, "<li>$1</li>")
    // Paragraphs
    .replace(/\n\n/g, "</p><p>")
    .replace(/\n/g, "<br>");
  html = "<p>" + html + "</p>";
  // Wrap consecutive <li> items in <ul>
  html = html.replace(/((?:<li>.*?<\/li><br>?)+)/g, (match) => {
    const items = match.replace(/<br>/g, "");
    return "<ul>" + items + "</ul>";
  });
  return html;
}

export function WriteMode() {
  const [files, setFiles] = useState<MarkdownFileEntry[]>([]);
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [content, setContent] = useState("");
  const [fileName, setFileName] = useState("");
  const [dirty, setDirty] = useState(false);
  const [preview, setPreview] = useState(false);
  const [newFileName, setNewFileName] = useState("");
  const [showNewFile, setShowNewFile] = useState(false);

  const loadFiles = useCallback(async () => {
    try {
      const list = await app.ListMarkdownFiles();
      setFiles(list ?? []);
    } catch { /* ignore */ }
  }, []);

  const openFile = useCallback(async (relPath: string) => {
    try {
      const data = await app.ReadMarkdownFile(relPath);
      if (data) {
        setSelectedFile(relPath);
        setContent(data.content);
        setFileName(data.name);
        setDirty(false);
        setPreview(false);
      }
    } catch { /* ignore */ }
  }, []);

  const saveFile = useCallback(async () => {
    if (!selectedFile) return;
    try {
      await app.SaveMarkdownFile(selectedFile, content);
      setDirty(false);
    } catch { /* ignore */ }
  }, [selectedFile, content]);

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

  // Keyboard shortcut: Cmd/Ctrl+S
  useEffect(() => {
    const handle = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "s") {
        e.preventDefault();
        if (dirty && selectedFile) saveFile();
      }
    };
    window.addEventListener("keydown", handle);
    return () => window.removeEventListener("keydown", handle);
  }, [dirty, selectedFile, saveFile]);

  const handleContentChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setContent(e.target.value);
    setDirty(true);
  };

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
              <button onClick={() => setPreview(!preview)} style={{
                ...iconBtnStyle, fontWeight: preview ? 700 : 400,
                color: preview ? "var(--accent)" : "var(--fg)",
              }}>
                {preview ? <Edit3 size={12} /> : <Eye size={12} />}
                <span style={{ fontSize: 10 }}>{preview ? "Edit" : "Preview"}</span>
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
          <div style={{ flex: 1, minHeight: 0, overflow: "hidden" }}>
            {preview ? (
              <div
                className="write-mode-preview"
                dangerouslySetInnerHTML={{ __html: renderMarkdown(content) }}
                style={{
                  padding: "16px 24px", overflow: "auto", height: "100%",
                  fontSize: 14, lineHeight: 1.7, color: "var(--fg)",
                }}
              />
            ) : (
              <textarea
                value={content}
                onChange={handleContentChange}
                style={{
                  width: "100%", height: "100%", border: "none", resize: "none",
                  padding: "16px", fontSize: 13, lineHeight: 1.6,
                  fontFamily: "var(--mono, monospace)", color: "var(--fg)",
                  background: "var(--bg)", outline: "none",
                }}
                spellCheck={false}
              />
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
