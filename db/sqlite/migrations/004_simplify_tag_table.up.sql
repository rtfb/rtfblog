alter table tag rename to tmp_tag;

create table tag (
    id integer primary key not null,
    tag text
);

insert into tag(id, tag)
select id, url
from tmp_tag;

drop table tmp_tag;
