alter table post rename to tmp_post;

create table post (
    id integer primary key not null,
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
