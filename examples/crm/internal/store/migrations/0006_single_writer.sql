-- The database is the last line of defence of the lifecycles
-- (inv:lifecycle-single-writer). For the app role, a deal or a task may be
-- created only in its machine's initial state, and its state may change only
-- along an edge of its machine and only inside the transition path, which
-- sets crm.transition in the transaction. The service fills lifecycle_edges
-- from its tables at start; the admin role is not restricted.
CREATE TABLE lifecycle_edges (
	machine text NOT NULL,
	source  text NOT NULL, -- empty for the initial state
	target  text NOT NULL,
	PRIMARY KEY (machine, source, target)
);
GRANT SELECT ON lifecycle_edges TO crm_app;

-- lifecycle_guard(machine, state column) guards one table.
CREATE FUNCTION lifecycle_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
	m_name    text := TG_ARGV[0];
	old_state text;
	new_state text := to_jsonb(NEW) ->> TG_ARGV[1];
BEGIN
	IF current_user <> 'crm_app' THEN
		RETURN NEW;
	END IF;
	IF TG_OP = 'INSERT' THEN
		IF NOT EXISTS (SELECT 1 FROM lifecycle_edges e
				WHERE e.machine = m_name AND e.source = '' AND e.target = new_state) THEN
			RAISE EXCEPTION 'lifecycle: a % may not be created in state %', m_name, new_state;
		END IF;
		RETURN NEW;
	END IF;
	old_state := to_jsonb(OLD) ->> TG_ARGV[1];
	IF old_state IS NOT DISTINCT FROM new_state THEN
		RETURN NEW;
	END IF;
	IF coalesce(current_setting('crm.transition', true), '') = '' THEN
		RAISE EXCEPTION 'lifecycle: % % -> % outside the transition path', m_name, old_state, new_state;
	END IF;
	IF NOT EXISTS (SELECT 1 FROM lifecycle_edges e
			WHERE e.machine = m_name AND e.source = old_state AND e.target = new_state) THEN
		RAISE EXCEPTION 'lifecycle: % % -> % is not an edge of the machine', m_name, old_state, new_state;
	END IF;
	RETURN NEW;
END $$;
CREATE TRIGGER deals_lifecycle BEFORE INSERT OR UPDATE OF stage ON deals
	FOR EACH ROW EXECUTE FUNCTION lifecycle_guard('deal', 'stage');
CREATE TRIGGER tasks_lifecycle BEFORE INSERT OR UPDATE OF status ON tasks
	FOR EACH ROW EXECUTE FUNCTION lifecycle_guard('task', 'status');
