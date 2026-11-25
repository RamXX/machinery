-- The record of a won deal, written by the recordWin actor inside the
-- transaction of the win. The service may read and insert it.
CREATE TABLE deal_wins (
	deal_id      uuid PRIMARY KEY REFERENCES deals (id),
	amount_cents bigint NOT NULL CHECK (amount_cents > 0),
	won_at       timestamptz NOT NULL DEFAULT now()
);
GRANT SELECT, INSERT ON deal_wins TO crm_app;
