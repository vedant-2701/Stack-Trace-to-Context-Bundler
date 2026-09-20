> This is a stack-trace-bundler bundle: an exception chain with own-code snippets, git blame, and resolved dependency versions, packaged for pasting into an AI chat.
- Language: java
- OS: linux
- Runtime: jvm 17.0.9 (inferred from local \`java \-version\`)
- Git: main @ aabbccd (clean)
- Fingerprint: ee55ff66aa77bb88

### java.lang.NullPointerException
> Cannot invoke "String\.length\(\)" because "name" is null
at Greeter.greet (src/main/java/com/example/Greeter.java:18) — own
```java
  13 | public class Greeter {
  14 | 
  15 |   private final String prefix;
  16 | 
  17 |   public String greet(String name) {
→ 18 |     return prefix + name.length();
  19 |   }
  20 | 
  21 |   public Greeter(String prefix) {
  22 |     this.prefix = prefix;
  23 |   }
```
| Lines | Commit | Author | Date | Summary |
| --- | --- | --- | --- | --- |
| 13-23 | aabbccd | vedant | 2026-05-02 | add greeter service |
at run (java.base/java.lang.Thread:840) — runtime

<details><summary>Raw input</summary>

```
java.lang.NullPointerException: Cannot invoke "String.length()" because "name" is null
	at com.example.Greeter.greet(Greeter.java:18)
	at java.base/java.lang.Thread.run(Thread.java:840)
```
</details>
