SET search_path TO public;

-- Support tickets: a user opens one, the conversation is a list of messages, and
-- an admin answers from the admin panel. The panel writes straight to Postgres
-- (it doesn't go through the API's event bus), so the worker watches for the
-- admin's replies and pushes them to the user — that's what `delivered_at` on
-- support_messages is for.
--
-- The conversation is turn-based: each side must wait for the other to answer.
-- `turn` says whose move it is; a send is only allowed when it's that side's
-- turn, and completing it flips the turn. Enforced in code on both sides (the
-- API for the user, the panel's Reply action for the admin).

CREATE TABLE support_tickets (
    id           uuid DEFAULT uuid_generate_v4() NOT NULL,
    user_id      uuid NOT NULL,
    subject      text NOT NULL,
    status       varchar(16) NOT NULL DEFAULT 'OPEN',
    -- Whose turn it is to send next. A brand-new ticket carries the user's first
    -- message, so it opens already awaiting the admin.
    turn         varchar(8) NOT NULL DEFAULT 'ADMIN',
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    -- Bumped on every message so the panel and the user's list sort by activity.
    last_message_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT support_tickets_pkey PRIMARY KEY (id),
    CONSTRAINT support_tickets_user_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT support_tickets_status_chk CHECK (status IN ('OPEN', 'CLOSED')),
    CONSTRAINT support_tickets_turn_chk CHECK (turn IN ('USER', 'ADMIN'))
);
CREATE INDEX idx_support_tickets_user ON support_tickets (user_id, last_message_at DESC);
CREATE INDEX idx_support_tickets_activity ON support_tickets (last_message_at DESC);

CREATE TABLE support_messages (
    id         uuid DEFAULT uuid_generate_v4() NOT NULL,
    ticket_id  uuid NOT NULL,
    -- USER = the ticket owner, ADMIN = staff (via the panel), SYSTEM = automated.
    sender     varchar(8) NOT NULL,
    body       text NOT NULL,
    -- NULL until the worker has pushed this message to the user. Only ever set
    -- for ADMIN/SYSTEM messages — a USER message is already "delivered" (the user
    -- wrote it), and the API pings Discord for it synchronously. This column is
    -- the hand-off between the panel (which can't emit events) and the worker.
    delivered_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT support_messages_pkey PRIMARY KEY (id),
    CONSTRAINT support_messages_ticket_fk FOREIGN KEY (ticket_id) REFERENCES support_tickets(id) ON DELETE CASCADE,
    CONSTRAINT support_messages_sender_chk CHECK (sender IN ('USER', 'ADMIN', 'SYSTEM'))
);
CREATE INDEX idx_support_messages_ticket ON support_messages (ticket_id, created_at);
-- The worker's claim query: find admin/system messages not yet pushed. Partial
-- index keeps it tiny — only undelivered rows, which is a short-lived state.
CREATE INDEX idx_support_messages_undelivered ON support_messages (created_at)
    WHERE delivered_at IS NULL AND sender <> 'USER';
