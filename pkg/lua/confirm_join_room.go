package lua

import "github.com/redis/go-redis/v9"

func ConfirmJoinRoom() *redis.Script {
	return redis.NewScript(`
	local result = redis.pcall("HGETALL", KEYS[1])
	if type(result) == 'table' and result.err then
		return result.err
	end

	local confirmCount = 0

	for k, v in pairs(result) do
  		if k == ARGV[1] then
			v = "confirm"
		end 
		
		if v == "confirm" then
			confirmCount = confirmCount + 1
		end
	end

	return confirmCount
	`)
}
