 #!/bin/bash

 sqlite3 ./reference/rc_domain_embeds.sqlite3 "SELECT d2.domain, d2.country, d2.distance FROM domains AS d2
 WHERE d2.embedding MATCH (SELECT embedding FROM domains WHERE domain = 'ford.com') AND k = 10
 ORDER BY d2.distance
 "


