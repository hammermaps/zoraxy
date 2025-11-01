#!/bin/bash
# Script to validate Jenkinsfile syntax

set -e

echo "================================================"
echo "Jenkinsfile Validation Script"
echo "================================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to validate a Jenkinsfile
validate_jenkinsfile() {
    local file=$1
    echo -e "${YELLOW}Validating: ${file}${NC}"
    
    if [ ! -f "$file" ]; then
        echo -e "${RED}✗ File not found: ${file}${NC}"
        return 1
    fi
    
    # Check for basic syntax issues
    if grep -q "pipeline {" "$file"; then
        echo -e "${GREEN}✓ Contains pipeline block${NC}"
    else
        echo -e "${RED}✗ No pipeline block found${NC}"
        return 1
    fi
    
    if grep -q "stages {" "$file"; then
        echo -e "${GREEN}✓ Contains stages block${NC}"
    else
        echo -e "${RED}✗ No stages block found${NC}"
        return 1
    fi
    
    # Check for balanced braces
    local open_braces=$(grep -o "{" "$file" | wc -l)
    local close_braces=$(grep -o "}" "$file" | wc -l)
    
    if [ "$open_braces" -eq "$close_braces" ]; then
        echo -e "${GREEN}✓ Balanced braces: ${open_braces} opening, ${close_braces} closing${NC}"
    else
        echo -e "${RED}✗ Unbalanced braces: ${open_braces} opening, ${close_braces} closing${NC}"
        return 1
    fi
    
    # Check for required stages
    if grep -q "stage(" "$file"; then
        local stage_count=$(grep -c "stage(" "$file")
        echo -e "${GREEN}✓ Found ${stage_count} stage(s)${NC}"
    else
        echo -e "${RED}✗ No stages found${NC}"
        return 1
    fi
    
    # Check for agent definition
    if grep -q "agent" "$file"; then
        echo -e "${GREEN}✓ Contains agent definition${NC}"
    else
        echo -e "${YELLOW}⚠ No agent definition found${NC}"
    fi
    
    echo -e "${GREEN}✓ ${file} passed basic validation${NC}"
    echo ""
    return 0
}

# Validate each Jenkinsfile
cd "$(dirname "$0")/.."

echo "Found Jenkinsfiles:"
find . -maxdepth 1 -name "Jenkinsfile*" -type f
echo ""

FAILED=0

for jenkinsfile in Jenkinsfile Jenkinsfile.advanced Jenkinsfile.docker; do
    if validate_jenkinsfile "$jenkinsfile"; then
        echo ""
    else
        FAILED=$((FAILED + 1))
        echo ""
    fi
done

echo "================================================"
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All Jenkinsfiles validated successfully! ✓${NC}"
    echo "================================================"
    exit 0
else
    echo -e "${RED}${FAILED} Jenkinsfile(s) failed validation ✗${NC}"
    echo "================================================"
    exit 1
fi
