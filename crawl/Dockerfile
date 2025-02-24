FROM python:3.10.16-alpine3.21

WORKDIR /crawl

COPY requirements.txt .

RUN pip install --no-cache-dir -r requirements.txt

COPY wait-for-data.sh .
COPY crawl.py .
COPY currency.txt .
COPY start.sh .

CMD ["sh", "./start.sh"]
