-- 002_order_lines: persist checkout lines so tickets can be generated on payment.
CREATE TABLE IF NOT EXISTS order_lines (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT UNSIGNED NOT NULL,
    ticket_type_id BIGINT UNSIGNED NOT NULL,
    qty INT NOT NULL,
    price_cents BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT fk_ol_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    CONSTRAINT fk_ol_tt FOREIGN KEY (ticket_type_id) REFERENCES ticket_types(id) ON DELETE RESTRICT,
    KEY idx_ol_order (order_id)
) ENGINE=InnoDB;
