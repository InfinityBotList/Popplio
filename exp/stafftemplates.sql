CREATE TABLE staff_template_types (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    icon TEXT NOT NULL,
    short TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO staff_template_types (
    id,
    name,
    icon,
    short
) VALUES (
    'approval',
    'Approval Templates',
    'material-symbols:check',
    'Choose the best approval reason based on the tags, title and text!'
);

INSERT INTO staff_template_types (
    id,
    name,
    icon,
    short
) VALUES (
    'denial',
    'Denial Templates',
    'material-symbols:close',
    'Choose the best denial reason based on the tags, title and text!'
);

CREATE TABLE staff_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    emoji TEXT NOT NULL,
    tags TEXT[] NOT NULL,
    description TEXT NOT NULL,
    type TEXT NOT NULL REFERENCES staff_template_types(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Then run convertTemplatesToSql.py