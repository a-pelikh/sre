CREATE TABLE projects (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT        NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 120),
    description TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tasks (
    id          BIGSERIAL PRIMARY KEY,
    project_id  BIGINT      NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title       TEXT        NOT NULL CHECK (char_length(btrim(title)) BETWEEN 1 AND 200),
    description TEXT        NOT NULL DEFAULT '',
    status      TEXT        NOT NULL DEFAULT 'todo'
                            CHECK (status IN ('todo', 'in_progress', 'done')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX tasks_project_id_idx ON tasks(project_id);
CREATE INDEX tasks_status_idx ON tasks(status);
