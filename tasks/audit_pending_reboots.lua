id = "audit_pending_reboots"
name = "Check Pending Reboots & Updates"
description = "Checks if the system has updates pending or requires a restart due to package updates."
rollback_possible = false
backup_files = {}
risk_level = "low"
distros = {}

function pre_check()
    return true, 0
end

-- Helper to check reboot requirement
local function check_reboot_required()
    if file_exists("/var/run/reboot-required") then
        log("WARNING: System reboot is required! /var/run/reboot-required exists.")
        
        -- Try to read which packages triggered it (on Debian/Ubuntu)
        local pkgs, err = read_file("/var/run/reboot-required.pkgs")
        if err == nil and pkgs ~= "" then
            log("Reboot triggered by packages:")
            for pkg in string.gmatch(pkgs, "[^\r\n]+") do
                log("  - " .. pkg)
            end
        end
        return true
    end
    log("No pending reboot required (/var/run/reboot-required not found).")
    return false
end

-- Helper to check pending packages (apt for debian, dnf for rhel)
local function check_pending_updates()
    -- Check what package managers are present
    local has_apt = false
    local has_dnf = false
    
    local _, code_apt = run("which apt-get 2>/dev/null")
    if code_apt == 0 then has_apt = true end
    
    local _, code_dnf = run("which dnf 2>/dev/null")
    if code_dnf == 0 then has_dnf = true end

    if has_apt then
        log("Checking for pending updates using apt-get (Debian/Ubuntu)...")
        -- Run simulated apt-get upgrade to count packages
        local out, code = run("apt-get -s upgrade 2>/dev/null")
        if code == 0 and out ~= nil then
            -- Find line like "X upgraded, Y newly installed, Z to remove and W not upgraded."
            local upgraded = string.match(out, "(%d+) upgraded")
            local newly_installed = string.match(out, "(%d+) newly installed")
            local not_upgraded = string.match(out, "(%d+) not upgraded")
            
            if upgraded then
                log("Pending upgrades: " .. upgraded .. " package(s) can be upgraded.")
                if tonumber(upgraded) > 0 then
                    log("WARNING: Packages are outdated. Run 'sudo apt-get update && sudo apt-get upgrade' to update.")
                end
            else
                log("No pending package upgrades found.")
            end
        else
            log("Failed to simulate apt upgrade.")
        end
    elseif has_dnf then
        log("Checking for pending updates using dnf (RHEL/Fedora)...")
        -- Run dnf check-update (returns 100 if updates are available, 0 if not, other codes on error)
        local out, code = run("dnf check-update -q 2>/dev/null")
        if code == 100 then
            -- Count lines of updates
            local count = 0
            for _ in string.gmatch(out, "[^\r\n]+") do
                count = count + 1
            end
            log("Pending upgrades: " .. count .. " package(s) can be upgraded.")
            log("WARNING: Packages are outdated. Run 'sudo dnf upgrade' to update.")
        elseif code == 0 then
            log("No pending package upgrades found.")
        else
            log("Failed to run dnf check-update.")
        end
    else
        log("No supported package manager (apt-get, dnf) detected for update check.")
    end
end

function exec()
    check_reboot_required()
    check_pending_updates()
    return nil
end

function post_check()
    return nil
end
