#!/usr/bin/env bash
# Regenerates the REMS Go client from the live Swagger spec.
# Usage: ./gen.sh

set -euo pipefail

SWAGGER_URL="https://rems.test.biocommons.org.au/swagger.json"

echo "1/3 Downloading Swagger spec..."
curl -sSL "$SWAGGER_URL" -o swagger.json

echo "2/3 Converting Swagger 2.0 to OpenAPI 3.0..."
swagger2openapi swagger.json --outfile openapi3.json --patch --warnOnly

echo "3/3 Generating Go client..."
oapi-codegen -config ./oapi-codegen.yaml openapi3.json

echo "Done."
