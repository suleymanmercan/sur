id = "sysctl_hardening"
name = "Kernel Parameter Hardening (sysctl)"
description = "Hardens TCP/IP stack and system settings against IP spoofing, redirects, Martian packets, and SYN floods."
rollback_possible = true
backup_files = { "/etc/sysctl.d/99-security-hardening.conf" }
risk_level = "medium"
distros = {} -- Supports all distributions

function pre_check()
    -- If the file doesn't exist, we definitely need to run
    if not file_exists("/etc/sysctl.d/99-security-hardening.conf") then
        return true, 0
    end

    -- If the file exists, read it and check if SYN cookies are enabled in it
    local content, err = read_file("/etc/sysctl.d/99-security-hardening.conf")
    if err ~= nil then
        log("Could not read security-hardening.conf: " .. err)
        return true, 0
    end

    if not string.find(content, "net.ipv4.tcp_syncookies%s*=%s*1") then
        return true, 0
    end

    return false, 0
end

function exec()
    log("Writing security hardening sysctl configurations...")
    local config = [[
# Created by sur - Linux Hardening Assistant
# Protects against IP spoofing, SYN flood attacks, and ICMP redirects

# Enable TCP SYN Cookies
net.ipv4.tcp_syncookies = 1

# IP Spoofing protection (Reverse Path Filtering)
net.ipv4.conf.all.rp_filter = 1
net.ipv4.conf.default.rp_filter = 1

# Ignore ICMP broadcast requests (prevent Smurf attacks)
net.ipv4.icmp_echo_ignore_broadcasts = 1

# Ignore bogus ICMP error responses
net.ipv4.icmp_ignore_bogus_error_responses = 1

# Disable ICMP redirect acceptance (prevent MITM attacks)
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.default.accept_redirects = 0
net.ipv6.conf.all.accept_redirects = 0
net.ipv6.conf.default.accept_redirects = 0

# Do not send ICMP redirects (this host is not a router)
net.ipv4.conf.all.send_redirects = 0
net.ipv4.conf.default.send_redirects = 0

# Disable source routed packets (prevent routing exploits)
net.ipv4.conf.all.accept_source_route = 0
net.ipv4.conf.default.accept_source_route = 0
net.ipv6.conf.all.accept_source_route = 0
net.ipv6.conf.default.accept_source_route = 0

# Log Martian packets
net.ipv4.conf.all.log_martians = 1
net.ipv4.conf.default.log_martians = 1
]]

    local write_err = write_file("/etc/sysctl.d/99-security-hardening.conf", config)
    if write_err ~= nil then
        return "Could not write /etc/sysctl.d/99-security-hardening.conf: " .. write_err
    end

    log("Applying new sysctl rules...")
    local out, code = run("sysctl --system")
    if code ~= 0 then
        return "Failed to apply sysctl rules: " .. out
    end

    return nil
end

function post_check()
    -- Check if SYN cookies are active in kernel
    local out, code = run("sysctl -n net.ipv4.tcp_syncookies")
    if code ~= 0 then
        return "Failed to read net.ipv4.tcp_syncookies state: " .. out
    end

    if string.find(out, "1") == nil then
        return "net.ipv4.tcp_syncookies is not active"
    end

    -- Check if accept_redirects is disabled
    local redirect_out, redirect_code = run("sysctl -n net.ipv4.conf.all.accept_redirects")
    if redirect_code ~= 0 then
        return "Failed to read net.ipv4.conf.all.accept_redirects state"
    end

    if string.find(redirect_out, "0") == nil then
        return "net.ipv4.conf.all.accept_redirects is not disabled"
    end

    return nil
end

function rollback(backup_path)
    if backup_path == nil or backup_path == "" then
        log("No backup file existed. Removing /etc/sysctl.d/99-security-hardening.conf...")
        run("rm -f /etc/sysctl.d/99-security-hardening.conf")
    else
        log("Backup restored by runner. Re-applying sysctl config...")
    end

    -- Apply current system configurations to reload default parameters
    local out, code = run("sysctl --system")
    if code ~= 0 then
        return "Failed to reload sysctl config during rollback: " .. out
    end

    return nil
end
