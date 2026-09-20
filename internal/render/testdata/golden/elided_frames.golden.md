> This is a stack-trace-bundler bundle: an exception chain with own-code snippets, git blame, and resolved dependency versions, packaged for pasting into an AI chat.
- Language: typescript
- OS: linux
- Runtime: node 20.11.0
- Git: main @ 3344556 (clean)
- Fingerprint: aa22bb33cc44dd55

### RuntimeException
> top\-level failure
at run (/repo/src/main.ts:10:5) — own
```typescript
   5 | import { readFile } from './io';
   6 | 
   7 | export async function run() {
   8 |   try {
   9 |     const data = await readFile('/tmp/input.txt');
→ 10 |     console.log(data);
  11 |   } catch (err) {
  12 |     throw err;
  13 |   }
  14 | }
  15 | 
```
| Lines | Commit | Author | Date | Summary |
| --- | --- | --- | --- | --- |
| 5-15 | 2233445 | vedant | 2026-04-11 | wire up main entrypoint |
at processTicksAndRejections (node:internal/process/task_queues:95:5) — runtime

Caused by ↓

### IOException
> disk read failed
at readFile (/repo/src/io.ts:22:8) — own
```typescript
  17 | import * as fs from 'fs/promises';
  18 | 
  19 | export async function readFile(path: string): Promise<string> {
  20 |   const buffer = await fs.readFile(path);
  21 | 
→ 22 |   return buffer.toString('utf-8');
  23 | }
  24 | 
  25 | export async function writeFile(path: string, data: string) {
  26 |   await fs.writeFile(path, data);
  27 | }
```
| Lines | Commit | Author | Date | Summary |
| --- | --- | --- | --- | --- |
| 17-27 | 3344556 | vedant | 2026-04-15 | add file IO helpers |
... 6 more frames (shared with enclosing exception)

## Dependencies
- lodash — declared ^4.17.21, resolved 4.17.21

<details><summary>Raw input</summary>

```
RuntimeException: top-level failure
    at run (/repo/src/main.ts:10:5)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)
Caused by: IOException: disk read failed
    at readFile (/repo/src/io.ts:22:8)
    ... 6 more frames (shared with enclosing exception)
```
</details>
