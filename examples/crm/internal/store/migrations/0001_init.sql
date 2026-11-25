-- The role the service connects as. Migrations run as the admin role.
CREATE ROLE crm_app LOGIN PASSWORD 'crm_app' NOSUPERUSER NOCREATEDB NOCREATEROLE;
GRANT CONNECT ON DATABASE crm TO crm_app;
