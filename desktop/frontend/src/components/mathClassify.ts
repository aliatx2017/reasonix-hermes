export function isLikelyInlineMath(math: string): boolean {
  if (!math || math !== math.trim() || math.includes('\n')) return false;
  if (math.includes('://') || math.includes('](')) return false;
  if (/^\$?\d+(?:\.\d+)?%?$/.test(math)) return false;

  // LaTeX command (\alpha, \frac{x}{y}, \tfrac12, ...). Do not require a
  // word boundary after the command name: \tfrac12 / \sqrt2 / \log3 have no
  // boundary between the command letters and the trailing digit.
  if (/\\[A-Za-z]+/.test(math)) return true;
  if (/[\^_{}|]/.test(math)) return true;
<<<<<<< HEAD
  if (
    /\b(?:alpha|beta|gamma|sum|int|prod|lim|infty|sqrt|frac|sin|cos|tan|log|ln|max|min|partial|nabla|left|right)\b/.test(
      math,
    )
  )
    return true;
  if (/^[A-Za-z]\s*\([^)]{1,80}\)$/.test(math)) return true;
  if (/[A-Za-z0-9)\]}]\s*[+\-*/=<>]\s*[A-Za-z0-9([{\\]/.test(math)) return true;
=======
  // Lone math operators are common in physics prose: "the sign of $+$",
  // "the $<$ relation".
  if (/^[+\-=<>±∓]$/.test(math)) return true;
  if (/\b(?:alpha|beta|gamma|sum|int|prod|lim|infty|sqrt|frac|sin|cos|tan|log|ln|max|min|partial|nabla|left|right)\b/.test(math)) return true;
  // Short identifiers followed by parenthesised arguments cover both
  // function calls (f(x)) and group notation (SO(3,1), SU(2), GL(n)).
  if (/^[A-Za-z]{1,6}\s*\([^)]{1,80}\)$/.test(math)) return true;
  // Primed function call: f'(x), g''(t).
  if (/^[A-Za-z]'{1,}\s*\([^)]{1,80}\)$/.test(math)) return true;
  // Permutation cycle notation: (12), (123), (12)(34).
  if (/^(?:\(\d+\))+$/.test(math)) return true;
  // Binary operator between operands. The RHS may start with a unary sign:
  // K = -iJ, p = +\alpha, a = -b.
  if (/[A-Za-z0-9)\]}]\s*[+\-*/=<>]\s*[+\-]?\s*[A-Za-z0-9([{\\]/.test(math)) return true;
  // One-sided comparison: < B, > 0, B < — comparison against an implicit
  // operand is common in prose.
  if (/^(?:<=?|>=?|≠|≤|≥)\s*[A-Za-z0-9]|[A-Za-z0-9]\s*(?:<=?|>=?|≠|≤|≥)$/.test(math)) return true;
  // Comma-separated tokens: ordered pairs, tuples, sets (A, B), 1, 2, 3,
  // \alpha, \beta. Currency/env-var usage never looks like this.
  if (/^\(?(?:[A-Za-z0-9]|\\[A-Za-z]+)(?:\s*,\s*(?:[A-Za-z0-9]|\\[A-Za-z]+)){1,10}\)?$/.test(math)) return true;
>>>>>>> upstream/main-v2

  // Bracketed labels/indexes such as [56], [8], [N], [\mathbf{56}], [56,0^+]
  // are common in group theory and arrays. Markdown links are rejected above
  // by the "](" guard.
  if (/^\[[A-Za-z0-9^_+\-,.\\\s{}]+\]$/.test(math)) return true;

  if (/[A-Za-z]\s+[A-Za-z]/.test(math)) return false;
  if (/^[A-Z][A-Z0-9_]{1,}$/.test(math)) return false;
  if (/^v\d+(?:\.\d+)*$/i.test(math)) return false;
  if (/^[A-Za-z]{2,}$/.test(math)) return false;

<<<<<<< HEAD
  // Only allow a single lowercase letter as a math token. Capitalised
  // single letters (I, A, V) are too often English / Roman numerals /
  // acronyms to risk misclassifying them.
  return /^[a-z]$/.test(math);
=======
  // Letter or Greek command followed by one or more primes: x', S', y'', \psi'.
  if (/^(?:[A-Za-z]|\\[A-Za-z]+)'{1,}$/.test(math)) return true;

  // Single letter (a-z, A-Z). Uppercase single letters (S, A, G, …) are
  // common math names (sets, algebras, groups) and $X$ is essentially
  // never written for the English word I/A by hand.
  return /^[A-Za-z]$/.test(math);
>>>>>>> upstream/main-v2
}
