> This is a stack-trace-bundler bundle: an exception chain with own-code snippets, git blame, and resolved dependency versions, packaged for pasting into an AI chat.
- Language: typescript
- OS: linux
- Runtime: node 20.11.0
- Git: main @ 1234567 (clean)
- Fingerprint: ee33ff44aa11bb22

### Error
> missing repo context
at slugify (/repo/src/utils/slugify.ts:10:3) — own
```typescript
   5 | export function slugify(input: string): string {
   6 |   const trimmed = input.trim();
   7 |   const lower = trimmed.toLowerCase();
   8 | 
   9 |   return lower
→ 10 |     .replace(/[^a-z0-9]+/g, '-')
  11 |     .replace(/^-+|-+$/g, '');
  12 | }
  13 | 
  14 | export function unslugify(slug: string): string {
  15 |   return slug.replace(/-/g, ' ');
```
⚠ no git repository found at this path
at processTicksAndRejections (node:internal/process/task_queues:95:5) — runtime

## Dependencies
- lodash — declared ^4.17.21, resolved 4.17.21

<details><summary>Raw input</summary>

```
Error: missing repo context
    at slugify (/repo/src/utils/slugify.ts:10:3)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)
```
</details>
