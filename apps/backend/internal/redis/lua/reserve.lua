local session_id = KEYS[1]
local category = ARGV[1]
local ttl = tonumber(ARGV[2])

local purchased_set = "purchased:sessions"
local hold_key = "hold:" .. session_id
local available_set = "tickets:available:" .. category

-- 1. Verify session does not already exist in purchased:sessions
if redis.call("SISMEMBER", purchased_set, session_id) == 1 then
    return { "ERR", "PURCHASE_LIMIT_EXCEEDED" }
end

-- 2. Verify session does not already have an active hold
if redis.call("EXISTS", hold_key) == 1 then
    return { "ERR", "ACTIVE_HOLD_EXISTS" }
end

-- 3. Verify there is at least one ticket ID in the set
local count = redis.call("SCARD", available_set)
if count == 0 then
    return { "ERR", "TICKET_UNAVAILABLE" }
end

-- 5. Pop ticket ID and set hold key
local ticket_id = redis.call("SPOP", available_set)
if not ticket_id then
    return { "ERR", "TICKET_UNAVAILABLE" }
end

redis.call("SET", hold_key, ticket_id .. ":" .. category, "EX", ttl)
return { "OK", ticket_id }
