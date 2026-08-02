SET search_path TO public;

-- AI assistant conversations + messages. The model only produces text; proposed
-- actions live on the assistant message's `actions` JSONB (each gains a result
-- once the client runs a write or the worker runs a read). Persisted for reload,
-- audit of what was proposed, and token/cost accounting.
CREATE TYPE ai_message_role AS ENUM ('user', 'assistant');
CREATE TYPE ai_message_status AS ENUM ('pending', 'awaiting_approval', 'done', 'failed');

CREATE TABLE ai_conversations (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    title character varying(200),
    memory text NOT NULL DEFAULT '',       -- rolling summary of folded-in old turns
    summarized_until timestamp without time zone, -- cursor: messages <= this are in memory
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    CONSTRAINT ai_conversations_pkey PRIMARY KEY (id),
    CONSTRAINT fk_ai_conv_user FOREIGN KEY (user_id)
      REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX idx_ai_conversations_user ON ai_conversations(user_id, updated_at DESC);

CREATE TABLE ai_messages (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    conversation_id uuid NOT NULL,
    role ai_message_role NOT NULL,
    text text NOT NULL DEFAULT '',
    actions jsonb NOT NULL DEFAULT '[]'::jsonb,
    status ai_message_status NOT NULL DEFAULT 'done',
    model character varying(128), -- provider/model that produced an assistant turn
    prompt_tokens integer NOT NULL DEFAULT 0,
    completion_tokens integer NOT NULL DEFAULT 0,
    total_tokens integer NOT NULL DEFAULT 0,
    created_at timestamp without time zone DEFAULT now(),
    CONSTRAINT ai_messages_pkey PRIMARY KEY (id),
    CONSTRAINT fk_ai_msg_conv FOREIGN KEY (conversation_id)
      REFERENCES ai_conversations(id) ON DELETE CASCADE
);
CREATE INDEX idx_ai_messages_conv ON ai_messages(conversation_id, created_at);
