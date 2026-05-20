id = "audit_dns_resolvers"
name = "Audit DNS Resolver Settings"
description = "Audits /etc/resolv.conf configuration and verifies if name servers are responsive."
rollback_possible = false
backup_files = {}
risk_level = "low"
distros = {}

function pre_check()
    return true, 0
end

function exec()
    log("Reading /etc/resolv.conf...")
    
    local content, err = read_file("/etc/resolv.conf")
    if err ~= nil then
        return "Could not read /etc/resolv.conf: " .. err
    end

    local nameservers = {}
    local options = {}
    local search_domains = {}

    for line in string.gmatch(content, "[^\r\n]+") do
        -- Trim comments
        local clean_line = string.gsub(line, "#.*", "")
        clean_line = string.gsub(clean_line, ";.*", "")
        
        -- Parse nameservers
        local ns = string.match(clean_line, "^%s*nameserver%s+([%a%d%.:]+)")
        if ns then
            table.insert(nameservers, ns)
        end

        -- Parse options
        local opts = string.match(clean_line, "^%s*options%s+(.+)")
        if opts then
            table.insert(options, opts)
        end

        -- Parse search domains
        local s = string.match(clean_line, "^%s*search%s+(.+)")
        if s then
            table.insert(search_domains, s)
        end
    end

    if #nameservers == 0 then
        log("WARNING: No active nameservers configured in /etc/resolv.conf!")
        return nil
    end

    log("Configured nameservers (" .. #nameservers .. " found):")
    for _, ns in ipairs(nameservers) do
        log("  - " .. ns)
    end

    if #search_domains > 0 then
        log("Search domains: " .. table.concat(search_domains, ", "))
    end
    if #options > 0 then
        log("Resolver options: " .. table.concat(options, ", "))
    end

    -- Test availability of each nameserver
    log("Checking nameserver responsiveness...")
    local responsive_count = 0
    for _, ns in ipairs(nameservers) do
        -- Check port 53 (DNS) with timeout of 2 seconds using nc (netcat)
        local test_cmd = "nc -z -w 2 " .. ns .. " 53 2>/dev/null"
        -- Also support testing with timeout command or bash if nc is missing
        local _, code = run(test_cmd)
        
        if code == 0 then
            log("  - " .. ns .. ": Responsive (TCP port 53 open)")
            responsive_count = responsive_count + 1
        else
            -- Try ping as fallback
            local ping_cmd = "ping -c 1 -W 2 " .. ns .. " >/dev/null 2>&1"
            local _, ping_code = run(ping_cmd)
            if ping_code == 0 then
                log("  - " .. ns .. ": Responsive to ping")
                responsive_count = responsive_count + 1
            else
                log("  - " .. ns .. ": UNRESPONSIVE (Port 53 closed / Ping failed)")
            end
        end
    end

    if responsive_count == 0 then
        log("WARNING: All configured nameservers are unresponsive! Outbound DNS resolution may be completely broken.")
    elseif responsive_count < #nameservers then
        log("WARNING: Some configured nameservers are unresponsive. Consider updating nameservers configuration.")
    else
        log("All configured nameservers are healthy and responsive.")
    end

    return nil
end

function post_check()
    return nil
end
