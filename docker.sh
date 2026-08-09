#!/usr/bin/env bash

set -euo pipefail

MYSQL_CONTAINER="taoxi-mysql"
MYSQL_ROOT_PASSWORD="123"
MYSQL_GAME_DATABASE="taoxi-game"
MYSQL_USER_DATABASE="taoxi-user"

MONGO_CONTAINER="taoxi-mongo"
MONGO_ROOT_USERNAME="root"
MONGO_ROOT_PASSWORD="123"
MONGO_DATABASE="taoxi"

REDIS_CONTAINER="taoxi-redis"

start_existing_container() {
  local container_name="$1"

  if docker container inspect "${container_name}" >/dev/null 2>&1; then
    if [ "$(docker container inspect -f '{{.State.Running}}' "${container_name}")" != "true" ]; then
      docker start "${container_name}" >/dev/null
    fi
    return 0
  fi

  return 1
}

wait_for_mysql() {
  local retry
  for retry in $(seq 1 60); do
    if docker exec "${MYSQL_CONTAINER}" mysqladmin ping \
      -h127.0.0.1 \
      -uroot \
      "-p${MYSQL_ROOT_PASSWORD}" \
      --silent >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  echo "MySQL 启动超时" >&2
  return 1
}

wait_for_mongo() {
  local retry
  for retry in $(seq 1 60); do
    if docker exec "${MONGO_CONTAINER}" mongosh \
      --quiet \
      --username "${MONGO_ROOT_USERNAME}" \
      --password "${MONGO_ROOT_PASSWORD}" \
      --authenticationDatabase admin \
      --eval 'db.runCommand({ ping: 1 }).ok' >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  echo "MongoDB 启动超时" >&2
  return 1
}

wait_for_redis() {
  local retry
  for retry in $(seq 1 60); do
    if docker exec "${REDIS_CONTAINER}" redis-cli ping 2>/dev/null | grep -q '^PONG$'; then
      return 0
    fi
    sleep 1
  done

  echo "Redis 启动超时" >&2
  return 1
}

if ! start_existing_container "${MYSQL_CONTAINER}"; then
  docker run -d \
    --name "${MYSQL_CONTAINER}" \
    --restart unless-stopped \
    -e MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD}" \
    -p 3306:3306 \
    -v taoxi-mysql-data:/var/lib/mysql \
    mysql:8.0 \
    --character-set-server=utf8mb4 \
    --collation-server=utf8mb4_unicode_ci >/dev/null
fi

wait_for_mysql

docker exec "${MYSQL_CONTAINER}" mysql \
  -uroot \
  "-p${MYSQL_ROOT_PASSWORD}" \
  -e "CREATE DATABASE IF NOT EXISTS \`${MYSQL_GAME_DATABASE}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; CREATE DATABASE IF NOT EXISTS \`${MYSQL_USER_DATABASE}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

if ! start_existing_container "${MONGO_CONTAINER}"; then
  docker run -d \
    --name "${MONGO_CONTAINER}" \
    --restart unless-stopped \
    -e MONGO_INITDB_ROOT_USERNAME="${MONGO_ROOT_USERNAME}" \
    -e MONGO_INITDB_ROOT_PASSWORD="${MONGO_ROOT_PASSWORD}" \
    -e MONGO_INITDB_DATABASE="${MONGO_DATABASE}" \
    -p 27017:27017 \
    -v taoxi-mongo-data:/data/db \
    mongo:7.0 >/dev/null
fi

wait_for_mongo

# MongoDB 只有写入数据后才会真正创建业务库。
docker exec "${MONGO_CONTAINER}" mongosh \
  --quiet \
  --username "${MONGO_ROOT_USERNAME}" \
  --password "${MONGO_ROOT_PASSWORD}" \
  --authenticationDatabase admin \
  --eval "db.getSiblingDB('${MONGO_DATABASE}').createCollection('_bootstrap')" >/dev/null 2>&1 || true

if ! start_existing_container "${REDIS_CONTAINER}"; then
  docker run -d \
    --name "${REDIS_CONTAINER}" \
    --restart unless-stopped \
    -p 6379:6379 \
    -v taoxi-redis-data:/data \
    redis:7-alpine \
    redis-server --appendonly yes >/dev/null
fi

wait_for_redis

echo "本地依赖启动完成："
echo "  MySQL  localhost:3306  root/${MYSQL_ROOT_PASSWORD}  DB=${MYSQL_GAME_DATABASE},${MYSQL_USER_DATABASE}"
echo "  MongoDB localhost:27017  ${MONGO_ROOT_USERNAME}/${MONGO_ROOT_PASSWORD}  DB=${MONGO_DATABASE} authSource=admin"
echo "  Redis   localhost:6379  无密码  DB=1,2"
