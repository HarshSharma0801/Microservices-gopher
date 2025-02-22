CREATE TABLE IF NOT EXISTS orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    amount INTEGER NOT NULL,
    description TEXT,
    payment_status TEXT DEFAULT 'pending',
    FOREIGN KEY (user_id) REFERENCES users(id)
);