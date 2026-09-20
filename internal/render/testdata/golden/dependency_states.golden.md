> This is a stack-trace-bundler bundle: an exception chain with own-code snippets, git blame, and resolved dependency versions, packaged for pasting into an AI chat.
- Language: typescript
- OS: linux
- Runtime: node 20.11.0
- Git: main @ 8899001 (clean)
- Fingerprint: ff77aa88bb99cc00

### Error
> dependency resolution issue
at bootstrap (/repo/src/app.ts:5:3) — own
```typescript
   1 | import React from 'react';
   2 | import _ from 'lodash';
   3 | 
   4 | export function bootstrap() {
→  5 |   const config = _.merge({}, defaults, overrides);
   6 |   return config;
   7 | }
   8 | 
   9 | export function teardown() {
  10 |   console.log('shutting down');
  11 |   return true;
```
| Lines | Commit | Author | Date | Summary |
| --- | --- | --- | --- | --- |
| 1-11 | 8899001 | vedant | 2026-01-22 | wire up app bootstrap |
at processTicksAndRejections (node:internal/process/task_queues:95:5) — runtime

## Dependencies
- leftpad — resolved unresolved (no package\-lock\.json entry found for this package)
- lodash — resolved 4.17.21 (resolved via top\-level lookup, not tied to this exact frame path)
- react — declared ^18.2.0, resolved 18.2.5

<details><summary>Raw input</summary>

```
Error: dependency resolution issue
    at bootstrap (/repo/src/app.ts:5:3)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)
```
</details>
