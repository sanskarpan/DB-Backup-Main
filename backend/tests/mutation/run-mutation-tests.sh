#!/bin/bash
# Mutation Testing Runner for db-backup
# Requires: go-mutesting (github.com/zimmski/go-mutesting)

set -e

echo "======================================"
echo "DB Backup Mutation Testing"
echo "======================================"

# Install go-mutesting if not present
if ! command -v go-mutesting &> /dev/null; then
    echo "Installing go-mutesting..."
    go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest
fi

# Create reports directory
mkdir -p tests/mutation/reports

# Run mutation tests
echo "Running mutation tests..."
go-mutesting \
    --verbose \
    --timeout 30m \
    --parallel 4 \
    --exclude="*_test.go" \
    --exclude="mock_*.go" \
    --output tests/mutation/reports/mutation-report.txt \
    ./internal/...

# Generate HTML report
echo "Generating HTML report..."
go-mutesting-report \
    --input tests/mutation/reports/mutation-report.txt \
    --output tests/mutation/reports/mutation-report.html

# Calculate mutation score
total_mutations=$(grep -c "PASSED\|FAILED" tests/mutation/reports/mutation-report.txt || true)
killed_mutations=$(grep -c "FAILED" tests/mutation/reports/mutation-report.txt || true)

if [ "$total_mutations" -gt 0 ]; then
    mutation_score=$(awk "BEGIN {printf \"%.2f\", ($killed_mutations/$total_mutations)*100}")
    echo "======================================"
    echo "Mutation Testing Results:"
    echo "Total Mutations: $total_mutations"
    echo "Killed Mutations: $killed_mutations"
    echo "Mutation Score: $mutation_score%"
    echo "======================================"

    # Check threshold
    threshold=80
    if (( $(echo "$mutation_score < $threshold" | bc -l) )); then
        echo "ERROR: Mutation score ($mutation_score%) is below threshold ($threshold%)"
        exit 1
    fi

    echo "SUCCESS: Mutation score meets threshold"
else
    echo "WARNING: No mutations found"
fi

echo "Report available at: tests/mutation/reports/mutation-report.html"
