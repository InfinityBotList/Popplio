alter table poppypaw rename to user_notifications;
CREATE VIEW poppypaw AS SELECT * FROM user_notifications;

-- TODO, after migration remove poppypaw view