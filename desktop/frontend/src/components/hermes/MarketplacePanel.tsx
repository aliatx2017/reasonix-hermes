import { useState, useEffect } from 'react';
import { Download, RefreshCw, Search, Star, Tag } from 'lucide-react';
import { app } from '../../lib/bridge';
import type { MarketplaceEntryView } from '../../lib/types';

interface MarketplacePanelProps {
  // standalone panel — no external props needed
}

export function MarketplacePanel(_props: MarketplacePanelProps) {
  const [skills, setSkills] = useState<MarketplaceEntryView[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [activeTag, setActiveTag] = useState('');
  const [installed, setInstalled] = useState<Set<string>>(new Set());
  const [statusMsg, setStatusMsg] = useState('');
  const [syncing, setSyncing] = useState(false);

  useEffect(() => {
    app
      .MarketplaceRegistry()
      .then((entries) => setSkills(entries))
      .catch(() => {
        /* bridge not ready */
      })
      .finally(() => setLoading(false));
  }, []);

  // Collect unique tags from the real registry.
  const allTags = [...new Set(skills.flatMap((s) => s.tags))].sort();

  const filtered = skills.filter((s) => {
    if (activeTag && !s.tags.some((t) => t.toLowerCase() === activeTag.toLowerCase())) return false;
    if (search) {
      const q = search.toLowerCase();
      if (
        !s.name.toLowerCase().includes(q) &&
        !s.description.toLowerCase().includes(q) &&
        !s.tags.some((t) => t.toLowerCase().includes(q))
      )
        return false;
    }
    return true;
  });

  const handleInstall = async (skill: MarketplaceEntryView) => {
    const cmd = `reasonix install-source install --source ${skill.url}`;
    try {
      await navigator.clipboard.writeText(cmd);
      setInstalled((prev) => new Set(prev).add(skill.name));
      setStatusMsg(`Copied install command for "${skill.name}"`);
      setTimeout(() => setStatusMsg(''), 3000);
    } catch {
      setStatusMsg(`Install command: ${cmd}`);
      setTimeout(() => setStatusMsg(''), 5000);
    }
  };

  const handleSync = async () => {
    setSyncing(true);
    setStatusMsg('Syncing from LobeHub marketplace...');
    try {
      const result = await app.SyncLobeHubMarketplace('', '');
      setStatusMsg(`Synced from LobeHub (${result.fetched} skills fetched)`);
      // Reload skill list
      const entries = await app.MarketplaceRegistry();
      setSkills(entries);
    } catch (err: any) {
      setStatusMsg(`Sync error: ${err?.message || err}`);
    } finally {
      setSyncing(false);
      setTimeout(() => setStatusMsg(''), 8000);
    }
  };

  if (loading) {
    return (
      <div
        className="hermes-panel"
        style={{ padding: 12, fontSize: 12, color: 'var(--color-text-muted)' }}
      >
        Loading marketplace...
      </div>
    );
  }

  return (
    <div className="hermes-panel" style={{ padding: 12, fontSize: 12 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
        <Search size={14} style={{ color: 'var(--color-accent)' }} />
        <span style={{ fontWeight: 600 }}>Skill Marketplace</span>
        <span style={{ color: 'var(--color-text-muted)', fontSize: 10 }}>
          {skills.length} skills · agentskills.io-compatible
        </span>
        <div style={{ flex: 1 }} />
        <button
          onClick={handleSync}
          disabled={syncing}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 3,
            padding: '2px 8px',
            borderRadius: 4,
            border: '1px solid var(--color-accent)',
            background: 'transparent',
            color: 'var(--color-accent)',
            cursor: syncing ? 'not-allowed' : 'pointer',
            fontSize: 10,
            fontWeight: 500,
            opacity: syncing ? 0.6 : 1,
          }}
          title="Sync skills from LobeHub community marketplace"
        >
          <RefreshCw size={10} className={syncing ? 'spin' : ''} />{' '}
          {syncing ? 'Syncing…' : 'Sync from LobeHub'}
        </button>
      </div>

      {/* Search input */}
      <div style={{ marginBottom: 6 }}>
        <input
          type="text"
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setActiveTag('');
          }}
          placeholder="Search skills..."
          style={{
            width: '100%',
            padding: '4px 8px',
            fontSize: 12,
            borderRadius: 4,
            border: '1px solid var(--color-border)',
            background: 'var(--color-surface)',
            color: 'var(--color-text)',
            boxSizing: 'border-box',
          }}
        />
      </div>

      {/* Tag chips */}
      {allTags.length > 0 && (
        <div style={{ marginBottom: 8 }}>
          <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
            <span
              onClick={() => {
                setActiveTag('');
                setSearch('');
              }}
              style={{
                padding: '1px 6px',
                borderRadius: 4,
                fontSize: 10,
                cursor: 'pointer',
                background:
                  activeTag === '' && !search
                    ? 'var(--color-accent)'
                    : 'var(--color-surface-raised)',
                color: activeTag === '' && !search ? '#fff' : 'var(--color-text-muted)',
                border: '1px solid var(--color-border)',
              }}
            >
              All
            </span>
            {allTags.map((tag) => (
              <span
                key={tag}
                onClick={() => {
                  setActiveTag(activeTag === tag ? '' : tag);
                  setSearch('');
                }}
                style={{
                  padding: '1px 6px',
                  borderRadius: 4,
                  fontSize: 10,
                  cursor: 'pointer',
                  background:
                    activeTag === tag ? 'var(--color-accent)' : 'var(--color-surface-raised)',
                  color: activeTag === tag ? '#fff' : 'var(--color-text-muted)',
                  border: '1px solid var(--color-border)',
                }}
              >
                {tag}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Skill list */}
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 4,
          maxHeight: 300,
          overflowY: 'auto',
        }}
      >
        {filtered.length === 0 ? (
          <div
            style={{
              color: 'var(--color-text-muted)',
              fontStyle: 'italic',
              padding: 16,
              textAlign: 'center',
            }}
          >
            No skills match your search.
          </div>
        ) : (
          filtered.map((skill) => (
            <div
              key={skill.name}
              style={{
                display: 'flex',
                alignItems: 'flex-start',
                gap: 8,
                padding: '6px 8px',
                borderRadius: 6,
                background: 'var(--color-surface-raised)',
                border: '1px solid var(--color-border)',
              }}
            >
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                  <span style={{ fontWeight: 600, fontSize: 12 }}>{skill.name}</span>
                  <span
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 2,
                      color: 'var(--color-accent)',
                      fontSize: 10,
                    }}
                  >
                    <Star size={10} /> {skill.rating}
                  </span>
                </div>
                <div
                  style={{
                    color: 'var(--color-text-muted)',
                    fontSize: 10,
                    marginTop: 2,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {skill.description}
                </div>
                <div style={{ display: 'flex', gap: 3, marginTop: 3, flexWrap: 'wrap' }}>
                  {skill.tags.map((tag) => (
                    <span
                      key={tag}
                      onClick={() => setActiveTag(tag)}
                      style={{
                        display: 'inline-flex',
                        alignItems: 'center',
                        gap: 2,
                        fontSize: 9,
                        color: 'var(--color-text-muted)',
                        cursor: 'pointer',
                      }}
                    >
                      <Tag size={8} />
                      {tag}
                    </span>
                  ))}
                  {skill.author && (
                    <span style={{ fontSize: 9, color: 'var(--color-text-muted)' }}>
                      by {skill.author}
                    </span>
                  )}
                </div>
              </div>
              <button
                onClick={() => handleInstall(skill)}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 3,
                  padding: '3px 8px',
                  borderRadius: 4,
                  border: `1px solid ${installed.has(skill.name) ? 'var(--color-success, #4caf50)' : 'var(--color-accent)'}`,
                  background: installed.has(skill.name)
                    ? 'var(--color-surface-raised)'
                    : 'transparent',
                  color: installed.has(skill.name)
                    ? 'var(--color-success, #4caf50)'
                    : 'var(--color-accent)',
                  cursor: 'pointer',
                  fontSize: 10,
                  fontWeight: 500,
                  flexShrink: 0,
                }}
                title={`Copy install command for ${skill.name}`}
              >
                <Download size={10} /> {installed.has(skill.name) ? 'Copied!' : 'Install'}
              </button>
            </div>
          ))
        )}
      </div>

      {/* Status message */}
      {statusMsg && (
        <div
          style={{
            marginTop: 8,
            color: 'var(--color-success, #4caf50)',
            fontSize: 10,
            fontStyle: 'italic',
          }}
        >
          {statusMsg}
        </div>
      )}

      <div style={{ marginTop: 8, color: 'var(--color-text-muted)', fontSize: 10 }}>
        CLI:{' '}
        <code style={{ fontFamily: 'monospace' }}>reasonix marketplace search &lt;query&gt;</code>
      </div>
    </div>
  );
}
