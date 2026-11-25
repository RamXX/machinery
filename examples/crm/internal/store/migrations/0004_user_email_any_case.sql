-- No two users share an email in any letter case (inv:user-email-unique).
CREATE UNIQUE INDEX users_email_lower ON users (lower(email));
