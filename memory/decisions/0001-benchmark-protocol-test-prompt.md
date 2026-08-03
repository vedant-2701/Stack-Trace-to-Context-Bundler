You are benchmarking Maven and Gradle's offline-mode dependency resolution
behavior to determine whether a hard timeout, combined with --offline, is
sufficient to prevent a CLI tool from ever hanging when shelling out to
these build tools. Report real measured data — do not estimate or guess.

SAFETY — READ FIRST:
Wrap every single command in an external kill switch so a genuine hang
doesn't freeze your own session: `timeout 60s <command>` for offline-mode
tests, `timeout 180s <command>` for the two online/network-blocked
baseline tests (M6, G7). Time every run with `time` or equivalent. Run
each scenario 3 times minimum and report min/median/max, not one sample.
For each run, record: exact command, exit code, last ~20 lines of
stdout+stderr, and `mvn -version` / `gradle --version` / `java -version` /
`uname -a`.

TEST PROJECT:
Don't use a trivial single-module hello-world. Generate synthetic
multi-module Gradle projects at 3 sizes — 1 module, 10 modules, 30
modules — each with 2-3 common plugins applied (java, checkstyle, a
shadow/fat-jar plugin). Configuration-phase cost scales with module/plugin
count, so a trivial project understates real-world cost.

MAVEN SCENARIOS:
M1. rm -rf ~/.m2/repository, then `timeout 60 mvn -o dependency:tree`.
    Expect a plugin-prefix failure. This confirms "plugin itself uncached"
    is a distinct failure mode from "target dependency uncached."
M2. Maven previously used successfully for a DIFFERENT project (plugin
    cached) but not this one's dependencies. Isolates tool-cold from
    target-cold.
M3. Fully warm (one prior successful online run), then repeat offline.
M4. Multi-module reactor, warm — does per-module cost scale meaningfully
    for Maven the way it does for Gradle?
M5. A dependency whose version comes from a parent POM / BOM, not a
    literal string — does offline resolution behave differently?
M6. ONLINE mode (no -o), network blackholed (unreachable proxy via
    -DproxyHost=10.255.255.1 -DproxyPort=1, or OS-level egress block).
    Wrap in timeout 180. Document how long it takes to fail and how.

GRADLE SCENARIOS:
G1. rm -rf ~/.gradle/caches/, then `timeout 60 ./gradlew --offline
    :app:dependencies`. Repeat 3x per project size above.
G2. CRITICAL — the untested gap: rm -rf ~/.gradle/wrapper/dists AND block
    network at the OS/firewall level (not just via the flag), then
    `timeout 60 ./gradlew --offline :app:dependencies`. The wrapper script
    downloads its Gradle distribution before Gradle itself starts, which
    --offline cannot prevent. Record whether this fails fast or hangs
    until the external timeout kills it.
G3. Fully warm (daemon up, caches warm).
G4. `./gradlew --stop` immediately before running, cache otherwise warm —
    isolates daemon-startup cost from configuration-phase cost.
G5. A dependency declared as a dynamic/range version (e.g. `1.+` or
    `[1.0,2.0)`), offline, everything else warm. Known Gradle limitation
    (gradle/gradle#10934) — record the actual error text.
G6. Truncate a cached jar under ~/.gradle/caches/modules-2 (`truncate -s
    100 <jar>`), run offline. Does Gradle fail fast on corruption or
    attempt a network repair (defeating offline mode)?
G7. ONLINE mode, network blackholed, timeout 180. Same purpose as M6.

OUT OF SCOPE for this pass: JAVA_HOME misconfiguration, plugin-portal
failures beyond G5/G6, Defender on/off A-B testing unless Windows is
actually available to you.

REPORT FORMAT — one row per scenario:
| Scenario | Real time (min/median/max of 3) | Exit code | Outcome | Did it touch the network? (yes/no + how you verified — "egress hard-blocked at OS level" counts, "the flag says offline" doesn't) |

For G2, M6, and G7 specifically, add one line: did the command exit on its
own, or did the external timeout have to kill it?

Run on whatever OS you have. If you only have Linux, say so explicitly
rather than presenting Linux numbers as if they cover Windows/macOS too —
JVM/Maven/Gradle cold-start timing is known to vary significantly across
platforms, especially Windows under antivirus real-time scanning.