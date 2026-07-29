alter table bots add column self_status text;

alter table bots add constraint bots_self_status_check check (self_status is null or self_status in ('online', 'idle', 'dnd', 'offline'));
