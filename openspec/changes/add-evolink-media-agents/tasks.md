## 1. Implementation
- [x] 1.1 Create `pkg/agentgo/providers/evolink` with shared HTTP client, config, and task polling utilities aligned with the EvoLink docs (Sora-2 video + GPT-4O image-series).
- [x] 1.2 Add `pkg/agentgo/models/evolink/video`, `image`, and `text` packages that expose typed constructors (`New`) hooking into the provider and enforcing parameter validation (aspect ratios, durations, sizes, counts, mask/reference limits, HTTPS callbacks).
- [x] 1.3 Integrate the models into at least one agent pipeline. — Added `cmd/examples/evolink_media_pipeline/` that composes a `workflow.Workflow` with custom `imageNode`/`videoNode` wrappers driven by an agent text step, proving the three evolink models compose with the rest of the framework.
- [x] 1.4 Update `website/examples/evolink-media-agents.md` and the zh mirror to document how to configure the provider, including env vars, sample Go snippets, and compliance callouts.

## 2. Validation
- [x] 2.1 `go test ./pkg/agentgo/providers/evolink/... ./pkg/agentgo/models/evolink/...` covering success/failure flows with httptest servers.
- [x] 2.2 Run or simulate a workflow using the new models. — `cmd/examples/evolink_media_pipeline/main_test.go` (`TestMediaPipelineEndToEnd`) mocks all three EvoLink endpoints via `httptest` and asserts the full text → image → video chain succeeds; passes under `make test`.
- [ ] 2.3 `npm run docs:build` (inside `website/`). — Deferred: `website/` has no committed `package.json` / `node_modules/`, so VitePress build is run out-of-band in the docs deploy workflow. Tracked as Sprint 0 CI step.
