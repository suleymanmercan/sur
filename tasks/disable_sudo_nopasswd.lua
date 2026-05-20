id = "disable_sudo_nopasswd"
name = "Disable Sudo NOPASSWD"
description = "Removes NOPASSWD directives in /etc/sudoers to enforce password prompt for all sudo commands."
rollback_possible = true
backup_files = { "/etc/sudoers" }
risk_level = "medium"
distros = {} -- Supports all distributions

function pre_check()
    local content, err = read_file("/etc/sudoers")
    if err ~= nil then
        log("Could not read /etc/sudoers: " .. err)
        return false, 1
    end

    for line in string.gmatch(content, "[^\r\n]+") do
        local trimmed = string.gsub(line, "^%s*", "")
        if string.sub(trimmed, 1, 1) ~= "#" then
            if string.find(trimmed, "NOPASSWD%s*:") then
                return true, 0
            end
        end
    end

    return false, 0
end

function exec()
    log("Disabling NOPASSWD in /etc/sudoers...")
    local content, err = read_file("/etc/sudoers")
    if err ~= nil then return err end

    local new_lines = {}
    local changed = false

    for line in string.gmatch(content, "[^\r\n]+") do
        local trimmed = string.gsub(line, "^%s*", "")
        if string.sub(trimmed, 1, 1) ~= "#" and string.find(trimmed, "NOPASSWD%s*:") then
            local new_line = string.gsub(line, "NOPASSWD%s*:%s*", "")
            table.insert(new_lines, new_line)
            changed = true
        else
            table.insert(new_lines, line)
        end
    end

    if not changed then
        log("No active NOPASSWD entries found in /etc/sudoers")
        return nil
    end

    local new_content = table.concat(new_lines, "\n") .. "\n"

    -- Write to a temporary file first for validation
    local tmp_path = "/tmp/sudoers.tmp"
    local w_err = write_file(tmp_path, new_content)
    if w_err ~= nil then
        return "Could not write to validation file: " .. w_err
    end

    -- Run visudo validation
    local out, code = run("visudo -cf " .. tmp_path)
    run("rm -f " .. tmp_path) -- clean up tmp file

    if code ~= 0 then
        return "Modified sudoers file is invalid: " .. out
    end

    -- Write verified content to /etc/sudoers
    local final_err = write_file("/etc/sudoers", new_content)
    if final_err ~= nil then
        return "Could not update /etc/sudoers: " .. final_err
    end

    -- Warn user if there are NOPASSWD entries in drop-in config files under /etc/sudoers.d/
    local files_out, files_code = run("ls /etc/sudoers.d 2>/dev/null")
    if files_code == 0 then
        for f in string.gmatch(files_out, "%S+") do
            local path = "/etc/sudoers.d/" .. f
            local file_content, file_err = read_file(path)
            if file_err == nil then
                if string.find(file_content, "NOPASSWD%s*:") then
                    log("WARNING: NOPASSWD entry also detected in " .. path .. ". Please audit and disable manually.")
                end
            end
        end
    end

    log("Successfully disabled NOPASSWD entries in /etc/sudoers")
    return nil
end

function post_check()
    local content, err = read_file("/etc/sudoers")
    if err ~= nil then return err end

    for line in string.gmatch(content, "[^\r\n]+") do
        local trimmed = string.gsub(line, "^%s*", "")
        if string.sub(trimmed, 1, 1) ~= "#" then
            if string.find(trimmed, "NOPASSWD%s*:") then
                return "NOPASSWD is still present in /etc/sudoers!"
            end
        end
    end

    return nil
end

function rollback(backup_path)
    log("Restoring original sudoers configuration...")
    -- The runner has already restored /etc/sudoers
    local out, code = run("visudo -cf /etc/sudoers")
    if code ~= 0 then
        return "Restored sudoers file is invalid: " .. out
    end
    return nil
end
