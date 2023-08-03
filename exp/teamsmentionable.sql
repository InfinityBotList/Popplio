alter table team_members add column itag UUID PRIMARY KEY DEFAULT uuid_generate_v4();

alter table team_members add column mentionable boolean not null default true;