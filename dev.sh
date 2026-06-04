#!/usr/bin/env bash
set -e

ROOT="$(cd "$(dirname "$0")" && pwd)"
BACKEND="$ROOT/backend"
FRONTEND="$ROOT/frontend"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

cleanup() {
  echo -e "\n${YELLOW}Stopping servers...${NC}"
  kill "$BACKEND_PID" "$FRONTEND_PID" 2>/dev/null || true
  wait "$BACKEND_PID" "$FRONTEND_PID" 2>/dev/null || true
  echo -e "${GREEN}Done.${NC}"
}
trap cleanup EXIT INT TERM

# Check dependencies
command -v go >/dev/null 2>&1 || { echo -e "${RED}Error: go not found${NC}"; exit 1; }
command -v node >/dev/null 2>&1 || { echo -e "${RED}Error: node not found${NC}"; exit 1; }
command -v npm >/dev/null 2>&1 || { echo -e "${RED}Error: npm not found${NC}"; exit 1; }

echo -e "${GREEN}Installing frontend dependencies...${NC}"
(cd "$FRONTEND" && npm install --silent)

echo -e "${GREEN}Starting backend on :7070...${NC}"
(cd "$BACKEND" && DEV=1 go run . ) &
BACKEND_PID=$!

echo -e "${GREEN}Starting frontend on :5173...${NC}"
(cd "$FRONTEND" && npm run dev) &
FRONTEND_PID=$!

echo -e "${GREEN}Dev servers running. Open http://localhost:5173${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop.${NC}"

wait "$BACKEND_PID" "$FRONTEND_PID"
