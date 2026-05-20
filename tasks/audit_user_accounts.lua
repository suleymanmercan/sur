id = "audit_user_accounts"
name = "Audit System User Accounts"
description = "Audits /etc/passwd and /etc/shadow for empty passwords, root-equivalent accounts, or interactive shell anomalies."
rollback_possible = false
backup_files = {}
risk_level = "medium"
distros = {}

function pre_check()
    return true, 0
end

local function audit_passwd_file()
    local content, err = read_file("/etc/passwd")
    if err ~= nil then
        log("ERROR: Could not read /etc/passwd: " .. err)
        return false
    end

    local uid_zero_users = {}
    local interactive_system_users = {}
    local total_users = 0

    -- Interactive shells patterns
    local interactive_shells = {
        ["/bin/sh"] = true,
        ["/bin/bash"] = true,
        ["/bin/zsh"] = true,
        ["/bin/dash"] = true,
        ["/usr/bin/bash"] = true,
        ["/usr/bin/zsh"] = true,
        ["/bin/ash"] = true
    }

    -- System usernames we want to watch out for
    local standard_system_users = {
        ["bin"] = true, ["daemon"] = true, ["sys"] = true, ["sync"] = true,
        ["games"] = true, ["man"] = true, ["lp"] = true, ["mail"] = true,
        ["news"] = true, ["uucp"] = true, ["proxy"] = true, ["www-data"] = true,
        ["backup"] = true, ["list"] = true, ["irc"] = true, ["gnats"] = true,
        ["nobody"] = true, ["systemd-network"] = true, ["systemd-resolve"] = true,
        ["messagebus"] = true, ["sshd"] = true
    }

    for line in string.gmatch(content, "[^\r\n]+") do
        total_users = total_users + 1
        -- Format: username:password:UID:GID:info:home:shell
        local parts = {}
        for part in string.gmatch(line .. ":", "([^:]*):") do
            table.insert(parts, part)
        end

        if #parts >= 7 then
            local username = parts[1]
            local uid = tonumber(parts[3])
            local shell = parts[7]

            -- Check for UID 0
            if uid == 0 then
                table.insert(uid_zero_users, username)
            end

            -- Check if system user has an interactive shell
            if standard_system_users[username] and interactive_shells[shell] then
                table.insert(interactive_system_users, username .. " (shell: " .. shell .. ")")
            end
        end
    end

    log("Total accounts in /etc/passwd: " .. total_users)

    -- Report UID 0 users
    if #uid_zero_users > 1 then
        log("WARNING: Multiple users have UID 0 (root access): " .. table.concat(uid_zero_users, ", "))
    else
        log("Only 'root' has UID 0 (standard).")
    end

    -- Report interactive system users
    if #interactive_system_users > 0 then
        log("WARNING: Standard system accounts configured with interactive shells (should be nologin/false):")
        for _, u in ipairs(interactive_system_users) do
            log("  - " .. u)
        end
    else
        log("System accounts have secure non-interactive shells.")
    end

    return true
end

local function audit_shadow_file()
    local content, err = read_file("/etc/shadow")
    if err ~= nil then
        log("Could not read /etc/shadow directly (this is normal if not running with root/sudo permissions: " .. err .. ")")
        -- Fallback: try checking via getent shadow
        local out, code = run("sudo getent shadow 2>/dev/null")
        if code == 0 and out ~= nil and out ~= "" then
            content = out
        else
            log("Could not run getent shadow. Skipping empty password checks.")
            return false
        end
    end

    local empty_password_users = {}

    for line in string.gmatch(content, "[^\r\n]+") do
        -- Format: username:password_hash:last_changed:min:max:warn:inactive:expire:reserved
        local parts = {}
        for part in string.gmatch(line .. ":", "([^:]*):") do
            table.insert(parts, part)
        end

        if #parts >= 2 then
            local username = parts[1]
            local passwd = parts[2]
            
            -- If password hash field is empty, the user can log in without password
            if passwd == "" then
                table.insert(empty_password_users, username)
            end
        end
    end

    if #empty_password_users > 0 then
        log("CRITICAL WARNING: User accounts found with empty passwords (no password required to log in!):")
        for _, username in ipairs(empty_password_users) do
            log("  - " .. username)
        end
        log("CRITICAL: Set passwords for these accounts immediately using 'passwd <username>' or lock them using 'passwd -l <username>'.")
    else
        log("No accounts with empty passwords detected in /etc/shadow.")
    end

    return true
end

function exec()
    log("Auditing system user accounts...")
    audit_passwd_file()
    audit_shadow_file()
    return nil
end

function post_check()
    return nil
end
