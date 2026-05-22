-- Redis stack lifecycle hooks

function install(ctx)
  ctx.log("Redis stack install hook: AOF persistence enabled by default.")
end

function update(ctx)
  ctx.log("Redis update hook: check CHANGELOG for breaking changes between major versions.")
end

function backup(ctx)
  ctx.log("Redis backup hook: AOF file is in ./data/appendonly.aof — copy it to backups/ if needed.")
end

function status(ctx)
  ctx.log("Redis status hook.")
end
