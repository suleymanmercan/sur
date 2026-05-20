id = "audit_ssh_failures"
name = "Audit SSH Login Failures"
description = "Analyzes auth logs or systemd journal for failed SSH login attempts in the last 24 hours."
rollback_possible = false
backup_files = {}
risk_level = "low"
distros = {} -- Empty means supports all distros

function pre_check()
    -- This is an informational audit check, always run it
    return true, 0
end

function exec()
    log("Scanning system logs for SSH authentication failures in the last 24 hours...")
    
    -- Try journalctl first, then fallback to /var/log/auth.log and /var/log/secure
    local log_data = ""
    local out, code = run("journalctl -u ssh -u sshd --since '24 hours ago' --no-pager 2>/dev/null")
    
    if code == 0 and out ~= nil and out ~= "" then
        log_data = out
    else
        log("journalctl failed or returned no logs, falling back to auth.log and secure files...")
        if file_exists("/var/log/auth.log") then
            local auth_content, err = read_file("/var/log/auth.log")
            if err == nil then log_data = auth_content end
        elseif file_exists("/var/log/secure") then
            local secure_content, err = read_file("/var/log/secure")
            if err == nil then log_data = secure_content end
        end
    end

    if log_data == "" then
        log("No SSH logs could be retrieved. Make sure the task is run with sufficient privileges (sudo).")
        return nil
    end

    -- Process log data line by line to count failures and extract IP/user details
    local total_failures = 0
    local ip_counts = {}
    local user_counts = {}

    for line in string.gmatch(log_data, "[^\r\n]+") do
        -- Patterns for SSH failures:
        -- "Failed password for invalid user <user> from <ip> port <port> ssh2"
        -- "Failed password for <user> from <ip> port <port> ssh2"
        -- "Connection closed by authenticating user <user> <ip> port <port>"
        if string.find(line, "[Ff]ailed password") or string.find(line, "Connection closed by authenticating user") then
            total_failures = total_failures + 1
            
            -- Attempt to extract IP address (simple IPv4 pattern: %d+.%d+.%d+.%d+)
            local ip = string.match(line, "from%s+(%d+%.%d+%.%d+%.%d+)")
            if not ip then
                ip = string.match(line, "user%s+[^%s]+%s+(%d+%.%d+%.%d+%.%d+)")
            end
            
            -- Attempt to extract username
            local user = string.match(line, "for%s+invalid%s+user%s+([^%s]+)")
            if not user then
                user = string.match(line, "for%s+([^%s]+)")
            end
            if not user then
                user = string.match(line, "authenticating%s+user%s+([^%s]+)")
            end

            if ip then
                ip_counts[ip] = (ip_counts[ip] or 0) + 1
            end
            if user then
                user_counts[user] = (user_counts[user] or 0) + 1
            end
        end
    end

    log("Total SSH login failures detected: " .. total_failures)

    if total_failures > 0 then
        -- Find top offending IPs
        log("Top failing IP addresses:")
        local sorted_ips = {}
        for ip, count in pairs(ip_counts) do
            table.insert(sorted_ips, {ip = ip, count = count})
        end
        table.sort(sorted_ips, function(a, b) return a.count > b.count end)
        
        local ip_limit = 5
        if #sorted_ips < ip_limit then ip_limit = #sorted_ips end
        for i = 1, ip_limit do
            log("  - IP: " .. sorted_ips[i].ip .. " (" .. sorted_ips[i].count .. " failures)")
        end

        -- Find top targeted users
        log("Top targeted usernames:")
        local sorted_users = {}
        for user, count in pairs(user_counts) do
            table.insert(sorted_users, {user = user, count = count})
        end
        table.sort(sorted_users, function(a, b) return a.count > b.count end)
        
        local user_limit = 5
        if #sorted_users < user_limit then user_limit = #sorted_users end
        for i = 1, user_limit do
            log("  - User: " .. sorted_users[i].user .. " (" .. sorted_users[i].count .. " attempts)")
        end
        
        if total_failures > 50 then
            log("WARNING: High number of SSH login failures detected! Consider securing SSH port, disabling password authentication, or configuring Fail2Ban.")
        end
    else
        log("No SSH login failures detected in the processed logs.")
    end

    return nil
end

function post_check()
    return nil
end
