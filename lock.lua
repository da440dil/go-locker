local token = redis.call("get", KEYS[1])
if token == false then
	redis.call("set", KEYS[1], ARGV[1], "px", ARGV[2])
	return -3
end
if token == ARGV[1] then
	redis.call("pexpire", KEYS[1], ARGV[2])
	return -4
end
return redis.call("pttl", KEYS[1])