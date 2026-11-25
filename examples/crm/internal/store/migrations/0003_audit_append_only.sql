-- The audit trail is append-only for every role, the admin role included:
-- UPDATE, DELETE and TRUNCATE raise an exception.
CREATE FUNCTION audit_entries_append_only() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
	RAISE EXCEPTION 'audit_entries is append-only: % is not allowed', TG_OP;
END $$;
CREATE TRIGGER audit_entries_no_change BEFORE UPDATE OR DELETE ON audit_entries
	FOR EACH ROW EXECUTE FUNCTION audit_entries_append_only();
CREATE TRIGGER audit_entries_no_truncate BEFORE TRUNCATE ON audit_entries
	FOR EACH STATEMENT EXECUTE FUNCTION audit_entries_append_only();
