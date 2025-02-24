#!/bin/sh

apk add --no-cache postgresql-client

# check valid currency
FILE="currency.txt"

# convert into SQL format
CURRENCIES=$(awk '{printf "\047%s\047,", $0}' $FILE | sed 's/,$//')

# count the number of lines in the file
REQUIRED_COUNT=$(wc -l < $FILE | tr -d '[:space:]')

echo "Checking Currencies: ($CURRENCIES), Required count: $REQUIRED_COUNT" 

while true; do
    # execute the postgres sqlquery
    QUERY_COUNT=$(psql $DB_SOURCE -t -c "SELECT COUNT(DISTINCT currency) FROM currencies WHERE currency IN ($CURRENCIES);" | tr -d '[:space:]')

    # compare the query count with the required count
    if [ $QUERY_COUNT -eq $REQUIRED_COUNT ]; then
        echo "✅ All required categories are present ($QUERY_COUNT/$REQUIRED_COUNT)"
        exit 0
    fi

    echo "⏳ Waiting for required categories... ($QUERY_COUNT/$REQUIRED_COUNT)"
    exit 1
done

# start next command
# exec "$@"
