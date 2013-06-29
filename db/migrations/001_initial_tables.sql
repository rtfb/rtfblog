-- +goose Up
create table tag (
    id integer not null primary key,
    name text,
    url text
);
create table author (
    id integer not null primary key,
    disp_name text,
    salt text,
    passwd text,
    full_name text,
    email text,
    www text
);
create table post (
    id integer not null primary key,
    author_id integer not null references author(id) on delete cascade on update cascade,
    title text,
    date long,
    url text,
    body text
);
create table tagmap (
    id integer not null primary key,
    tag_id integer not null references tag(id) on delete cascade on update cascade,
    post_id integer not null references post(id) on delete cascade on update cascade
);
create table commenter (
    id integer not null primary key,
    name text,
    email text,
    www text,
    ip text
);
create table comment (
    id integer not null primary key,
    commenter_id integer not null references commenter(id) on delete cascade on update cascade,
    post_id integer not null references post(id) on delete cascade on update cascade,
    timestamp long,
    body text
);

-- +goose Down
drop table tag;
drop table author;
drop table post;
drop table tagmap;
drop table commenter;
drop table comment;
