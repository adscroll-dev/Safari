#!/usr/bin/env python3
"""Normalize poms in the local sandbox Maven repo (~/.m2-vyro).

Problem: some Kotlin poms declare the same dependency several times in
one <dependencies> block (once required, then again with
<optional>true</optional>). Maven uses the first declaration and ignores
the rest; Gradle merges them and treats the dependency as optional, so
the jar is silently dropped from plugin classpaths and the build later
fails with NoClassDefFoundError (e.g. KotlinBasePlugin).

Fix: within each top-level <dependencies> block (not inside
<dependencyManagement>), keep the FIRST declaration of a
(groupId, artifactId) and drop later duplicates that are identical
except for the <optional> flag. Only the local repo copies are touched;
upstream poms are never modified. Safe to re-run (idempotent).
"""
import re
from pathlib import Path

REPO = Path.home() / ".m2-vyro"
DEP_RE = re.compile(r"<dependency>.*?</dependency>", re.S)


def dep_key(dep: str):
    g = re.search(r"<groupId>(.*?)</groupId>", dep)
    a = re.search(r"<artifactId>(.*?)</artifactId>", dep)
    return (g.group(1) if g else "", a.group(1) if a else "")


def strip_optional(dep: str) -> str:
    return re.sub(r"\s*<optional>.*?</optional>", "", dep, flags=re.S)


def normalize_block(block_inner: str):
    """Dedupe <dependency> entries inside one <dependencies> block.
    Returns (new_inner, dropped_count)."""
    deps = DEP_RE.findall(block_inner)
    if len(deps) <= 1:
        return block_inner, 0
    seen = {}
    keep = set()
    dropped = 0
    for idx, dep in enumerate(deps):
        key = dep_key(dep)
        if key in seen:
            if strip_optional(dep) == strip_optional(seen[key]):
                dropped += 1
                continue
        else:
            seen[key] = dep
        keep.add(idx)
    if not dropped:
        return block_inner, 0
    kept = [d for i, d in enumerate(deps) if i in keep]
    # Rebuild: replace dependency elements in order, dropping removed ones.
    it = iter(kept)
    out = []
    pos = 0
    for m in DEP_RE.finditer(block_inner):
        out.append(block_inner[pos:m.start()])
        try:
            out.append(next(it))
        except StopIteration:
            pass  # dropped duplicate: emit nothing for this element
        pos = m.end()
    out.append(block_inner[pos:])
    return "".join(out), dropped


def normalize_pom(path: Path) -> int:
    text = path.read_text(errors="replace")
    total = 0
    # Mask <dependencyManagement>...</dependencyManagement> so its inner
    # <dependencies> block is left alone.
    masked = {}

    def mask(m):
        key = f"\x00MASK{len(masked)}\x00"
        masked[key] = m.group(0)
        return key

    tmp = re.sub(r"<dependencyManagement>.*?</dependencyManagement>", mask, text, flags=re.S)

    def repl(m):
        nonlocal total
        new_inner, n = normalize_block(m.group(1))
        total += n
        return "<dependencies>" + new_inner + "</dependencies>"

    tmp = re.sub(r"<dependencies>(.*?)</dependencies>", repl, tmp, flags=re.S)
    if total == 0:
        return 0
    for key, orig in masked.items():
        tmp = tmp.replace(key, orig)
    path.write_text(tmp)
    return total


def main():
    total_files = 0
    total_dropped = 0
    for pom in sorted(REPO.rglob("*.pom")):
        n = normalize_pom(pom)
        if n:
            total_files += 1
            total_dropped += n
            print(f"normalized {pom.relative_to(REPO)}: dropped {n} duplicate optional dep(s)")
    print(f"done: {total_files} pom(s) changed, {total_dropped} duplicate(s) dropped")


if __name__ == "__main__":
    main()
