#!/usr/bin/env python3
"""Keep OpenAPI component IDs stable across Go module major versions.

Swag derives definition names from Go import paths. IAM's Go module moved through /v3 and /v4,
but its public REST/OpenAPI contract remains v2. Rewriting only the encoded module
prefix prevents an implementation-level Go module change from renaming every
published OpenAPI component.
"""

from pathlib import Path
import re


ROOT = Path(__file__).resolve().parent.parent
TARGET_FILES = (
    ROOT / "internal/apiserver/docs/docs.go",
    ROOT / "internal/apiserver/docs/swagger.json",
    ROOT / "internal/apiserver/docs/swagger.yaml",
    ROOT / "api/rest/authn.v2.yaml",
    ROOT / "api/rest/authz.v3.yaml",
    ROOT / "api/rest/identity.v2.yaml",
    ROOT / "api/rest/idp.v2.yaml",
    ROOT / "api/rest/suggest.v2.yaml",
)
GENERATED_PREFIX = re.compile(r"github_com_FangcunMount_iam_v[34]_")
STABLE_WIRE_PREFIX = "github_com_FangcunMount_iam_v2_"


def main() -> int:
    replacements = 0
    for path in TARGET_FILES:
        source = path.read_text(encoding="utf-8")
        count = len(GENERATED_PREFIX.findall(source))
        if count == 0:
            continue
        path.write_text(GENERATED_PREFIX.sub(STABLE_WIRE_PREFIX, source), encoding="utf-8")
        replacements += count
    print(f"Normalized {replacements} Swagger schema references to the stable REST v2 prefix.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
