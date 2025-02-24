CREATE TABLE IF NOT EXISTS "currencies" (
    "currency" varchar PRIMARY KEY,
    "rate" float NOT NULL,
    "created_at" timestamp NOT NULL DEFAULT (now())
);