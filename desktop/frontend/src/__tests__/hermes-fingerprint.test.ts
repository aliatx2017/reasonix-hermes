// Run: tsx src/__tests__/hermes-fingerprint.test.ts
//
// SYSTEM-WIDE guard: verifies every Hermes code block injected into
// upstream-shared files still exists. Upstream merges can silently
// drop injected code when git merge doesn't flag a conflict.
//
// Documented losses:
//   - hotbar handler in App.tsx (lost 9 days, Jun 12–21 2026, fixed b6de38ac)
//   - sqz CompressToolOutput wiring in boot.go (lost, fixed h52)
//   - SettingsView Hotbar/Profiles fields (lost, fixed h52)
//   - render.go Hermes TOML sections (lost, fixed h11)
//
// Every fingerprint was verified against the live codebase before commit.
// If a fingerprint fails, an upstream merge just destroyed Hermes code.

import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

let passed = 0;
let failed = 0;

function ok(cond: boolean, label: string) {
  if (cond) {
    process.stdout.write(`  PASS  ${label}\n`);
    passed += 1;
  } else {
    process.stdout.write(`  FAIL  ${label}\n`);
    failed += 1;
  }
}

const here = dirname(fileURLToPath(import.meta.url));
const frontendRoot = resolve(here, '..');

function source(rel: string): string {
  return readFileSync(resolve(frontendRoot, rel), 'utf8');
}

console.log('\nhermes fingerprint — desktop frontend');

// ── App.tsx — the #1 merge hotspot ──
{
  const s = source('App.tsx');

  // Hotbar digit-key handler (restored b6de38ac after 9-day gap)
  ok(s.includes('case "1"') && s.includes('case "7"'),
    'App.tsx: hotbar digit-key handler (cases 1-7)');
  ok(/case\s+"1":[\s\S]*?openPalette/.test(s),
    'App.tsx: key 1 → openPalette');
  ok(s.includes('setRightDockMode((cur) => cur === "context" ? "none" : "context")'),
    'App.tsx: key 5 dock toggle with "none" fallback');
  // RightDockMode type was moved to store/layout.ts by upstream refactoring
  ok(source('store/layout.ts').includes('"none"') && source('store/layout.ts').includes('type RightDockMode'),
    'store/layout.ts: RightDockMode includes "none"');
  // Hermes integration: the HermesSettings component is rendered via
  // SettingsPanel, not imported directly in App.tsx. The import lives in
  // SettingsPanel.tsx (verified separately below).
}

// ── SettingsPanel.tsx — Hermes tab injection ──
{
  const s = source('components/SettingsPanel.tsx');

  ok(s.includes('HermesLiveSection'),
    'SettingsPanel.tsx: HermesLiveSection component');
  ok(s.includes('useHermesLiveData'),
    'SettingsPanel.tsx: useHermesLiveData hook imported');
  ok(s.includes('HermesSettings') && s.includes('from "./hermes/HermesSettings"'),
    'SettingsPanel.tsx: HermesSettings imported from hermes/');
}

// ── bridge.ts — Hermes Wails bindings ──
{
  const s = source('lib/bridge.ts');

  ok(s.includes('Hermes bindings (restored after upstream merge)'),
    'bridge.ts: Hermes bindings block header');
  ok(s.includes('SetDesktopHotbar'),
    'bridge.ts: SetDesktopHotbar binding');
  ok(s.includes('SetProfiles'),
    'bridge.ts: SetProfiles binding');
  ok(s.includes('LearnedPatterns'),
    'bridge.ts: LearnedPatterns binding');
  ok(s.includes('CompressStats'),
    'bridge.ts: CompressStats binding');
  ok(s.includes('ScheduleDashboard'),
    'bridge.ts: ScheduleDashboard binding');
  ok(s.includes('CostSummary'),
    'bridge.ts: CostSummary binding');
  ok(s.includes('SyncLobeHubMarketplace'),
    'bridge.ts: SyncLobeHubMarketplace binding');
  ok(s.includes('hotbar:') && s.includes('profiles:') && s.includes('activeProfile:'),
    'bridge.ts: hotbar/profiles/activeProfile in mock defaults');
}

// ── types.ts — Hermes type definitions ──
{
  const s = source('lib/types.ts');

  ok(s.includes('HotbarView'),
    'types.ts: HotbarView interface');
  ok(s.includes('ProfileView'),
    'types.ts: ProfileView interface');
  ok(s.includes('hotbar: HotbarView'),
    'types.ts: hotbar field on SettingsView');
  ok(s.includes('activeProfile:'),
    'types.ts: activeProfile field on SettingsView');
}

// ── theme.ts — Hermes theme style ──
{
  const s = source('lib/theme.ts');

  ok(s.includes("'hermes'"),
    "theme.ts: 'hermes' in theme style definitions");
}

// ── StatusBar.tsx — Hermes status bar group ──
{
  const s = source('components/StatusBar.tsx');

  ok(s.includes('statusbar__group--hermes'),
    'StatusBar.tsx: statusbar__group--hermes class');
}

// ── index.html — window title ──
{
  const s = readFileSync(resolve(frontendRoot, '..', 'index.html'), 'utf8');

  ok(s.includes('Reasonix-Hermes'),
    'index.html: <title>Reasonix-Hermes</title>');
}

console.log(`\n${passed} passed, ${failed} failed`);
if (failed > 0) process.exit(1);
