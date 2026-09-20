> This is a stack-trace-bundler bundle: an exception chain with own-code snippets, git blame, and resolved dependency versions, packaged for pasting into an AI chat.
- Language: typescript
- OS: linux
- Runtime: node 20.11.0
- Git: main @ 5566778 (clean)
- Fingerprint: cc44dd55ee66ff77

### TypeError
> Expected Map\<string, number\> but got List\<String\>
at convert (/repo/src/converter.ts:12:4) — own
```typescript
   7 | export function convert<T>(input: List<T>): Map<string, T> {
   8 |   const result = new Map<string, T>();
   9 |   let index = 0;
  10 | 
  11 |   for (const item of input) {
→ 12 |     result.set(String(index), item);
  13 |     index++;
  14 |   }
  15 | 
  16 |   return result;
  17 | }
```
| Lines | Commit | Author | Date | Summary |
| --- | --- | --- | --- | --- |
| 7-17 | 5566778 | vedant | 2026-02-10 | add generic type converter |
at processTicksAndRejections (node:internal/process/task_queues:95:5) — runtime

## Dependencies
- lodash — declared ^4.17.21, resolved 4.17.21

<details><summary>Raw input</summary>

```
TypeError: Expected Map<string, number> but got List<String>
    at convert (/repo/src/converter.ts:12:4)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)
```
</details>
