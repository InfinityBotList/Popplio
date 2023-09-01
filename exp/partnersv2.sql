CREATE TABLE partners (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    short TEXT NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(user_id),
    image TEXT NOT NULL,
    links jsonb NOT NULL,
    type TEXT NOT NULL REFERENCES partner_types(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE partner_types (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    short TEXT NOT NULL,
    icon TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO partner_types (
    id,
    name,
    short,
    icon
) VALUES (
    'bot',
    'Bot Partner',
    'Meet some great discord bots officially partnered with us!',
    'bxs:bot'
);

INSERT INTO partner_types (
    id,
    name,
    short,
    icon
) VALUES (
    'featured',
    'Featured Partners',
    'These are the partners that we have chosen to featured on our website.',
    'material-symbols:featured-play-list'
);

INSERT INTO partner_types (
    id,
    name,
    short,
    icon
) VALUES (
    'botlist',
    'Bot List Partner',
    'These are the bot lists who we trust!',
    'material-symbols:list'
);
