alter table post add column hidden boolean;

update post set hidden=FALSE;

alter table post alter column hidden set not null;
