-- 001_init: core tables for the ticket marketplace.
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(64) PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    email VARCHAR(190) NOT NULL,
    phone VARCHAR(40) NULL,
    password_hash VARCHAR(255) NOT NULL,
    role ENUM('buyer','seller','super_admin') NOT NULL DEFAULT 'buyer',
    seller_verified TINYINT(1) NOT NULL DEFAULT 0,
    bank_name VARCHAR(100) NULL,
    bank_account VARCHAR(100) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_users_email (email),
    KEY idx_users_role (role)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS events (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    seller_id BIGINT UNSIGNED NOT NULL,
    title VARCHAR(200) NOT NULL,
    description TEXT NULL,
    category VARCHAR(80) NULL,
    venue VARCHAR(200) NULL,
    address TEXT NULL,
    maps_url VARCHAR(500) NULL,
    starts_at DATETIME NOT NULL,
    ends_at DATETIME NOT NULL,
    banner_url VARCHAR(500) NULL,
    status ENUM('draft','pending_review','published','rejected','finished') NOT NULL DEFAULT 'draft',
    refund_policy TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_events_seller FOREIGN KEY (seller_id) REFERENCES users(id) ON DELETE RESTRICT,
    KEY idx_events_status (status),
    KEY idx_events_starts (starts_at),
    KEY idx_events_seller (seller_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS ticket_types (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    event_id BIGINT UNSIGNED NOT NULL,
    name VARCHAR(120) NOT NULL,
    price_cents BIGINT NOT NULL DEFAULT 0,
    quota INT NOT NULL DEFAULT 0,
    sold INT NOT NULL DEFAULT 0,
    sale_starts_at DATETIME NULL,
    sale_ends_at DATETIME NULL,
    CONSTRAINT fk_tt_event FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE,
    KEY idx_tt_event (event_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS orders (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    order_code VARCHAR(40) NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    event_id BIGINT UNSIGNED NOT NULL,
    subtotal_cents BIGINT NOT NULL DEFAULT 0,
    commission_cents BIGINT NOT NULL DEFAULT 0,
    gateway_fee_cents BIGINT NOT NULL DEFAULT 0,
    gateway_fee_bearer ENUM('buyer','seller','platform') NOT NULL DEFAULT 'buyer',
    gross_cents BIGINT NOT NULL DEFAULT 0,
    status ENUM('pending','paid','failed','cancelled','expired','refund_requested','refunded') NOT NULL DEFAULT 'pending',
    snap_token VARCHAR(255) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_orders_event FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE RESTRICT,
    UNIQUE KEY uq_orders_code (order_code),
    KEY idx_orders_user (user_id),
    KEY idx_orders_status (status),
    KEY idx_orders_created (created_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS order_items (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT UNSIGNED NOT NULL,
    ticket_type_id BIGINT UNSIGNED NOT NULL,
    ticket_code VARCHAR(40) NOT NULL,
    qr_payload VARCHAR(200) NOT NULL,
    used_at DATETIME NULL,
    price_cents BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT fk_oi_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    CONSTRAINT fk_oi_tt FOREIGN KEY (ticket_type_id) REFERENCES ticket_types(id) ON DELETE RESTRICT,
    UNIQUE KEY uq_oi_ticket_code (ticket_code),
    KEY idx_oi_order (order_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS payments (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT UNSIGNED NOT NULL,
    midtrans_order_id VARCHAR(80) NOT NULL,
    transaction_id VARCHAR(80) NOT NULL,
    payment_type VARCHAR(60) NULL,
    transaction_status VARCHAR(40) NOT NULL,
    gross_amount BIGINT NOT NULL DEFAULT 0,
    raw_notification JSON NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_pay_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    UNIQUE KEY uq_pay_idem (midtrans_order_id, transaction_id, transaction_status),
    KEY idx_pay_order (order_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS refunds (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT UNSIGNED NOT NULL,
    amount_cents BIGINT NOT NULL DEFAULT 0,
    reason TEXT NULL,
    status ENUM('requested','approved','rejected') NOT NULL DEFAULT 'requested',
    handled_by BIGINT UNSIGNED NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_ref_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE RESTRICT,
    KEY idx_ref_order (order_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS payouts (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    seller_id BIGINT UNSIGNED NOT NULL,
    event_id BIGINT UNSIGNED NOT NULL,
    amount_cents BIGINT NOT NULL DEFAULT 0,
    status ENUM('pending','paid','cancelled') NOT NULL DEFAULT 'pending',
    proof_url VARCHAR(500) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_payout_seller FOREIGN KEY (seller_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_payout_event FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE RESTRICT,
    KEY idx_payout_seller (seller_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS settings (
    `key` VARCHAR(100) NOT NULL PRIMARY KEY,
    value_encrypted TEXT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    actor_id BIGINT UNSIGNED NULL,
    action VARCHAR(120) NOT NULL,
    detail TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_audit_action (action),
    KEY idx_audit_created (created_at)
) ENGINE=InnoDB;
