id = "audit_ssh_keys"
name = "Audit Authorized SSH Keys"
description = "Audits authorized_keys files for all users on the system, checking for weak algorithms or key sizes."
rollback_possible = false
backup_files = {}
risk_level = "low"
distros = {}

function pre_check()
    return true, 0
end

local function audit_user_keys(username, home_dir)
    local keys_path = home_dir .. "/.ssh/authorized_keys"
    if not file_exists(keys_path) then
        return
    end

    log("Auditing keys for user: " .. username .. " at " .. keys_path)

    -- Use ssh-keygen to check the keys in authorized_keys
    local out, code = run("ssh-keygen -l -f " .. keys_path .. " 2>/dev/null")
    if code ~= 0 or out == nil or out == "" then
        log("  - Could not parse authorized_keys file with ssh-keygen (file might be empty or unreadable).")
        return
    end

    local keys_count = 0
    for line in string.gmatch(out, "[^\r\n]+") do
        keys_count = keys_count + 1
        
        -- Line format: <size> <fingerprint> <comment> (<type>)
        -- Examples:
        -- "2048 SHA256:abcd... user@host (RSA)"
        -- "256 SHA256:xyz... user@host (ED25519)"
        -- "1024 SHA256:123... user@host (DSA)"
        local size_str, fingerprint, comment, key_type = string.match(line, "^(%d+)%s+([^%s]+)%s+(.-)%s+%(([^%)]+)%)")
        if not size_str then
            -- Fallback match if comment is missing
            size_str, fingerprint, key_type = string.match(line, "^(%d+)%s+([^%s]+)%s+%(([^%)]+)%)")
            comment = "No Comment"
        end

        if size_str and key_type then
            local size = tonumber(size_str)
            local is_weak = false
            local warning_reason = ""

            key_type = string.upper(key_type)

            -- Audit Rules:
            -- 1. DSA/DSS is completely obsolete
            if key_type == "DSA" or key_type == "DSS" then
                is_weak = true
                warning_reason = "Obsolete and insecure DSA algorithm"
            -- 2. RSA under 2048 bits is insecure
            elseif key_type == "RSA" and size < 2048 then
                is_weak = true
                warning_reason = "Weak RSA key size (" .. size .. " bits < 2048)"
            -- 3. RSA 2048 is acceptable but 3072+ is recommended today
            elseif key_type == "RSA" and size == 2048 then
                log("  - Key #" .. keys_count .. ": RSA 2048 bit key (" .. comment .. "). Safe, but 3072+ bit or ED25519 is recommended.")
            end

            if is_weak then
                log("  - WARNING: Weak Key #" .. keys_count .. " (" .. key_type .. ", " .. size .. " bits, comment: '" .. comment .. "'): " .. warning_reason .. "!")
            else
                if key_type ~= "RSA" or size > 2048 then
                    log("  - Key #" .. keys_count .. ": Secure " .. key_type .. " key (" .. size .. " bits, comment: '" .. comment .. "').")
                end
            end
        else
            log("  - Could not parse key info line: " .. line)
        end
    end
end

function exec()
    log("Scanning system for user home directories and SSH authorized_keys...")
    
    local content, err = read_file("/etc/passwd")
    if err ~= nil then
        return "Could not read /etc/passwd: " .. err
    end

    local scanned_users = 0
    for line in string.gmatch(content, "[^\r\n]+") do
        -- username:passwd:UID:GID:gecos:home:shell
        local parts = {}
        for part in string.gmatch(line .. ":", "([^:]*):") do
            table.insert(parts, part)
        end

        if #parts >= 6 then
            local username = parts[1]
            local home_dir = parts[6]
            
            -- Scan authorized keys if home directory exists
            if home_dir ~= "" and home_dir ~= "/" then
                audit_user_keys(username, home_dir)
                scanned_users = scanned_users + 1
            end
        end
    end

    log("Scan complete. Scanned home directories of " .. scanned_users .. " user account(s).")
    return nil
end

function post_check()
    return nil
end
