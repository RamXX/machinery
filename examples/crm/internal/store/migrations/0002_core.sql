-- The core tables. The service, as crm_app, may read, insert and update the
-- records and only read and append the audit trail.
CREATE TABLE users (
	id    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	email text NOT NULL UNIQUE,
	role  text NOT NULL CHECK (role IN ('Admin', 'Rep'))
);
CREATE TABLE deals (
	id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	title        text NOT NULL,
	stage        text NOT NULL,
	amount_cents bigint NOT NULL DEFAULT 0 CHECK (amount_cents >= 0),
	owner_id     uuid NOT NULL REFERENCES users (id)
);
CREATE TABLE tasks (
	id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	title       text NOT NULL,
	status      text NOT NULL,
	deal_id     uuid NOT NULL REFERENCES deals (id),
	assignee_id uuid NOT NULL REFERENCES users (id)
);
CREATE TABLE audit_entries (
	id       bigserial PRIMARY KEY,
	actor_id uuid NOT NULL REFERENCES users (id),
	action   text NOT NULL,
	subject  text NOT NULL,
	outcome  text NOT NULL CHECK (outcome IN ('applied', 'refused', 'failed')),
	at       timestamptz NOT NULL DEFAULT now()
);
GRANT SELECT, INSERT, UPDATE ON users, deals, tasks TO crm_app;
GRANT SELECT, INSERT ON audit_entries TO crm_app;
GRANT USAGE ON SEQUENCE audit_entries_id_seq TO crm_app;
