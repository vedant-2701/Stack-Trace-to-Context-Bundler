> This is a stack-trace-bundler bundle: an exception chain with own-code snippets, git blame, and resolved dependency versions, packaged for pasting into an AI chat.
- Language: typescript
- OS: darwin
- Runtime: node 20.11.0
- Git: feature/query\-fix @ 7654321 (uncommitted changes)
- Fingerprint: 73be0bb68b5104a7

### TypeError
> Cannot read properties of undefined \(reading 'id'\)
at handleRequest (/repo/src/handler.ts:27:14) — own
```typescript
  22 | import { Request, Response } from 'express';
  23 | import * as service from './service';
  24 | 
  25 | export function handleRequest(req: Request) {
  26 |   const payload = req.body;
→ 27 |   return service.queryDatabase(payload.id);
  28 | }
  29 | 
  30 | export function handleError(err: Error) {
  31 |   console.error(err);
  32 | }
```
| Lines | Commit | Author | Date | Summary |
| --- | --- | --- | --- | --- |
| 22-32 | 89abcde | vedant | 2026-07-29 | validate request payload before dispatch |
at get (/repo/node_modules/lodash/lodash.js:11812:3) — dependency: lodash
at processTicksAndRejections (node:internal/process/task_queues:95:5) — runtime

Caused by ↓

### Error
> ECONNREFUSED
at queryDatabase (/repo/src/service.ts:63:9) — own
```typescript
  58 | import { Pool } from 'pg';
  59 | 
  60 | const pool = new Pool();
  61 | 
  62 | export async function queryDatabase(id: string) {
→ 63 |   const conn = await pool.connect();
  64 |   return conn.query(id);
  65 | }
  66 | 
  67 | export function closePool(): void {
  68 |   pool.end();
```
| Lines | Commit | Author | Date | Summary |
| --- | --- | --- | --- | --- |
| 58-68 | 7654321 | vedant | 2026-08-01 | add connection pooling for query path |

## Dependencies
- express — declared ^4.18.2, resolved unresolved (no package\-lock\.json entry found for this package)
- lodash — declared ^4.17.21, resolved 4.17.21

<details><summary>Raw input</summary>

```
TypeError: Cannot read properties of undefined (reading 'id')
    at handleRequest (/repo/src/handler.ts:27:14)
    at Object.get (/repo/node_modules/lodash/lodash.js:11812:3)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)
Caused by: Error: ECONNREFUSED
    at queryDatabase (/repo/src/service.ts:63:9)
Node.js v20.11.0
```
</details>
