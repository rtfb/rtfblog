-- +goose Up
alter table tag rename to tmp_tag;

create table tag (
    id integer primary key not null,
    tag text
);

insert into tag(id, tag)
select id, url
from tmp_tag;

drop table tmp_tag;

-- +goose Down
alter table tag rename to tmp_tag;

create table tag (
    id integer primary key not null,
    name text,
    url text
);

insert into tag(id, name, url)
select id, tag, tag
from tmp_tag;

drop table tmp_tag;
