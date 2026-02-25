\c bingwa;

BEGIN;

-- ============================================
-- CUSTOM TYPES
-- ============================================
CREATE TYPE offer_type AS ENUM ('data', 'sms', 'voice', 'combo');
CREATE TYPE offer_units AS ENUM ('GB', 'MB', 'KB', 'minutes', 'sms', 'units');
CREATE TYPE offer_status AS ENUM ('active', 'inactive', 'paused', 'suspended', 'archived');
CREATE TYPE transaction_status AS ENUM ('pending', 'processing', 'success', 'failed', 'cancelled', 'reversed');
CREATE TYPE subscription_status AS ENUM ('active', 'inactive', 'expired', 'cancelled', 'suspended');
CREATE TYPE ussd_processing_type AS ENUM ('express', 'multistep', 'callback');
CREATE TYPE renewal_period AS ENUM ('daily', 'weekly', 'monthly', 'quarterly', 'yearly');
CREATE TYPE payment_method AS ENUM ('mpesa', 'airtel_money', 'tigopesa', 'card', 'bank', 'agent_balance');

-- ============================================
-- AGENT CUSTOMERS (Non-login users)
-- ============================================
CREATE TABLE IF NOT EXISTS agent_customers (
    id BIGSERIAL PRIMARY KEY,
    agent_identity_id BIGINT NOT NULL, -- The agent who owns this customer
    customer_reference VARCHAR(50) UNIQUE NOT NULL, -- Unique ref for customer (e.g., CUST-XXXXX)
    
    -- Customer details
    full_name VARCHAR(255),
    phone_number VARCHAR(20) NOT NULL, -- Primary contact
    alt_phone_number VARCHAR(20), -- Alternative phone
    email VARCHAR(255),
    
    -- Status and flags
    is_active BOOLEAN DEFAULT TRUE,
    is_verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    
    -- Additional info
    notes TEXT, -- Agent notes about customer
    tags VARCHAR(50)[], -- e.g., ['vip', 'regular', 'corporate']
    metadata JSONB, -- Flexible field for additional data
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    -- Constraints
    CONSTRAINT fk_agent_identity FOREIGN KEY (agent_identity_id) 
        REFERENCES auth_identities(id) ON DELETE CASCADE,
    CONSTRAINT unique_agent_customer_phone UNIQUE(agent_identity_id, phone_number)
);

-- Index for fast lookups
CREATE INDEX idx_agent_customers_agent ON agent_customers(agent_identity_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_agent_customers_phone ON agent_customers(phone_number) WHERE deleted_at IS NULL;
CREATE INDEX idx_agent_customers_active ON agent_customers(agent_identity_id, is_active) WHERE deleted_at IS NULL;

-- ============================================
-- AGENT OFFERS
-- ============================================
CREATE TABLE IF NOT EXISTS agent_offers (
    id BIGSERIAL PRIMARY KEY,
    agent_identity_id BIGINT NOT NULL,
    offer_code VARCHAR(50) UNIQUE NOT NULL, -- Unique code for offer (e.g., DATA-5GB-30D)
    
    -- Offer details
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type offer_type NOT NULL,
    amount NUMERIC(10, 2) NOT NULL, -- Amount of data/minutes/sms
    units offer_units NOT NULL,
    
    -- Pricing
    price NUMERIC(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'KES',
    discount_percentage NUMERIC(5, 2) DEFAULT 0,
    
    -- Validity
    validity_days INT NOT NULL, -- Number of days the offer is valid
    validity_label VARCHAR(50), -- e.g., "30 days", "1 month", "unlimited"
    
    -- USSD Configuration
    ussd_code_template VARCHAR(255) NOT NULL, -- e.g., *181*{phone}*{amount}#
    ussd_processing_type ussd_processing_type NOT NULL DEFAULT 'express',
    ussd_expected_response VARCHAR(255), -- Expected success response pattern
    ussd_error_pattern VARCHAR(255), -- Pattern to detect errors
    
    -- Features
    is_featured BOOLEAN DEFAULT FALSE,
    is_recurring BOOLEAN DEFAULT FALSE, -- Can be auto-renewed
    max_purchases_per_customer INT, -- Limit purchases per customer
    
    -- Status
    status offer_status NOT NULL DEFAULT 'active',
    available_from TIMESTAMPTZ,
    available_until TIMESTAMPTZ,
    
    -- Metadata
    tags VARCHAR(50)[], -- e.g., ['popular', 'weekend-special']
    metadata JSONB,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT fk_offer_agent FOREIGN KEY (agent_identity_id) 
        REFERENCES auth_identities(id) ON DELETE CASCADE
);

CREATE INDEX idx_agent_offers_agent ON agent_offers(agent_identity_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_agent_offers_status ON agent_offers(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_agent_offers_type ON agent_offers(type) WHERE deleted_at IS NULL;
CREATE INDEX idx_agent_offers_price ON agent_offers(price) WHERE deleted_at IS NULL;

-- Table for USSD codes
BEGIN;
CREATE TABLE IF NOT EXISTS offer_ussd_codes (
    id BIGSERIAL PRIMARY KEY,
    offer_id BIGINT NOT NULL,
    
    -- USSD code details
    ussd_code VARCHAR(255) NOT NULL,
    signature_pattern VARCHAR(255), -- For learning signature matching
    priority INT NOT NULL DEFAULT 1, -- Lower number = higher priority
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Processing details
    expected_response VARCHAR(255),
    error_pattern VARCHAR(255),
    processing_type ussd_processing_type NOT NULL DEFAULT 'express',
    
    -- Usage statistics
    success_count INT DEFAULT 0,
    failure_count INT DEFAULT 0,
    last_used_at TIMESTAMPTZ,
    last_success_at TIMESTAMPTZ,
    last_failure_at TIMESTAMPTZ,
    
    -- Metadata
    metadata JSONB,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT fk_ussd_offer FOREIGN KEY (offer_id) 
        REFERENCES agent_offers(id) ON DELETE CASCADE,
    CONSTRAINT unique_offer_ussd UNIQUE (offer_id, ussd_code)
);

CREATE INDEX idx_offer_ussd_codes_offer ON offer_ussd_codes(offer_id);
CREATE INDEX idx_offer_ussd_codes_priority ON offer_ussd_codes(offer_id, priority) WHERE is_active = TRUE;
CREATE INDEX idx_offer_ussd_codes_success ON offer_ussd_codes(offer_id, success_count DESC);
COMMIT;

-- Keep the old columns for backward compatibility, but they'll reference the primary USSD
-- In migration, we can move existing data to the new table

-- ============================================
-- OFFER REQUESTS (M-Pesa payments)
-- ============================================
CREATE TABLE IF NOT EXISTS offer_requests (
    id BIGSERIAL PRIMARY KEY,
    request_reference VARCHAR(50) UNIQUE NOT NULL, -- Unique request reference
    
    -- Offer and customer info
    offer_id BIGINT NOT NULL,
    agent_identity_id BIGINT NOT NULL,
    customer_id BIGINT, -- NULL if customer not registered
    customer_phone VARCHAR(20) NOT NULL,
    customer_name VARCHAR(255),
    
    -- Payment details
    payment_method payment_method NOT NULL,
    amount_paid NUMERIC(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'KES',
    
    -- M-Pesa specific
    mpesa_transaction_id VARCHAR(50),
    mpesa_receipt_number VARCHAR(50),
    mpesa_transaction_date TIMESTAMPTZ,
    mpesa_phone_number VARCHAR(20),
    mpesa_message TEXT,
    
    -- Request details
    request_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    status transaction_status NOT NULL DEFAULT 'pending',
    failure_reason TEXT,
    retry_count INT DEFAULT 0,
    
    -- Metadata
    device_info JSONB, -- Device that made the request
    metadata JSONB,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT fk_request_offer FOREIGN KEY (offer_id) 
        REFERENCES agent_offers(id) ON DELETE CASCADE,
    CONSTRAINT fk_request_agent FOREIGN KEY (agent_identity_id) 
        REFERENCES auth_identities(id) ON DELETE CASCADE,
    CONSTRAINT fk_request_customer FOREIGN KEY (customer_id) 
        REFERENCES agent_customers(id) ON DELETE SET NULL
);

CREATE INDEX idx_offer_requests_offer ON offer_requests(offer_id);
CREATE INDEX idx_offer_requests_agent ON offer_requests(agent_identity_id);
CREATE INDEX idx_offer_requests_customer ON offer_requests(customer_id);
CREATE INDEX idx_offer_requests_phone ON offer_requests(customer_phone);
CREATE INDEX idx_offer_requests_status ON offer_requests(status);
CREATE INDEX idx_offer_requests_mpesa ON offer_requests(mpesa_transaction_id) WHERE mpesa_transaction_id IS NOT NULL;
CREATE INDEX idx_offer_requests_created ON offer_requests(created_at DESC);

-- ============================================
-- OFFER REDEMPTIONS (USSD Processing)
-- ============================================
CREATE TABLE IF NOT EXISTS offer_redemptions (
    id BIGSERIAL PRIMARY KEY,
    redemption_reference VARCHAR(50) UNIQUE NOT NULL,
    
    -- Related entities
    offer_id BIGINT,
    offer_request_id BIGINT NOT NULL,
    agent_identity_id BIGINT NOT NULL,
    customer_id BIGINT,
    customer_phone VARCHAR(20) NOT NULL,
    
    -- Redemption details
    amount NUMERIC(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'KES',
    ussd_code_used VARCHAR(255) NOT NULL, -- Actual USSD code sent
    
    -- USSD Response
    ussd_response TEXT,
    ussd_session_id VARCHAR(100),
    ussd_processing_time INT, -- Milliseconds
    
    -- Status
    redemption_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    status transaction_status NOT NULL DEFAULT 'pending',
    failure_reason TEXT,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    
    -- Validity
    valid_from TIMESTAMPTZ,
    valid_until TIMESTAMPTZ,
    
    -- Metadata
    metadata JSONB,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT fk_redemption_offer FOREIGN KEY (offer_id) 
        REFERENCES agent_offers(id) ON DELETE CASCADE,
    CONSTRAINT fk_redemption_request FOREIGN KEY (offer_request_id) 
        REFERENCES offer_requests(id) ON DELETE CASCADE,
    CONSTRAINT fk_redemption_agent FOREIGN KEY (agent_identity_id) 
        REFERENCES auth_identities(id) ON DELETE CASCADE,
    CONSTRAINT fk_redemption_customer FOREIGN KEY (customer_id) 
        REFERENCES agent_customers(id) ON DELETE SET NULL
);

CREATE INDEX idx_redemptions_offer ON offer_redemptions(offer_id) WHERE offer_id IS NOT NULL;
CREATE INDEX idx_redemptions_request ON offer_redemptions(offer_request_id);
CREATE INDEX idx_redemptions_agent ON offer_redemptions(agent_identity_id);
CREATE INDEX idx_redemptions_customer ON offer_redemptions(customer_id) WHERE customer_id IS NOT NULL;
CREATE INDEX idx_redemptions_status ON offer_redemptions(status);
CREATE INDEX idx_redemptions_created ON offer_redemptions(created_at DESC);

-- ============================================
-- SCHEDULED OFFERS (Auto-renewal)
-- ============================================
CREATE TABLE IF NOT EXISTS scheduled_offers (
    id BIGSERIAL PRIMARY KEY,
    schedule_reference VARCHAR(50) UNIQUE NOT NULL,
    
    -- Related entities
    offer_id BIGINT NOT NULL,
    agent_identity_id BIGINT NOT NULL,
    customer_id BIGINT,
    customer_phone VARCHAR(20) NOT NULL,
    
    -- Schedule details
    scheduled_time TIMESTAMPTZ NOT NULL,
    next_renewal_date TIMESTAMPTZ,
    last_renewal_date TIMESTAMPTZ,
    
    -- Auto-renewal configuration
    auto_renew BOOLEAN DEFAULT FALSE,
    renewal_period renewal_period,
    renewal_count INT DEFAULT 0,
    renewal_limit INT, -- NULL = unlimited
    renew_until TIMESTAMPTZ, -- Stop renewals after this date
    
    -- Status
    status subscription_status NOT NULL DEFAULT 'active',
    paused_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancellation_reason TEXT,
    
    -- Metadata
    metadata JSONB,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT fk_scheduled_offer FOREIGN KEY (offer_id) 
        REFERENCES agent_offers(id) ON DELETE CASCADE,
    CONSTRAINT fk_scheduled_agent FOREIGN KEY (agent_identity_id) 
        REFERENCES auth_identities(id) ON DELETE CASCADE,
    CONSTRAINT fk_scheduled_customer FOREIGN KEY (customer_id) 
        REFERENCES agent_customers(id) ON DELETE CASCADE
);

CREATE INDEX idx_scheduled_offers_offer ON scheduled_offers(offer_id);
CREATE INDEX idx_scheduled_offers_agent ON scheduled_offers(agent_identity_id);
CREATE INDEX idx_scheduled_offers_customer ON scheduled_offers(customer_id);
CREATE INDEX idx_scheduled_offers_status ON scheduled_offers(status);
CREATE INDEX idx_scheduled_offers_next_renewal ON scheduled_offers(next_renewal_date) WHERE status = 'active';

-- ============================================
-- SCHEDULED OFFER HISTORY
-- ============================================
CREATE TABLE IF NOT EXISTS scheduled_offer_history (
    id BIGSERIAL PRIMARY KEY,
    
    -- Related entities
    scheduled_offer_id BIGINT NOT NULL,
    offer_redemption_id BIGINT, -- Link to actual redemption
    customer_id BIGINT,
    customer_phone VARCHAR(20) NOT NULL,
    
    -- Renewal details
    renewal_time TIMESTAMPTZ NOT NULL,
    renewal_number INT NOT NULL, -- Which renewal attempt this was
    status transaction_status NOT NULL DEFAULT 'pending',
    failure_reason TEXT,
    
    -- Metadata
    metadata JSONB,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT fk_history_scheduled FOREIGN KEY (scheduled_offer_id) 
        REFERENCES scheduled_offers(id) ON DELETE CASCADE,
    CONSTRAINT fk_history_redemption FOREIGN KEY (offer_redemption_id) 
        REFERENCES offer_redemptions(id) ON DELETE SET NULL,
    CONSTRAINT fk_history_customer FOREIGN KEY (customer_id) 
        REFERENCES agent_customers(id) ON DELETE SET NULL
);

CREATE INDEX idx_history_scheduled ON scheduled_offer_history(scheduled_offer_id);
CREATE INDEX idx_history_customer ON scheduled_offer_history(customer_id);
CREATE INDEX idx_history_status ON scheduled_offer_history(status);
CREATE INDEX idx_history_created ON scheduled_offer_history(created_at DESC);

-- ============================================
-- SUBSCRIPTION PLANS
-- ============================================
CREATE TABLE IF NOT EXISTS subscription_plans (
    id BIGSERIAL PRIMARY KEY,
    plan_code VARCHAR(50) UNIQUE NOT NULL,
    
    -- Plan details
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Pricing
    price NUMERIC(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'KES',
    setup_fee NUMERIC(10, 2) DEFAULT 0,
    
    -- Billing
    billing_usage INT NOT NULL, -- Number of requests/redemptions allowed
    billing_cycle renewal_period NOT NULL,
    overage_charge NUMERIC(10, 2), -- Charge per extra request
    
    -- Features (what agents get)
    max_offers INT, -- Max offers agent can create
    max_customers INT, -- Max customers agent can have
    features JSONB, -- Additional features (analytics, bulk operations, etc.)
    
    -- Status
    status subscription_status NOT NULL DEFAULT 'active',
    is_public BOOLEAN DEFAULT TRUE, -- Can be subscribed to directly
    
    -- Metadata
    metadata JSONB,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_subscription_plans_status ON subscription_plans(status);
CREATE INDEX idx_subscription_plans_price ON subscription_plans(price);

-- ============================================
-- PROMOTIONAL CAMPAIGNS
-- ============================================
CREATE TABLE IF NOT EXISTS promotional_campaigns (
    id BIGSERIAL PRIMARY KEY,
    campaign_code VARCHAR(50) UNIQUE NOT NULL,
    
    -- Campaign details
    name VARCHAR(255) NOT NULL,
    description TEXT,
    promotional_code VARCHAR(50) NOT NULL UNIQUE,
    
    -- Discount
    discount_type VARCHAR(20) NOT NULL, -- percentage, fixed_amount, free_trial
    discount_value NUMERIC(10, 2) NOT NULL,
    max_discount_amount NUMERIC(10, 2), -- Cap for percentage discounts
    
    -- Validity
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    
    -- Usage limits
    max_uses INT, -- Total uses allowed
    uses_per_user INT DEFAULT 1,
    current_uses INT DEFAULT 0,
    
    -- Targeting
    applicable_plans BIGINT[], -- Array of plan IDs
    target_user_types VARCHAR(50)[], -- e.g., ['new_users', 'existing_users']
    
    -- Status
    status subscription_status NOT NULL DEFAULT 'active',
    
    -- Metadata
    metadata JSONB,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_campaigns_status ON promotional_campaigns(status);
CREATE INDEX idx_campaigns_code ON promotional_campaigns(promotional_code);
CREATE INDEX idx_campaigns_dates ON promotional_campaigns(start_date, end_date) WHERE status = 'active';

-- ============================================
-- AGENT SUBSCRIPTIONS
-- ============================================
CREATE TABLE IF NOT EXISTS agent_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    subscription_reference VARCHAR(50) UNIQUE NOT NULL,
    
    -- Related entities
    agent_identity_id BIGINT NOT NULL,
    subscription_plan_id BIGINT NOT NULL,
    promotional_campaign_id BIGINT,
    
    -- Subscription period
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ,
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,
    
    -- Renewal
    auto_renew BOOLEAN DEFAULT TRUE,
    renewal_count INT DEFAULT 0,
    next_billing_date TIMESTAMPTZ,
    
    -- Usage tracking
    requests_used INT DEFAULT 0,
    requests_limit INT,
    
    -- Pricing (snapshot at subscription time)
    plan_price NUMERIC(10, 2) NOT NULL,
    discount_applied NUMERIC(10, 2) DEFAULT 0,
    amount_paid NUMERIC(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'KES',
    
    -- Status
    status subscription_status NOT NULL DEFAULT 'active',
    cancelled_at TIMESTAMPTZ,
    cancellation_reason TEXT,
    
    -- Metadata
    metadata JSONB,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT fk_subscription_agent FOREIGN KEY (agent_identity_id) 
        REFERENCES auth_identities(id) ON DELETE CASCADE,
    CONSTRAINT fk_subscription_plan FOREIGN KEY (subscription_plan_id) 
        REFERENCES subscription_plans(id) ON DELETE CASCADE,
    CONSTRAINT fk_subscription_campaign FOREIGN KEY (promotional_campaign_id) 
        REFERENCES promotional_campaigns(id) ON DELETE SET NULL
);

CREATE INDEX idx_subscriptions_agent ON agent_subscriptions(agent_identity_id);
CREATE INDEX idx_subscriptions_plan ON agent_subscriptions(subscription_plan_id);
CREATE INDEX idx_subscriptions_status ON agent_subscriptions(status);
CREATE INDEX idx_subscriptions_next_billing ON agent_subscriptions(next_billing_date) WHERE status = 'active';

-- ============================================
-- AGENT CONFIGURATIONS
-- ============================================
CREATE TABLE IF NOT EXISTS agent_configs (
    id BIGSERIAL PRIMARY KEY,
    agent_identity_id BIGINT NOT NULL,
    
    -- Config details
    config_key VARCHAR(255) NOT NULL,
    config_value JSONB NOT NULL,
    description TEXT,
    
    -- Scope
    device_id VARCHAR(255), -- Config specific to a device
    is_global BOOLEAN DEFAULT TRUE, -- Applies to all devices
    
    -- Metadata
    metadata JSONB,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT fk_config_agent FOREIGN KEY (agent_identity_id) 
        REFERENCES auth_identities(id) ON DELETE CASCADE,
    CONSTRAINT unique_agent_config_key UNIQUE(agent_identity_id, config_key, device_id)
);

CREATE INDEX idx_agent_configs_agent ON agent_configs(agent_identity_id);
CREATE INDEX idx_agent_configs_key ON agent_configs(config_key);

-- ============================================
-- TRIGGERS FOR UPDATED_AT
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_agent_customers_updated_at BEFORE UPDATE ON agent_customers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_agent_offers_updated_at BEFORE UPDATE ON agent_offers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_offer_requests_updated_at BEFORE UPDATE ON offer_requests
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_offer_redemptions_updated_at BEFORE UPDATE ON offer_redemptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_scheduled_offers_updated_at BEFORE UPDATE ON scheduled_offers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_agent_subscriptions_updated_at BEFORE UPDATE ON agent_subscriptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_agent_configs_updated_at BEFORE UPDATE ON agent_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMIT;