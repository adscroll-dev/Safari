#!/bin/bash
# Iterate the Android release build, fetching each missing Maven
# artifact with curl (see fetch-maven.sh) until the build succeeds
# or no new artifact can be found. Run from mobile/android.
# Usage: prime-build.sh [max-iterations]
set -u
cd "$(dirname "$0")/../android" || exit 1
# JDK lives in ~/java (Temurin 17, downloaded with curl) because the
# system JDK under /usr/lib/jvm does NOT survive VM restarts here.
export JAVA_HOME="${JAVA_HOME:-$(echo ~/java/jdk-17*/bin/java | xargs dirname | xargs dirname)}"
export PATH="$JAVA_HOME/bin:$PATH"
export GRADLE_USER_HOME=/home/hatch/.gradle
export GRADLE_OPTS="-Djava.net.preferIPv4Stack=true"
export JAVA_OPTS="-Djava.net.preferIPv4Stack=true"
# Reaches EVERY JVM the build forks (client, daemon AND worker
# daemons): without it the worker daemon cannot talk to the Gradle
# daemon over loopback and fails with "Never received a connection".
export JAVA_TOOL_OPTIONS="-Djava.net.preferIPv4Stack=true"
MAX="${1:-40}"
for i in $(seq 1 "$MAX"); do
  echo "=== iteration $i ==="
  OUT=$(./gradlew assembleRelease --no-daemon --offline 2>&1)
  if echo "$OUT" | grep -q "BUILD SUCCESSFUL"; then
    echo "$OUT" | tail -5
    echo "BUILD SUCCEEDED on iteration $i"
    exit 0
  fi
  # Missing artifacts look like:
  #   Could not resolve com.android.tools.build:gradle:7.3.0.
  #   Could not find com.example:thing:1.0.
  MISSING=$(echo "$OUT" | grep -oE "Could not (resolve|find) [A-Za-z0-9._-]+:[A-Za-z0-9._-]+:[0-9][A-Za-z0-9._-]*" \
    | sed -E 's/Could not (resolve|find) //; s/\.$//' | sort -u)
  # Classifier artifacts look like:
  #   Could not find semver4j-0.16.4-nodeps.jar (com.github.gundy:semver4j:0.16.4).
  # The classifier is the filename remainder after <artifact>-<version>-.
  CLASSIFIER_MISSING=$(echo "$OUT" | grep -oE "Could not find [A-Za-z0-9._-]+\.jar \([A-Za-z0-9._-]+:[A-Za-z0-9._-]+:[0-9][A-Za-z0-9._-]*\)" | sort -u)
  if [ -n "$CLASSIFIER_MISSING" ]; then
    echo "$CLASSIFIER_MISSING" | while IFS= read -r line; do
      file=$(echo "$line" | sed -E 's/Could not find ([^ ]+).*/\1/')
      gav=$(echo "$line" | sed -E 's/.*\(([^)]+)\).*/\1/')
      group="${gav%%:*}"; rest="${gav#*:}"
      artifact="${rest%%:*}"; version="${rest##*:}"
      classifier="${file#$artifact-$version-}"
      classifier="${classifier%.jar}"
      if [ "$classifier" != "$file" ] && [ -n "$classifier" ]; then
        echo "missing (classifier): $gav:$classifier"
        ../scripts/fetch-maven.sh "$group:$artifact:$version:$classifier" || true
      fi
    done
    python3 ../scripts/normalize-poms.py >/dev/null 2>&1 || true
    continue
  fi
  if [ -z "$MISSING" ]; then
    echo "$OUT" | grep -E "What went wrong|Could not|FAILED|error" | head -20
    echo "NO PARSEABLE MISSING ARTIFACT — stopping"
    exit 1
  fi
  echo "missing: $MISSING" | tr '\n' ' '; echo
  # shellcheck disable=SC2086
  ../scripts/fetch-maven.sh $MISSING || true
  # Some Kotlin poms repeat a dependency as <optional>true</optional>;
  # Gradle then drops the jar and the build dies with NoClassDefFoundError.
  python3 ../scripts/normalize-poms.py >/dev/null 2>&1 || true
done
echo "GAVE UP after $MAX iterations"
exit 1
