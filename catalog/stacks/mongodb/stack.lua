-- MongoDB stack lifecycle hooks

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
  ctx.log("Waiting for MongoDB to be ready...")
  local ok = false
  for i = 1, 20 do
    local ready, _ = ctx.exec(
      "docker", "compose", "--project-directory", ctx.dir,
      "exec", "-T", "mongodb",
      "mongosh", "--quiet", "--eval", "db.adminCommand('ping').ok"
    )
    if ready then ok = true; break end
    ctx.exec("sleep", "3")
  end
  if ok then
    ctx.log("MongoDB is ready.")
  else
    ctx.log("Warning: MongoDB did not become ready in time.")
  end
end

-- ─── backup ───────────────────────────────────────────────────────────────────

function backup(ctx)
  local user   = read_env(ctx.dir, "USERNAME")
  local db     = read_env(ctx.dir, "DB_NAME")
  local ts     = os.date("%Y-%m-%dT%H-%M-%S")
  local dst    = ctx.dir .. "/backups/mongodump_" .. ts

  ctx.log("Running mongodump → " .. dst)

  local ok, out = ctx.exec(
    "docker", "compose", "--project-directory", ctx.dir,
    "exec", "-T", "mongodb",
    "mongodump",
    "--username=" .. user,
    "--password=$(cat /run/secrets/password)",
    "--authenticationDatabase=admin",
    "--db=" .. db,
    "--out=/tmp/mongodump_" .. ts
  )

  if not ok then
    ctx.log("ERROR: mongodump failed: " .. out)
    return
  end

  -- Copy from container to host
  local ok2, out2 = ctx.exec(
    "docker", "compose", "--project-directory", ctx.dir,
    "cp", "mongodb:/tmp/mongodump_" .. ts, dst
  )

  if ok2 then
    -- tar + gzip
    ctx.exec("tar", "-czf", dst .. ".tar.gz", "-C", ctx.dir .. "/backups", "mongodump_" .. ts)
    ctx.exec("rm", "-rf", dst)
    ctx.log("Backup written to " .. dst .. ".tar.gz")
  else
    ctx.log("ERROR: could not copy backup from container: " .. out2)
  end
end

-- ─── update ───────────────────────────────────────────────────────────────────

function update(ctx)
  ctx.log("MongoDB update: check for breaking changes in the release notes before major upgrades.")
  ctx.log("Safe to proceed with minor version updates.")
end

-- ─── rotate ───────────────────────────────────────────────────────────────────

function rotate(ctx)
  local user    = read_env(ctx.dir, "USERNAME")
  local newpass = ctx.read_secret("password")

  if newpass == "" then
    ctx.log("ERROR: new password is empty.")
    error("empty password")
  end

  ctx.log("Rotating password for MongoDB user '" .. user .. "'...")

  local js = string.format(
    'db.getSiblingDB("admin").changeUserPassword("%s", "%s")',
    user, newpass:gsub('"', '\\"')
  )

  local ok, out = ctx.exec(
    "docker", "compose", "--project-directory", ctx.dir,
    "exec", "-T", "mongodb",
    "mongosh",
    "--username=" .. user,
    "--password=$(cat /run/secrets/password)",
    "--authenticationDatabase=admin",
    "--quiet",
    "--eval", js
  )

  if ok then
    ctx.log("Password rotated successfully.")
  else
    ctx.log("ERROR: changeUserPassword failed: " .. out)
    error("mongo changeUserPassword failed")
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
    "exec", "-T", "mongodb",
    "mongosh",
    "--username=" .. user,
    "--password=$(cat /run/secrets/password)",
    "--authenticationDatabase=admin",
    "--quiet",
    "--eval", string.format('db.getSiblingDB("%s").stats().dataSize', db)
  )
  if ok then
    ctx.log("Data size (bytes): " .. size)
  end
end
