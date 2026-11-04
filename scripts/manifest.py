#!/usr/bin/env python3
"""Read an extension's manifest: extensions/<id>/persona-extension.json (persona-plugin.json before the rename).

  scripts/manifest.py path <id>       the manifest's path
  scripts/manifest.py version <id>    its version
  scripts/manifest.py ids             every extension in extensions/
  scripts/manifest.py check           every manifest is sound (what CI runs)
  scripts/manifest.py changed <ref>   the extensions whose version is not what it was at <ref> (new ones too)
"""
import json, os, re, subprocess, sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
NEW, OLD = "persona-extension.json", "persona-plugin.json"
SEMVER = re.compile(r"^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$")


def ids():
    d = os.path.join(ROOT, "extensions")
    return sorted(e for e in os.listdir(d) if os.path.isdir(os.path.join(d, e))) if os.path.isdir(d) else []


def path(id):
    for name in (NEW, OLD):
        p = os.path.join(ROOT, "extensions", id, name)
        if os.path.exists(p):
            return p
    sys.exit("extensions/%s has no %s" % (id, NEW))


def load(id):
    return json.load(open(path(id)))


def legacy(id):
    """The manifest for hubs that read only persona-plugin.json (nwverse/persona#1844)."""
    p = os.path.join(ROOT, "extensions", id, OLD)
    return json.load(open(p)) if os.path.exists(p) else load(id)


def version_at(ref, id):
    """The version at a git ref, or None when the extension wasn't there (also under its pre-monorepo path)."""
    for p in ("extensions/%s/%s" % (id, NEW), "extensions/%s/%s" % (id, OLD), "%s/%s" % (id, OLD)):
        r = subprocess.run(["git", "-C", ROOT, "show", "%s:%s" % (ref, p)], capture_output=True, text=True)
        if r.returncode == 0:
            return json.loads(r.stdout).get("version")
    return None


def check():
    bad = []
    for id in ids():
        m = load(id)
        if m.get("id") != id:
            bad.append("extensions/%s: the manifest's id is %r, not the directory's name" % (id, m.get("id")))
        if not SEMVER.match(str(m.get("version", ""))):
            bad.append("extensions/%s: version %r is not MAJOR.MINOR.PATCH" % (id, m.get("version")))
        old = os.path.join(ROOT, "extensions", id, OLD)
        if os.path.exists(os.path.join(ROOT, "extensions", id, NEW)) and os.path.exists(old) and json.load(open(old)) != m:
            bad.append("extensions/%s: %s and %s differ (keep them the same until the old name goes)" % (id, NEW, OLD))
    for b in bad:
        print(b, file=sys.stderr)
    return not bad


if __name__ == "__main__":
    cmd = sys.argv[1] if len(sys.argv) > 1 else ""
    if cmd == "path":
        print(path(sys.argv[2]))
    elif cmd == "version":
        print(load(sys.argv[2])["version"])
    elif cmd == "ids":
        print("\n".join(ids()))
    elif cmd == "changed":
        print("\n".join(id for id in ids() if version_at(sys.argv[2], id) != load(id)["version"]))
    elif cmd == "check":
        sys.exit(0 if check() else 1)
    else:
        sys.exit(__doc__)
