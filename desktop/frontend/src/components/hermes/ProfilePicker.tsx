import { Sliders } from 'lucide-react';
import type { ProfileView } from '../../lib/types';

interface ProfilePickerProps {
  profiles: Record<string, ProfileView>;
  activeProfile: string;
  onChangeProfile: (name: string, profile: ProfileView) => void;
  onClearProfile: () => void;
}

export function ProfilePicker({
  profiles,
  activeProfile,
  onChangeProfile,
  onClearProfile,
}: ProfilePickerProps) {
  const entries = Object.entries(profiles ?? {}) as [string, ProfileView][];

  if (entries.length === 0) return null;

  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const name = e.target.value;
    if (name === '' || name === '__clear__') {
      onClearProfile();
    } else {
      const profile = profiles[name];
      if (profile) onChangeProfile(name, profile);
    }
  };

  const active = entries.find(([n]) => n === activeProfile);

  return (
    <div
      className="profile-picker"
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 6,
        flexShrink: 0,
      }}
    >
      <Sliders size={14} style={{ color: 'var(--color-text-muted)', flexShrink: 0 }} />
      <select
        value={activeProfile || '__none__'}
        onChange={handleChange}
        title={
          active
            ? `${active[0]}: ${active[1].description || ''} (model: ${active[1].model || 'default'})`
            : 'Select harness profile'
        }
        style={{
          padding: '2px 4px',
          fontSize: 11,
          borderRadius: 4,
          border: `1px solid ${activeProfile ? 'var(--color-accent)' : 'var(--color-border)'}`,
          background: activeProfile ? 'var(--color-accent-subtle)' : 'var(--color-surface)',
          color: 'var(--color-text)',
          maxWidth: 120,
          cursor: 'pointer',
        }}
      >
        <option value="__none__">profile</option>
        <option value="__clear__" disabled={!activeProfile}>
          — clear —
        </option>
        {entries.map(([name]) => (
          <option key={name} value={name}>
            {name}
          </option>
        ))}
      </select>
    </div>
  );
}
