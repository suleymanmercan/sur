id = "ssh_idle_timeout"
name = "Configure SSH Idle Timeout"
description = "Sets ClientAliveInterval to 300 and ClientAliveCountMax to 2 in sshd_config to automatically close idle connections after 10 minutes."
rollback_possible = true
backup_files = { "/etc/ssh/sshd_config" }
risk_level = "low"
distros = {} -- Supports all distributions

function pre_check()
    local interval = nil
    local count = nil

    -- Try running sshd -T first to get effective values
    local out, code = run("sshd -T 2>/dev/null")
    if code == 0 then
        for line in string.gmatch(out, "[^\r\n]+") do
            local parts = {}
            for part in string.gmatch(line, "%S+") do
                table.insert(parts, part)
            end
            if #parts >= 2 then
                local key = string.lower(parts[1])
                if key == "clientaliveinterval" then
                    interval = tonumber(parts[2])
                elseif key == "clientalivecountmax" then
                    count = tonumber(parts[2])
                end
            end
        end
    end

    -- Fallback to reading sshd_config directly if sshd -T failed or wasn't clean
    if interval == nil or count == nil then
        local content, err = read_file("/etc/ssh/sshd_config")
        if err == nil then
            for line in string.gmatch(content, "[^\r\n]+") do
                local trimmed = string.gsub(line, "^%s*", "")
                if string.sub(trimmed, 1, 1) ~= "#" then
                    local parts = {}
                    for part in string.gmatch(trimmed, "%S+") do
                        table.insert(parts, part)
                    end
                    if #parts >= 2 then
                        local key = string.lower(parts[1])
                        if key == "clientaliveinterval" then
                            interval = tonumber(parts[2])
                        elseif key == "clientalivecountmax" then
                            count = tonumber(parts[2])
                        end
                    end
                end
            end
        end
    end

    if interval == nil or count == nil then
        return true, 0
    end

    if interval ~= 300 or count ~= 2 then
        return true, 0
    end

    return false, 0
end

function exec()
    log("Configuring SSH idle timeout settings...")
    local content, err = read_file("/etc/ssh/sshd_config")
    if err ~= nil then return err end

    local new_lines = {}
    local interval_set = false
    local count_set = false

    for line in string.gmatch(content, "[^\r\n]+") do
        local replaced = false
        
        -- Match ClientAliveInterval (possibly commented out)
        if string.find(line, "^[#%s]*[Cc]lient[Aa]live[Ii]nterval%s+") then
            table.insert(new_lines, "ClientAliveInterval 300")
            interval_set = true
            replaced = true
        -- Match ClientAliveCountMax (possibly commented out)
        elseif string.find(line, "^[#%s]*[Cc]lient[Aa]live[Cc]ount[Mm]ax%s+") then
            table.insert(new_lines, "ClientAliveCountMax 2")
            count_set = true
            replaced = true
        end

        if not replaced then
            table.insert(new_lines, line)
        end
    end

    if not interval_set then
        table.insert(new_lines, "ClientAliveInterval 300")
    end
    if not count_set then
        table.insert(new_lines, "ClientAliveCountMax 2")
    end

    local new_content = table.concat(new_lines, "\n") .. "\n"

    -- Write to sshd_config
    local w_err = write_file("/etc/ssh/sshd_config", new_content)
    if w_err ~= nil then return w_err end

    -- Validate SSH configuration using sshd -t
    log("Validating SSH daemon configuration...")
    local val_out, val_code = run("sshd -t")
    if val_code ~= 0 then
        return "Invalid SSH daemon configuration: " .. val_out
    end

    -- Reload SSH service
    log("Reloading SSH daemon...")
    local reload_out, reload_code = run("systemctl reload ssh || systemctl reload sshd || service ssh reload || service sshd reload")
    if reload_code ~= 0 then
        return "Failed to reload SSH daemon: " .. reload_out
    end

    return nil
end

function post_check()
    local out, code = run("sshd -T 2>/dev/null")
    if code ~= 0 then
        return "Failed to verify SSH configuration with sshd -T"
    end

    local interval = nil
    local count = nil

    for line in string.gmatch(out, "[^\r\n]+") do
        local parts = {}
        for part in string.gmatch(line, "%S+") do
            table.insert(parts, part)
        end
        if #parts >= 2 then
            local key = string.lower(parts[1])
            if key == "clientaliveinterval" then
                interval = tonumber(parts[2])
            elseif key == "clientalivecountmax" then
                count = tonumber(parts[2])
            end
        end
    end

    if interval ~= 300 then
        return "ClientAliveInterval is " .. tostring(interval) .. " instead of 300"
    end

    if count ~= 2 then
        return "ClientAliveCountMax is " .. tostring(count) .. " instead of 2"
    end

    return nil
end

function rollback(backup_path)
    log("Reloading SSH daemon to restore original settings...")
    -- The runner has already restored sshd_config
    local out, code = run("systemctl reload ssh || systemctl reload sshd || service ssh reload || service sshd reload")
    if code ~= 0 then
        return "Failed to reload SSH daemon on rollback: " .. out
    end
    return nil
end
