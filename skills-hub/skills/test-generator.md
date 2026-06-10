---
name: test-generator
description: Generate unit tests for a given function or module with TDD patterns.
runAs: subagent
allowedTools:
  - read_file
  - grep
  - glob
  - write_file
  - bash
---

# Test Generator

Generate comprehensive unit tests following TDD best practices.

## Process

1. **Read the target code** — understand the function signatures, branches, and edge cases.
2. **Identify test cases**:
   - Happy path (normal inputs, expected outputs)
   - Edge cases (empty, nil, zero, max/min values)
   - Error cases (invalid inputs, failure modes)
   - Boundary conditions
3. **Generate tests** using the project's existing test framework:
   - Go: `testing` package, table-driven tests
   - TypeScript/JavaScript: Jest, Vitest, or Mocha
   - Python: pytest
   - Rust: `#[test]` with assertions
   - Java: JUnit 5
4. **Run the tests** to verify they compile and pass.

## Output

- The test file with all test cases
- A summary: "Generated N test cases covering X branches"
