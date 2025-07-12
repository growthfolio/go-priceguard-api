-- Initial schema for PriceGuard API
-- Create UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    google_id VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    picture TEXT,
    avatar TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- User settings table
CREATE TABLE user_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    theme VARCHAR(20) DEFAULT 'dark',
    default_timeframe VARCHAR(10) DEFAULT '1h',
    default_view VARCHAR(20) DEFAULT 'overview',
    notifications_email BOOLEAN DEFAULT true,
    notifications_push BOOLEAN DEFAULT true,
    notifications_sms BOOLEAN DEFAULT false,
    risk_profile VARCHAR(20) DEFAULT 'moderate',
    favorite_symbols TEXT[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id)
);

-- Cryptocurrencies table
CREATE TABLE cryptocurrencies (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    market_type VARCHAR(20) DEFAULT 'Spot',
    image_url TEXT,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Alerts table
CREATE TABLE alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    symbol VARCHAR(20) NOT NULL,
    alert_type VARCHAR(50) NOT NULL, -- 'price', 'rsi', 'ema_cross', etc.
    condition_type VARCHAR(20) NOT NULL, -- 'above', 'below', 'crosses'
    target_value DECIMAL(20, 8) NOT NULL,
    timeframe VARCHAR(10) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    notify_via TEXT[] DEFAULT '{app}', -- ['app', 'email', 'sms']
    triggered_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Notifications table
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alert_id UUID REFERENCES alerts(id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    notification_type VARCHAR(50) NOT NULL, -- 'alert_triggered', 'system', etc.
    read_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Price history table (for caching historical data)
CREATE TABLE price_history (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    timeframe VARCHAR(10) NOT NULL,
    open_price DECIMAL(20, 8) NOT NULL,
    high_price DECIMAL(20, 8) NOT NULL,
    low_price DECIMAL(20, 8) NOT NULL,
    close_price DECIMAL(20, 8) NOT NULL,
    volume DECIMAL(30, 8) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(symbol, timeframe, timestamp)
);

-- Technical indicators cache table
CREATE TABLE technical_indicators (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    timeframe VARCHAR(10) NOT NULL,
    indicator_type VARCHAR(50) NOT NULL, -- 'rsi', 'ema', 'sma', 'supertrend', etc.
    value DECIMAL(20, 8),
    metadata JSONB, -- For storing additional indicator data
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(symbol, timeframe, indicator_type, timestamp)
);

-- Sessions table (for JWT token management)
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX idx_users_google_id ON users(google_id);
CREATE INDEX idx_users_email ON users(email);

CREATE INDEX idx_alerts_user_id ON alerts(user_id);
CREATE INDEX idx_alerts_symbol ON alerts(symbol);
CREATE INDEX idx_alerts_enabled ON alerts(enabled);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_read_at ON notifications(read_at);
CREATE INDEX idx_notifications_created_at ON notifications(created_at);

CREATE INDEX idx_price_history_symbol_timeframe ON price_history(symbol, timeframe);
CREATE INDEX idx_price_history_timestamp ON price_history(timestamp);

CREATE INDEX idx_technical_indicators_symbol_timeframe ON technical_indicators(symbol, timeframe);
CREATE INDEX idx_technical_indicators_type ON technical_indicators(indicator_type);
CREATE INDEX idx_technical_indicators_timestamp ON technical_indicators(timestamp);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Insert some default cryptocurrencies
INSERT INTO cryptocurrencies (symbol, name, market_type, image_url) VALUES
('BTCUSDT', 'Bitcoin', 'Futures', 'https://bin.bnbstatic.com/static/assets/logos/BTC.png'),
('ETHUSDT', 'Ethereum', 'Futures', 'https://bin.bnbstatic.com/static/assets/logos/ETH.png'),
('BNBUSDT', 'Binance Coin', 'Futures', 'https://bin.bnbstatic.com/static/assets/logos/BNB.png'),
('ADAUSDT', 'Cardano', 'Futures', 'https://bin.bnbstatic.com/static/assets/logos/ADA.png'),
('SOLUSDT', 'Solana', 'Futures', 'https://bin.bnbstatic.com/static/assets/logos/SOL.png'),
('XRPUSDT', 'XRP', 'Futures', 'https://bin.bnbstatic.com/static/assets/logos/XRP.png'),
('DOTUSDT', 'Polkadot', 'Futures', 'https://bin.bnbstatic.com/static/assets/logos/DOT.png'),
('AVAXUSDT', 'Avalanche', 'Futures', 'https://bin.bnbstatic.com/static/assets/logos/AVAX.png'),
('MATICUSDT', 'Polygon', 'Futures', 'https://bin.bnbstatic.com/static/assets/logos/MATIC.png'),
('LINKUSDT', 'Chainlink', 'Futures', 'https://bin.bnbstatic.com/static/assets/logos/LINK.png');
