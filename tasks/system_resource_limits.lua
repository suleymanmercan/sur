id = "system_resource_limits"
name = "Check System Resource Limits"
description = "Checks disk space usage, memory limits, and CPU load averages to ensure the system is not over-utilised."
rollback_possible = false
backup_files = {}
risk_level = "low"
distros = {}

function pre_check()
    return true, 0
end

-- Helper to count CPU cores
local function get_cpu_cores()
    local cpuinfo, err = read_file("/proc/cpuinfo")
    if err ~= nil or cpuinfo == "" then
        -- fallback to nproc command
        local out, code = run("nproc 2>/dev/null")
        if code == 0 then
            local cores = tonumber(string.match(out, "%d+"))
            if cores then return cores end
        end
        return 1 -- fallback default
    end
    
    local cores = 0
    for line in string.gmatch(cpuinfo, "[^\r\n]+") do
        if string.find(line, "^processor%s*:") then
            cores = cores + 1
        end
    end
    if cores == 0 then return 1 end
    return cores
end

-- Helper to check CPU load average
local function check_cpu_load(cores)
    local loadavg, err = read_file("/proc/loadavg")
    local load1 = nil
    if err == nil and loadavg ~= "" then
        load1 = tonumber(string.match(loadavg, "^([%d%.]+)"))
    else
        local out, code = run("uptime 2>/dev/null")
        if code == 0 then
            load1 = tonumber(string.match(out, "load average%s*:%s*([%d%.]+)"))
        end
    end

    if load1 then
        log("CPU Load Average (1 min): " .. load1 .. " (Cores: " .. cores .. ")")
        local threshold = cores * 1.5 -- warning threshold at 150% capacity
        if load1 > threshold then
            log("WARNING: CPU load is very high (" .. load1 .. " > " .. threshold .. ").")
            return false
        end
    else
        log("Could not read CPU load average.")
    end
    return true
end

-- Helper to check Memory usage
local function check_memory()
    local meminfo, err = read_file("/proc/meminfo")
    local total_kb = nil
    local avail_kb = nil
    
    if err == nil and meminfo ~= "" then
        total_kb = tonumber(string.match(meminfo, "MemTotal:%s+(%d+)"))
        avail_kb = tonumber(string.match(meminfo, "MemAvailable:%s+(%d+)"))
    end
    
    -- Fallback to free -k if /proc/meminfo is missing or parsing failed
    if not total_kb or not avail_kb then
        local out, code = run("free -k 2>/dev/null")
        if code == 0 then
            -- Find the 'Mem:' line and parse total, available
            for line in string.gmatch(out, "[^\r\n]+") do
                if string.find(line, "^Mem:") then
                    local parts = {}
                    for num in string.gmatch(line, "%d+") do
                        table.insert(parts, tonumber(num))
                    end
                    -- free -k columns: total, used, free, shared, buff/cache, available
                    if #parts >= 6 then
                        total_kb = parts[1]
                        avail_kb = parts[6]
                    elseif #parts >= 3 then
                        total_kb = parts[1]
                        avail_kb = parts[3] -- fallback to free if available column is missing
                    end
                end
            end
        end
    end

    if total_kb and avail_kb then
        local used_kb = total_kb - avail_kb
        local used_pct = math.floor((used_kb / total_kb) * 100)
        local total_mb = math.floor(total_kb / 1024)
        local avail_mb = math.floor(avail_kb / 1024)
        
        log("Memory Usage: " .. used_pct .. "% (Total: " .. total_mb .. " MB, Available: " .. avail_mb .. " MB)")
        if used_pct > 90 then
            log("WARNING: Memory usage is critically high (" .. used_pct .. "%).")
            return false
        end
    else
        log("Could not read memory information.")
    end
    return true
end

-- Helper to check Disk usage
local function check_disk()
    local out, code = run("df -k / 2>/dev/null")
    if code ~= 0 then
        log("Could not run df command.")
        return true
    end

    local use_pct = nil
    local mount = nil
    for line in string.gmatch(out, "[^\r\n]+") do
        if string.find(line, "^/") or string.find(line, "%d+%%") then
            local pct = string.match(line, "(%d+)%%")
            if pct then
                use_pct = tonumber(pct)
            end
        end
    end

    if use_pct then
        log("Root Disk Usage: " .. use_pct .. "%")
        if use_pct > 85 then
            log("WARNING: Root disk usage is high (" .. use_pct .. "%).")
            return false
        end
    else
        log("Could not parse disk usage from df output.")
    end
    return true
end

function exec()
    log("Checking system resource utilisation...")
    local cores = get_cpu_cores()
    
    local cpu_ok = check_cpu_load(cores)
    local mem_ok = check_memory()
    local disk_ok = check_disk()

    if not (cpu_ok and mem_ok and disk_ok) then
        log("Action recommended: Check processes consuming excessive resources (e.g., via 'htop' or 'top') or clear disk space.")
    else
        log("All system resources are within acceptable limits.")
    end

    return nil
end

function post_check()
    return nil
end
