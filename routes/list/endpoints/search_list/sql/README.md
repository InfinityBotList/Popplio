# Filter format:

-- Filtername filter (X-[X+1])
AND (servers >= $X)
AND (($X+1 = 0) OR (servers <= $X+1))

