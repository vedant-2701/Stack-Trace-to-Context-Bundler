> This is a stack-trace-bundler bundle: an exception chain with own-code snippets, git blame, and resolved dependency versions, packaged for pasting into an AI chat.
- Language: typescript
- OS: linux
- Runtime: node 20.11.0
- Git: main @ 6677889 (clean)
- Fingerprint: dd55ee66ff77aa88

### Error
> bad markdown in error message
at render (/repo/src/render.ts:9:2) — own
```typescript
   4 | import { formatOutput } from './format';
   5 | 
   6 | export function render(input: string): string {
   7 |   const trimmed = input.trim();
   8 | 
→  9 |   return formatOutput(trimmed);
  10 | }
  11 | 
  12 | export function renderRaw(input: string): string {
  13 |   return input;
  14 | }
```
| Lines | Commit | Author | Date | Summary |
| --- | --- | --- | --- | --- |
| 4-14 | 6677889 | vedant | 2026-02-14 | add render helpers |
at processTicksAndRejections (node:internal/process/task_queues:95:5) — runtime

## Dependencies
- lodash — declared ^4.17.21, resolved 4.17.21

<details><summary>Raw input</summary>

````
Error: bad markdown in error message
```
some embedded code block
```
    at render (/repo/src/render.ts:9:2)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)
````
</details>
