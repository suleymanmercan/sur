id = "audit_log_anomalies"
name = "Audit System Log Anomalies"
description = "Scans system logs for critical anomalies like OOM-kills, kernel segfaults, or disk I/O errors."
rollback_possible = false
backup_files = {}
risk_level = "low"
distros = {}

function pre_check()
    return true, 0
end

function exec()
    log("Scanning system logs for anomalies and critical errors...")
    
    local log_data = ""
    local has_journal = false
    
    -- Try journalctl for recent error-level logs (priority 3 and below: err, crit, alert, emerg)
    local out, code = run("journalctl -p 3 -b --no-pager -n 100 2>/dev/null")
    if code == 0 and out ~= nil and out ~= "" then
        log_data = out
        has_journal = true
        log("Retrieved last 100 error logs from systemd journal.")
    else
        log("journalctl not available, searching log files directly...")
        local paths = {"/var/log/syslog", "/var/log/messages", "/var/log/kern.log"}
        for _, path in ipairs(paths) do
            if file_exists(path) then
                local content, err = read_file(path)
                if err == nil and content ~= nil and content ~= "" then
                    -- Get the last 2000 lines to avoid blowing memory
                    local lines = {}
                    for line in string.gmatch(content, "[^\r\n]+") do
                        table.insert(lines, line)
                    end
                    local start_idx = 1
                    if #lines > 2000 then start_idx = #lines - 2000 end
                    
                    local tail_lines = {}
                    for i = start_idx, #lines do
                        table.insert(tail_lines, lines[i])
                    end
                    log_data = table.concat(tail_lines, "\n")
                    log("Retrieved logs from " .. path .. " (last " .. (#lines - start_idx + 1) .. " lines).")
                    break
                end
            end
        end
    end

    if log_data == "" then
        -- Try dmesg as fallback
        local dmesg_out, dmesg_code = run("dmesg -T 2>/dev/null")
        if dmesg_code == 0 and dmesg_out ~= nil and dmesg_out ~= "" then
            log_data = dmesg_out
            log("Retrieved logs from dmesg.")
        else
            log("No log sources could be read. Ensure task is run as root/sudo.")
            return nil
        end
    end

    -- Scrapers
    local oom_kills = {}
    local segfaults = {}
    local io_errors = {}
    local ext4_errors = {}

    for line in string.gmatch(log_data, "[^\r\n]+") do
        local lower_line = string.lower(line)
        
        -- OOM Kill detection
        if string.find(lower_line, "out of memory") or string.find(lower_line, "oom%-kill") or string.find(lower_line, "killed process") then
            table.insert(oom_kills, line)
        end

        -- Segfault detection
        if string.find(lower_line, "segfault") or string.find(lower_line, "general protection fault") or string.find(lower_line, "segmentation fault") then
            table.insert(segfaults, line)
        end

        -- Disk I/O errors
        if string.find(lower_line, "i/o error") or string.find(lower_line, "read-only file system") then
            table.insert(io_errors, line)
        end

        -- Filesystem errors
        if string.find(lower_line, "ext4%-fs error") or string.find(lower_line, "xfs_force_shutdown") or string.find(lower_line, "fs error") then
            table.insert(ext4_errors, line)
        end
    end

    -- Report findings
    local anomalies_found = false

    if #oom_kills > 0 then
        log("WARNING: OOM-Kills detected! The system is running out of memory and has terminated processes:")
        local show_limit = 5
        if #oom_kills < show_limit then show_limit = #oom_kills end
        for i = #oom_kills - show_limit + 1, #oom_kills do
            log("  - " .. oom_kills[i])
        end
        anomalies_found = true
    end

    if #segfaults > 0 then
        log("WARNING: Process segmentation faults detected! Software crashes occurred:")
        local show_limit = 5
        if #segfaults < show_limit then show_limit = #segfaults end
        for i = #segfaults - show_limit + 1, #segfaults do
            log("  - " .. segfaults[i])
        end
        anomalies_found = true
    end

    if #io_errors > 0 then
        log("CRITICAL: Disk I/O errors detected! Potential hardware failure or filesystem read-only state:")
        for _, err_line in ipairs(io_errors) do
            log("  - " .. err_line)
        end
        anomalies_found = true
    end

    if #ext4_errors > 0 then
        log("CRITICAL: Filesystem errors detected! Filesystem integrity might be compromised:")
        for _, err_line in ipairs(ext4_errors) do
            log("  - " .. err_line)
        end
        anomalies_found = true
    end

    if not anomalies_found then
        log("No critical log anomalies (OOM, Segfaults, I/O or Filesystem errors) were found in the scanned logs.")
    else
        log("WARNING: Log anomalies were detected. Please run 'dmesg' or inspect log files for further diagnostics.")
    end

    return nil
end

function post_check()
    return nil
end
