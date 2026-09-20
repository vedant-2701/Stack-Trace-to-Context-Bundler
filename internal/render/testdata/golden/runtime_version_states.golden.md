> This is a stack-trace-bundler bundle: an exception chain with own-code snippets, git blame, and resolved dependency versions, packaged for pasting into an AI chat.
- Language: java
- OS: linux
- Runtime: jvm (version unknown)
- Git: main @ 9900112 (clean)
- Fingerprint: aa88bb99cc00dd11

### java.lang.RuntimeException
> unsupported configuration detected
at Loader.load (src/main/java/com/example/Loader.java:9) — own
```java
   4 | public class Loader {
   5 | 
   6 |   private final Config config;
   7 | 
   8 |   public void load() {
→  9 |     if (config == null) {
  10 |       throw new RuntimeException("unsupported configuration detected");
  11 |     }
  12 |   }
  13 | 
  14 |   public Loader(Config config) {
```
| Lines | Commit | Author | Date | Summary |
| --- | --- | --- | --- | --- |
| 4-14 | 9900112 | vedant | 2026-01-25 | add config loader |
at run (java.base/java.lang.Thread:840) — runtime

## Dependencies
- com.example:config-lib — declared 1.2.0, resolved 1.2.0

<details><summary>Raw input</summary>

```
java.lang.RuntimeException: unsupported configuration detected
	at com.example.Loader.load(Loader.java:9)
	at java.base/java.lang.Thread.run(Thread.java:840)
```
</details>
