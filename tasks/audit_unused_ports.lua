id = "audit_insecure_ports"
name = "Audit Insecure Ports and Services (Lua)"
description = "Checks if insecure services like Telnet (23), FTP (21) or Rsh (514) are listening"
rollback_possible = false
backup_files = {}
risk_level = "low"
distros = {} -- Empty means supports all distros

function pre_check()
    -- Always run this task to audit ports
    return true, 0
end

function exec()
    log("Auditing active network connections...")
    
    local out, code = run("ss -tulpn")
    if code ~= 0 then
        -- fallback to netstat if ss is not installed
        out, code = run("netstat -tulpn")
        if code ~= 0 then
            return "Both ss and netstat are unavailable for port auditing"
        end
    end

    local insecure_found = false

    -- Check for default insecure ports: FTP (21), Telnet (23), RCP/Rsh (514)
    if string.find(out, ":21 ") then
        log("WARNING: FTP service (port 21) appears to be listening!")
        insecure_found = true
    end
    if string.find(out, ":23 ") then
        log("WARNING: Telnet service (port 23) appears to be listening!")
        insecure_found = true
    end
    if string.find(out, ":514 ") then
        log("WARNING: Rsh/Rexec service (port 514) appears to be listening!")
        insecure_found = true
    end

    if insecure_found then
        log("Action recommended: Disable the identified insecure services.")
    else
        log("No default insecure services (FTP, Telnet, Rsh) detected running.")
    end

    return nil
end

function post_check()
    -- Audit is informational, post check always succeeds
    return nil
end
