> This is a stack-trace-bundler bundle: an exception chain with own-code snippets, git blame, and resolved dependency versions, packaged for pasting into an AI chat.
- Language: typescript
- OS: linux
- Runtime: node 20.11.0
- Fingerprint: aa11bb22cc33dd44

### RangeError
> Invalid array length
at buildMatrix (/repo/src/matrix.ts:14:10) — own
```typescript
   9 | export function buildMatrix(rows: number, cols: number) {
  10 |   if (rows < 0 || cols < 0) {
  11 |     throw new RangeError('Invalid array length');
  12 |   }
  13 | 
→ 14 |   const matrix = new Array(rows * cols);
  15 |   for (let i = 0; i < matrix.length; i++) {
  16 |     matrix[i] = 0;
  17 |   }
  18 |   return matrix;
  19 | }
```
| Lines | Commit | Author | Date | Summary |
| --- | --- | --- | --- | --- |
| 9-19 | 1122334 | vedant | 2026-06-10 | add matrix builder utility |
at processTicksAndRejections (node:internal/process/task_queues:95:5) — runtime

## Dependencies
- lodash — declared ^4.17.21, resolved 4.17.21

<details><summary>Raw input</summary>

```
RangeError: Invalid array length
    at buildMatrix (/repo/src/matrix.ts:14:10)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)
```
</details>
