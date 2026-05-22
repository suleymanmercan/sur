-- MySQL stack lifecycle hooks

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

-- ─── install ──────────────────────────────────────────────────────────────────

function install(ctx)
  ctx.log("Waiting for MySQL to be ready...")
  local ok = false
  for i = 1, 20 do
    local ready, _ = ctx.exec(
      "docker", "compose", "--project-directory", ctx.dir,
      "exec", "-T", "mysql",
      "mysqladmin", "ping", "-h", "localhost", "--silent"
    )
    if ready then ok = true; break end
    ctx.exec("sleep", "3")
  end
  if ok then
    ctx.log("MySQL is ready.")
  else
    ctx.log("Warning: MySQL did not become ready in time.")
  end
end

-- ─── backup ───────────────────────────────────────────────────────────────────

function backup(ctx)
  local user   = read_env(ctx.dir, "USERNAME")
  local db     = read_env(ctx.dir, "DB_NAME")
  local ts     = os.date("%Y-%m-%dT%H-%M-%S")
  local tmp    = ctx.dir .. "/backups/mysqldump_" .. ts .. ".sql"
  local dump   = tmp .. ".gz"

  ctx.log("Running mysqldump → " .. dump)

  local ok, out = ctx.exec(
    "docker", "compose", "--project-directory", ctx.dir,
    "exec", "-T", "mysql",
    "mysqldump",
    "--user=" .. user,
    "--password=$(cat /run/secrets/password)",
    "--single-transaction",
    "--quick",
    db
  )

  if not ok then
    ctx.log("ERROR: mysqldump failed: " .. out)
    return
  end

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
  ctx.log("MySQL update: no major version check implemented — verify image tag manually.")
  ctx.log("Safe to proceed with minor version updates.")
end

-- ─── rotate ───────────────────────────────────────────────────────────────────

function rotate(ctx)
  local user    = read_env(ctx.dir, "USERNAME")
  local db      = read_env(ctx.dir, "DB_NAME")
  local newpass = ctx.read_secret("password")

  if newpass == "" then
    ctx.log("ERROR: new password is empty.")
    error("empty password")
  end

  ctx.log("Applying new password to MySQL user '" .. user .. "'...")

  local safe = newpass:gsub("'", "''")
  local sql   = "ALTER USER '" .. user .. "'@'%' IDENTIFIED BY '" .. safe .. "'; FLUSH PRIVILEGES;"

  local ok, out = ctx.exec(
    "docker", "compose", "--project-directory", ctx.dir,
    "exec", "-T", "mysql",
    "mysql",
    "--user=root",
    "--password-file=/run/secrets/root_password",
    "-e", sql
  )

  if ok then
    ctx.log("Password rotated successfully.")
  else
    ctx.log("ERROR: ALTER USER failed: " .. out)
    error("mysql ALTER USER failed")
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
    "exec", "-T", "mysql",
    "mysql",
    "--user=" .. user,
    "--password-file=/run/secrets/password",
    "-e", "SELECT ROUND(SUM(data_length+index_length)/1024/1024,2) AS 'Size (MB)' FROM information_schema.tables WHERE table_schema='" .. db .. "';",
    "--batch", "--skip-column-names"
  )
  if ok then
    ctx.log("Database size: " .. size .. " MB")
  end
end
