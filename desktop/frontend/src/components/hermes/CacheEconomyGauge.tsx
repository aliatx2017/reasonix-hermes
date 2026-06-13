import { Cpu } from "lucide-react";

interface CacheEconomyView {
  hitTokens: number;
  missTokens: number;
  totalTokens: number;
  hitRate: number;
}

interface CacheEconomyGaugeProps {
  cache: CacheEconomyView | null;
}

function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return String(n);
}

export function CacheEconomyGauge({ cache }: CacheEconomyGaugeProps) {
  if (!cache || cache.totalTokens === 0) return null;

  const rate = cache.hitRate;
  const color = rate >= 75 ? "var(--color-green)" : rate >= 50 ? "var(--color-yellow)" : "var(--color-warn)";

  return (
    <div
      className="hermes-cache-gauge"
      title={`Cache: ${cache.hitRate.toFixed(1)}% hit rate (${formatTokens(cache.hitTokens)} hit / ${formatTokens(cache.missTokens)} miss)`}
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 4,
        fontSize: 11,
        padding: "1px 6px",
        borderRadius: 4,
        background: "var(--color-surface-raised)",
        border: "1px solid var(--color-border)",
        cursor: "default",
        whiteSpace: "nowrap",
      }}
    >
      <Cpu size={12} style={{ color }} />
      <span style={{ fontWeight: 600, color }}>{rate.toFixed(0)}%</span>
      <span style={{ color: "var(--color-text-muted)" }}>
        {formatTokens(cache.hitTokens)}/{formatTokens(cache.totalTokens)}
      </span>
    </div>
  );
}
