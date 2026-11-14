#!/bin/bash
# Fetch a Maven artifact (pom + jar/aar) with curl into the local
# file-based Maven repo at ~/.m2-vyro, because Java/Gradle cannot do
# TLS through this sandbox's egress proxy (curl can).
# Usage: fetch-maven.sh group:artifact:version[:classifier] [...]
# The classifier form fetches $artifact-$version-$classifier.$ext
# (e.g. semver4j-0.16.4-nodeps.jar, required by kotlin-gradle-plugin).
# The repo layout is the standard Maven layout, so Gradle can use it
# as maven { url = uri("file:///home/hatch/.m2-vyro") }.
set -u
REPO="${M2_VYRO:-/home/hatch/.m2-vyro}"
export https_proxy="http://hatch-runtime:957a031631744c70ac6df3fb0161b4bc@hatch-egress-proxy:3128"
export http_proxy="$https_proxy"

fetch_one() {
  local gav="$1"
  local group="${gav%%:*}" rest="${gav#*:}"
  local artifact="${rest%%:*}" rest2="${rest#*:}"
  local version="${rest2%%:*}" classifier="${rest2#*:}"
  [ "$classifier" = "$version" ] && classifier=""
  local gpath="${group//.//}"
  local dir="$REPO/$gpath/$artifact/$version"
  mkdir -p "$dir"
  local base="https://repo.maven.apache.org/maven2/$gpath/$artifact/$version/$artifact-$version"
  local google="https://dl.google.com/dl/android/maven2/$gpath/$artifact/$version/$artifact-$version"
  # Third fallback: Flutter's own Maven repo (engine embedding artifacts
  # such as io.flutter:flutter_embedding_release live only here).
  local flutter="https://storage.googleapis.com/download.flutter.io/$gpath/$artifact/$version/$artifact-$version"
  # Fourth fallback: JitPack (git-based artifacts such as
  # com.github.davidliu:audioswitch, used by flutter_webrtc).
  local jitpack="https://jitpack.io/$gpath/$artifact/$version/$artifact-$version"
  # POM first (decides packaging, and Gradle needs it to resolve).
  if [ ! -f "$dir/$artifact-$version.pom" ]; then
    if ! curl -fsSL --max-time 60 -o "$dir/$artifact-$version.pom" "$base.pom" 2>/dev/null; then
      if ! curl -fsSL --max-time 60 -o "$dir/$artifact-$version.pom" "$google.pom" 2>/dev/null; then
        if ! curl -fsSL --max-time 60 -o "$dir/$artifact-$version.pom" "$flutter.pom" 2>/dev/null; then
          curl -fsSL --max-time 60 -o "$dir/$artifact-$version.pom" "$jitpack.pom" || {
            echo "FAIL pom $gav"; rm -f "$dir/$artifact-$version.pom"; return 1;
          }
        fi
      fi
    fi
  fi
  local packaging
  packaging=$(grep -oPm1 "(?<=<packaging>)[^<]+" "$dir/$artifact-$version.pom" || echo jar)
  # A classifier artifact is always a real binary (e.g. aapt2's
  # -linux.jar) even when the base pom declares packaging "pom".
  if [ -z "$classifier" ] && { [ -z "$packaging" ] || [ "$packaging" = "pom" ]; }; then
    echo "OK $gav (pom-only)"
    return 0
  fi
  # With a classifier, the file extension is jar unless the pom says aar.
  local ext="jar"
  [ "$packaging" = "aar" ] && ext="aar"
  local file="$artifact-$version"
  [ -n "$classifier" ] && file="$file-$classifier"
  if [ ! -f "$dir/$file.$ext" ]; then
    if ! curl -fsSL --max-time 120 -o "$dir/$file.$ext" "$base${classifier:+-$classifier}.$ext" 2>/dev/null; then
      if ! curl -fsSL --max-time 120 -o "$dir/$file.$ext" "$google${classifier:+-$classifier}.$ext" 2>/dev/null; then
        if ! curl -fsSL --max-time 120 -o "$dir/$file.$ext" "$flutter${classifier:+-$classifier}.$ext" 2>/dev/null; then
          curl -fsSL --max-time 120 -o "$dir/$file.$ext" "$jitpack${classifier:+-$classifier}.$ext" || {
            echo "FAIL $ext $gav"; rm -f "$dir/$file.$ext"; return 1;
          }
        fi
      fi
    fi
  fi
  echo "OK $gav ($ext)"
}

for gav in "$@"; do fetch_one "$gav"; done
