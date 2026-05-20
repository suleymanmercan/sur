id = "audit_suid_files"
name = "Audit SUID/SGID Files"
description = "Scans common executable paths for files with SUID or SGID permissions to detect potential local privilege escalation vulnerabilities."
rollback_possible = false
backup_files = {}
risk_level = "low"
distros = {}

function pre_check()
    return true, 0
end

function exec()
    log("Scanning system executable paths (/bin, /sbin, /usr/bin, /usr/sbin) for SUID/SGID files...")

    local out, code = run("find /bin /sbin /usr/bin /usr/sbin -perm /6000 -type f 2>/dev/null")
    if code ~= 0 then
        -- try search without all directories if some fail
        out, code = run("find /usr/bin -perm -4000 -o -perm -2000 -type f 2>/dev/null")
        if code ~= 0 then
            return "Could not run find command to scan SUID/SGID files."
        end
    end

    -- Whitelist of standard SUID/SGID files found on typical Linux servers
    local whitelist = {
        ["/bin/ping"] = true,
        ["/bin/ping6"] = true,
        ["/bin/su"] = true,
        ["/bin/mount"] = true,
        ["/bin/umount"] = true,
        ["/usr/bin/ping"] = true,
        ["/usr/bin/ping6"] = true,
        ["/usr/bin/su"] = true,
        ["/usr/bin/mount"] = true,
        ["/usr/bin/umount"] = true,
        ["/usr/bin/sudo"] = true,
        ["/usr/bin/sudoedit"] = true,
        ["/usr/bin/passwd"] = true,
        ["/usr/bin/chsh"] = true,
        ["/usr/bin/chfn"] = true,
        ["/usr/bin/gpasswd"] = true,
        ["/usr/bin/newgrp"] = true,
        ["/usr/bin/chage"] = true,
        ["/usr/bin/expiry"] = true,
        ["/usr/bin/pkexec"] = true,
        ["/usr/bin/newuidmap"] = true,
        ["/usr/bin/newgidmap"] = true,
        ["/usr/bin/crontab"] = true,
        ["/usr/bin/ssh-agent"] = true,
        ["/usr/sbin/exim4"] = true,
        ["/usr/sbin/postdrop"] = true,
        ["/usr/sbin/postqueue"] = true,
        ["/usr/lib/dbus-1.0/dbus-daemon-launch-helper"] = true,
        ["/usr/lib/policykit-1/polkit-agent-helper-1"] = true,
        ["/usr/lib/openssh/ssh-keysign"] = true
    }

    local files_found = 0
    local non_whitelisted = {}

    for line in string.gmatch(out, "[^\r\n]+") do
        files_found = files_found + 1
        -- Clean up path (sometimes paths contain double slashes)
        local path = string.gsub(line, "//", "/")
        if not whitelist[path] then
            table.insert(non_whitelisted, path)
        end
    end

    log("Total SUID/SGID files found in executable paths: " .. files_found)

    if #non_whitelisted > 0 then
        log("Non-standard SUID/SGID files detected (manual review recommended):")
        for _, path in ipairs(non_whitelisted) do
            -- Get detailed info about the file
            local ls_out, _ = run("ls -l " .. path .. " 2>/dev/null")
            if ls_out ~= nil and ls_out ~= "" then
                log("  - " .. string.gsub(ls_out, "%s*$", ""))
            else
                log("  - " .. path)
            end
        end
        log("WARNING: Review the files listed above. Untrusted SUID/SGID binaries can be abused by local attackers to gain root access.")
    else
        log("No non-standard SUID/SGID files detected in standard executable paths.")
    end

    return nil
end

function post_check()
    return nil
end
