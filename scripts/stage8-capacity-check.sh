#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FAILED=0

positive_int() {
  local name="$1"
  local value="$2"
  if ! [[ "$value" =~ ^[0-9]+$ ]] || [ "$value" -le 0 ]; then
    echo "FAIL $name must be a positive integer"
    FAILED=1
  fi
}

non_negative_int() {
  local name="$1"
  local value="$2"
  if ! [[ "$value" =~ ^[0-9]+$ ]]; then
    echo "FAIL $name must be a non-negative integer"
    FAILED=1
  fi
}

detect_cpu_millicores() {
  local cores
  cores="$(getconf _NPROCESSORS_ONLN 2>/dev/null || true)"
  if [[ "$cores" =~ ^[0-9]+$ ]] && [ "$cores" -gt 0 ]; then
    echo $((cores * 1000))
    return
  fi
  echo 2000
}

detect_memory_mb() {
  if [ -r /proc/meminfo ]; then
    awk '/MemTotal:/ { print int($2 / 1024); found=1 } END { if (!found) print 4096 }' /proc/meminfo
    return
  fi
  if command -v sysctl >/dev/null 2>&1; then
    local bytes
    bytes="$(sysctl -n hw.memsize 2>/dev/null || true)"
    if [[ "$bytes" =~ ^[0-9]+$ ]] && [ "$bytes" -gt 0 ]; then
      echo $((bytes / 1024 / 1024))
      return
    fi
  fi
  echo 4096
}

available_disk_mb() {
  local path="$1"
  df -Pk "$path" | awk 'NR == 2 { print int($4 / 1024) }'
}

HI_PROCESS_COUNT="${STAGE8_HI_PROCESS_COUNT:-5}"
DB_MAX_CONNS_VALUE="${DB_MAX_CONNS:-10}"
POSTGRES_MAX_CONNECTIONS="${STAGE8_POSTGRES_MAX_CONNECTIONS:-100}"
POSTGRES_RESERVED_CONNECTIONS="${STAGE8_POSTGRES_RESERVED_CONNECTIONS:-20}"

CPU_CAPACITY_MILLICORES="${STAGE8_CAPACITY_CPU_MILLICORES:-$(detect_cpu_millicores)}"
CPU_RESERVED_MILLICORES="${STAGE8_RESERVED_CPU_MILLICORES:-0}"
CPU_PER_HI_PROCESS_MILLICORES="${STAGE8_HI_PROCESS_CPU_MILLICORES:-100}"

MEMORY_CAPACITY_MB="${STAGE8_CAPACITY_MEMORY_MB:-$(detect_memory_mb)}"
MEMORY_RESERVED_MB="${STAGE8_RESERVED_MEMORY_MB:-0}"
MEMORY_PER_HI_PROCESS_MB="${STAGE8_HI_PROCESS_MEMORY_MB:-128}"

CAPACITY_PATH="${STAGE8_CAPACITY_PATH:-$ROOT_DIR}"
MIN_FREE_DISK_MB="${STAGE8_MIN_FREE_DISK_MB:-1024}"
BACKUP_DAILY_ESTIMATE_MB="${STAGE8_BACKUP_DAILY_ESTIMATE_MB:-64}"
BACKUP_RETENTION_DAYS="${STAGE8_BACKUP_RETENTION_DAYS:-7}"

positive_int STAGE8_HI_PROCESS_COUNT "$HI_PROCESS_COUNT"
positive_int DB_MAX_CONNS "$DB_MAX_CONNS_VALUE"
positive_int STAGE8_POSTGRES_MAX_CONNECTIONS "$POSTGRES_MAX_CONNECTIONS"
non_negative_int STAGE8_POSTGRES_RESERVED_CONNECTIONS "$POSTGRES_RESERVED_CONNECTIONS"
positive_int STAGE8_CAPACITY_CPU_MILLICORES "$CPU_CAPACITY_MILLICORES"
non_negative_int STAGE8_RESERVED_CPU_MILLICORES "$CPU_RESERVED_MILLICORES"
positive_int STAGE8_HI_PROCESS_CPU_MILLICORES "$CPU_PER_HI_PROCESS_MILLICORES"
positive_int STAGE8_CAPACITY_MEMORY_MB "$MEMORY_CAPACITY_MB"
non_negative_int STAGE8_RESERVED_MEMORY_MB "$MEMORY_RESERVED_MB"
positive_int STAGE8_HI_PROCESS_MEMORY_MB "$MEMORY_PER_HI_PROCESS_MB"
positive_int STAGE8_MIN_FREE_DISK_MB "$MIN_FREE_DISK_MB"
positive_int STAGE8_BACKUP_DAILY_ESTIMATE_MB "$BACKUP_DAILY_ESTIMATE_MB"
positive_int STAGE8_BACKUP_RETENTION_DAYS "$BACKUP_RETENTION_DAYS"

if [ "$FAILED" -ne 0 ]; then
  exit 1
fi

postgres_app_budget=$((POSTGRES_MAX_CONNECTIONS - POSTGRES_RESERVED_CONNECTIONS))
postgres_required=$((DB_MAX_CONNS_VALUE * HI_PROCESS_COUNT))
if [ "$postgres_app_budget" -le 0 ] || [ "$postgres_required" -gt "$postgres_app_budget" ]; then
  echo "FAIL postgres connection budget: required=$postgres_required budget=$postgres_app_budget"
  FAILED=1
fi

cpu_budget=$((CPU_CAPACITY_MILLICORES - CPU_RESERVED_MILLICORES))
cpu_required=$((CPU_PER_HI_PROCESS_MILLICORES * HI_PROCESS_COUNT))
if [ "$cpu_budget" -le 0 ] || [ "$cpu_required" -gt "$cpu_budget" ]; then
  echo "FAIL cpu budget: required_millicores=$cpu_required budget_millicores=$cpu_budget"
  FAILED=1
fi

memory_budget=$((MEMORY_CAPACITY_MB - MEMORY_RESERVED_MB))
memory_required=$((MEMORY_PER_HI_PROCESS_MB * HI_PROCESS_COUNT))
if [ "$memory_budget" -le 0 ] || [ "$memory_required" -gt "$memory_budget" ]; then
  echo "FAIL memory budget: required_mb=$memory_required budget_mb=$memory_budget"
  FAILED=1
fi

if [ ! -d "$CAPACITY_PATH" ]; then
  echo "FAIL STAGE8_CAPACITY_PATH must be an existing directory"
  FAILED=1
else
  disk_available_mb="$(available_disk_mb "$CAPACITY_PATH")"
  disk_required_mb=$((MIN_FREE_DISK_MB + BACKUP_DAILY_ESTIMATE_MB * BACKUP_RETENTION_DAYS))
  if [ "$disk_available_mb" -lt "$disk_required_mb" ]; then
    echo "FAIL disk budget: available_mb=$disk_available_mb required_mb=$disk_required_mb path=$CAPACITY_PATH"
    FAILED=1
  fi
fi

if [ "$FAILED" -ne 0 ]; then
  exit 1
fi

echo "stage8 capacity check passed: hi_processes=$HI_PROCESS_COUNT postgres_required=$postgres_required postgres_budget=$postgres_app_budget cpu_required_millicores=$cpu_required cpu_budget_millicores=$cpu_budget memory_required_mb=$memory_required memory_budget_mb=$memory_budget retention_days=$BACKUP_RETENTION_DAYS"
