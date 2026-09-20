> This is a stack-trace-bundler bundle: an exception chain with own-code snippets, git blame, and resolved dependency versions, packaged for pasting into an AI chat.
- Language: typescript
- OS: linux
- Runtime: node 20.11.0
- Git: main @ 1234567 (uncommitted changes)
- Fingerprint: dd22ee33ff44aa11

### Error
> stale checkout mismatch
at formatDate (/repo/src/utils/formatDate.ts:8:5) — own
⚠ file has uncommitted local changes; snippet may not match the trace
at processTicksAndRejections (node:internal/process/task_queues:95:5) — runtime

## Dependencies
- lodash — declared ^4.17.21, resolved 4.17.21

<details><summary>Raw input</summary>

```
Error: stale checkout mismatch
    at formatDate (/repo/src/utils/formatDate.ts:8:5)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)
```
</details>
