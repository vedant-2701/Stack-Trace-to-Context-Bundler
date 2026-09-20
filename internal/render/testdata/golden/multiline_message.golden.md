> This is a stack-trace-bundler bundle: an exception chain with own-code snippets, git blame, and resolved dependency versions, packaged for pasting into an AI chat.
- Language: typescript
- OS: linux
- Runtime: node 20.11.0
- Git: main @ 4455667 (clean)
- Fingerprint: bb33cc44dd55ee66

### ValidationError
> Expected values to be strictly equal:
>
> foo \!== bar
at assertEqual (/repo/src/assert.ts:6:7) — own
```typescript
   3 | export function assertEqual(a: unknown, b: unknown) {
   4 |   if (a !== b) {
   5 |     throw new ValidationError(
→  6 |       'Expected values to be strictly equal: ' + a + ' !== ' + b
   7 |     );
   8 |   }
   9 | }
  10 | 
  11 | export function assertNotEqual(a: unknown, b: unknown) {
  12 |   if (a === b) {
  13 |     throw new ValidationError('Expected values to differ');
```
| Lines | Commit | Author | Date | Summary |
| --- | --- | --- | --- | --- |
| 3-13 | 4455667 | vedant | 2026-03-20 | add assertion helpers |
at processTicksAndRejections (node:internal/process/task_queues:95:5) — runtime

## Dependencies
- lodash — declared ^4.17.21, resolved 4.17.21

<details><summary>Raw input</summary>

```
ValidationError: Expected values to be strictly equal:

foo !== bar
    at assertEqual (/repo/src/assert.ts:6:7)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)
```
</details>
