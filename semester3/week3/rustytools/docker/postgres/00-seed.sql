CREATE TABLE orders (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id bigint NOT NULL CHECK (customer_id > 0),
    status text NOT NULL CHECK (status IN ('pending', 'paid', 'shipped', 'delivered', 'cancelled', 'refunded')),
    total_cents bigint NOT NULL CHECK (total_cents >= 0),
    created_at timestamptz NOT NULL
);

CREATE INDEX orders_customer_created_at_idx
    ON orders (customer_id, created_at DESC);

INSERT INTO orders (customer_id, status, total_cents, created_at) VALUES
    (1, 'delivered', 2599,  '2026-07-02 09:15:00+00'),
    (1, 'shipped',   8499,  '2026-07-18 14:20:00+00'),
    (1, 'paid',      1299,  '2026-07-29 08:05:00+00'),
    (1, 'pending',   15999, '2026-08-04 16:45:00+00'),
    (2, 'cancelled', 4999,  '2026-06-11 11:30:00+00'),
    (2, 'delivered', 3299,  '2026-07-07 10:10:00+00'),
    (2, 'refunded',  2199,  '2026-07-22 18:35:00+00'),
    (2, 'shipped',   6799,  '2026-08-03 12:00:00+00'),
    (3, 'delivered', 999,   '2026-05-19 07:40:00+00'),
    (3, 'delivered', 4599,  '2026-06-28 13:25:00+00'),
    (3, 'paid',      7500,  '2026-07-31 20:15:00+00'),
    (3, 'pending',   1899,  '2026-08-05 01:30:00+00');
