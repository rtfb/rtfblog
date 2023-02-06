alter table author add column salt text;
insert into author(salt) values('');
