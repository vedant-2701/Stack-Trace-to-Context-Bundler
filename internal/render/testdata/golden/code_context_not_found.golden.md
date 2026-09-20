> This is a stack-trace-bundler bundle: an exception chain with own-code snippets, git blame, and resolved dependency versions, packaged for pasting into an AI chat.
- Language: typescript
- OS: linux
- Runtime: node 20.11.0
- Git: main @ 1234567 (clean)
- Fingerprint: cc11dd22ee33ff44

### Error
> legacy path removed
at parseLegacy (/repo/src/legacy/oldParser.ts:5:3) — own
⚠ file not found in current checkout \(deleted or renamed since the trace was captured\)
at processTicksAndRejections (node:internal/process/task_queues:95:5) — runtime

## Dependencies
- lodash — declared ^4.17.21, resolved 4.17.21

<details><summary>Raw input</summary>

```
Error: legacy path removed
    at parseLegacy (/repo/src/legacy/oldParser.ts:5:3)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)
```
</details>
