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
