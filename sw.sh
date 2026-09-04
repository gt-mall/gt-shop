#!/bin/bash
cd "$(dirname "$0")/admin-web" && \
npx -y swagger-typescript-api@13.12.0 generate \
  --axios \
  --unwrap-response-data \
  --templates ../scripts/swagger-templates \
  -p ../admin-server/docs/swagger.json \
  -o ./src/framework/http \
  -n generated_api.ts 
#   --custom-config ../scripts/swagger-typescript-api.config.cjs
