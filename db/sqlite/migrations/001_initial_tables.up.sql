create table tag (
    id integer primary key not null,
    name text,
    url text
);
create table author (
    id integer primary key not null,
    disp_name text,
    salt text,
    passwd text,
    full_name text,
    email text,
    www text
);
create table post (
    id integer primary key not null,
    author_id integer not null references author(id) on delete cascade on update cascade,
    title text,
    date bigint,
    url text,
    body text
);
create table tagmap (
    id integer primary key not null,
    tag_id integer not null references tag(id) on delete cascade on update cascade,
    post_id integer not null references post(id) on delete cascade on update cascade
);
create table commenter (
    id integer primary key not null,
    name text,
    email text,
    www text,
    ip text
);
create table comment (
    id integer primary key not null,
    commenter_id integer not null references commenter(id) on delete cascade on update cascade,
    post_id integer not null references post(id) on delete cascade on update cascade,
    timestamp bigint,
    body text
);
