-- PostgreSQL stack lifecycle hooks
-- All functions are optional. sur calls them when present.

function install(ctx)
  -- Called after files are copied and .env is written, before docker compose up.
  -- ctx.log(msg)   — write a log line
  -- ctx.dir        — installed stack directory (/opt/sur/stacks/postgres)
  ctx.log("PostgreSQL stack install hook: nothing extra to do.")
end

function update(ctx)
  -- Called before docker compose pull + up.
  ctx.log("PostgreSQL update hook: remember to check major version compatibility before upgrading.")
end

function backup(ctx)
  -- Called by the Backup action in the TUI.
  -- sur handles directory creation; this hook can dump the DB.
  ctx.log("PostgreSQL backup hook: run pg_dump manually if needed until automated backup is implemented.")
end

function status(ctx)
  -- Called to provide extra status info beyond 'docker compose ps'.
  ctx.log("PostgreSQL status hook.")
end
