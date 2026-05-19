id = "disable_root_ssh_lua"
name = "Disable Root SSH Login (Lua version)"
description = "Updates sshd_config to disable PermitRootLogin and restarts sshd"
rollback_possible = true
backup_files = { "/etc/ssh/sshd_config" }
risk_level = "medium"
distros = { "debian", "ubuntu", "fedora", "centos", "rhel", "almalinux", "rocky" }

function pre_check()
    -- check if PermitRootLogin yes or uncommented yes is set
    local content, err = read_file("/etc/ssh/sshd_config")
    if err ~= nil then
        log("Could not read sshd_config: " .. err)
        return false, 1
    end

    -- Look for non-commented PermitRootLogin yes
    if string.find(content, "\nPermitRootLogin yes") or string.find(content, "^PermitRootLogin yes") then
        return true, 0
    end
    
    return false, 0
end

function exec()
    log("Disabling root SSH login...")
    local content, err = read_file("/etc/ssh/sshd_config")
    if err ~= nil then
        return "Could not read sshd_config: " .. err
    end

    -- replace PermitRootLogin yes with PermitRootLogin no
    local new_content = string.gsub(content, "PermitRootLogin yes", "PermitRootLogin no")
    
    local write_err = write_file("/etc/ssh/sshd_config", new_content)
    if write_err ~= nil then
        return "Could not write to sshd_config: " .. write_err
    end

    -- Restart ssh service
    log("Restarting SSH daemon...")
    local out, code = run("systemctl restart sshd || systemctl restart ssh")
    if code ~= 0 then
        return "Failed to restart SSH daemon: " .. out
    end

    return nil
end

function post_check()
    local content, err = read_file("/etc/ssh/sshd_config")
    if err ~= nil then
        return "Could not read sshd_config in post_check"
    end

    if string.find(content, "\nPermitRootLogin yes") or string.find(content, "^PermitRootLogin yes") then
        return "PermitRootLogin yes is still present!"
    end

    return nil
end

function rollback(backup_path)
    log("Restoring sshd_config from backup: " .. backup_path)
    -- The Go runner already restores the file if it's in backup_files.
    -- We just need to restart the sshd service to apply the restored file.
    local out, code = run("systemctl restart sshd || systemctl restart ssh")
    if code ~= 0 then
        return "Failed to restart SSH daemon on rollback: " .. out
    end
    return nil
end
