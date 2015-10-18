-- +goose Up
alter table post add column hidden boolean;

update post set hidden=0;

alter table post rename to tmp_post;

create table post (
    id serial primary key,
    author_id integer not null references author(id) on delete cascade on update cascade,
    title text,
    date bigint,
    url text,
    body text,
    hidden boolean not null
);

insert into post(id, author_id, title, date, url, body, hidden)
select id, author_id, title, date, url, body, hidden
from tmp_post;

drop table tmp_post;

-- +goose Down
alter table post rename to tmp_post;

create table post (
    id serial primary key,
    author_id integer not null references author(id) on delete cascade on update cascade,
    title text,
    date bigint,
    url text,
    body text
);

insert into post(id, author_id, title, date, url, body)
select id, author_id, title, date, url, body
from tmp_post;

drop table tmp_post;
