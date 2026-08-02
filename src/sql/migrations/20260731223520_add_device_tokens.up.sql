SET search_path TO public;

-- One row per app install that can receive a push. The token is unique table-wide,
-- not per user: registering an existing token moves it, so a phone handed to a
-- second account stops getting the first one's notifications.
CREATE TABLE device_tokens (
    id           uuid DEFAULT uuid_generate_v4() NOT NULL,
    user_id      uuid NOT NULL,
    token        text NOT NULL,
    platform     varchar(16) NOT NULL,
    locale       varchar(8),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT device_tokens_pkey PRIMARY KEY (id),
    CONSTRAINT device_tokens_token_key UNIQUE (token),
    CONSTRAINT device_tokens_user_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT device_tokens_platform_chk CHECK (platform IN ('android', 'ios', 'web'))
);
CREATE INDEX idx_device_tokens_user ON device_tokens (user_id);
