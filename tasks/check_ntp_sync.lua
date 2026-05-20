id = "check_ntp_sync"
name = "Verify Time Sync (NTP)"
description = "Checks NTP service status and verifies system time is synchronized."
rollback_possible = false
backup_files = {}
risk_level = "low"
distros = {}

function pre_check()
    return true, 0
end

function exec()
    log("Checking time synchronization status...")
    
    local out, code = run("timedatectl status 2>/dev/null")
    if code == 0 and out ~= nil and out ~= "" then
        -- Output typically looks like:
        --                Local time: Wed 2026-05-20 12:00:00 UTC
        --            Universal time: Wed 2026-05-20 12:00:00 UTC
        --                  RTC time: Wed 2026-05-20 12:00:00
        --                 Time zone: UTC (UTC, +0000)
        -- System clock synchronized: yes
        --               NTP service: active
        --           RTC in local TZ: no
        
        local synced = string.match(out, "System clock synchronized:%s*([%a]+)")
        local service = string.match(out, "NTP service:%s*([%a]+)")
        
        -- Support alternative timedatectl output formats (older systemd)
        if not synced then
            synced = string.match(out, "NTP synchronized:%s*([%a]+)")
        end
        if not service then
            service = string.match(out, "network time on:%s*([%a]+)")
        end
        
        if service then
            log("NTP Service Status: " .. service)
            if service == "active" or service == "yes" then
                log("NTP / time synchronization service is enabled.")
            else
                log("WARNING: NTP service is not active/enabled!")
            end
        else
            log("NTP Service Status: Unknown (Could not parse timedatectl output)")
        end

        if synced then
            log("System Clock Synchronized: " .. synced)
            if synced == "yes" then
                log("System clock is correctly synchronized.")
            else
                log("WARNING: System clock is NOT synchronized!")
            end
        else
            log("System Clock Synchronized: Unknown (Could not parse timedatectl output)")
        end

        if (service == "active" or service == "yes") and synced == "yes" then
            log("System time synchronization is configured and working correctly.")
        else
            log("WARNING: Time synchronization issues detected. Consider running 'sudo timedatectl set-ntp true' or starting chronyd/systemd-timesyncd.")
        end
    else
        log("timedatectl status failed or not available. Checking common NTP services via systemctl...")
        
        local ntp_services = {"systemd-timesyncd", "chrony", "ntp", "ntpd"}
        local active_found = false
        
        for _, svc in ipairs(ntp_services) do
            local svc_out, svc_code = run("systemctl is-active " .. svc .. " 2>/dev/null")
            if svc_code == 0 and string.find(svc_out, "active") then
                log("Time service '" .. svc .. "' is active.")
                active_found = true
                break
            end
        end
        
        if active_found then
            log("An NTP service was found active. Clock synchronization status could not be verified directly.")
        else
            log("WARNING: No active NTP service (systemd-timesyncd, chrony, ntp, ntpd) was found. Clock drift may occur.")
        end
    end

    return nil
end

function post_check()
    return nil
end
