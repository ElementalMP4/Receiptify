Reference = {}

local state_file = REFERENCE_DATA_FOLDER .. "/refs.ini"
local refs = {}

local function load_refs()
    local f = io.open(state_file, "r")
    if not f then return end
    for line in f:lines() do
        local key, val = line:match("^(.-)=(%d+)$")
        if key and val then
            refs[key] = tonumber(val)
        end
    end
    f:close()
end

local function save_refs()
    local f = io.open(state_file, "w")
    if not f then return end
    for k, v in pairs(refs) do
        f:write(k .. "=" .. v .. "\n")
    end
    f:close()
end

function Reference.ref(target)
    refs[target] = (refs[target] or 0) + 1
    save_refs()
    return target .. "-" .. refs[target]
end

load_refs()

return Reference
