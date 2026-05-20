id = "secure_shared_memory"
name = "Secure Shared Memory (/dev/shm)"
description = "Hardens shared memory (/dev/shm) by adding 'noexec,nosuid,nodev' options in /etc/fstab to prevent execution of binary files."
rollback_possible = true
backup_files = { "/etc/fstab" }
risk_level = "medium"
distros = {} -- Supports all distributions

function pre_check()
    local content, err = read_file("/etc/fstab")
    if err ~= nil then
        log("Could not read /etc/fstab: " .. err)
        return false, 1
    end

    local found = false
    local secured = false

    for line in string.gmatch(content, "[^\r\n]+") do
        -- Trim spaces and ignore comments
        local trimmed = string.gsub(line, "^%s*", "")
        if string.sub(trimmed, 1, 1) ~= "#" then
            local parts = {}
            for part in string.gmatch(trimmed, "%S+") do
                table.insert(parts, part)
            end

            if #parts >= 4 and parts[2] == "/dev/shm" then
                found = true
                local opts = parts[4]
                if string.find(opts, "noexec") and string.find(opts, "nosuid") and string.find(opts, "nodev") then
                    secured = true
                end
                break
            end
        end
    end

    if found and secured then
        return false, 0
    end
    return true, 0
end

function exec()
    log("Modifying /etc/fstab to secure /dev/shm...")
    local content, err = read_file("/etc/fstab")
    if err ~= nil then return err end

    local new_lines = {}
    local found = false

    for line in string.gmatch(content, "[^\r\n]+") do
        local trimmed = string.gsub(line, "^%s*", "")
        local is_shm = false
        if string.sub(trimmed, 1, 1) ~= "#" then
            local parts = {}
            for part in string.gmatch(trimmed, "%S+") do
                table.insert(parts, part)
            end
            if #parts >= 4 and parts[2] == "/dev/shm" then
                is_shm = true
            end
        end

        if is_shm then
            table.insert(new_lines, "tmpfs /dev/shm tmpfs defaults,noexec,nosuid,nodev 0 0")
            found = true
        else
            table.insert(new_lines, line)
        end
    end

    if not found then
        table.insert(new_lines, "tmpfs /dev/shm tmpfs defaults,noexec,nosuid,nodev 0 0")
    end

    local new_content = table.concat(new_lines, "\n") .. "\n"
    local w_err = write_file("/etc/fstab", new_content)
    if w_err ~= nil then return w_err end

    log("Remounting /dev/shm with new options...")
    local out, code = run("mount -o remount /dev/shm || mount /dev/shm")
    if code ~= 0 then
        return "Failed to remount /dev/shm: " .. out
    end

    return nil
end

function post_check()
    -- Read from /proc/mounts to see active mount options
    local content, err = read_file("/proc/mounts")
    if err ~= nil then
        -- Fallback to running mount command
        local out, code = run("mount")
        if code ~= 0 then return "Failed to run mount command" end
        content = out
    end

    for line in string.gmatch(content, "[^\r\n]+") do
        local parts = {}
        for part in string.gmatch(line, "%S+") do
            table.insert(parts, part)
        end
        if #parts >= 4 and parts[2] == "/dev/shm" then
            local opts = parts[4]
            if string.find(opts, "noexec") and string.find(opts, "nosuid") and string.find(opts, "nodev") then
                return nil
            else
                return "Shared memory mount is active but missing security flags: " .. opts
            end
        end
    end

    return "No mount point found for /dev/shm"
end

function rollback(backup_path)
    log("Restoring original mount settings for /dev/shm...")
    -- Backup is already restored to /etc/fstab by the Go runner.
    -- Remount /dev/shm so the options from fstab are re-applied.
    local out, code = run("mount -o remount /dev/shm || mount /dev/shm")
    if code ~= 0 then
        return "Failed to remount /dev/shm on rollback: " .. out
    end
    return nil
end
