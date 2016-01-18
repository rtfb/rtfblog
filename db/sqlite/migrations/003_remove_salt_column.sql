-- +goose Up
alter table author rename to tmp_author;

create table author (
    id integer primary key not null,
    disp_name text,
    passwd text,
    full_name text,
    email text,
    www text
);

insert into author(id, disp_name, passwd, full_name, email, www)
select id, disp_name, passwd, full_name, email, www
from tmp_author;

drop table tmp_author;

-- +goose Down
alter table author add column salt text;
insert into author(salt) values('');
