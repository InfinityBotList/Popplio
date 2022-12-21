/*
Filter format:

-- Filtername filter (X-[X+1])
AND (servers >= $X)
AND (($X+1 = 0) OR (servers <= $X+1))
*/

SELECT DISTINCT {{.Cols}} FROM bots
WHERE (type = 'approved' OR type = 'certified')

-- Guild count filter (1-2)
AND (servers >= $1)
AND (($2 = 0) OR (servers <= $2))

-- Votes filter (3-4)
AND (votes >= $3)
AND (($4 = 0) OR (votes <= $4))

-- Shards filter (5-6)
AND (shards >= $5)
AND (($6 = 0) OR (shards <= $6))

-- Tags filter
AND (cardinality($7::text[]) = 0 OR tags {{.TagMode}} $7) -- Where TagMode is one of @> = all, && = any

{{if .Query}}
AND (queue_name ILIKE $8 OR vanity ILIKE $8 OR owner @@ $9 OR short @@ $9) 
{{end}}

ORDER BY votes DESC, type DESC LIMIT 12
