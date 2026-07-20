# AgentSDK Analyzer Extraction

Date: 2026-07-20

The dependency analyzer was extracted from `agentSDK/tools/dependency-analyzer` into the standalone repository `~/projects/go-dependency-analysis`.

## Decision

- Module path: `github.com/bizshuk/go-dependency-analysis`
- Dependency boundary: Go standard library only
- Integration: external CLI invocation, never a Go import from analyzed projects
- Behavior: read-only text/JSON/Mermaid analysis with optional JSON policy
- Self-exclusion: module identity, independent of checkout directory name
- AgentSDK policy: retained as `examples/agentsdk.json`, not loaded by default

## Why

The analyzer is a general-purpose tool layer component. Keeping it inside AgentSDK made workspace module counts, self-exclusion, documentation, and release lifecycle depend on one application repository. Extraction gives it an independent CLI lifecycle and allows any Go workspace to use it without cross-project imports.
