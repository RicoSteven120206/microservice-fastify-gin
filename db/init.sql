CREATE TABLE IF NOT EXISTS products (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    tsv tsvector GENERATED ALWAYS AS
        (to_tsvector('simple', name || ' ' || description)) STORED
);

INSERT INTO products (name, description) VALUES
  ('Laptop Gaming ASUS ROG',  'Laptop gaming performa tinggi dengan RTX dan layar 165Hz'),
  ('Laptop Ultrabook Lenovo', 'Laptop tipis ringan untuk kerja dan kuliah'),
  ('Mouse Wireless Logitech', 'Mouse nirkabel ergonomis baterai awet'),
  ('Keyboard Mekanikal',      'Keyboard mekanikal switch biru untuk gaming dan mengetik'),
  ('Monitor 27 inch 4K',      'Monitor 4K IPS untuk desain dan produktivitas'),
  ('Headset Gaming',          'Headset gaming surround dengan mikrofon noise cancelling'),
  ('Kursi Ergonomis',         'Kursi kantor ergonomis dengan sandaran lumbar'),
  ('Webcam Full HD',          'Webcam 1080p untuk meeting online dan streaming');

INSERT INTO products (name, description)
SELECT 
    initcap(w.word) || ' ' || a.adj || ' merk' || (i % 1000) || ' seri ' || i
    || CASE WHEN i % 100000 = 7 THEN ' zebra' ELSE '' END,
    'Produk ' || w.word || ' ' || a.adj || ' nomor ' || i
    || ' untuk kebutuhan sehari-hari dan profesional'
FROM generate_series(9, 100000) AS i,
LATERAL (SELECT (ARRAY['laptop','mouse','keyboard','monitor','headset',
    'kursi','webcam','speaker','tablet','kamera'])[1 + (i % 10)::int] AS word) w,
LATERAL (SELECT(ARRAY['gaming','kantor','murah','premium','wireless','portable',
    'ergonomis','compact'])[1 + ((i / 10) % 8)::int] AS adj) a;

CREATE INDEX IF NOT EXISTS products_tsv_idx ON products USING GIN (tsv);
ANALYZE products; 