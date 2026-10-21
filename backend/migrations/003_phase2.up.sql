-- 003_phase2: promo codes, reviews, ticket transfers, refund/payout phase-2 fields.
CREATE TABLE IF NOT EXISTS promo_codes (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(60) NOT NULL,
    type ENUM('percent','flat') NOT NULL,
    value DECIMAL(12,2) NOT NULL,
    min_spend_cents BIGINT NOT NULL DEFAULT 0,
    quota INT NULL,
    per_user_limit INT NULL,
    starts_at DATETIME NULL,
    ends_at DATETIME NULL,
    scope ENUM('global','event') NOT NULL DEFAULT 'global',
    event_id BIGINT UNSIGNED NULL,
    seller_id BIGINT UNSIGNED NOT NULL,
    active TINYINT(1) NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_pc_event FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE,
    CONSTRAINT fk_pc_seller FOREIGN KEY (seller_id) REFERENCES users(id) ON DELETE RESTRICT,
    UNIQUE KEY uq_pc_code (code),
    KEY idx_pc_seller (seller_id),
    KEY idx_pc_event (event_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS promo_usages (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    promo_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    order_id BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_pu_promo FOREIGN KEY (promo_id) REFERENCES promo_codes(id) ON DELETE CASCADE,
    CONSTRAINT fk_pu_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_pu_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    UNIQUE KEY uq_pu_order (promo_id, order_id),
    KEY idx_pu_promo_user (promo_id, user_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS reviews (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    event_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    rating TINYINT NOT NULL,
    text TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_rev_event FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE,
    CONSTRAINT fk_rev_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT chk_rev_rating CHECK (rating BETWEEN 1 AND 5),
    UNIQUE KEY uq_rev_event_user (event_id, user_id),
    KEY idx_rev_event (event_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS ticket_transfers (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    ticket_code VARCHAR(40) NOT NULL,
    from_user_id BIGINT UNSIGNED NOT NULL,
    to_user_id BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_tt_from FOREIGN KEY (from_user_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_tt_to FOREIGN KEY (to_user_id) REFERENCES users(id) ON DELETE RESTRICT,
    KEY idx_tt_from (from_user_id),
    KEY idx_tt_to (to_user_id)
) ENGINE=InnoDB;

ALTER TABLE order_items ADD COLUMN holder_user_id BIGINT UNSIGNED NULL;
ALTER TABLE order_items ADD CONSTRAINT fk_oi_holder FOREIGN KEY (holder_user_id) REFERENCES users(id) ON DELETE RESTRICT;
ALTER TABLE order_items ADD KEY idx_oi_holder (holder_user_id);

ALTER TABLE orders ADD COLUMN discount_cents BIGINT NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN promo_code VARCHAR(60) NULL;

ALTER TABLE refunds ADD COLUMN midtrans_refund_id VARCHAR(120) NULL;
ALTER TABLE refunds ADD COLUMN raw_response TEXT NULL;
ALTER TABLE refunds ADD COLUMN note VARCHAR(255) NULL;

ALTER TABLE payouts ADD COLUMN method VARCHAR(20) NOT NULL DEFAULT 'manual';
ALTER TABLE payouts ADD COLUMN iris_reference_no VARCHAR(120) NULL;
ALTER TABLE payouts ADD COLUMN iris_status VARCHAR(40) NULL;
ALTER TABLE payouts MODIFY COLUMN status ENUM('pending','processing','paid','cancelled') NOT NULL DEFAULT 'pending';
