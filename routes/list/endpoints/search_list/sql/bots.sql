/*
Filter format:

-- Filtername filter (X-[X+1])
AND (servers >= $X)
AND (($X+1 = -1) OR (servers <= $X+1))
*/

SELECT DISTINCT {cols} FROM bots
WHERE (type = 'approved' OR type = 'certified')
AND (queue_name ILIKE $2 OR vanity ILIKE $2 OR owner @@ $1 OR short @@ $1) 

-- Guild count filter (3-4)
AND (servers >= $3)
AND (($4 = -1) OR (servers <= $4))

-- Votes filter (5-6)
AND (votes >= $5)
AND (($6 = -1) OR (votes <= $6))

-- Shards filter (7-8)
AND (shards >= $7)
AND (($8 = -1) OR (shards <= $8))

-- Tags filter
AND (cardinality($9::text[]) = 0 OR tags {op} $9) -- Where op is one of @> = all, && = any

ORDER BY votes DESC, type DESC LIMIT 6