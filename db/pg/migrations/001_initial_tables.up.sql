create table tag (
    id serial primary key,
    name text,
    url text
);
create table author (
    id serial primary key,
    disp_name text,
    salt text,
    passwd text,
    full_name text,
    email text,
    www text
);
create table post (
    id serial primary key,
    author_id integer not null references author(id) on delete cascade on update cascade,
    title text,
    date bigint,
    url text,
    body text
);
create table tagmap (
    id serial primary key,
    tag_id integer not null references tag(id) on delete cascade on update cascade,
    post_id integer not null references post(id) on delete cascade on update cascade
);
create table commenter (
    id serial primary key,
    name text,
    email text,
    www text,
    ip text
);
create table comment (
    id serial primary key,
    commenter_id integer not null references commenter(id) on delete cascade on update cascade,
    post_id integer not null references post(id) on delete cascade on update cascade,
    timestamp bigint,
    body text
);
