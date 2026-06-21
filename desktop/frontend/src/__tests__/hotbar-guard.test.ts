// Run: tsx src/__tests__/hotbar-guard.test.ts
//
// Verifies the hotbar keyboard handler is present in App.tsx.
// The handler was added in 4069ee47 (Jun 12 2026), destroyed by
// the v1.6.0 upstream merge at 78a9d76a, and not restored until
// b6de38ac (Jun 21 2026) — ~30 sessions / 9 days of broken hotbar.
// This test EXISTS to prevent a repeat of that pattern.

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
const appSource = readFileSync(resolve(here, '../App.tsx'), 'utf8');

console.log('\nhotbar guard');

// 1. The hotbar digit-key handler useEffect must exist
ok(
  appSource.includes('case "1"') && appSource.includes('case "7"'),
  'hotbar digit-key handler present in App.tsx (case "1" through case "7")',
);

// 2. Key 1 opens the palette
ok(
  /case\s+"1":[\s\S]*?openPalette/.test(appSource),
  'key 1 dispatches to openPalette',
);

// 3. Key 2 toggles workspace
ok(
  /case\s+"2":[\s\S]*?setWorkspacePanelOpen/.test(appSource),
  'key 2 toggles workspace panel',
);

// 4. Key 3 opens a new tab
ok(
  /case\s+"3":[\s\S]*?handleNewTab/.test(appSource),
  'key 3 opens a new tab',
);

// 5. Key 4 opens all history
ok(
  /case\s+"4":[\s\S]*?openAllHistory/.test(appSource),
  'key 4 opens all history',
);

// 6. Key 5 toggles the right dock
ok(
  /case\s+"5":[\s\S]*?setRightDockMode/.test(appSource),
  'key 5 toggles right dock',
);

// 7. Key 6 toggles the sidebar
ok(
  /case\s+"6":[\s\S]*?setSidebarCollapsed/.test(appSource),
  'key 6 toggles sidebar',
);

// 8. Key 7 opens settings
ok(
  /case\s+"7":[\s\S]*?setSettingsTarget/.test(appSource),
  'key 7 opens settings',
);

// 9. "none" must be in RightDockMode (required by key 5 dock toggle)
ok(
  /type RightDockMode\s*=\s*"none"/.test(appSource),
  'RightDockMode includes "none" (required for dock toggle)',
);

// 10. Handler guards against modifier keys
ok(
  /if\s*\(e\.ctrlKey\s*\|\|\s*e\.metaKey\s*\|\|\s*e\.altKey\)\s*return/.test(appSource),
  'hotbar handler ignores modified keys (ctrl/meta/alt)',
);

// 11. Handler skips input/textarea/select elements
ok(
  /tag\s*===\s*"INPUT"\s*\|\|\s*tag\s*===\s*"TEXTAREA"\s*\|\|\s*tag\s*===\s*"SELECT"/.test(appSource),
  'hotbar handler skips input/textarea/select elements',
);

console.log(`\n${passed} passed, ${failed} failed`);
if (failed > 0) process.exit(1);
