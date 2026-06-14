---
name: shell-quoting-ssh
description: "Shell quoting patterns for SSH remote commands and bash tool invocations. Work on first try, not third. Covers single-quote nesting, heredoc alternatives, and common escaping pitfalls."
version: 1.0.0
author: Theshire (adapted for Reasonix-Hermes)
tags: [shell, bash, ssh, quoting, escaping, tool-use]
---

# Shell Quoting Patterns

## When to Use

- Running `bash` tool with complex commands containing quotes, variables, or special chars
- Executing commands over SSH where quoting nests multiple times
- Debugging shell quoting errors ("unexpected token", "command not found" where it should work)

## Fundamental Rules

1. **Single quotes** — everything literal, no expansion. `'$HOME'` → literal `$HOME`
2. **Double quotes** — variables and `$()` expand. `"$HOME"` → `/Users/alex`
3. **No quotes** — word splitting + glob expansion. Almost never what you want.

## Single Quote Nesting

You CANNOT nest single quotes directly. `'it's'` fails. Workarounds:

```bash
# End single-quote, insert escaped quote, resume single-quote
echo 'it'"'"'s working'

# Or use double quotes (if no $variables to protect)
echo "it's working"

# Or use heredoc (best for multi-line)
cat <<'EOF'
it's working
EOF
```

## SSH Quoting — Triple Nesting

SSH adds one level of quoting. Complex commands need careful nesting:

```bash
# WRONG — inner quotes break
ssh host "grep 'pattern' file"

# RIGHT — outer double, inner single
ssh host "grep 'pattern' file"

# HARD — command with both quote types
# Use heredoc to avoid nightmare quoting
ssh host <<'ENDSSH'
  grep "it's done" /var/log/*.log
ENDSSH

# OR base64-encode the command
CMD=$(echo 'grep "pattern" file' | base64)
ssh host "echo $CMD | base64 -d | bash"
```

## Common Pitfalls

| Pattern | Wrong | Right |
|---------|-------|-------|
| Variable in single quotes | `'$HOME/path'` | `"$HOME/path"` |
| Nested single quotes | `'it's'` | `'it'"'"'s'` |
| Backticks in quotes | `` '`cmd`' `` | `"$(cmd)"` |
| Exclamation in double quotes | `"hello!"` | `'hello!'` or `"hello"\!"` |

## bash Tool Specific

The `bash` tool in Reasonix runs commands via a configured shell. Prefer heredocs for multi-line scripts:

```bash
cat <<'SCRIPT' > /tmp/script.sh
#!/bin/bash
# complex logic here
SCRIPT
bash /tmp/script.sh
```

## Verification

After any SSH or complex bash command, verify the output is what you expect. A silent success with no output ≠ it worked.

## Related

- Project skill: `ready-means-tested` — verify, don't assume
- Tool: `bash` — runs commands through configured shell interpreter
