-- 003_phase2 down: reverse every change from the up migration.
UPDATE payouts SET status='pending' WHERE status='processing';
ALTER TABLE payouts MODIFY COLUMN status ENUM('pending','paid','cancelled') NOT NULL DEFAULT 'pending';
ALTER TABLE payouts DROP COLUMN iris_status;
ALTER TABLE payouts DROP COLUMN iris_reference_no;
ALTER TABLE payouts DROP COLUMN method;

ALTER TABLE refunds DROP COLUMN note;
ALTER TABLE refunds DROP COLUMN raw_response;
ALTER TABLE refunds DROP COLUMN midtrans_refund_id;

ALTER TABLE orders DROP COLUMN promo_code;
ALTER TABLE orders DROP COLUMN discount_cents;

ALTER TABLE order_items DROP FOREIGN KEY fk_oi_holder;
ALTER TABLE order_items DROP KEY idx_oi_holder;
ALTER TABLE order_items DROP COLUMN holder_user_id;

DROP TABLE IF EXISTS ticket_transfers;
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS promo_usages;
DROP TABLE IF EXISTS promo_codes;
