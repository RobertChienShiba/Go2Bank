import os
import requests
import psycopg2

# download the exchange rate data from the website
url = 'https://rate.bot.com.tw/xrt/flcsv/0/day'
rate = requests.get(url)
rate.encoding = 'utf-8'
rts = rate.text.strip().split('\n')

currency_set = set()

file_path = os.path.dirname(__file__) + "/currency.txt"
with open(file_path, "r", encoding="utf-8") as file:
    for line in file:
        currency_set.add(line.strip()) 

# load the environment variables
db_source = os.getenv("DB_SOURCE")

# link to the database
conn = psycopg2.connect(db_source)
cur = conn.cursor()

# clear the data in the currencies table
cur.execute("""
    TRUNCATE TABLE currencies
""")
conn.commit()

# create currencies table (if not exists)
cur.execute("""
    CREATE TABLE IF NOT EXISTS currencies (
        currency VARCHAR(50) PRIMARY KEY,
        rate FLOAT NOT NULL,
        created_at TIMESTAMP DEFAULT NOW
    )
""")
conn.commit()

# parse the exchange rate data and insert into the database
data = []
for i in rts:
    try:
        a = i.split(',')
        if a[0] in currency_set:
            currency = a[0].strip()  # currency
            rate = float(a[12].strip())  # exchange rate
            data.append((currency, rate))
    except (IndexError, ValueError):
        continue  

# insert data into the database
if data:
    cur.executemany("""
        INSERT INTO currencies (currency, rate)
        VALUES (%s, %s)
        ON CONFLICT (currency) DO UPDATE SET rate = EXCLUDED.rate
    """, data)
    conn.commit()

# close the connection
cur.close()
conn.close()

print("Done!")

