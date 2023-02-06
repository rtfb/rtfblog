alter table post add column hidden integer;

update post set hidden=0;

alter table post rename to tmp_post;

create table post (
    id integer primary key not null,
    author_id integer not null references author(id) on delete cascade on update cascade,
    title text,
    date bigint,
    url text,
    body text,
    hidden integer not null
);

insert into post(id, author_id, title, date, url, body, hidden)
select id, author_id, title, date, url, body, hidden
from tmp_post;

drop table tmp_post;
