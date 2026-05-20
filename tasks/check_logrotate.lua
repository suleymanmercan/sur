id = "check_logrotate"
name = "Verify Logrotate and Log File Sizes"
description = "Audits logrotate configuration and searches for unrotated log files exceeding 500MB."
rollback_possible = false
backup_files = {}
risk_level = "low"
distros = {}

function pre_check()
    return true, 0
end

function exec()
    log("Checking if logrotate is installed...")
    
    local has_logrotate = false
    local out, code = run("which logrotate 2>/dev/null")
    if code == 0 and out ~= nil and out ~= "" then
        has_logrotate = true
    else
        if file_exists("/usr/sbin/logrotate") or file_exists("/usr/bin/logrotate") then
            has_logrotate = true
        end
    end

    if not has_logrotate then
        log("WARNING: logrotate utility was not found. System logs may grow infinitely!")
    else
        log("logrotate utility is installed.")
        
        -- Check if it is running via systemd timer or cron
        local has_timer = false
        local timer_out, timer_code = run("systemctl is-active logrotate.timer 2>/dev/null")
        if timer_code == 0 and string.find(timer_out, "active") then
            has_timer = true
            log("logrotate systemd timer is active.")
        end

        local has_cron = file_exists("/etc/cron.daily/logrotate")
        if has_cron then
            log("logrotate daily cron script exists.")
        end

        if not has_timer and not has_cron then
            log("WARNING: logrotate is installed but neither the systemd timer nor cron job was verified to be active.")
        end
    end

    -- Check for excessively large log files in /var/log
    log("Scanning /var/log for files exceeding 500MB...")
    local size_out, size_code = run("find /var/log -type f -size +500M -exec ls -lh {} \\; 2>/dev/null")
    
    if size_code == 0 and size_out ~= nil and size_out ~= "" then
        log("Large log files detected:")
        for line in string.gmatch(size_out, "[^\r\n]+") do
            log("  - " .. line)
        end
        log("WARNING: Large log files detected. Consider manually rotating them, reducing logging verbosity, or configuring shorter logrotate periods.")
    else
        log("No log files larger than 500MB were found under /var/log.")
    end

    return nil
end

function post_check()
    return nil
end
