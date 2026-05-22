-- PostgreSQL stack lifecycle hooks
-- ctx.log(msg)               — TUI'ya satır yaz
-- ctx.exec(cmd, arg, ...)    — komut çalıştır → (ok, output)
-- ctx.read_secret(name)      — secrets/<name>.txt oku
-- ctx.write_secret(name, v)  — secrets/<name>.txt yaz
-- ctx.dir                    — /opt/sur/stacks/postgres
-- ctx.env(key)               — environment variable oku

-- .env dosyasından bir değer oku
local function read_env(dir, key)
  local f = io.open(dir .. "/.env", "r")
  if not f then return "" end
  for line in f:lines() do
    local k, v = line:match("^([%w_]+)=(.*)$")
    if k == key then f:close(); return v end
  end
  f:close()
  return ""
end

-- Postgres container adını bul
local function container_name(dir)
  local ok, out = ctx.exec("docker", "compose", "--project-directory", dir, "ps", "-q", "postgres")
  if not ok or out == "" then return nil end
  return out:match("^%S+")
end

-- psql çalıştır (container içinde)
local function psql(dir, sql)
  local user = read_env(dir, "USERNAME")
  local db   = read_env(dir, "DB_NAME")
  return ctx.exec(
    "docker", "compose", "--project-directory", dir,
    "exec", "-T", "postgres",
    "psql", "-U", user, "-d", db, "-c", sql
  )
end

-- ─── install ──────────────────────────────────────────────────────────────────

function install(ctx)
  ctx.log("PostgreSQL install hook: waiting for DB to be ready...")
  local ok = false
  for i = 1, 20 do
    local ready = ctx.exec(
      "docker", "compose", "--project-directory", ctx.dir,
      "exec", "-T", "postgres",
      "pg_isready", "-q"
    )
    if ready then ok = true; break end
    ctx.exec("sleep", "2")
  end
  if ok then
    ctx.log("PostgreSQL is ready.")
  else
    ctx.log("Warning: PostgreSQL did not become ready in time. Check logs.")
  end
end

-- ─── backup ───────────────────────────────────────────────────────────────────

function backup(ctx)
  local user   = read_env(ctx.dir, "USERNAME")
  local db     = read_env(ctx.dir, "DB_NAME")
  local ts     = os.date("%Y-%m-%dT%H-%M-%S")
  local dump   = ctx.dir .. "/backups/pg_dump_" .. ts .. ".sql.gz"

  ctx.log("Running pg_dump → " .. dump)

  -- pg_dump inside container, gzip on host via shell redirect isn't available
  -- so we use docker exec + write output to a temp file then gzip.
  local tmp = ctx.dir .. "/backups/pg_dump_" .. ts .. ".sql"

  local ok, out = ctx.exec(
    "docker", "compose", "--project-directory", ctx.dir,
    "exec", "-T", "postgres",
    "pg_dump", "-U", user, "-d", db, "--no-password"
  )

  if not ok then
    ctx.log("ERROR: pg_dump failed: " .. out)
    return
  end

  -- Write output to tmp file
  local f = io.open(tmp, "w")
  if f then
    f:write(out)
    f:close()
    ctx.exec("gzip", "-f", tmp)
    ctx.log("Backup written to " .. dump)
  else
    ctx.log("ERROR: could not write backup file.")
  end
end

-- ─── update ───────────────────────────────────────────────────────────────────

function update(ctx)
  ctx.log("Checking PostgreSQL major version before update...")

  -- Read running version
  local ok, running = ctx.exec(
    "docker", "compose", "--project-directory", ctx.dir,
    "exec", "-T", "postgres",
    "psql", "-U", read_env(ctx.dir, "USERNAME"), "-d", read_env(ctx.dir, "DB_NAME"),
    "-tAc", "SHOW server_version_num;"
  )

  -- Read target image version from compose.yml
  local ok2, target_img = ctx.exec(
    "docker", "compose", "--project-directory", ctx.dir,
    "images", "--format", "{{.Image}}", "postgres"
  )

  ctx.log("Running version: " .. (running or "unknown"))
  ctx.log("Target image:   " .. (target_img or "unknown"))

  -- Extract major versions (server_version_num: 160004 → 16)
  local running_major = tonumber((running or ""):match("^(%d%d)")) or 0
  local target_major  = tonumber((target_img or ""):match(":(%d+)")) or 0

  if target_major > 0 and running_major > 0 and target_major ~= running_major then
    ctx.log("⚠️  MAJOR VERSION CHANGE DETECTED: " .. running_major .. " → " .. target_major)
    ctx.log("⚠️  pg_upgrade required! sur will NOT auto-upgrade major versions.")
    ctx.log("⚠️  Run backup first, then migrate manually. Update cancelled.")
    error("major version mismatch — manual migration required")
  end

  ctx.log("Minor version update, safe to proceed.")
end

-- ─── rotate ───────────────────────────────────────────────────────────────────

function rotate(ctx)
  -- sur has already written the new password to secrets/password.txt
  -- We need to tell PostgreSQL about it.
  local user    = read_env(ctx.dir, "USERNAME")
  local newpass = ctx.read_secret("password")

  if newpass == "" then
    ctx.log("ERROR: new password is empty, aborting rotate.")
    error("empty password")
  end

  ctx.log("Applying new password to PostgreSQL user '" .. user .. "'...")

  -- Escape single quotes in password (SQL injection guard)
  local safe = newpass:gsub("'", "''")
  local sql   = "ALTER USER \"" .. user .. "\" WITH PASSWORD '" .. safe .. "';"

  local ok, out = ctx.exec(
    "docker", "compose", "--project-directory", ctx.dir,
    "exec", "-T", "postgres",
    "psql", "-U", user, "-d", read_env(ctx.dir, "DB_NAME"),
    "-c", sql
  )

  if ok then
    ctx.log("Password updated successfully in PostgreSQL.")
  else
    ctx.log("ERROR: ALTER USER failed: " .. out)
    error("psql ALTER USER failed")
  end
end

-- ─── status ───────────────────────────────────────────────────────────────────

function status(ctx)
  local user = read_env(ctx.dir, "USERNAME")
  local db   = read_env(ctx.dir, "DB_NAME")
  local port = read_env(ctx.dir, "PORT")

  ctx.log(string.format("DB: %s  User: %s  Port: %s", db, user, port))

  local ok, size = ctx.exec(
    "docker", "compose", "--project-directory", ctx.dir,
    "exec", "-T", "postgres",
    "psql", "-U", user, "-d", db,
    "-tAc", "SELECT pg_size_pretty(pg_database_size('" .. db .. "'));"
  )
  if ok then
    ctx.log("Database size: " .. size)
  end
end
