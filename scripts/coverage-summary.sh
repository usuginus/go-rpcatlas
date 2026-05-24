#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "usage: $0 <go-test-output> <coverage-profile>" >&2
  exit 2
fi

test_output=$1
coverage_profile=$2
total=$(go tool cover -func="$coverage_profile" | awk '/^total:/ { print $3 }')
module=$(go list -m 2>/dev/null || true)

awk -v total="$total" -v module="$module" '
  BEGIN {
    print "## Coverage Report"
    print ""
  }

  /^ok[[:space:]]+/ && match($0, /coverage: [0-9.]+% of statements/) {
    package_name = $2
    display_name = package_name
    prefix = module "/"
    if (module != "" && index(package_name, prefix) == 1) {
      display_name = substr(package_name, length(prefix) + 1)
    }
    coverage = substr($0, RSTART + 10, RLENGTH - 24)
    rows[++count] = "| `" display_name "` | " coverage " |"
  }

  END {
    if (count == 0) {
      print "No coverage data found."
      exit 1
    }

    print "Total: **" total "**"
    print ""
    print "| package | coverage |"
    print "| --- | ---: |"
    for (i = 1; i <= count; i++) {
      print rows[i]
    }
  }
' "$test_output"
