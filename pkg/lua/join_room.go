package lua

import (
	"github.com/redis/go-redis/v9"
)

func JoinRoom() *redis.Script {
	return redis.NewScript(`
	local pushResult = redis.pcall("RPUSH", KEYS[1], ARGV[2])
	if type(pushResult) == 'table' and pushResult.err then
		return pushResult.err
	end

	local result = redis.pcall("LRANGE", KEYS[1], 0, ARGV[1]-1)
	if type(result) == 'table' and result.err then
		return result.err
	end

	if #result < tonumber(ARGV[1]) then
		return result
	end

	local trimResult = redis.pcall("LTRIM", KEYS[1], ARGV[1], -1)
	if type(trimResult) == 'table' and trimResult.err then
		return trimResult.err
	end

	return result
	`)
}
