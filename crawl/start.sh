#!/bin/sh

python3 crawl.py

chmod +x crawl.py

echo "0       0       *       *       *       python3 /crawl/crawl.py" >> /var/spool/cron/crontabs/root

crond -f -l 2
