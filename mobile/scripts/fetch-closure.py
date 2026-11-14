#!/usr/bin/env python3
"""Recursively fetch a Maven artifact closure into the local file repo.

Because Java/Gradle cannot do TLS through this sandbox's egress proxy
(curl can), artifacts are pre-fetched with curl into ~/.m2-vyro and
Gradle resolves from there (see ~/.gradle/init.d/local-repo.gradle).

Usage: fetch-closure.py group:artifact:version [group:artifact:version ...]
Only compile/runtime, non-optional dependencies are followed. Artifacts
already present (pom + binary) are skipped. Parent POMs are fetched to
resolve versions and properties.
"""
import os
import re
import subprocess
import sys
import xml.etree.ElementTree as ET

REPO = os.environ.get("M2_VYRO", "/home/hatch/.m2-vyro")
NS = {"m": "http://maven.apache.org/POM/4.0.0"}
SOURCES = [
    "https://dl.google.com/dl/android/maven2",
    "https://repo.maven.apache.org/maven2",
    "https://plugins.gradle.org/m2",
    "https://storage.googleapis.com/download.flutter.io",
    "https://jitpack.io",
]
visited = set()


def gav_path(g, a, v):
    return os.path.join(REPO, g.replace(".", "/"), a, v)


def curl(url, dest):
    r = subprocess.run(
        ["curl", "-fsSL", "--max-time", "120", "-o", dest, url],
        capture_output=True, text=True)
    if r.returncode != 0:
        if os.path.exists(dest):
            os.remove(dest)
        return False
    return True


def fetch_file(g, a, v, filename, bases=None):
    """Fetch one file (pom/jar/aar) from the first source that has it."""
    gp = g.replace(".", "/")
    for base in (bases or SOURCES):
        if curl(f"{base}/{gp}/{a}/{v}/{filename}",
                os.path.join(gav_path(g, a, v), filename)):
            return base
    return None


def get_pom(g, a, v):
    """Return parsed POM root, fetching it (and parents) as needed."""
    d = gav_path(g, a, v)
    pom = os.path.join(d, f"{a}-{v}.pom")
    if not os.path.exists(pom):
        os.makedirs(d, exist_ok=True)
        if not fetch_file(g, a, v, f"{a}-{v}.pom"):
            print(f"  FAIL pom {g}:{a}:{v}")
            return None
    try:
        root = ET.parse(pom).getroot()
    except ET.ParseError:
        print(f"  BAD pom {g}:{a}:{v}")
        return None
    return root


def merge_props(root, props):
    p = root.find("m:properties", NS)
    if p is not None:
        for e in p:
            props.setdefault(e.tag.split("}")[1], e.text or "")


def resolve(value, props):
    if not value:
        return value
    for _ in range(5):
        m = re.search(r"\$\{([^}]+)\}", value)
        if not m:
            break
        value = value.replace(m.group(0), props.get(m.group(1), m.group(0)))
    return value


def pom_chain(g, a, v):
    """Walk parent POMs; return (root, props, depMgmt) merged bottom-up."""
    props, mgmt = {}, {}
    chain = []
    cur = (g, a, v)
    for _ in range(6):
        root = get_pom(*cur)
        if root is None:
            break
        chain.append(root)
        par = root.find("m:parent", NS)
        if par is None:
            break
        cur = (par.findtext("m:groupId", "", NS),
               par.findtext("m:artifactId", "", NS),
               par.findtext("m:version", "", NS))
    for root in reversed(chain):
        merge_props(root, props)
        dm = root.find("m:dependencyManagement/m:dependencies", NS)
        if dm is not None:
            for d in dm.findall("m:dependency", NS):
                key = (d.findtext("m:groupId", "", NS),
                       d.findtext("m:artifactId", "", NS))
                mgmt.setdefault(key, d.findtext("m:version", "", NS))
    # Follow BOM imports (scope=import, type=pom) in dependencyManagement:
    # Gradle needs those POMs to parse the parent POM at all.
    for root in chain:
        dm = root.find("m:dependencyManagement/m:dependencies", NS)
        if dm is None:
            continue
        for d in dm.findall("m:dependency", NS):
            if (d.findtext("m:scope", "", NS) == "import"
                    and d.findtext("m:type", "", NS) == "pom"):
                ig = d.findtext("m:groupId", "", NS)
                ia = d.findtext("m:artifactId", "", NS)
                iv = resolve(d.findtext("m:version", "", NS), props)
                if not iv:
                    iv = mgmt.get((ig, ia), "")
                iv = resolve(iv, props)
                if iv and (ig, ia, iv) not in pom_seen:
                    pom_seen.add((ig, ia, iv))
                    pom_chain(ig, ia, iv)
    return chain[0] if chain else None, props, mgmt


pom_seen = set()


def fetch_artifact(g, a, v, depth=0):
    key = (g, a, v)
    if key in visited:
        return
    visited.add(key)
    pad = "  " * depth
    root, props, mgmt = pom_chain(g, a, v)
    if root is None:
        return
    packaging = root.findtext("m:packaging", "jar", NS) or "jar"
    d = gav_path(g, a, v)
    if packaging not in ("pom",):
        ext = "aar" if packaging == "aar" else "jar"
        f = os.path.join(d, f"{a}-{v}.{ext}")
        if not os.path.exists(f):
            if fetch_file(g, a, v, f"{a}-{v}.{ext}"):
                print(f"{pad}OK {g}:{a}:{v} ({ext})")
            else:
                print(f"{pad}FAIL file {g}:{a}:{v} ({ext})")
    else:
        print(f"{pad}OK {g}:{a}:{v} (pom-only)")
    for dep in root.findall(".//m:dependencies/m:dependency", NS):
        dg = dep.findtext("m:groupId", "", NS)
        da = dep.findtext("m:artifactId", "", NS)
        dv = dep.findtext("m:version", "", NS)
        scope = dep.findtext("m:scope", "compile", NS)
        opt = dep.findtext("m:optional", "", NS)
        if scope not in ("compile", "runtime", "") or opt == "true":
            continue
        if not dv:
            dv = mgmt.get((dg, da), "")
        dv = resolve(dv, props)
        if not dv or dv.startswith("${"):
            print(f"{pad}  SKIP unresolved {dg}:{da} version={dv!r}")
            continue
        fetch_artifact(dg, da, dv, depth + 1)


if __name__ == "__main__":
    for spec in sys.argv[1:]:
        parts = spec.split(":")
        if len(parts) != 3:
            print(f"bad spec: {spec} (want group:artifact:version)")
            continue
        print(f"Fetching closure for {spec}")
        fetch_artifact(*parts)
    print("Done.")
