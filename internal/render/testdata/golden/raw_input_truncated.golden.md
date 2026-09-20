> This is a stack-trace-bundler bundle: an exception chain with own-code snippets, git blame, and resolved dependency versions, packaged for pasting into an AI chat.
- Language: typescript
- OS: linux
- Runtime: node 20.11.0
- Git: main @ 7788990 (clean)
- Fingerprint: ee66ff77aa88bb99

### Error
> input too large to fully capture
at handleLargeRequest (/repo/src/handler.ts:17:9) — own
```typescript
  10 | export function handleLargeRequest(req: Request) {
  11 |   const body = req.rawBody;
  12 | 
  13 |   if (!body) {
  14 |     throw new Error('missing body');
  15 |   }
  16 | 
→ 17 |   const parsed = JSON.parse(body);
  18 |   return parsed;
  19 | }
  20 | 
```
| Lines | Commit | Author | Date | Summary |
| --- | --- | --- | --- | --- |
| 10-20 | 7788990 | vedant | 2026-01-18 | handle large request bodies |
at processTicksAndRejections (node:internal/process/task_queues:95:5) — runtime

## Dependencies
- lodash — declared ^4.17.21, resolved 4.17.21

<details><summary>Raw input</summary>

⚠ input truncated at the 512 KB cap

```
Error: input too large to fully capture
    at handleLargeRequest (/repo/src/handler.ts:17:9)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)
```
</details>
