CREATE TABLE IF NOT EXISTS tickets (
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    stock      INT NOT NULL CHECK (stock >= 0),
    price      NUMERIC(12, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS transactions (
    id         SERIAL PRIMARY KEY,
    ticket_id  INT NOT NULL REFERENCES tickets(id),
    buyer_name VARCHAR(100) NOT NULL,
    status     VARCHAR(20) NOT NULL DEFAULT 'success',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO tickets (name, stock, price) VALUES ('VIP Concert Ticket', 1, 1500000);
