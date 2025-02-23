
-- creates table for products. Product SKU's should be unique.
CREATE TABLE IF NOT EXISTS product (
	id SERIAL PRIMARY KEY,
	sku TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	price decimal(12,2) NOT NULL,
	qty INTEGER NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- initial insert of products.
INSERT INTO product (sku, name, price, qty)VALUES
('120P90', 'Google TV', 49.99, 10),
('43N23P', 'Macbook Pro', 5399.99, 5),
('A304SD', 'Alexa Speaker', 109.50, 10),
('234234', 'Raspberry Pi', 30.00, 2);